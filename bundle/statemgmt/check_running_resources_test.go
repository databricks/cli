package statemgmt

import (
	"testing"

	"github.com/databricks/cli/bundle/internal/bundletest"
	"github.com/databricks/cli/libs/testserver"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func notFound(testserver.Request) any {
	return testserver.Response{
		StatusCode: 404,
		Body:       map[string]string{"error_code": "RESOURCE_DOES_NOT_EXIST", "message": "not found"},
	}
}

func TestIsAnyResourceRunningWithEmptyState(t *testing.T) {
	w, _ := bundletest.NewClients(t)
	err := checkAnyResourceRunning(t.Context(), w, nil)
	require.NoError(t, err)
}

func TestIsAnyResourceRunningWithJob(t *testing.T) {
	w, server := bundletest.NewClients(t)
	resources := ExportedResourcesMap{
		"resources.jobs.job1": {ID: "123"},
	}

	runs := []map[string]any{{"run_id": 1234}}
	server.Handle("GET", "/api/2.2/jobs/runs/list", func(req testserver.Request) any {
		assert.Equal(t, "123", req.URL.Query().Get("job_id"))
		assert.Equal(t, "true", req.URL.Query().Get("active_only"))
		return map[string]any{"runs": runs}
	})

	err := checkAnyResourceRunning(t.Context(), w, resources)
	require.ErrorContains(t, err, "job 123 is running")

	runs = nil
	err = checkAnyResourceRunning(t.Context(), w, resources)
	require.NoError(t, err)
}

func TestIsAnyResourceRunningWithPipeline(t *testing.T) {
	w, server := bundletest.NewClients(t)
	resources := ExportedResourcesMap{
		"resources.pipelines.pipeline1": {ID: "123"},
	}

	state := "RUNNING"
	server.Handle("GET", "/api/2.0/pipelines/{pipeline_id}", func(req testserver.Request) any {
		assert.Equal(t, "123", req.Vars["pipeline_id"])
		return map[string]any{"pipeline_id": "123", "state": state}
	})

	err := checkAnyResourceRunning(t.Context(), w, resources)
	require.ErrorContains(t, err, "pipeline 123 is running")

	state = "IDLE"
	err = checkAnyResourceRunning(t.Context(), w, resources)
	require.NoError(t, err)
}

func TestIsAnyResourceRunningWithAPIFailure(t *testing.T) {
	w, server := bundletest.NewClients(t)
	resources := ExportedResourcesMap{
		"resources.pipelines.pipeline1": {ID: "123"},
	}

	server.Handle("GET", "/api/2.0/pipelines/{pipeline_id}", func(testserver.Request) any {
		return testserver.Response{
			StatusCode: 400,
			Body:       map[string]string{"error_code": "INVALID_PARAMETER_VALUE", "message": "API failure"},
		}
	})

	err := checkAnyResourceRunning(t.Context(), w, resources)
	require.ErrorContains(t, err, "API failure")
}

func TestIsAnyResourceRunningWithDeletedJob(t *testing.T) {
	w, server := bundletest.NewClients(t)
	resources := ExportedResourcesMap{
		"resources.jobs.job1": {ID: "123"},
	}

	server.Handle("GET", "/api/2.2/jobs/runs/list", notFound)

	err := checkAnyResourceRunning(t.Context(), w, resources)
	require.NoError(t, err)
}

func TestIsAnyResourceRunningWithDeletedPipeline(t *testing.T) {
	w, server := bundletest.NewClients(t)
	resources := ExportedResourcesMap{
		"resources.pipelines.pipeline1": {ID: "123"},
	}

	server.Handle("GET", "/api/2.0/pipelines/{pipeline_id}", notFound)

	err := checkAnyResourceRunning(t.Context(), w, resources)
	require.NoError(t, err)
}
