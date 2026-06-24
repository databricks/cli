package fuzz

import (
	"encoding/json"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestCaptureJobCreateDirect(t *testing.T) {
	job := GenerateJob(newRNG(1))

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
	job := GenerateJob(newRNG(1))

	body, err := captureJobCreate(t.Context(), t, job, "terraform")
	require.NoError(t, err)
	require.NotEmpty(t, body)

	var payload map[string]any
	require.NoError(t, json.Unmarshal(body, &payload))
	assert.Equal(t, job.Name, payload["name"])
}
