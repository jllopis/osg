package operations

import (
	"context"
	"fmt"
	"io"
	"os"
	"strings"
	"sync"
	"time"
)

// State is the in-memory lifecycle state of an operation. It is wider
// than the persisted Status because it includes transitional values
// (starting/stopping) that are useful for the UI but not worth recording
// per-run.
type State string

const (
	StateIdle      State = "idle"
	StateStarting  State = "starting"
	StateRunning   State = "running"
	StateStopping  State = "stopping"
	StateError     State = "error"
	StateCancelled State = "cancelled"
)

// RunFunc is the body of an operation. It must respect ctx for
// cancellation and write log output to logW.
type RunFunc func(ctx context.Context, params map[string]any, logW io.Writer) error

// Definition is a static description of a runnable operation. The
// Runner is constructed with a list of these and never adds new ones at
// runtime.
type Definition struct {
	Name        string  // unique
	Kind        string  // KindService or KindTask
	Description string  // shown on the cards
	Addr        string  // optional, e.g. ":1313" for serve
	Run         RunFunc // function the runner invokes
}

// Run is one execution of an operation.
type Run struct {
	ID        int64  // assigned by the Store
	Name      string // back-reference to Definition.Name
	Kind      string
	State     State
	StartedAt time.Time
	EndedAt   time.Time
	LastError string
	Params    map[string]any

	// Internals.
	cancel context.CancelFunc
	done   chan struct{}
	logs   *ringBuffer
}

// Snapshot is the per-definition view used by templates. State and
// LastRun help callers render either "idle / last ok 2m ago" or
// "running 12s" without scanning the full history.
type Snapshot struct {
	Definition Definition
	State      State
	Active     *Run        // nil when no run is in flight
	LastRun    *HistoryRun // most recent finished run, if any
	// LogTail carries recent log lines: from the active run while one is
	// in flight, otherwise from the most recently finished run captured
	// in memory (logs are not persisted to disk, so they survive only
	// across other runs of the same dashboard process).
	LogTail []string
}

// Runner manages a fixed set of Definitions, enforces one-per-name
// concurrency, captures logs into per-run ring buffers, and persists
// every Trigger to the Store for later inspection.
type Runner struct {
	mu     sync.Mutex
	defs   map[string]Definition
	order  []string
	active map[string]*Run
	// lastLogs keeps the tail of the most recently finished run per
	// operation name so the drawer can show "what just happened" after
	// the run leaves r.active. Replaced on each finish.
	lastLogs map[string][]string
	store    *Store
}

// New creates a runner with the given definitions and a backing store.
// store may be nil for tests; in that case runs simply aren't persisted.
func New(defs []Definition, store *Store) *Runner {
	r := &Runner{
		defs:     make(map[string]Definition, len(defs)),
		active:   make(map[string]*Run, len(defs)),
		lastLogs: make(map[string][]string, len(defs)),
		store:    store,
	}
	for _, d := range defs {
		if d.Kind == "" {
			d.Kind = KindTask
		}
		r.defs[d.Name] = d
		r.order = append(r.order, d.Name)
	}
	return r
}

// Trigger starts a new run for the named operation. Returns an error
// when the name is unknown or another run with the same name is in
// flight (concurrency by command type).
func (r *Runner) Trigger(name string, params map[string]any) (*Run, error) {
	r.mu.Lock()
	def, ok := r.defs[name]
	if !ok {
		r.mu.Unlock()
		return nil, fmt.Errorf("unknown operation: %s", name)
	}
	if existing, busy := r.active[name]; busy {
		r.mu.Unlock()
		return nil, fmt.Errorf("operation %q already running (started %s)",
			name, existing.StartedAt.Format(time.RFC3339))
	}

	startedAt := time.Now()
	run := &Run{
		Name:      name,
		Kind:      def.Kind,
		State:     StateStarting,
		StartedAt: startedAt,
		Params:    params,
		done:      make(chan struct{}),
		logs:      newRingBuffer(500),
	}
	if r.store != nil {
		if id, err := r.store.Begin(name, def.Kind, params, startedAt); err == nil {
			run.ID = id
		}
	}
	ctx, cancel := context.WithCancel(context.Background())
	run.cancel = cancel
	r.active[name] = run
	r.mu.Unlock()

	errCh := make(chan error, 1)
	go func() {
		defer close(run.done)
		// Tee logs to stderr so the operator running the dashboard still
		// sees output, plus the ring buffer for the UI.
		err := def.Run(ctx, params, io.MultiWriter(os.Stderr, run.logs))
		errCh <- err

		ended := time.Now()
		status := StatusOK
		errMsg := ""
		newState := StateIdle
		if err != nil {
			if ctx.Err() != nil {
				status = StatusCancelled
				errMsg = err.Error()
				newState = StateCancelled
			} else {
				status = StatusError
				errMsg = err.Error()
				newState = StateError
			}
		}

		r.mu.Lock()
		run.State = newState
		run.EndedAt = ended
		run.LastError = errMsg
		r.lastLogs[name] = run.logs.Tail(500)
		delete(r.active, name)
		r.mu.Unlock()

		if r.store != nil && run.ID > 0 {
			_ = r.store.Finish(run.ID, status, errMsg, ended)
		}
	}()

	// Brief startup window so callers can surface immediate failures
	// (e.g. port-already-in-use) synchronously instead of returning a
	// "running" state that flips to "error" milliseconds later.
	select {
	case err := <-errCh:
		if err != nil {
			return nil, err
		}
		return run, nil
	case <-time.After(200 * time.Millisecond):
		r.mu.Lock()
		if run.State == StateStarting {
			run.State = StateRunning
		}
		r.mu.Unlock()
		return run, nil
	}
}

