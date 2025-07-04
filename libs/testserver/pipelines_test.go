package testserver

import (
	"testing"

	"github.com/databricks/databricks-sdk-go/service/pipelines"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestPipelineStartUpdate_StoresUpdateParameters(t *testing.T) {
	workspace := NewFakeWorkspace("http://test")

	pipelineId := "test-pipeline-123"
	workspace.Pipelines[pipelineId] = pipelines.GetPipelineResponse{
		PipelineId: pipelineId,
		Name:       "Test Pipeline",
	}

	// Test case 1: Basic update with no parameters
	req1 := Request{
		Body: []byte(`{}`),
	}

	response1 := workspace.PipelineStartUpdate(req1, pipelineId)
	require.Equal(t, 0, response1.StatusCode)

	assert.Len(t, workspace.PipelineUpdates, 1)

	startUpdateResponse, ok := response1.Body.(pipelines.StartUpdateResponse)
	require.True(t, ok)
	updateId1 := startUpdateResponse.UpdateId

	storedUpdate1, exists := workspace.PipelineUpdates[updateId1]
	require.True(t, exists)
	assert.Equal(t, pipelineId, storedUpdate1.PipelineId)
	assert.False(t, storedUpdate1.FullRefresh)
	assert.False(t, storedUpdate1.ValidateOnly)
	assert.Nil(t, storedUpdate1.RefreshSelection)
	assert.Nil(t, storedUpdate1.FullRefreshSelection)

	// Test case 2: Update with refresh selection
	req2 := Request{
		Body: []byte(`{
			"refresh_selection": ["table1", "table2"]
		}`),
	}

	response2 := workspace.PipelineStartUpdate(req2, pipelineId)
	require.Equal(t, 0, response2.StatusCode)

	// Verify we now have 2 updates
	assert.Len(t, workspace.PipelineUpdates, 2)

	startUpdateResponse2, ok := response2.Body.(pipelines.StartUpdateResponse)
	require.True(t, ok)
	updateId2 := startUpdateResponse2.UpdateId

	storedUpdate2, exists := workspace.PipelineUpdates[updateId2]
	require.True(t, exists)
	assert.Equal(t, pipelineId, storedUpdate2.PipelineId)
	assert.Equal(t, []string{"table1", "table2"}, storedUpdate2.RefreshSelection)
	assert.False(t, storedUpdate2.FullRefresh)
	assert.False(t, storedUpdate2.ValidateOnly)

	// Test case 3: Update with full refresh
	req3 := Request{
		Body: []byte(`{
			"full_refresh": true,
			"full_refresh_selection": ["table3", "table4"]
		}`),
	}

	response3 := workspace.PipelineStartUpdate(req3, pipelineId)
	require.Equal(t, 0, response3.StatusCode)

	startUpdateResponse3, ok := response3.Body.(pipelines.StartUpdateResponse)
	require.True(t, ok)
	updateId3 := startUpdateResponse3.UpdateId

	storedUpdate3, exists := workspace.PipelineUpdates[updateId3]
	require.True(t, exists)
	assert.Equal(t, pipelineId, storedUpdate3.PipelineId)
	assert.True(t, storedUpdate3.FullRefresh)
	assert.Equal(t, []string{"table3", "table4"}, storedUpdate3.FullRefreshSelection)
	assert.False(t, storedUpdate3.ValidateOnly)

	// Test case 4: Update with validate only
	req4 := Request{
		Body: []byte(`{
			"validate_only": true
		}`),
	}

	response4 := workspace.PipelineStartUpdate(req4, pipelineId)
	require.Equal(t, 0, response4.StatusCode)

	startUpdateResponse4, ok := response4.Body.(pipelines.StartUpdateResponse)
	require.True(t, ok)
	updateId4 := startUpdateResponse4.UpdateId

	storedUpdate4, exists := workspace.PipelineUpdates[updateId4]
	require.True(t, exists)
	assert.Equal(t, pipelineId, storedUpdate4.PipelineId)
	assert.True(t, storedUpdate4.ValidateOnly)
	assert.False(t, storedUpdate4.FullRefresh)
}

func TestPipelineStartUpdate_HandlesInvalidJSON(t *testing.T) {
	workspace := NewFakeWorkspace("http://test")

	pipelineId := "test-pipeline-123"
	workspace.Pipelines[pipelineId] = pipelines.GetPipelineResponse{
		PipelineId: pipelineId,
		Name:       "Test Pipeline",
	}

	req := Request{
		Body: []byte(`{invalid json`),
	}

	response := workspace.PipelineStartUpdate(req, pipelineId)
	assert.Equal(t, 400, response.StatusCode)
	assert.Contains(t, response.Body.(string), "cannot unmarshal request body")
}

func TestPipelineStartUpdate_HandlesNonExistentPipeline(t *testing.T) {
	workspace := NewFakeWorkspace("http://test")

	req := Request{
		Body: []byte(`{}`),
	}

	response := workspace.PipelineStartUpdate(req, "non-existent-pipeline")
	assert.Equal(t, 404, response.StatusCode)

	assert.Len(t, workspace.PipelineUpdates, 0)
}

func TestPipelineStartUpdate_GeneratesUniqueUpdateIds(t *testing.T) {
	workspace := NewFakeWorkspace("http://test")

	pipelineId := "test-pipeline-123"
	workspace.Pipelines[pipelineId] = pipelines.GetPipelineResponse{
		PipelineId: pipelineId,
		Name:       "Test Pipeline",
	}

	req := Request{Body: []byte(`{}`)}

	response1 := workspace.PipelineStartUpdate(req, pipelineId)
	response2 := workspace.PipelineStartUpdate(req, pipelineId)
	response3 := workspace.PipelineStartUpdate(req, pipelineId)

	assert.Equal(t, 0, response1.StatusCode)
	assert.Equal(t, 0, response2.StatusCode)
	assert.Equal(t, 0, response3.StatusCode)

	startUpdateResponse1, ok := response1.Body.(pipelines.StartUpdateResponse)
	require.True(t, ok)
	startUpdateResponse2, ok := response2.Body.(pipelines.StartUpdateResponse)
	require.True(t, ok)
	startUpdateResponse3, ok := response3.Body.(pipelines.StartUpdateResponse)
	require.True(t, ok)

	updateIds := map[string]bool{
		startUpdateResponse1.UpdateId: true,
		startUpdateResponse2.UpdateId: true,
		startUpdateResponse3.UpdateId: true,
	}

	assert.Len(t, updateIds, 3, "All update IDs should be unique")
	assert.Len(t, workspace.PipelineUpdates, 3, "All updates should be stored")
}

func TestPipelineStartUpdate_ComplexParameterCombination(t *testing.T) {
	workspace := NewFakeWorkspace("http://test")

	pipelineId := "test-pipeline-123"
	workspace.Pipelines[pipelineId] = pipelines.GetPipelineResponse{
		PipelineId: pipelineId,
		Name:       "Test Pipeline",
	}

	complexRequest := `{
		"refresh_selection": ["table1", "table2", "table3"],
		"full_refresh_selection": ["table4"],
		"validate_only": true,
		"cause": "API_CALL"
	}`

	req := Request{
		Body: []byte(complexRequest),
	}

	response := workspace.PipelineStartUpdate(req, pipelineId)
	assert.Equal(t, 0, response.StatusCode)

	startUpdateResponse, ok := response.Body.(pipelines.StartUpdateResponse)
	require.True(t, ok)

	storedUpdate, exists := workspace.PipelineUpdates[startUpdateResponse.UpdateId]
	require.True(t, exists)

	assert.Equal(t, pipelineId, storedUpdate.PipelineId)
	assert.Equal(t, []string{"table1", "table2", "table3"}, storedUpdate.RefreshSelection)
	assert.Equal(t, []string{"table4"}, storedUpdate.FullRefreshSelection)
	assert.True(t, storedUpdate.ValidateOnly)
	assert.Equal(t, pipelines.StartUpdateCauseApiCall, storedUpdate.Cause)
}

func TestPipelineUpdates_Initialization(t *testing.T) {
	workspace := NewFakeWorkspace("http://test")

	assert.NotNil(t, workspace.PipelineUpdates)
	assert.Len(t, workspace.PipelineUpdates, 0)
}

func TestPipelineStartUpdate_EmptyBodyHandling(t *testing.T) {
	workspace := NewFakeWorkspace("http://test")

	pipelineId := "test-pipeline-123"
	workspace.Pipelines[pipelineId] = pipelines.GetPipelineResponse{
		PipelineId: pipelineId,
		Name:       "Test Pipeline",
	}

	req := Request{
		Body: []byte{},
	}

	response := workspace.PipelineStartUpdate(req, pipelineId)
	assert.Equal(t, 0, response.StatusCode)

	startUpdateResponse, ok := response.Body.(pipelines.StartUpdateResponse)
	require.True(t, ok)

	storedUpdate, exists := workspace.PipelineUpdates[startUpdateResponse.UpdateId]
	require.True(t, exists)

	assert.Equal(t, pipelineId, storedUpdate.PipelineId)
	assert.False(t, storedUpdate.FullRefresh)
	assert.False(t, storedUpdate.ValidateOnly)
	assert.Nil(t, storedUpdate.RefreshSelection)
	assert.Nil(t, storedUpdate.FullRefreshSelection)
}
