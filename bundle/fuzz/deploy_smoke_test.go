package fuzz

import (
	"encoding/json"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// TestCaptureJobCreateDirect is intentionally NOT gated behind requireFuzzOptIn,
// unlike the terraform parity suite. The direct engine needs no provisioned
// terraform, and one deterministic direct deploy is cheap, so this runs on every
// `task test` as a smoke test that the capture harness and the direct create path
// still work. The expensive part the opt-in protects against is the terraform
// side (two real deploys per seed), which stays opt-in via requireTerraform.
func TestCaptureJobCreateDirect(t *testing.T) {
	job := generateJob(newRNG(1))

	body, err := captureJobCreate(t.Context(), t, job, "direct")
	require.NoError(t, err)
	require.NotEmpty(t, body)

	var payload map[string]any
	require.NoError(t, json.Unmarshal(body, &payload))
	assert.Equal(t, job.Name, payload["name"])
	assert.Contains(t, payload, "tasks")
}

func TestCaptureJobCreateTerraform(t *testing.T) {
	requireTerraform(t)
	job := generateJob(newRNG(1))

	body, err := captureJobCreate(t.Context(), t, job, "terraform")
	require.NoError(t, err)
	require.NotEmpty(t, body)

	var payload map[string]any
	require.NoError(t, json.Unmarshal(body, &payload))
	assert.Equal(t, job.Name, payload["name"])
}
