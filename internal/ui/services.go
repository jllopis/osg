package ui

import (
	"context"
	"fmt"
	"io"
	"os"
	"strings"
	"sync"
	"time"
)

// ServiceState describes the lifecycle state of a managed service.
type ServiceState string

const (
	StateIdle     ServiceState = "idle"
	StateStarting ServiceState = "starting"
	StateRunning  ServiceState = "running"
	StateStopping ServiceState = "stopping"
	StateError    ServiceState = "error"
)

// ServiceRunner is the function the supervisor calls in a goroutine to
// actually run the service. It must respect ctx for shutdown and write
// human-readable log output to logW.
type ServiceRunner func(ctx context.Context, logW io.Writer) error

// ServiceMeta describes a service to the supervisor up-front.
type ServiceMeta struct {
	Name        string
	Description string
	Addr        string
	Runner      ServiceRunner
}

// Service is a snapshot of a managed service for templates.
type Service struct {
	Name        string
	Description string
	Addr        string
	State       ServiceState
	StartedAt   time.Time
	LastError   string
	LogTail     []string

	// Internal lifecycle.
	runner ServiceRunner
	cancel context.CancelFunc
	done   chan struct{}
	logs   *ringBuffer
}

// Supervisor manages a fixed set of services (serve, api) as goroutines.
// It captures their log output into per-service ring buffers and exposes
// start/stop/status operations.
type Supervisor struct {
	mu       sync.Mutex
	services map[string]*Service
	order    []string // stable display order
}

// NewSupervisor creates a supervisor with the given service definitions.
// Services start in the idle state.
func NewSupervisor(metas []ServiceMeta) *Supervisor {
	s := &Supervisor{services: make(map[string]*Service, len(metas))}
	for _, m := range metas {
		s.services[m.Name] = &Service{
			Name:        m.Name,
			Description: m.Description,
			Addr:        m.Addr,
			State:       StateIdle,
			runner:      m.Runner,
			logs:        newRingBuffer(500),
		}
		s.order = append(s.order, m.Name)
	}
	return s
}

// Snapshot returns a copy of all services suitable for rendering. The
// LogTail field on each service contains the most recent log lines.
func (s *Supervisor) Snapshot() []Service {
	s.mu.Lock()
	defer s.mu.Unlock()
	out := make([]Service, 0, len(s.services))
	for _, name := range s.order {
		svc, ok := s.services[name]
		if !ok {
			continue
		}
		// Shallow copy + a snapshot of the log tail.
		copy := *svc
		copy.LogTail = svc.logs.Tail(20)
		out = append(out, copy)
	}
	return out
}

// Start launches the named service in a new goroutine. Returns an error
// immediately if the service is unknown, already running, or fails within
// a short startup window.
func (s *Supervisor) Start(name string) error {
	s.mu.Lock()
	svc, ok := s.services[name]
	if !ok {
		s.mu.Unlock()
		return fmt.Errorf("unknown service: %s", name)
	}
	if svc.State == StateRunning || svc.State == StateStarting {
		s.mu.Unlock()
		return fmt.Errorf("service %q already %s", name, svc.State)
	}

	ctx, cancel := context.WithCancel(context.Background())
	svc.cancel = cancel
	svc.done = make(chan struct{})
	svc.State = StateStarting
	svc.StartedAt = time.Now()
	svc.LastError = ""
	svc.logs.Reset()
	runner := svc.runner
	logs := svc.logs
	s.mu.Unlock()

	errCh := make(chan error, 1)
	go func() {
		defer close(svc.done)
		// Tee logs to stderr so the operator running the dashboard still
		// sees output, plus the ring buffer for the UI.
		err := runner(ctx, io.MultiWriter(os.Stderr, logs))
		errCh <- err
		s.mu.Lock()
		defer s.mu.Unlock()
		if err != nil {
			svc.LastError = err.Error()
			svc.State = StateError
		} else {
			svc.State = StateIdle
		}
	}()

	// Wait briefly for an immediate failure (e.g. port in use). If the
	// runner is still going after the window, mark it running.
	select {
	case err := <-errCh:
		if err != nil {
			return err
		}
		return nil
	case <-time.After(200 * time.Millisecond):
		s.mu.Lock()
		if svc.State == StateStarting {
			svc.State = StateRunning
		}
		s.mu.Unlock()
		return nil
	}
}

