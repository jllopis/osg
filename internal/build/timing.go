package build

import (
	"encoding/json"
	"fmt"
	"log/slog"
	"os"
	"path/filepath"
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

// WriteJSON writes the timing data as JSON to the given file path.
func (bt *BuildTimings) WriteJSON(path string) error {
	type stageJSON struct {
		Name string  `json:"name"`
		Ms   float64 `json:"ms"`
	}
	type timingJSON struct {
		TotalMs float64     `json:"total_ms"`
		Stages  []stageJSON `json:"stages"`
	}

	data := timingJSON{
		TotalMs: float64(bt.Total.Milliseconds()),
	}
	for _, s := range bt.Stages {
		data.Stages = append(data.Stages, stageJSON{
			Name: s.Name,
			Ms:   float64(s.Duration.Milliseconds()),
		})
	}

	b, err := json.MarshalIndent(data, "", "  ")
	if err != nil {
		return err
	}
	if dir := filepath.Dir(path); dir != "." {
		if err := os.MkdirAll(dir, 0o755); err != nil {
			return err
		}
	}
	return os.WriteFile(path, b, 0o644)
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

const buildHistoryPath = ".osg/build-history.json"
const maxHistoryEntries = 100

// buildHistoryEntry records one build run.
type buildHistoryEntry struct {
	Timestamp string             `json:"timestamp"`
	TotalMs   float64            `json:"total_ms"`
	Rendered  int                `json:"rendered"`
	Cached    int                `json:"cached"`
	Errors    int                `json:"errors"`
	Stages    map[string]float64 `json:"stages"`
}

// appendBuildHistory adds a build entry to the persistent history file.
func appendBuildHistory(timings *BuildTimings, stats Stats) {
	// Load existing history.
	var history []buildHistoryEntry
	if data, err := os.ReadFile(buildHistoryPath); err == nil {
		_ = json.Unmarshal(data, &history)
	}

	stages := map[string]float64{}
	for _, s := range timings.Stages {
		stages[s.Name] = float64(s.Duration.Milliseconds())
	}

	entry := buildHistoryEntry{
		Timestamp: time.Now().Format(time.RFC3339),
		TotalMs:   float64(timings.Total.Milliseconds()),
		Rendered:  stats.Rendered,
		Cached:    stats.Cached,
		Errors:    stats.Errors,
		Stages:    stages,
	}
	history = append(history, entry)

	// Keep only the last N entries.
	if len(history) > maxHistoryEntries {
		history = history[len(history)-maxHistoryEntries:]
	}

	dir := filepath.Dir(buildHistoryPath)
	if err := os.MkdirAll(dir, 0o755); err != nil {
		return
	}
	data, err := json.MarshalIndent(history, "", "  ")
	if err != nil {
		return
	}
	_ = os.WriteFile(buildHistoryPath, data, 0o644)
}
