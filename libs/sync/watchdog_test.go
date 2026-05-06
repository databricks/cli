package sync

import (
	"context"
	"strconv"
	"sync/atomic"
	"testing"
	"time"

	"github.com/stretchr/testify/require"
)

// TestGroupRunParallelHonorsLimit asserts the parallelism limit passed into
// groupRunParallel actually caps in-flight work. Without this test, swapping
// the hardcoded limit out for SyncOptions.Concurrency could silently lose
// the cap.
func TestGroupRunParallelHonorsLimit(t *testing.T) {
	const limit = 3

	paths := make([]string, 20)
	for i := range paths {
		paths[i] = strconv.Itoa(i)
	}

	var inflight, peak atomic.Int32
	fn := func(ctx context.Context, path string) error {
		cur := inflight.Add(1)
		// Track the peak observed concurrency.
		for {
			p := peak.Load()
			if cur <= p || peak.CompareAndSwap(p, cur) {
				break
			}
		}
		// Hold long enough that other goroutines pile up if the limit is wrong.
		time.Sleep(20 * time.Millisecond)
		inflight.Add(-1)
		return nil
	}

	err := groupRunParallel(t.Context(), paths, limit, fn)
	require.NoError(t, err)
	require.LessOrEqual(t, int(peak.Load()), limit, "observed concurrency exceeded limit")
}
