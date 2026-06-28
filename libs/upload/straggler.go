package upload

// Straggler control for multipart part uploads. A wedged cloud connection
// trickles bytes for minutes while its peers stay fast; it trips neither the
// idle timeout (bytes keep moving) nor the response timeout (the response phase
// is fine), so a single slow part gates the whole upload (CompleteMultipart
// needs every ETag). doUploadOnePart cancels an attempt that runs far longer
// than the recent part-duration tail and re-issues it on a fresh connection.
//
// The trigger keys off the recent p95, not the median: at high concurrency the
// opening burst of connections makes many parts legitimately slow (tens of
// seconds) even though the median stays low, so a median-relative trigger would
// both miss the real outliers and falsely cancel healthy burst parts. A p95
// multiple tracks that legitimate tail and fires only well above it, while still
// adapting to a uniformly slow network (where p95 rises with everything else).

import (
	"errors"
	"slices"
	"sync"
	"time"
)

// Straggler tunables. Package variables (not constants) so tests can shrink
// them; production code never mutates them.
var (
	// slowAttemptFactor cancels an attempt that exceeds this multiple of the
	// recent p95 part duration. It is the main knob: lower catches stragglers
	// sooner but re-issues more healthy parts.
	slowAttemptFactor = 3

	// slowAttemptWarmup is the number of completed parts required before the guard
	// switches from the cold-start deadline to the p95-relative one.
	slowAttemptWarmup = 32

	// slowAttemptWindow bounds the rolling sample set so the p95 tracks recent
	// conditions rather than the whole upload.
	slowAttemptWindow = 128

	// slowAttemptMinDeadline floors the soft deadline so a fast network (tiny p95)
	// cannot produce a deadline that clips normal variance.
	slowAttemptMinDeadline = 5 * time.Second

	// slowAttemptColdDeadline is the absolute soft deadline for a part in the
	// opening wave, before slowAttemptWarmup parts have completed and the
	// p95-relative deadline is armed. Because the deadline is re-evaluated in
	// flight (see sendPart), this only bites in the first seconds of an upload or
	// for a file too small to warm the guard; once warmed, an in-flight early part
	// is caught at the tighter p95 deadline instead. It is set above the legitimate
	// opening-burst latency (tens of seconds at high concurrency).
	slowAttemptColdDeadline = 30 * time.Second

	// slowAttemptFirstPart is the soft deadline for the synchronous first part,
	// which runs alone before any worker (so it has no samples and, unlike the
	// opening wave, no burst contention). A healthy first PUT is a few seconds, so
	// a tight deadline re-issues a wedged first connection quickly without blocking
	// the whole upload at 0% on the cold-start floor.
	slowAttemptFirstPart = 10 * time.Second

	// slowAttemptCheckInterval is how often an in-flight attempt is re-checked
	// against the current deadline, so warmup and a falling p95 take effect while
	// the attempt is still running.
	slowAttemptCheckInterval = 1 * time.Second

	// slowAttemptMax caps soft re-issues per part; afterward the part rides out the
	// normal timeouts rather than being cancelled again, so it is never failed
	// solely for staying slow.
	slowAttemptMax = 2
)

// errSlowAttempt is the cancel cause for a part attempt that outlived its soft
// deadline. It never escapes doUploadOnePart; it only distinguishes the guard's
// own cancellation from the caller cancelling ctx.
var errSlowAttempt = errors.New("part attempt exceeded soft deadline")

// slowAttemptGuard tracks the duration of recently completed part attempts so an
// attempt that far exceeds the recent tail can be cancelled and re-issued. Safe
// for concurrent use by the upload workers.
type slowAttemptGuard struct {
	mu      sync.Mutex
	samples []time.Duration
}

// record adds a completed attempt's duration to the rolling window.
func (g *slowAttemptGuard) record(d time.Duration) {
	g.mu.Lock()
	defer g.mu.Unlock()
	g.samples = append(g.samples, d)
	if len(g.samples) > slowAttemptWindow {
		g.samples = g.samples[len(g.samples)-slowAttemptWindow:]
	}
}

// deadline returns the soft deadline for the next attempt: the cold-start
// deadline until enough attempts have completed for the p95 to be meaningful,
// then slowAttemptFactor x the recent p95 (floored at slowAttemptMinDeadline).
// The p95 ignores the rare wedged attempts in the window (they sit above it), so
// the deadline does not drift up toward the stragglers it is meant to catch.
func (g *slowAttemptGuard) deadline() time.Duration {
	g.mu.Lock()
	defer g.mu.Unlock()
	if len(g.samples) < slowAttemptWarmup {
		return slowAttemptColdDeadline
	}
	s := slices.Clone(g.samples)
	slices.Sort(s)
	p95 := s[min(len(s)*95/100, len(s)-1)]
	return max(time.Duration(slowAttemptFactor)*p95, slowAttemptMinDeadline)
}

// attemptDeadline returns the current soft deadline for a part attempt. It is
// called repeatedly while an attempt is in flight (see sendPart), so the value
// tracks the warming-up guard. The synchronous first part gets its own tight,
// contention-free deadline; once a part has been re-issued slowAttemptMax times
// the guard disarms (0) so the part rides out the normal timeouts instead of
// being cancelled again.
func (uc *uploadContext) attemptDeadline(isFirstPart bool, slowRetries int) time.Duration {
	switch {
	case slowRetries >= slowAttemptMax:
		return 0
	case isFirstPart:
		return slowAttemptFirstPart
	default:
		return uc.slowGuard.deadline()
	}
}
