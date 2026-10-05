package testserver

import (
	"net/http"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestAiTrainingValidateConfigAcceptsCurrentRequest(t *testing.T) {
	response, ok := aiTrainingValidateConfig(Request{Body: []byte(`{
		"task": {
			"experiment": "smoke-test",
			"deployments": [{
				"command_path": "/Workspace/Users/user@example.com/.air/cli_launch/smoke-test/command.sh",
				"compute": {
					"accelerator_type": "GPU_1xH100",
					"accelerator_count": 2,
					"provisioned_capacity_id": "capacity"
				}
			}],
			"priority_class": "CRITICAL",
			"mlflow_run": "run",
			"mlflow_experiment_directory": "/Workspace/Users/user@example.com/experiments",
			"mlflow_artifact_location": "dbfs:/Volumes/main/default/volume/artifacts"
		},
		"run_options": {
			"max_retries": 2,
			"timeout_minutes": 30,
			"idempotency_token": "token",
			"usage_policy_name": "policy",
			"env_variables": {"VALID_NAME": "value"},
			"secrets": {"TOKEN": "scope/key"}
		}
	}`)}).(aiTrainingValidateConfigResponse)
	require.True(t, ok)
	assert.Empty(t, response.Errors)
}

func TestAiTrainingValidateConfigRejectsUnknownFields(t *testing.T) {
	response, ok := aiTrainingValidateConfig(Request{Body: []byte(`{
		"task": {
			"experiment": "smoke-test",
			"deployments": [{
				"command_path": "/Workspace/command.sh",
				"compute": {
					"accelerator_type": "GPU_1xH100",
					"accelerator_count": 1,
					"unexpected": true
				}
			}]
		}
	}`)}).(Response)
	require.True(t, ok)
	assert.Equal(t, http.StatusBadRequest, response.StatusCode)
	body, ok := response.Body.(map[string]string)
	require.True(t, ok)
	assert.Equal(t, "INVALID_PARAMETER_VALUE", body["error_code"])
	assert.Contains(t, body["message"], `unknown field "unexpected"`)
}
