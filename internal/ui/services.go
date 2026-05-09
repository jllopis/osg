package ui

import (
	"time"

	"osg/internal/operations"
)

// ServiceState is an alias of operations.State so the existing services.html
// template can keep using its current field names. New code should refer to
// operations.State directly.
type ServiceState = operations.State

const (
	StateIdle      = operations.StateIdle
	StateStarting  = operations.StateStarting
	StateRunning   = operations.StateRunning
	StateStopping  = operations.StateStopping
	StateError     = operations.StateError
	StateCancelled = operations.StateCancelled
)

// Service is the template-facing view of a managed service. Built from
// an operations.Snapshot in serviceFromSnapshot below.
type Service struct {
	Name        string
	Description string
	Addr        string
	State       ServiceState
	StartedAt   time.Time
	LastError   string
	LogTail     []string
}

func serviceFromSnapshot(snap operations.Snapshot) Service {
	s := Service{
		Name:        snap.Definition.Name,
		Description: snap.Definition.Description,
		Addr:        snap.Definition.Addr,
		State:       snap.State,
		LogTail:     snap.LogTail,
	}
	if snap.Active != nil {
		s.StartedAt = snap.Active.StartedAt
		s.LastError = snap.Active.LastError
	} else if snap.LastRun != nil {
		s.LastError = snap.LastRun.Error
	}
	return s
}

// servicesFromRunner returns Service views for every service-kind
// definition the runner manages, in the runner's stable display order.
func servicesFromRunner(r *operations.Runner) []Service {
	out := []Service{}
	if r == nil {
		return out
	}
	for _, snap := range r.Snapshot() {
		if snap.Definition.Kind != operations.KindService {
			continue
		}
		out = append(out, serviceFromSnapshot(snap))
	}
	return out
}
