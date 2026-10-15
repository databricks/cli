package run

import (
	"encoding/json"
	"testing"

	"github.com/databricks/cli/bundle"
	"github.com/databricks/cli/bundle/config"
	"github.com/databricks/cli/bundle/config/resources"
	"github.com/databricks/cli/bundle/internal/bundletest"
	"github.com/databricks/cli/libs/cmdio"
	"github.com/databricks/cli/libs/testserver"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func newPipelineRunner(t *testing.T) (pipelineRunner, *testserver.Server) {
	pipeline := &resources.Pipeline{
		ID: "123",
	}

	b := &bundle.Bundle{
		Config: config.Root{
			Resources: config.Resources{
				Pipelines: map[string]*resources.Pipeline{
					"test_pipeline": pipeline,
				},
			},
		},
	}

	w, server := bundletest.NewClients(t)
	b.SetWorkpaceClient(w)
	return pipelineRunner{key: "test", bundle: b, pipeline: pipeline}, server
}

// handleStop serves the stop request and the pipeline poll that waits for IDLE.
func handleStop(t *testing.T, server *testserver.Server) *int {
	stops := 0
	server.Handle("POST", "/api/2.0/pipelines/{pipeline_id}/stop", func(req testserver.Request) any {
		assert.Equal(t, "123", req.Vars["pipeline_id"])
		stops++
		return map[string]any{}
	})
	server.Handle("GET", "/api/2.0/pipelines/{pipeline_id}", func(req testserver.Request) any {
		return map[string]any{"pipeline_id": "123", "state": "IDLE"}
	})
	return &stops
}

func TestPipelineRunnerCancel(t *testing.T) {
	runner, server := newPipelineRunner(t)
	stops := handleStop(t, server)

	err := runner.Cancel(t.Context())
	require.NoError(t, err)
	assert.Equal(t, 1, *stops)
}

func TestPipelineRunnerRestart(t *testing.T) {
	runner, server := newPipelineRunner(t)
	stops := handleStop(t, server)

	// The runner starts a new update.
	server.Handle("POST", "/api/2.0/pipelines/{pipeline_id}/updates", func(req testserver.Request) any {
		var body map[string]any
		assert.NoError(t, json.Unmarshal(req.Body, &body))
		// sdk-go repeats the pipeline_id path parameter in the body.
		assert.Equal(t, map[string]any{"pipeline_id": "123"}, body)
		return map[string]any{"update_id": "456"}
	})

	// The runner polls for events.
	server.Handle("GET", "/api/2.0/pipelines/{pipeline_id}/events", func(req testserver.Request) any {
		assert.Equal(t, "update_id = '456'", req.URL.Query().Get("filter"))
		assert.Equal(t, "100", req.URL.Query().Get("max_results"))
		return map[string]any{}
	})

	// The runner polls for the update status.
	server.Handle("GET", "/api/2.0/pipelines/{pipeline_id}/updates/{update_id}", func(req testserver.Request) any {
		assert.Equal(t, "456", req.Vars["update_id"])
		return map[string]any{"update": map[string]any{"state": "COMPLETED"}}
	})

	ctx := cmdio.MockDiscard(t.Context())
	_, err := runner.Restart(ctx, &Options{})
	require.NoError(t, err)
	assert.Equal(t, 1, *stops)
}
