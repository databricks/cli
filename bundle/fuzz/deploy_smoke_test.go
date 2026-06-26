package fuzz

import (
	"testing"

	"github.com/stretchr/testify/require"
)

// TestCaptureJobCreateDirect is intentionally NOT opt-in gated: a single direct
// deploy is cheap, so it runs on every `task test` as a smoke test of the capture
// harness and the invariants. The wider seed sweep stays opt-in via
// requireFuzzOptIn.
func TestCaptureJobCreateDirect(t *testing.T) {
	job := generateJob(newRNG(1))

	body, err := captureJobCreate(t.Context(), t, job)
	require.NoError(t, err)
	require.NotEmpty(t, body)

	checkJobInvariants(t, 1, job, body)
}