// Stop cancels the named service's context and waits for it to terminate
// (up to 5s). Returns an error if the service is unknown or already idle.
func (s *Supervisor) Stop(name string) error {
	s.mu.Lock()
	svc, ok := s.services[name]
	if !ok {
		s.mu.Unlock()
		return fmt.Errorf("unknown service: %s", name)
	}
	if svc.State != StateRunning && svc.State != StateStarting {
		s.mu.Unlock()
		return fmt.Errorf("service %q is not running", name)
	}
	cancel := svc.cancel
	done := svc.done
	svc.State = StateStopping
	s.mu.Unlock()

	if cancel != nil {
		cancel()
	}
	if done != nil {
		select {
		case <-done:
		case <-time.After(5 * time.Second):
			return fmt.Errorf("service %q did not stop within 5s", name)
		}
	}
	return nil
}

// Logs returns a subscription channel for the named service's log
// stream. The current buffered content is delivered first, then live
// updates flow until ctx is cancelled. Returns nil if name is unknown.
func (s *Supervisor) Logs(ctx context.Context, name string) <-chan string {
	s.mu.Lock()
	svc, ok := s.services[name]
	s.mu.Unlock()
	if !ok {
		return nil
	}
	return svc.logs.Subscribe(ctx)
}

// StopAll stops every running service and returns when they have all
// terminated (or the per-service timeout fires).
func (s *Supervisor) StopAll() {
	s.mu.Lock()
	names := make([]string, 0, len(s.services))
	for name, svc := range s.services {
		if svc.State == StateRunning || svc.State == StateStarting {
			names = append(names, name)
		}
	}
	s.mu.Unlock()
	for _, name := range names {
		_ = s.Stop(name)
	}
}

// ringBuffer is a fixed-capacity, line-oriented buffer that satisfies
// io.Writer. Each Write call is treated as one log entry; trailing
// newlines are stripped. Subscribers receive new entries via Subscribe.
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
	return &ringBuffer{
		lines: make([]string, capacity),
		cap:   capacity,
	}
}

func (r *ringBuffer) Write(p []byte) (int, error) {
	r.mu.Lock()
	line := strings.TrimRight(string(p), "\n")
	if line == "" {
		r.mu.Unlock()
		return len(p), nil
	}
	r.lines[r.next] = line
	r.next = (r.next + 1) % r.cap
	if r.next == 0 {
		r.full = true
	}
	subs := append([]chan string(nil), r.subs...)
	r.mu.Unlock()
	for _, ch := range subs {
		// Non-blocking send: a slow subscriber drops messages rather
		// than blocking the writer (real OSG service goroutine).
		select {
		case ch <- line:
		default:
		}
	}
	return len(p), nil
}

// Subscribe returns a channel that receives new log lines until ctx is
// cancelled. The current buffered content is delivered first so the
// subscriber sees the recent history before live updates.
func (r *ringBuffer) Subscribe(ctx context.Context) <-chan string {
	ch := make(chan string, 64)

	r.mu.Lock()
	// Snapshot current buffer first.
	current := r.tailLocked(r.cap)
	r.subs = append(r.subs, ch)
	r.mu.Unlock()

	go func() {
		// Replay history (non-blocking; if the consumer is slow, drop).
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

// tailLocked is the lock-held implementation of Tail. Caller must hold r.mu.
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
		start := r.next - n
		if start < 0 {
			start = 0
		}
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

// Reset empties the buffer.
func (r *ringBuffer) Reset() {
	r.mu.Lock()
	defer r.mu.Unlock()
	for i := range r.lines {
		r.lines[i] = ""
	}
	r.next = 0
	r.full = false
}
