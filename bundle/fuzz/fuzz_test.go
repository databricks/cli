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

// defaultParitySeeds is the number of random jobs TestJobCreateParity checks by
// default. Each seed runs two real deploys (direct + terraform), so the count is
// kept modest; override with FUZZ_SEEDS for a deeper local run.
const defaultParitySeeds = 20

// regressionSeeds are seeds that previously surfaced a terraform/direct create
// payload divergence. They are always checked (in addition to the rotating
// nightly window) so the divergence keeps being exercised even though the
// nightly window moves on every run and would otherwise never revisit them.
//
// When the nightly job reports a new failing FUZZ_SEED, add it here.
//
//   - 29: generates a single-node task-level new_cluster (num_workers 0, no
//     autoscale). The direct engine omits num_workers on task clusters while
//     terraform force-sends num_workers:0, so the create payloads diverge. That
//     specific shape is suppressed by defaultIgnoreRules (see
//     isBenignTaskNumWorkers), so seed 29 currently asserts only that nothing
//     else about this config diverges. Once the divergence is fixed and its
//     ignore rule removed, this seed becomes a full guard against it regressing.
//     Tracked under DECO-25361.
var regressionSeeds = []int64{29}

// TestJobCreateParity is the first DECO-25361 technique: for many random job
// configs, assert the terraform and direct engines produce equivalent create
// payloads. On divergence it prints the seed and the generated job so the failure
// can be reproduced and inspected.
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
// FUZZ_SEED (comma-separated list) runs exactly those seeds and overrides
// everything else. This is the knob the failure message prints so a single
// reported divergence can be reproduced with one command, without re-running
// every seed before it.
//
// Otherwise the test runs the regressionSeeds plus FUZZ_SEEDS seeds (default
// defaultParitySeeds) starting at FUZZ_SEED_OFFSET. The offset lets the nightly
// job shift the window every run (push.yml derives it from the run number) so CI
// explores configs it has never tested before instead of re-checking the same
// fixed set forever; the regressionSeeds are always included on top so known
// past divergences keep being verified.
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
	// Isolate from any ambient FUZZ_* in the developer's environment. FUZZ_SEED in
	// particular would short-circuit paritySeeds and break the cases below; an
	// inherited FUZZ_SEEDS/OFFSET would skew the expected window. paritySeeds
	// treats "" as unset, and subtests set only what they need on top.
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

// FuzzJobCreateParity exposes the same parity check to Go's native fuzzer
// (`go test -fuzz=FuzzJobCreateParity`). Note each input runs two real deploys,
// so this is intended for ad-hoc deep runs, not the default `go test` path.
func FuzzJobCreateParity(f *testing.F) {
	requireTerraform(f)
	for seed := range int64(5) {
		f.Add(seed)
	}
	// Seed the corpus with known past divergences so the fuzzer always starts
	// from inputs that previously exposed a bug.
	for _, seed := range regressionSeeds {
		f.Add(seed)
	}
	f.Fuzz(func(t *testing.T, seed int64) {
		checkJobParity(t, seed)
	})
}

// checkJobParity generates the job for seed, deploys it under both engines, and
// fails the test with reproduction details if the create payloads diverge.
//
// A deploy/capture failure is not a create-payload divergence, so the three
// outcomes are handled distinctly to keep nightly triage from misdirecting a
// deploy failure into regressionSeeds (which is only for real payload diffs):
//   - neither engine deployed: the generator produced a config nothing accepts,
//     so skip (logging both errors) rather than flag a parity bug.
//   - exactly one engine deployed: the engines disagree on whether the config
//     deploys at all. That is worth failing on, but it is a deploy/capture
//     difference rather than a payload diff, so it is reported separately. The
//     failing side's error (an API rejection, an unregistered route, etc.) is
//     included so triage can tell a true acceptance divergence from a harness gap.
//   - both deployed: compare the captured create payloads.
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

	diffs, err := diffPayloads(direct, terraform, defaultIgnoreRules)
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
