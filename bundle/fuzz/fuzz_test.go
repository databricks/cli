package fuzz

import (
	"encoding/json"
	"os"
	"strconv"
	"strings"
	"testing"

	"github.com/stretchr/testify/require"
)

// defaultParitySeeds is the number of random jobs TestJobCreateParity checks by
// default. Each seed runs two real deploys (direct + terraform), so the count is
// kept modest; override with FUZZ_SEEDS for a deeper local run.
const defaultParitySeeds = 20

// TestJobCreateParity is the first DECO-25361 technique: for many random job
// configs, assert the terraform and direct engines produce equivalent create
// payloads. On divergence it prints the seed and the generated job so the failure
// can be reproduced and inspected.
func TestJobCreateParity(t *testing.T) {
	RequireTerraform(t)

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
// Otherwise the test runs FUZZ_SEEDS seeds (default defaultParitySeeds) starting
// at FUZZ_SEED_OFFSET. The offset lets the nightly job shift the window every run
// (push.yml derives it from the run number) so CI explores configs it has never
// tested before instead of re-checking the same fixed set forever.
func paritySeeds(t *testing.T) []int64 {
	if v := os.Getenv("FUZZ_SEED"); v != "" {
		var seeds []int64
		for _, part := range strings.Split(v, ",") {
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
		require.Greaterf(t, n, 0, "FUZZ_SEEDS must be positive, got %d", n)
		count = n
	}

	var offset int64
	if v := os.Getenv("FUZZ_SEED_OFFSET"); v != "" {
		n, err := strconv.ParseInt(v, 10, 64)
		require.NoErrorf(t, err, "invalid FUZZ_SEED_OFFSET=%q", v)
		offset = n
	}

	seeds := make([]int64, 0, count)
	for i := range int64(count) {
		seeds = append(seeds, offset+i)
	}
	return seeds
}

// FuzzJobCreateParity exposes the same parity check to Go's native fuzzer
// (`go test -fuzz=FuzzJobCreateParity`). Note each input runs two real deploys,
// so this is intended for ad-hoc deep runs, not the default `go test` path.
func FuzzJobCreateParity(f *testing.F) {
	RequireTerraform(f)
	for seed := range int64(5) {
		f.Add(seed)
	}
	f.Fuzz(func(t *testing.T, seed int64) {
		checkJobParity(t, seed)
	})
}

// checkJobParity generates the job for seed, deploys it under both engines, and
// fails the test with reproduction details if the create payloads diverge.
func checkJobParity(t *testing.T, seed int64) {
	t.Helper()
	job := GenerateJob(newRNG(seed))

	diffs, err := CompareJobEngines(t.Context(), t, job)
	require.NoErrorf(t, err, "seed %d", seed)

	if len(diffs) > 0 {
		jobJSON, _ := json.MarshalIndent(job, "", "  ")
		t.Errorf("seed %d: terraform/direct create payloads diverge (%d differences):", seed, len(diffs))
		for _, d := range diffs {
			t.Errorf("  %s", d)
		}
		t.Logf("reproduce with: FUZZ_SEED=%d go test ./bundle/fuzz -run TestJobCreateParity\n%s", seed, jobJSON)
	}
}
