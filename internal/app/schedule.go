package app

// The scans nobody asks for: the full rescan a rebuilt derived half owes, and
// the incremental scan of every library on a schedule.
//
// # Why this is here at all, and why it was nearly not written
//
// 003 plan §3 gives `internal/app` *"the `library` subcommand (§6.7), the
// scheduled scan, and the start-time rebuild-and-rescan of §6.8"*; §6.9 spells
// the setting (`--scan-interval`, an environment fallback, a default measured in
// hours, `0` to disable) and §6.8 says a generation bump *"drops, recreates, and
// enqueues a full scan of every library"*. **No task in 003's list owned
// either.** The task list was written from spec §3 and §5, and neither the
// schedule nor the rebuild's rescan has a clause in §3 or an acceptance
// criterion in §5 — spec §2 states the schedule while putting *filesystem
// watching* out of scope, which is the one place in the specification it
// appears. That is how both fell through, and it is worth recording as a shape:
// **a behaviour named only in a scope note is a behaviour no task list derives.**
//
// They are taken at T14 rather than deferred because T14 is the last task in
// this feature that adds production code — T15 to T18 are tests and T19 and T20
// are documents — so a deferral would have been the second time nobody owned
// them. The other half of the argument is that `store.DerivedRebuilt()` already
// exists, is already logged by [Run], and is documented in three places as the
// signal that *"a full scan of every library is owed"*: shipping the feature
// with nothing acting on it would leave an installation whose derived half was
// rebuilt serving an empty library until an operator noticed and scanned by
// hand, which is ADR-0003's *"a rescan rather than an error"* half-kept.
//
// # What a scan being cancelled by a shutdown actually does
//
// 003 plan §6.9 said a scheduled scan is *"bound to the server's own lifetime
// and cancelled by shutdown, and a scan cancelled that way releases its claim on
// the way out"*. The last clause is **not** what happens and the plan is amended
// where it says so: a claim is released by [scan.Scanner.Scan] reaching its own
// end, and a cancellation is precisely the exit that does not reach it. What a
// cancelled scan leaves is what plan §7's *"a batch fails to commit"* row
// already describes — a claim left to go stale, bounded by
// [scan.DefaultStaleAfter] — and the alternative, releasing a claim over work
// that was abandoned half way, would write a summary document describing a scan
// that did not happen.

import (
	"context"
	"log/slog"
	"time"

	"github.com/vdatanet/atrium-go/internal/scan"
	"github.com/vdatanet/atrium-go/internal/store/sqlite"
)

// scheduledScans is the goroutine [Run] leaves behind it, and the handle that
// waits for it.
type scheduledScans struct {
	done chan struct{}
}

// startScheduledScans starts the server's own scanning and returns at once.
//
// `owed` is `store.DerivedRebuilt()`: this start found the derived half at
// another generation, dropped it and created it again, so **every library holds
// no items** and a full scan of all of them is owed. It runs before the first
// tick and it is a *full* re-examination, because there is nothing left of the
// previous scan for an incremental one to compare against — the change
// detection would find every file new anyway, and asking for the fast path over
// an empty store would be a lie about what is happening.
//
// `interval` of 0 disables the schedule and does not disable the rescan above:
// an operator who scans from a cron entry still expects a rebuilt installation
// to come back on its own, and the two settings are about different things.
//
// The scan is not part of the start. A synchronous full scan of every library
// would turn a generation bump into a start that takes minutes with the
// readiness gate shut for all of them, which is exactly what
// `ensureDerivedGeneration` says it is leaving to the entry layer.
func startScheduledScans(ctx context.Context, store *sqlite.Store, logger *slog.Logger,
	interval time.Duration, owed bool) *scheduledScans {
	scans := &scheduledScans{done: make(chan struct{})}

	go func() {
		defer close(scans.done)

		if owed {
			logger.Info("rescanning every library, because the derived half was rebuilt at this start")
			scanEveryLibrary(ctx, store, logger, scan.Options{Full: true})
		}
		if interval <= 0 {
			return
		}

		ticker := time.NewTicker(interval)
		defer ticker.Stop()
		for {
			select {
			case <-ctx.Done():
				return
			case <-ticker.C:
				// Incremental, always. 003 §3.8 makes the full re-examination
				// something an operator asks for, and a schedule that ignored
				// the change signal would re-read every file in the
				// installation twice a day.
				scanEveryLibrary(ctx, store, logger, scan.Options{})
			}
		}
	}()

	return scans
}

// wait blocks until the scanning goroutine has stopped.
//
// [Run] defers this **after** it defers the store's close, so this runs first:
// LIFO is what keeps a scan that is still writing from meeting a closed
// database. A scan already under way is not abandoned — the walk of the root it
// is on finishes, and the first store call after that fails on the cancelled
// context — so a stop waits for one root's walk and no longer.
//
// **Nothing asserts this, and that is measured rather than assumed.** Replacing
// this call with a discard survives every test in the package: no test stops a
// server while a scan of its own is still in flight, and constructing one would
// need a tree large enough that a walk outlasts a cancellation — a race dressed
// as a fixture. What the ordering prevents is a `database is closed` error at
// the end of a scan on a real installation, and it is recorded here in the
// shape 003 T12 and T13 both used for a mutation that survives.
func (s *scheduledScans) wait() { <-s.done }

// scanEveryLibrary runs one pass over every library this installation holds.
//
// **A failure is logged and never returned**, which is the difference between
// this and the subcommand: there is no operator reading an exit status, the next
// tick is the retry, and a library whose mount is missing must not stop the
// three that are fine. That is the same rule `library scan` follows over its
// selected set, one layer up.
func scanEveryLibrary(ctx context.Context, store *sqlite.Store, logger *slog.Logger, options scan.Options) {
	if ctx.Err() != nil {
		return
	}

	libraries, err := store.Libraries(ctx)
	if err != nil {
		logScanFailure(ctx, logger, "reading the libraries to scan", err)
		return
	}
	if len(libraries) == 0 {
		return
	}

	scanner, err := scan.New(scan.Config{
		Items:     store,
		Clock:     SystemClock(),
		ClaimedBy: ScannerName(),
		Logger:    logger,
	})
	if err != nil {
		logScanFailure(ctx, logger, "building the scanner", err)
		return
	}

	for _, lib := range libraries {
		if ctx.Err() != nil {
			return
		}
		// Never --allow-empty-root. That flag is an operator saying they
		// emptied a root on purpose (AC-16), and nothing unattended is in a
		// position to say it: a scheduled scan that assumed it would answer an
		// unmounted share by removing every item under it.
		if _, err := scanner.Scan(ctx, lib, options); err != nil {
			logScanFailure(ctx, logger, "scanning "+lib.Name, err)
		}
	}
}

// logScanFailure reports a failed scan, and says so quietly when the reason is
// that the server is stopping.
//
// A stop is not a fault, and a line at error level saying "context canceled" on
// every shutdown is how an operator learns to ignore the level.
func logScanFailure(ctx context.Context, logger *slog.Logger, what string, err error) {
	if ctx.Err() != nil {
		logger.Debug("scanning stopped by the shutdown", "what", what, "error", err)
		return
	}
	logger.Error("scheduled scan failed", "what", what, "error", err)
}
