package app

import (
	"context"
	"path/filepath"
	"time"

	"osg/internal/build"
	"osg/internal/config"
	"osg/internal/logging"
	"osg/internal/scheduler"
)

// schedulerFallback is the maximum sleep between checks when no
// PublishAt date is upcoming. Set conservatively: longer means a freshly
// added publish_at within a closer window may be missed for up to this
// duration after editing the file (the watcher service is the right
// answer for that — it triggers a rebuild on file change anyway).
const schedulerFallback = 5 * time.Minute

// RunScheduler watches for posts with a future publish_at frontmatter
// field and triggers a rebuild when each one becomes due. The function
// blocks until ctx is cancelled.
//
// It is intentionally simple: it scans the site after every rebuild,
// finds the earliest future PublishAt across all pages, and sleeps
// until that moment (or until schedulerFallback elapses, whichever is
// sooner). When the timer fires it runs RunBuild and re-scans.
func RunScheduler(ctx context.Context, opts CLIOptions) error {
	logger := logging.NewWithWriter(loadLoggingCfg(opts.ConfigPath), opts.Verbose, opts.LogWriter)
	logger.Info("scheduler starting")

	store, err := scheduler.NewStore(SchedulerDBPath(opts.ConfigPath))
	if err != nil {
		// Non-fatal: scheduler still works without an audit trail.
		logger.Warn("scheduler audit store unavailable", "error", err)
	}
	if store != nil {
		defer func() { _ = store.Close() }()
	}

	for {
		cfg, err := config.Load(opts.ConfigPath)
		if err != nil {
			logger.Warn("scheduler config load failed", "error", err)
			if !sleep(ctx, schedulerFallback) {
				return nil
			}
			continue
		}

		stats, err := build.ComputeStats(cfg)
		var sleepFor time.Duration
		next := time.Time{}
		if err != nil {
			logger.Warn("scheduler stats failed", "error", err)
			sleepFor = schedulerFallback
		} else {
			next = stats.NextScheduled
			if next.IsZero() {
				logger.Info("scheduler idle (no upcoming publish_at)", "fallback", schedulerFallback)
				sleepFor = schedulerFallback
			} else {
				sleepFor = time.Until(next)
				if sleepFor < 0 {
					sleepFor = 0
				}
				if sleepFor > schedulerFallback {
					sleepFor = schedulerFallback
				}
				logger.Info("scheduler waiting",
					"next_publish_at", next.Format(time.RFC3339),
					"in", sleepFor.Round(time.Second).String(),
				)
			}
		}

		if !sleep(ctx, sleepFor) {
			return nil
		}

		// Only run a build if we actually woke up because a scheduled
		// post is now due. Periodic wake-ups (no upcoming next) skip
		// the rebuild — the watcher service is the right way to pick up
		// content changes.
		if !next.IsZero() && !time.Now().Before(next) {
			logger.Info("scheduler triggering build",
				"due_at", next.Format(time.RFC3339),
			)
			o := opts
			o.SkipAI = true
			run := scheduler.Run{DueAt: next, RanAt: time.Now(), Status: "ok"}
			if err := RunBuild(ctx, o); err != nil {
				logger.Warn("scheduler build failed", "error", err)
				run.Status = "error"
				run.Error = err.Error()
			}
			run.RanAt = time.Now()
			if store != nil {
				if recErr := store.Record(run); recErr != nil {
					logger.Warn("scheduler audit record failed", "error", recErr)
				}
			}
		}
	}
}

// SchedulerDBPath returns the path of the scheduler audit database. It
// lives next to other OSG state under .osg/, derived from the config
// path's directory.
func SchedulerDBPath(configPath string) string {
	dir := filepath.Dir(configPath)
	if dir == "" || dir == "." {
		return ".osg/scheduler.db"
	}
	return filepath.Join(dir, ".osg", "scheduler.db")
}

// sleep blocks for d or until ctx is cancelled. Returns false if the
// context fired (caller should exit).
func sleep(ctx context.Context, d time.Duration) bool {
	if d <= 0 {
		return ctx.Err() == nil
	}
	t := time.NewTimer(d)
	defer t.Stop()
	select {
	case <-t.C:
		return true
	case <-ctx.Done():
		return false
	}
}

// loadLoggingCfg loads only the logging section of the config. Used by
// services that want a logger before doing their main work and treat
// config errors non-fatally.
func loadLoggingCfg(path string) config.LoggingConfig {
	cfg, err := config.Load(path)
	if err != nil {
		return config.Default().Logging
	}
	return cfg.Logging
}
