package testserver

import (
	"testing"

	"github.com/databricks/databricks-sdk-go/service/pipelines"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func createTestPipeline(t *testing.T, workspace *FakeWorkspace) string {
	createReq := Request{
		Body: []byte(`{
			"name": "Test Pipeline",
			"storage": "dbfs:/pipelines/test-pipeline"
		}`),
	}

	createResponse := workspace.PipelineCreate(createReq)
	// StatusCode 0 gets converted to 200 by normalizeResponse in the server
	require.Equal(t, 0, createResponse.StatusCode)

	createPipelineResponse, ok := createResponse.Body.(pipelines.CreatePipelineResponse)
	require.True(t, ok)
	return createPipelineResponse.PipelineId
}

func TestPipelineCreate_RejectsDottedTargetSchemaName(t *testing.T) {
	workspace := NewFakeWorkspace("http://test", "dbapi123")

	response := workspace.PipelineCreate(Request{
		Body: []byte(`{"name": "p", "catalog": "main", "target": "main.test_schema"}`),
	})
	assert.Equal(t, 400, response.StatusCode)

	body, ok := response.Body.(map[string]string)
	require.True(t, ok)
	assert.Equal(t, "INVALID_PARAMETER_VALUE", body["error_code"])
	assert.Contains(t, body["message"], `target_schema_name "main.test_schema"`)
}

func TestPipelineCreate_AllowsSingleSegmentTargetSchemaName(t *testing.T) {
	workspace := NewFakeWorkspace("http://test", "dbapi123")

	response := workspace.PipelineCreate(Request{
		Body: []byte(`{"name": "p", "catalog": "main", "target": "test_schema"}`),
	})
	assert.Equal(t, 0, response.StatusCode)
}

func TestPipelineStartUpdate_HandlesNonExistentPipeline(t *testing.T) {
	workspace := NewFakeWorkspace("http://test", "dbapi123")

	response := workspace.PipelineStartUpdate(Request{Body: []byte(`{}`)}, "non-existent-pipeline")
	assert.Equal(t, 404, response.StatusCode)
	assert.Contains(t, response.Body.(map[string]string)["message"], "The specified pipeline non-existent-pipeline was not found")
}

func TestPipelineGetUpdate_HandlesNonExistent(t *testing.T) {
	workspace := NewFakeWorkspace("http://test", "dbapi123")

	response := workspace.PipelineGetUpdate("non-existent-pipeline", "some-update-id")
	assert.Equal(t, 404, response.StatusCode)

	pipelineId := createTestPipeline(t, workspace)

	response = workspace.PipelineGetUpdate(pipelineId, "non-existent-update")
	assert.Equal(t, 404, response.StatusCode)
	assert.Contains(t, response.Body.(map[string]string)["message"], "The specified update non-existent-update was not found")
}

func TestPipelineStop_AfterUpdate(t *testing.T) {
	workspace := NewFakeWorkspace("http://test", "dbapi123")

	pipelineId := createTestPipeline(t, workspace)

	startResponse := workspace.PipelineStartUpdate(Request{Body: []byte(`{}`)}, pipelineId)
	assert.Equal(t, 0, startResponse.StatusCode)

	stopResponse := workspace.PipelineStop(pipelineId)
	assert.Equal(t, 0, stopResponse.StatusCode)

	stopBody, ok := stopResponse.Body.(pipelines.GetPipelineResponse)
	require.True(t, ok)
	assert.Equal(t, pipelineId, stopBody.PipelineId)
	assert.Equal(t, pipelines.PipelineStateIdle, stopBody.State)
}

func TestPipelineGetUpdatePreservesRefreshSelections(t *testing.T) {
	workspace := NewFakeWorkspace("http://workspace.test", "dbapi123")
	pipelineId := createTestPipeline(t, workspace)
	tests := []struct {
		name string
		body string
		want pipelines.UpdateInfo
	}{
		{
			name: "file selections",
			body: `{"refresh_selection_by_path":["/Workspace/src/orders.py","/Workspace/src/returns.py"],"full_refresh_selection_by_path":["/Workspace/src/customers.sql"]}`,
			want: pipelines.UpdateInfo{
				RefreshSelectionByPath:     []string{"/Workspace/src/orders.py", "/Workspace/src/returns.py"},
				FullRefreshSelectionByPath: []string{"/Workspace/src/customers.sql"},
			},
		},
		{
			name: "table selections",
			body: `{"refresh_selection":["orders"],"full_refresh_selection":["customers"]}`,
			want: pipelines.UpdateInfo{
				RefreshSelection:     []string{"orders"},
				FullRefreshSelection: []string{"customers"},
			},
		},
	}
	for i := range tests {
		response := workspace.PipelineStartUpdate(Request{Body: []byte(tests[i].body)}, pipelineId)
		require.Zero(t, response.StatusCode)
		tests[i].want.UpdateId = response.Body.(pipelines.StartUpdateResponse).UpdateId
		tests[i].want.State = pipelines.UpdateInfoStateCompleted
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			response := workspace.PipelineGetUpdate(pipelineId, tt.want.UpdateId)
			require.Zero(t, response.StatusCode)
			assert.Equal(t, &tt.want, response.Body.(pipelines.GetUpdateResponse).Update)
		})
	}
}
