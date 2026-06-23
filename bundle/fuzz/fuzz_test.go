package fuzz

import (
	"encoding/json"
	"os"
	"strconv"
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

	seeds := defaultParitySeeds
	if v := os.Getenv("FUZZ_SEEDS"); v != "" {
		n, err := strconv.Atoi(v)
		require.NoErrorf(t, err, "invalid FUZZ_SEEDS=%q", v)
		seeds = n
	}

	for seed := range int64(seeds) {
		t.Run("seed="+strconv.FormatInt(seed, 10), func(t *testing.T) {
			checkJobParity(t, seed)
		})
	}
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
		t.Logf("reproduce with GenerateJob(newRNG(%d)):\n%s", seed, jobJSON)
	}
}
