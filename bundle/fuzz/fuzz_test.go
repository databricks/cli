package fuzz

import (
	"encoding/json"
	"os"
	"strconv"
	"strings"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// defaultParitySeeds is how many random jobs TestJobCreateParity checks by default.
// Each seed runs two real deploys, so keep it modest; override with FUZZ_SEEDS.
const defaultParitySeeds = 20

// regressionSeeds are seeds that previously surfaced a divergence. They are always
// checked (on top of the rotating nightly window, which never revisits them) so a
// fixed divergence can't silently regress. When the nightly job reports a new
// failing FUZZ_SEED, add it here in the PR that fixes the divergence.
//
//   - 29: single-node task new_cluster; direct omitted num_workers while terraform
//     force-sent 0. Fixed by initializeNumWorkers on task clusters (DECO-25361).
var regressionSeeds = []int64{29}

// TestJobCreateParity asserts the terraform and direct engines produce equivalent
// create payloads for many random jobs, printing the seed on divergence.
func TestJobCreateParity(t *testing.T) {
	requireTerraform(t)

	for _, seed := range paritySeeds(t) {
		t.Run("seed="+strconv.FormatInt(seed, 10), func(t *testing.T) {
			checkJobParity(t, seed)
		})
	}
}

// paritySeeds returns the seeds TestJobCreateParity should check.
//
// FUZZ_SEED (comma-separated) runs exactly those seeds and overrides everything,
// so a reported divergence reproduces with one command. Otherwise it runs
// regressionSeeds plus FUZZ_SEEDS seeds (default defaultParitySeeds) from
// FUZZ_SEED_OFFSET; the nightly job shifts the offset each run so CI keeps
// exploring new configs.
func paritySeeds(t *testing.T) []int64 {
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

	count := defaultParitySeeds
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

// TestParitySeeds verifies paritySeeds composes the regression seeds with the
// rotating window, deduplicates overlaps, and lets FUZZ_SEED override both.
func TestParitySeeds(t *testing.T) {
	// Isolate from ambient FUZZ_* in the dev environment (paritySeeds treats "" as
	// unset); subtests set only what they need.
	t.Setenv("FUZZ_SEED", "")
	t.Setenv("FUZZ_SEEDS", "")
	t.Setenv("FUZZ_SEED_OFFSET", "")

	t.Run("default includes regression seeds then window", func(t *testing.T) {
		t.Setenv("FUZZ_SEEDS", "3")
		t.Setenv("FUZZ_SEED_OFFSET", "100")
		want := append(append([]int64{}, regressionSeeds...), 100, 101, 102)
		assert.Equal(t, want, paritySeeds(t))
	})

	t.Run("window overlapping a regression seed is deduplicated", func(t *testing.T) {
		t.Setenv("FUZZ_SEEDS", "5")
		t.Setenv("FUZZ_SEED_OFFSET", "27")
		seeds := paritySeeds(t)
		count := 0
		for _, s := range seeds {
			if s == 29 {
				count++
			}
		}
		assert.Equal(t, 1, count, "seed 29 must appear once even though it is both a regression seed and inside the window")
	})

	t.Run("FUZZ_SEED override ignores regression seeds", func(t *testing.T) {
		t.Setenv("FUZZ_SEED", "7, 8")
		assert.Equal(t, []int64{7, 8}, paritySeeds(t))
	})
}

// FuzzJobCreateParity exposes the parity check to Go's native fuzzer. Each input
// runs two real deploys, so it's for ad-hoc deep runs, not the default test path.
func FuzzJobCreateParity(f *testing.F) {
	requireTerraform(f)
	for seed := range int64(5) {
		f.Add(seed)
	}
	// Seed the corpus with known past divergences.
	for _, seed := range regressionSeeds {
		f.Add(seed)
	}
	f.Fuzz(func(t *testing.T, seed int64) {
		checkJobParity(t, seed)
	})
}

// checkJobParity deploys the seed's job under both engines and fails if the create
// payloads diverge. A deploy/capture failure is not a payload divergence, so the
// outcomes are kept distinct:
//   - neither deployed: skip (the config is unacceptable to both engines).
//   - one deployed: fail separately as a deploy/capture difference, not a diff.
//   - both deployed: compare the captured payloads.
func checkJobParity(t *testing.T, seed int64) {
	t.Helper()
	job := generateJob(newRNG(seed))

	ctx := t.Context()
	direct, directErr := captureJobCreate(ctx, t, job, "direct")
	terraform, tfErr := captureJobCreate(ctx, t, job, "terraform")

	switch {
	case directErr != nil && tfErr != nil:
		t.Skipf("seed %d: config did not deploy under either engine (not a parity divergence)\ndirect: %v\nterraform: %v", seed, directErr, tfErr)
	case directErr != nil:
		t.Fatalf("seed %d: terraform deployed but direct did not (deploy/capture difference, not a payload diff): %v", seed, directErr)
	case tfErr != nil:
		t.Fatalf("seed %d: direct deployed but terraform did not (deploy/capture difference, not a payload diff): %v", seed, tfErr)
	}

	diffs, err := diffPayloads(direct, terraform, defaultIgnorePaths)
	require.NoErrorf(t, err, "seed %d: comparing create payloads", seed)

	if len(diffs) > 0 {
		jobJSON, _ := json.MarshalIndent(job, "", "  ")
		t.Errorf("seed %d: terraform/direct create payloads diverge (%d differences):", seed, len(diffs))
		for _, d := range diffs {
			t.Errorf("  %s", d)
		}
		t.Logf("reproduce with: FUZZ_SEED=%d task test-fuzz\nonce fixed, add %d to regressionSeeds in bundle/fuzz/fuzz_test.go\n%s", seed, seed, jobJSON)
	}
}
