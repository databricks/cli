package aircmd

import (
	"context"
	"testing"
	"time"

	"github.com/databricks/cli/libs/testserver"
	"github.com/databricks/databricks-sdk-go"
	"github.com/databricks/databricks-sdk-go/service/jobs"
	"github.com/databricks/databricks-sdk-go/service/ml"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestGrantJobPermissions(t *testing.T) {
	server := testserver.New(t)
	t.Cleanup(server.Close)

	var requestBody string
	server.Handle("PATCH", "/api/2.0/permissions/jobs/123", func(req testserver.Request) any {
		requestBody = string(req.Body)
		return map[string]any{}
	})

	w, err := databricks.NewWorkspaceClient(&databricks.Config{Host: server.URL, Token: "token"})
	require.NoError(t, err)
	err = grantJobPermissions(t.Context(), w, "123", []permission{
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
	}`, requestBody)
}

func TestExperimentPermissionLevel(t *testing.T) {
	cases := []struct {
		job  string
		want ml.ExperimentPermissionLevel
		ok   bool
	}{
		{"CAN_VIEW", ml.ExperimentPermissionLevelCanRead, true},
		{"CAN_MANAGE_RUN", ml.ExperimentPermissionLevelCanEdit, true},
		{"CAN_MANAGE", ml.ExperimentPermissionLevelCanManage, true},
		{"IS_OWNER", ml.ExperimentPermissionLevelCanManage, true},
		{"CAN_ATTACH_TO", "", false},
	}
	for _, tc := range cases {
		got, ok := experimentPermissionLevel(tc.job)
		assert.Equal(t, tc.ok, ok, tc.job)
		assert.Equal(t, tc.want, got, tc.job)
	}
}

func TestGrantExperimentPermissions(t *testing.T) {
	server := testserver.New(t)
	t.Cleanup(server.Close)

	server.Handle("GET", "/api/2.0/mlflow/experiments/get-by-name", func(req testserver.Request) any {
		assert.Equal(t, "/Users/alice@example.com/submit-smoke", req.URL.Query().Get("experiment_name"))
		return map[string]any{"experiment": map[string]any{"experiment_id": "42"}}
	})
	var requestBody string
	server.Handle("PATCH", "/api/2.0/permissions/experiments/42", func(req testserver.Request) any {
		requestBody = string(req.Body)
		return map[string]any{}
	})

	w, err := databricks.NewWorkspaceClient(&databricks.Config{Host: server.URL, Token: "token"})
	require.NoError(t, err)
	err = grantExperimentPermissions(t.Context(), w, "/Users/alice@example.com/submit-smoke", []permission{
		{UserName: new("alice@example.com"), Level: "CAN_MANAGE"},
		{GroupName: new("data-team"), Level: "CAN_VIEW"},
		{ServicePrincipalName: new("training-sp"), Level: "CAN_MANAGE_RUN"},
	})
	require.NoError(t, err)

	// Job levels are mapped onto their nearest experiment equivalents.
	assert.JSONEq(t, `{
		"access_control_list": [
			{"user_name": "alice@example.com", "permission_level": "CAN_MANAGE"},
			{"group_name": "data-team", "permission_level": "CAN_READ"},
			{"service_principal_name": "training-sp", "permission_level": "CAN_EDIT"}
		]
	}`, requestBody)
}

func TestResolveExperimentIDTimesOut(t *testing.T) {
	server := testserver.New(t)
	t.Cleanup(server.Close)

	server.Handle("GET", "/api/2.0/mlflow/experiments/get-by-name", func(req testserver.Request) any {
		return testserver.Response{StatusCode: 404, Body: `{"error_code": "RESOURCE_DOES_NOT_EXIST", "message": "not found"}`}
	})

	w, err := databricks.NewWorkspaceClient(&databricks.Config{Host: server.URL, Token: "token"})
	require.NoError(t, err)

	// An already-expired context gives up on the first miss without polling.
	ctx, cancel := context.WithTimeout(t.Context(), time.Nanosecond)
	defer cancel()
	_, err = resolveExperimentID(ctx, w, "/Users/alice@example.com/missing")
	require.Error(t, err)
}

func TestSubmittedExperimentName(t *testing.T) {
	assert.Empty(t, submittedExperimentName(&jobs.Run{}))
	assert.Empty(t, submittedExperimentName(&jobs.Run{Tasks: []jobs.RunTask{{}}}))

	run := &jobs.Run{Tasks: []jobs.RunTask{{
		GenAiComputeTask: &jobs.GenAiComputeTask{MlflowExperimentName: "/Users/alice@example.com/submit-smoke"},
	}}}
	assert.Equal(t, "/Users/alice@example.com/submit-smoke", submittedExperimentName(run))
}
