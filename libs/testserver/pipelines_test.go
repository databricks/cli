package testserver

import (
	"strings"
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

func TestPipelineCreate_RequiresRunAsCreateTableOnSchema(t *testing.T) {
	const pipeline = `{"name": "p", "catalog": "main", "schema": "src", "run_as": {"service_principal_name": "sp"}}`

	tests := []struct {
		name               string
		createSchema       bool
		grants             map[string]string
		pipeline           string
		expectedStatusCode int
	}{
		{
			name:               "no grants",
			createSchema:       true,
			pipeline:           pipeline,
			expectedStatusCode: 403,
		},
		{
			name:               "no grants with target",
			createSchema:       true,
			pipeline:           `{"name": "p", "catalog": "main", "target": "src", "run_as": {"service_principal_name": "sp"}}`,
			expectedStatusCode: 403,
		},
		{
			name:               "grants without CREATE_TABLE",
			createSchema:       true,
			grants:             map[string]string{"schema/main.src": "USE_SCHEMA"},
			pipeline:           pipeline,
			expectedStatusCode: 403,
		},
		{
			name:         "CREATE_TABLE on schema",
			createSchema: true,
			grants:       map[string]string{"schema/main.src": "CREATE_TABLE"},
			pipeline:     pipeline,
		},
		{
			name:         "ALL_PRIVILEGES on schema",
			createSchema: true,
			grants:       map[string]string{"schema/main.src": "ALL_PRIVILEGES"},
			pipeline:     pipeline,
		},
		{
			name:         "CREATE_TABLE on catalog",
			createSchema: true,
			grants:       map[string]string{"catalog/main": "CREATE_TABLE"},
			pipeline:     pipeline,
		},
		{
			name:         "run_as schema owner",
			createSchema: true,
			pipeline:     `{"name": "p", "catalog": "main", "schema": "src", "run_as": {"user_name": "` + TestUser.UserName + `"}}`,
		},
		{
			name:         "no run_as",
			createSchema: true,
			pipeline:     `{"name": "p", "catalog": "main", "schema": "src"}`,
		},
		{
			name:     "schema not created in workspace",
			pipeline: pipeline,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			// A user token, so that the schema owner is TestUser.
			workspace := NewFakeWorkspace("http://test", UserNameTokenPrefix+"123")

			if tt.createSchema {
				response := workspace.SchemasCreate(Request{Body: []byte(`{"catalog_name": "main", "name": "src"}`)})
				require.Equal(t, 0, response.StatusCode)
			}

			for securable, privilege := range tt.grants {
				securableType, fullName, _ := strings.Cut(securable, "/")
				response := workspace.GrantsUpdate(Request{
					Body: []byte(`{"changes": [{"principal": "sp", "add": ["` + privilege + `"]}]}`),
				}, securableType, fullName)
				require.Equal(t, 0, response.StatusCode)
			}

			response := workspace.PipelineCreate(Request{Body: []byte(tt.pipeline)})
			assert.Equal(t, tt.expectedStatusCode, response.StatusCode)
			if tt.expectedStatusCode == 403 {
				assert.Equal(t, map[string]string{
					"error_code": "PERMISSION_DENIED",
					"message":    "User does not have CREATE TABLE on Schema 'main.src'.",
				}, response.Body)
			}
		})
	}
}

func TestPipelineStartUpdate_HandlesNonExistentPipeline(t *testing.T) {
	workspace := NewFakeWorkspace("http://test", "dbapi123")

	response := workspace.PipelineStartUpdate("non-existent-pipeline")
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

	startResponse := workspace.PipelineStartUpdate(pipelineId)
	assert.Equal(t, 0, startResponse.StatusCode)

	stopResponse := workspace.PipelineStop(pipelineId)
	assert.Equal(t, 0, stopResponse.StatusCode)

	stopBody, ok := stopResponse.Body.(pipelines.GetPipelineResponse)
	require.True(t, ok)
	assert.Equal(t, pipelineId, stopBody.PipelineId)
	assert.Equal(t, pipelines.PipelineStateIdle, stopBody.State)
}
