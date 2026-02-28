package build

import (
	"fmt"
	"log/slog"
	"strings"
	"time"
)

// StageTiming records the duration of a single build stage.
type StageTiming struct {
	Name     string
	Duration time.Duration
}

// BuildTimings collects per-stage timings for the entire build pipeline.
type BuildTimings struct {
	Stages []StageTiming
	Total  time.Duration
}

// stage starts a timer and returns a function that records the elapsed time.
// Usage:
//
//	done := timings.stage("assets")
//	// ... do work ...
//	done()
func (bt *BuildTimings) stage(name string) func() {
	start := time.Now()
	return func() {
		bt.Stages = append(bt.Stages, StageTiming{
			Name:     name,
			Duration: time.Since(start),
		})
	}
}

// Log emits a structured log summary of all stage timings.
func (bt *BuildTimings) Log(logger *slog.Logger) {
	if len(bt.Stages) == 0 {
		return
	}

	var parts []string
	for _, s := range bt.Stages {
		parts = append(parts, fmt.Sprintf("%s=%s", s.Name, s.Duration.Round(time.Millisecond)))
	}

	logger.Info("build timing",
		"total", bt.Total.Round(time.Millisecond).String(),
		"stages", strings.Join(parts, " "),
	)
}
