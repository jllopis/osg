package ui

import (
	"time"

	"osg/internal/operations"
)

// RebuildSnapshot is the template-facing summary of the most recent
// build run, surfaced on the /assets page. Built from the runner's
// snapshot of the "build" task.
type RebuildSnapshot struct {
	Available bool
	Running   bool
	LastRan   time.Time
	LastError string
	Duration  time.Duration
}

func rebuildSnapshotFromRunner(r *operations.Runner) RebuildSnapshot {
	if r == nil {
		return RebuildSnapshot{}
	}
	snap := RebuildSnapshot{}
	for _, s := range r.Snapshot() {
		if s.Definition.Name != "build" {
			continue
		}
		snap.Available = true
		if s.State == operations.StateRunning || s.State == operations.StateStarting {
			snap.Running = true
		}
		if s.LastRun != nil {
			snap.LastRan = s.LastRun.EndedAt
			snap.LastError = s.LastRun.Error
			if !s.LastRun.StartedAt.IsZero() && !s.LastRun.EndedAt.IsZero() {
				snap.Duration = s.LastRun.EndedAt.Sub(s.LastRun.StartedAt)
			}
		}
		break
	}
	return snap
}
