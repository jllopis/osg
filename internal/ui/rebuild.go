package ui

import (
	"context"
	"errors"
	"sync"
	"time"
)

// rebuildState tracks ad-hoc on-demand rebuilds triggered from the UI's
// "Rebuild now" button. Only one rebuild runs at a time; concurrent
// triggers return a busy error.
type rebuildState struct {
	fn func(ctx context.Context) error

	mu       sync.Mutex
	running  bool
	lastRan  time.Time
	lastErr  string
	duration time.Duration
}

// RebuildSnapshot is the read-side view of the rebuilder for templates.
type RebuildSnapshot struct {
	Available bool
	Running   bool
	LastRan   time.Time
	LastError string
	Duration  time.Duration
}

func newRebuildState(fn func(ctx context.Context) error) *rebuildState {
	return &rebuildState{fn: fn}
}

// Trigger starts a rebuild in the background and returns immediately.
// Returns an error if a rebuild is already in flight or no build
// function was configured.
func (r *rebuildState) Trigger() error {
	if r == nil || r.fn == nil {
		return errors.New("rebuild not available")
	}
	r.mu.Lock()
	if r.running {
		r.mu.Unlock()
		return errors.New("rebuild already in progress")
	}
	r.running = true
	r.mu.Unlock()

	go func() {
		started := time.Now()
		err := r.fn(context.Background())
		r.mu.Lock()
		r.running = false
		r.lastRan = time.Now()
		r.duration = r.lastRan.Sub(started)
		if err != nil {
			r.lastErr = err.Error()
		} else {
			r.lastErr = ""
		}
		r.mu.Unlock()
	}()
	return nil
}

func (r *rebuildState) Snapshot() RebuildSnapshot {
	if r == nil {
		return RebuildSnapshot{}
	}
	r.mu.Lock()
	defer r.mu.Unlock()
	return RebuildSnapshot{
		Available: r.fn != nil,
		Running:   r.running,
		LastRan:   r.lastRan,
		LastError: r.lastErr,
		Duration:  r.duration,
	}
}