// Stop cancels the active run with the given name. Returns nil if the
// operation is unknown or already idle (idempotent).
func (r *Runner) Stop(name string) error {
	r.mu.Lock()
	run, ok := r.active[name]
	if !ok {
		r.mu.Unlock()
		return nil
	}
	cancel := run.cancel
	done := run.done
	run.State = StateStopping
	r.mu.Unlock()

	if cancel != nil {
		cancel()
	}
	if done != nil {
		select {
		case <-done:
		case <-time.After(5 * time.Second):
			return fmt.Errorf("operation %q did not stop within 5s", name)
		}
	}
	return nil
}

// Restart is a convenience for service-style operations: stops the
// active run (if any) and triggers a fresh one with the same params.
// No-op if the operation is currently idle.
func (r *Runner) Restart(name string) error {
	r.mu.Lock()
	run, busy := r.active[name]
	if !busy {
		r.mu.Unlock()
		return nil
	}
	params := run.Params
	r.mu.Unlock()
	if err := r.Stop(name); err != nil {
		return err
	}
	_, err := r.Trigger(name, params)
	return err
}

// StopAll cancels every active run and waits for them to terminate.
// Persists the cancellation in the store via MarkInterruptedRunning so
// shutdown is visible in /history.
func (r *Runner) StopAll() {
	r.mu.Lock()
	names := make([]string, 0, len(r.active))
	for name := range r.active {
		names = append(names, name)
	}
	r.mu.Unlock()
	for _, name := range names {
		_ = r.Stop(name)
	}
	if r.store != nil {
		_, _ = r.store.MarkInterruptedRunning(time.Now())
	}
}

// Snapshot returns the per-definition view for templates. LastRun is
// the most recent finished run from the store; nil when none exists.
func (r *Runner) Snapshot() []Snapshot {
	r.mu.Lock()
	defer r.mu.Unlock()
	out := make([]Snapshot, 0, len(r.order))
	for _, name := range r.order {
		def := r.defs[name]
		snap := Snapshot{Definition: def, State: StateIdle}
		if active, ok := r.active[name]; ok {
			snap.State = active.State
			activeCopy := *active
			snap.Active = &activeCopy
			snap.LogTail = active.logs.Tail(20)
		} else if tail, ok := r.lastLogs[name]; ok {
			snap.LogTail = tail
		}
		if r.store != nil {
			// Pull a couple of rows so we can skip the currently-running
			// one (already surfaced as snap.Active) and fall back to the
			// previous completed run.
			rows, err := r.store.Recent(Filter{Name: name, Limit: 2})
			if err == nil {
				for _, row := range rows {
					if snap.Active != nil && row.Status == StatusRunning {
						continue
					}
					last := row
					snap.LastRun = &last
					break
				}
			}
		}
		out = append(out, snap)
	}
	return out
}

// Active returns the currently running Run for a name, if any.
func (r *Runner) Active(name string) (*Run, bool) {
	r.mu.Lock()
	defer r.mu.Unlock()
	run, ok := r.active[name]
	return run, ok
}

