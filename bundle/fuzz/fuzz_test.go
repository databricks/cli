package fuzz

import (
	"encoding/json"
	"errors"
	"os"
	"strconv"
	"strings"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// defaultInvariantSeeds is how many random jobs TestJobInvariants checks by
// default. Each seed runs a real deploy, so keep it modest; override with
// FUZZ_SEEDS.
const defaultInvariantSeeds = 20

// regressionSeeds are seeds that previously broke an invariant. They are always
// checked (on top of the rotating nightly window, which never revisits them) so a
// fixed bug can't silently regress. When the nightly job reports a new failing
// FUZZ_SEED, add it here in the PR that fixes it. Empty until the first such bug.
var regressionSeeds = []int64{}

// TestJobInvariants asserts the engine produces a create payload satisfying the
// invariants in checkJobInvariants for many random jobs, printing the seed on
// failure.
func TestJobInvariants(t *testing.T) {
	requireFuzzOptIn(t)

	for _, seed := range invariantSeeds(t) {
		t.Run("seed="+strconv.FormatInt(seed, 10), func(t *testing.T) {
			checkJob(t, seed)
		})
	}
}

// invariantSeeds returns the seeds TestJobInvariants should check.
//
// FUZZ_SEED (comma-separated) runs exactly those seeds and overrides everything,
// so a reported failure reproduces with one command. Otherwise it runs
// regressionSeeds plus FUZZ_SEEDS seeds (default defaultInvariantSeeds) from
// FUZZ_SEED_OFFSET; the nightly job shifts the offset each run so CI keeps
// exploring new configs.
func invariantSeeds(t *testing.T) []int64 {
	if v := os.Getenv("FUZZ_SEED"); v != "" {
		var seeds []int64
		for part := range strings.SplitSeq(v, ",") {
			part = strings.TrimSpace(part)
			if part == "" {
				continue
			}
			n, err := strconv.ParseInt(part, 10, 64)
			require.NoErrorf(t, err, "invalid FUZZ_SEED entry %q", part)
			seeds = append(seeds, n)
		}
		require.NotEmptyf(t, seeds, "FUZZ_SEED=%q contained no seeds", v)
		return seeds
	}

	count := defaultInvariantSeeds
	if v := os.Getenv("FUZZ_SEEDS"); v != "" {
		n, err := strconv.Atoi(v)
		require.NoErrorf(t, err, "invalid FUZZ_SEEDS=%q", v)
		require.Positivef(t, n, "FUZZ_SEEDS must be positive, got %d", n)
		count = n
	}

	var offset int64
	if v := os.Getenv("FUZZ_SEED_OFFSET"); v != "" {
		n, err := strconv.ParseInt(v, 10, 64)
		require.NoErrorf(t, err, "invalid FUZZ_SEED_OFFSET=%q", v)
		offset = n
	}

	seeds := make([]int64, 0, len(regressionSeeds)+count)
	seen := make(map[int64]bool, len(regressionSeeds)+count)
	for _, s := range regressionSeeds {
		if !seen[s] {
			seen[s] = true
			seeds = append(seeds, s)
		}
	}
	for i := range int64(count) {
		s := offset + i
		if !seen[s] {
			seen[s] = true
			seeds = append(seeds, s)
		}
	}
	return seeds
}

// TestInvariantSeeds verifies invariantSeeds composes the regression seeds with
// the rotating window, deduplicates overlaps, and lets FUZZ_SEED override both.
func TestInvariantSeeds(t *testing.T) {
	// Isolate from ambient FUZZ_* in the dev environment (invariantSeeds treats ""
	// as unset); subtests set only what they need.
	t.Setenv("FUZZ_SEED", "")
	t.Setenv("FUZZ_SEEDS", "")
	t.Setenv("FUZZ_SEED_OFFSET", "")

	t.Run("default is regression seeds then the window", func(t *testing.T) {
		t.Setenv("FUZZ_SEEDS", "3")
		t.Setenv("FUZZ_SEED_OFFSET", "100")
		want := append(append([]int64{}, regressionSeeds...), 100, 101, 102)
		assert.Equal(t, want, invariantSeeds(t))
	})

	t.Run("FUZZ_SEED override ignores regression seeds", func(t *testing.T) {
		t.Setenv("FUZZ_SEED", "7, 8")
		assert.Equal(t, []int64{7, 8}, invariantSeeds(t))
	})
}

// FuzzJobInvariants exposes the invariant check to Go's native fuzzer. Each input
// runs a real deploy, so it's for ad-hoc deep runs, not the default test path.
func FuzzJobInvariants(f *testing.F) {
	requireFuzzOptIn(f)
	for seed := range int64(5) {
		f.Add(seed)
	}
	// Seed the corpus with known past failures.
	for _, seed := range regressionSeeds {
		f.Add(seed)
	}
	f.Fuzz(func(t *testing.T, seed int64) {
		checkJob(t, seed)
	})
}

// checkJob validates and deploys the seed's job, then asserts its create payload
// satisfies the invariants. It separates the two fuzzing outcomes:
//   - the config doesn't validate: skip, since invalid input can't be a bug.
//   - a validated config that fails to deploy or breaks an invariant: fail (a
//     config the CLI accepted must deploy and produce a sound payload).
func checkJob(t *testing.T, seed int64) {
	t.Helper()
	job := generateJob(newRNG(seed))

	payload, err := captureJobCreate(t.Context(), t, job)
	if errors.Is(err, errInvalidConfig) {
		t.Skipf("seed %d: config did not validate, so it can't violate an invariant: %v", seed, err)
	}
	require.NoErrorf(t, err, "seed %d: validated config failed to deploy", seed)

	checkJobInvariants(t, seed, job, payload)

	if t.Failed() {
		jobJSON, _ := json.MarshalIndent(job, "", "  ")
		t.Logf("reproduce with: FUZZ_SEED=%d task test-fuzz\nonce fixed, add %d to regressionSeeds in bundle/fuzz/fuzz_test.go\n%s", seed, seed, jobJSON)
	}
}
