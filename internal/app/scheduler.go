package app

import (
	"context"
	"time"

	"osg/internal/build"
	"osg/internal/config"
	"osg/internal/logging"
	"osg/internal/operations"
	"osg/internal/publish"
)

// schedulerFallback is the maximum sleep between checks when no
// PublishAt date is upcoming. Set conservatively: longer means a freshly
// added publish_at within a closer window may be missed for up to this
// duration after editing the file (the watcher service is the right
// answer for that — it triggers a rebuild on file change anyway).
const schedulerFallback = 5 * time.Minute

// schedulerMinRetry bounds how often the publish flow re-runs for the same
// overdue publish_at. Without it, a draft that is due but cannot be promoted
// (e.g. an unwritable vault file) leaves NextScheduled in the past, making
// time.Until(next) zero and spinning the loop — full ComputeStats +
// update-content + build back-to-back with no pause. The backoff only applies
// when the same due time recurs, so a successful promotion still triggers
// promptly.
const schedulerMinRetry = 30 * time.Second

// schedulerStore is the subset of operations.Store the scheduler needs
// to record build triggers. Accepting an interface keeps tests light
// (nil is allowed and simply skips audit) and avoids a hard dependency
// on the SQLite dialect from this package.
type schedulerStore interface {
	Begin(name, kind string, params map[string]any, started time.Time) (int64, error)
	Finish(id int64, status, errMsg string, ended time.Time) error
}

// RunScheduler watches for posts with a future publish_at frontmatter
// field and triggers a rebuild when each one becomes due. The function
// blocks until ctx is cancelled.
//
// It is intentionally simple: it scans the site after every rebuild,
// finds the earliest future PublishAt across all pages, and sleeps
// until that moment (or until schedulerFallback elapses, whichever is
// sooner). When the timer fires it runs RunBuild and re-scans.
//
// store may be nil; in that case scheduler triggers are not persisted
// to the audit log.
func RunScheduler(ctx context.Context, opts CLIOptions, store schedulerStore) error {
	logger := logging.NewWithWriter(loadLoggingCfg(opts.ConfigPath), opts.Verbose, opts.LogWriter)
	logger.Info("scheduler starting")

	// lastDue is the publish_at we last triggered the flow for. If the next
	// scan still reports the same overdue time, promotion isn't clearing it,
	// so we back off instead of busy-looping.
	var lastDue time.Time

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
				sleepFor = max(time.Until(next), 0)
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

		// Only run the publish flow if we actually woke up because a
		// scheduled post is now due. Periodic wake-ups (no upcoming
		// next) skip everything — the watcher service is the right
		// way to pick up content changes.
		if !next.IsZero() && !time.Now().Before(next) {
			// Same overdue due time as last trigger: promotion didn't clear
			// it, so pause before retrying rather than spinning.
			if next.Equal(lastDue) {
				logger.Warn("scheduler publish_at still due after previous trigger; backing off",
					"due_at", next.Format(time.RFC3339),
					"retry_in", schedulerMinRetry.String(),
				)
				if !sleep(ctx, schedulerMinRetry) {
					return nil
				}
			}
			lastDue = next

			logger.Info("scheduler triggering publish flow",
				"due_at", next.Format(time.RFC3339),
			)
			startedAt := time.Now()
			var auditID int64
			if store != nil {
				params := map[string]any{"due_at": next.Format(time.RFC3339Nano)}
				id, err := store.Begin("scheduler:trigger", operations.KindTask, params, startedAt)
				if err != nil {
					logger.Warn("scheduler audit begin failed", "error", err)
				} else {
					auditID = id
				}
			}

			o := opts
			o.SkipAI = true
			status := operations.StatusOK
			errMsg := ""

			// 1. Promote drafts whose publish_at has just arrived: the
			//    vault file's osg.publish flips draft → true, atomically.
			vaultPath, vErr := config.ResolveVaultPath(cfg)
			if vErr != nil {
				logger.Warn("scheduler vault resolve failed", "error", vErr)
			} else if promoted, pErr := publish.PromoteDueDrafts(vaultPath, time.Now(), logger); pErr != nil {
				logger.Warn("scheduler promote failed", "error", pErr)
			} else if len(promoted) > 0 {
				logger.Info("scheduler promoted drafts", "count", len(promoted), "paths", promoted)
			}

			// 2. Sync the (possibly rewritten) vault into content/ so
			//    the build sees the new state.
			if err := RunUpdateContent(ctx, o); err != nil {
				logger.Warn("scheduler update-content failed", "error", err)
				status = operations.StatusError
				errMsg = err.Error()
			}

			// 3. Build. Drafts are still drafts (filtered out); newly-
			//    promoted posts are now non-draft and past-dated, so
			//    they reach public/ and the next deploy.
			if status == operations.StatusOK {
				if err := RunBuild(ctx, o); err != nil {
					logger.Warn("scheduler build failed", "error", err)
					status = operations.StatusError
					errMsg = err.Error()
				}
			}
			if store != nil && auditID > 0 {
				if err := store.Finish(auditID, status, errMsg, time.Now()); err != nil {
					logger.Warn("scheduler audit finish failed", "error", err)
				}
			}
		}
	}
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
