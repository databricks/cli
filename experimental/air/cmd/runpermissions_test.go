package aircmd

import (
	"encoding/json"
	"net/http"
	"testing"

	"github.com/databricks/cli/libs/testserver"
	"github.com/databricks/databricks-sdk-go"
	"github.com/databricks/databricks-sdk-go/service/iam"
	"github.com/databricks/databricks-sdk-go/service/ml"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestGetOrCreateMLflowExperimentCreatesWithArtifactLocation(t *testing.T) {
	server := testserver.New(t)
	t.Cleanup(server.Close)

	server.Handle("GET", "/api/2.0/mlflow/experiments/get-by-name", func(req testserver.Request) any {
		return testserver.Response{
			StatusCode: http.StatusNotFound,
			Body: map[string]string{
				"error_code": "RESOURCE_DOES_NOT_EXIST",
				"message":    "experiment does not exist",
			},
		}
	})
	server.Handle("POST", "/api/2.0/mlflow/experiments/create", func(req testserver.Request) any {
		var got ml.CreateExperiment
		require.NoError(t, json.Unmarshal(req.Body, &got))
		assert.Equal(t, "/Users/alice@example.com/training", got.Name)
		assert.Equal(t, "dbfs:/Volumes/main/default/artifacts", got.ArtifactLocation)
		return ml.CreateExperimentResponse{ExperimentId: "exp-456"}
	})

	w, err := databricks.NewWorkspaceClient(&databricks.Config{Host: server.URL, Token: "token"})
	require.NoError(t, err)
	experimentID, err := getOrCreateMLflowExperiment(t.Context(), w, "/Users/alice@example.com/training", "dbfs:/Volumes/main/default/artifacts")
	require.NoError(t, err)
	assert.Equal(t, "exp-456", experimentID)
}

func TestExperimentPermissionLevel(t *testing.T) {
	tests := []struct {
		job        string
		experiment iam.PermissionLevel
	}{
		{"CAN_VIEW", iam.PermissionLevelCanRead},
		{"CAN_MANAGE_RUN", iam.PermissionLevelCanEdit},
		{"CAN_MANAGE", iam.PermissionLevelCanManage},
		{"IS_OWNER", iam.PermissionLevelCanManage},
	}
	for _, tt := range tests {
		t.Run(tt.job, func(t *testing.T) {
			got, err := experimentPermissionLevel(tt.job)
			require.NoError(t, err)
			assert.Equal(t, tt.experiment, got)
		})
	}
}

func TestGrantWorkloadPermissionsRejectsUnsupportedLevelBeforeRequests(t *testing.T) {
	server := testserver.New(t)
	t.Cleanup(server.Close)

	w, err := databricks.NewWorkspaceClient(&databricks.Config{Host: server.URL, Token: "token"})
	require.NoError(t, err)
	server.RequestCallback = func(req *testserver.Request) {
		t.Errorf("unexpected permission request: %s %s", req.Method, req.URL.Path)
	}
	err = grantWorkloadPermissions(t.Context(), w, "123", "exp-456", []permission{
		{GroupName: new("data-team"), Level: "CAN_USE"},
	})
	require.EqualError(t, err, `unsupported AIR permission level "CAN_USE"`)
}

func TestGrantWorkloadPermissionsUpdatesJobAndExperiment(t *testing.T) {
	server := testserver.New(t)
	t.Cleanup(server.Close)

	requests := make(map[string]string)
	for _, objectPath := range []string{"jobs/123", "experiments/exp-456"} {
		server.Handle("PATCH", "/api/2.0/permissions/"+objectPath, func(req testserver.Request) any {
			requests[objectPath] = string(req.Body)
			return map[string]any{}
		})
	}

	w, err := databricks.NewWorkspaceClient(&databricks.Config{Host: server.URL, Token: "token"})
	require.NoError(t, err)
	err = grantWorkloadPermissions(t.Context(), w, "123", "exp-456", []permission{
		{UserName: new("alice@example.com"), Level: "CAN_MANAGE"},
		{GroupName: new("data-team"), Level: "CAN_VIEW"},
		{ServicePrincipalName: new("training-sp"), Level: "CAN_MANAGE_RUN"},
	})
	require.NoError(t, err)

	assert.JSONEq(t, `{
		"access_control_list": [
			{"user_name": "alice@example.com", "permission_level": "CAN_MANAGE"},
			{"group_name": "data-team", "permission_level": "CAN_VIEW"},
			{"service_principal_name": "training-sp", "permission_level": "CAN_MANAGE_RUN"}
		]
	}`, requests["jobs/123"])
	assert.JSONEq(t, `{
		"access_control_list": [
			{"user_name": "alice@example.com", "permission_level": "CAN_MANAGE"},
			{"group_name": "data-team", "permission_level": "CAN_READ"},
			{"service_principal_name": "training-sp", "permission_level": "CAN_EDIT"}
		]
	}`, requests["experiments/exp-456"])
}