// Logs returns a channel that receives log lines from the currently
// active run with the given name. The current ring buffer content is
// replayed first, then live updates flow until ctx is cancelled or the
// run ends. Returns nil when there is no active run.
func (r *Runner) Logs(ctx context.Context, name string) <-chan string {
	r.mu.Lock()
	run, ok := r.active[name]
	r.mu.Unlock()
	if !ok {
		return nil
	}
	return run.logs.Subscribe(ctx)
}

// RunFlow triggers the given operations sequentially. Each step waits
// for the previous run to finish; the chain aborts as soon as one
// returns an error or is cancelled. Returns the name of the failed
// step (or empty if the whole chain succeeded).
//
// This is the building block for the "Run from here" affordance on
// /actions: the UI passes a slice starting at the chosen node so the
// downstream pipeline is executed in order.
func (r *Runner) RunFlow(names []string) (string, error) {
	for _, name := range names {
		run, err := r.Trigger(name, nil)
		if err != nil {
			return name, fmt.Errorf("trigger %s: %w", name, err)
		}
		// Block until the run goroutine signals completion. The done
		// channel is closed regardless of success/failure.
		<-run.done
		r.mu.Lock()
		state := run.State
		lastErr := run.LastError
		r.mu.Unlock()
		if state == StateError || state == StateCancelled {
			if lastErr == "" {
				lastErr = string(state)
			}
			return name, fmt.Errorf("%s: %s", name, lastErr)
		}
	}
	return "", nil
}

// History returns audit-log rows from the store with the given filter.
func (r *Runner) History(filter Filter) ([]HistoryRun, error) {
	if r.store == nil {
		return nil, nil
	}
	return r.store.Recent(filter)
}

// Definitions exposes the configured definitions in stable order.
func (r *Runner) Definitions() []Definition {
	r.mu.Lock()
	defer r.mu.Unlock()
	out := make([]Definition, 0, len(r.order))
	for _, name := range r.order {
		out = append(out, r.defs[name])
	}
	return out
}

// ringBuffer is a fixed-capacity, line-oriented buffer that satisfies
// io.Writer. Each Write is one log entry; trailing newlines are
// stripped. Subscribers receive new entries via Subscribe.
type ringBuffer struct {
	mu    sync.Mutex
	lines []string
	cap   int
	next  int
	full  bool
	subs  []chan string
}

func newRingBuffer(capacity int) *ringBuffer {
	if capacity <= 0 {
		capacity = 1
	}
	return &ringBuffer{lines: make([]string, capacity), cap: capacity}
}

func (r *ringBuffer) Write(p []byte) (int, error) {
	r.mu.Lock()
	defer r.mu.Unlock()
	line := strings.TrimRight(string(p), "\n")
	if line == "" {
		return len(p), nil
	}
	r.lines[r.next] = line
	r.next = (r.next + 1) % r.cap
	if r.next == 0 {
		r.full = true
	}
	// Deliver under the lock: Subscribe removes a channel from r.subs before
	// closing it (also under the lock), so holding it here guarantees we never
	// send on a closed channel. Sends are non-blocking, so the lock is brief.
	for _, ch := range r.subs {
		select {
		case ch <- line:
		default:
		}
	}
	return len(p), nil
}

// Subscribe returns a channel that receives new log lines until ctx is
// cancelled. Buffered history is replayed first.
func (r *ringBuffer) Subscribe(ctx context.Context) <-chan string {
	ch := make(chan string, 64)
	r.mu.Lock()
	current := r.tailLocked(r.cap)
	r.subs = append(r.subs, ch)
	r.mu.Unlock()

	go func() {
		for _, line := range current {
			select {
			case ch <- line:
			default:
			}
		}
		<-ctx.Done()
		r.mu.Lock()
		for i, c := range r.subs {
			if c == ch {
				r.subs = append(r.subs[:i], r.subs[i+1:]...)
				break
			}
		}
		r.mu.Unlock()
		close(ch)
	}()
	return ch
}

// Tail returns up to n most recent entries in chronological order.
func (r *ringBuffer) Tail(n int) []string {
	r.mu.Lock()
	defer r.mu.Unlock()
	return r.tailLocked(n)
}

func (r *ringBuffer) tailLocked(n int) []string {
	size := r.next
	if r.full {
		size = r.cap
	}
	if n > size {
		n = size
	}
	out := make([]string, 0, n)
	if !r.full {
		start := max(r.next-n, 0)
		for i := start; i < r.next; i++ {
			out = append(out, r.lines[i])
		}
		return out
	}
	start := (r.next + (r.cap - n)) % r.cap
	for i := 0; i < n; i++ {
		out = append(out, r.lines[(start+i)%r.cap])
	}
	return out
}
