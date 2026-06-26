package fuzz

import (
	"encoding/json"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// TestCaptureJobCreateDirect is intentionally NOT opt-in gated: a single direct
// deploy is cheap, so it runs on every `task test` as a smoke test of the capture
// harness. The expensive terraform side stays opt-in via requireTerraform.
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
