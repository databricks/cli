package files

// Straggler control for multipart part uploads. A wedged cloud connection
// trickles bytes for minutes while its peers stay fast; it trips neither the
// idle timeout (bytes keep moving) nor the response timeout (the response phase
// is fine), so a single slow part gates the whole upload (completeMultipart
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

// errSlowAttempt is the cancel cause for a part attempt that outlived its soft
// deadline. It never escapes doUploadOnePart; it only distinguishes the guard's
// own cancellation from the caller cancelling ctx.
var errSlowAttempt = errors.New("part attempt exceeded soft deadline")

// slowAttemptGuard tracks the duration of recently completed part attempts so an
// attempt that far exceeds the recent tail can be cancelled and re-issued. Safe
// for concurrent use by the upload workers. Its policy comes from tun (see the
// slow* fields on tunables).
type slowAttemptGuard struct {
	tun     tunables
	mu      sync.Mutex
	samples []time.Duration
}

// record adds a completed attempt's duration to the rolling window.
func (g *slowAttemptGuard) record(d time.Duration) {
	g.mu.Lock()
	defer g.mu.Unlock()
	g.samples = append(g.samples, d)
	if len(g.samples) > g.tun.slowWindow {
		g.samples = g.samples[len(g.samples)-g.tun.slowWindow:]
	}
}

// deadline returns the soft deadline for the next attempt: the cold-start
// deadline until enough attempts have completed for the p95 to be meaningful,
// then slowFactor x the recent p95 (floored at slowMinDeadline). The p95 ignores
// the rare wedged attempts in the window (they sit above it), so the deadline
// does not drift up toward the stragglers it is meant to catch.
func (g *slowAttemptGuard) deadline() time.Duration {
	g.mu.Lock()
	defer g.mu.Unlock()
	if len(g.samples) < g.tun.slowWarmup {
		return g.tun.slowColdDeadline
	}
	s := slices.Clone(g.samples)
	slices.Sort(s)
	p95 := s[min(len(s)*95/100, len(s)-1)]
	return max(time.Duration(g.tun.slowFactor)*p95, g.tun.slowMinDeadline)
}

// attemptDeadline returns the current soft deadline for a part attempt. It is
// called repeatedly while an attempt is in flight (see sendPart), so the value
// tracks the warming-up guard. The synchronous first part gets its own tight,
// contention-free deadline; once a part has been re-issued slowMaxReissue times
// the guard disarms (0) so the part rides out the normal timeouts instead of
// being cancelled again.
func (uc *uploadContext) attemptDeadline(isFirstPart bool, slowRetries int) time.Duration {
	switch {
	case slowRetries >= uc.tun.slowMaxReissue:
		return 0
	case isFirstPart:
		return uc.tun.slowFirstPartDeadline
	default:
		return uc.slowGuard.deadline()
	}
}
