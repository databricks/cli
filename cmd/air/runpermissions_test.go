package aircmd

import (
	"context"
	"testing"
	"time"

	"github.com/databricks/cli/libs/sdkclients"
	"github.com/databricks/cli/libs/sdktypes/accessmanagement/v1"
	"github.com/databricks/cli/libs/sdktypes/jobs/v2"
	"github.com/databricks/cli/libs/testserver"
	"github.com/databricks/databricks-sdk-go/config"
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

	w, err := sdkclients.New(t.Context(), &config.Config{Host: server.URL, Token: "token"})
	require.NoError(t, err)
	err = grantJobPermissions(t.Context(), w, "123", []permission{
		{UserName: new("alice@example.com"), Level: "CAN_MANAGE"},
		{GroupName: new("data-team"), Level: "CAN_VIEW"},
		{ServicePrincipalName: new("training-sp"), Level: "CAN_MANAGE_RUN"},
	})
	require.NoError(t, err)

	assert.JSONEq(t, `{
		"request_object_type": "jobs",
		"request_object_id": "123",
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
		want accessmanagement.PermissionLevel
		ok   bool
	}{
		{"CAN_VIEW", accessmanagement.PermissionLevel_CanRead, true},
		{"CAN_MANAGE_RUN", accessmanagement.PermissionLevel_CanEdit, true},
		{"CAN_MANAGE", permissionLevelCanManage, true},
		{"IS_OWNER", permissionLevelCanManage, true},
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

	// The experiment id is the run's own, resolved from runs/get-output keyed by
	// the task run id.
	server.Handle("GET", "/api/2.2/jobs/runs/get-output", func(req testserver.Request) any {
		assert.Equal(t, "456", req.URL.Query().Get("run_id"))
		return map[string]any{"ai_runtime_task_output": map[string]any{"mlflow_experiment_id": "42"}}
	})
	var requestBody string
	server.Handle("PATCH", "/api/2.0/permissions/experiments/42", func(req testserver.Request) any {
		requestBody = string(req.Body)
		return map[string]any{}
	})

	w, err := sdkclients.New(t.Context(), &config.Config{Host: server.URL, Token: "token"})
	require.NoError(t, err)
	run := &jobs.GetRunResponse{Tasks: []jobs.RunTask{{RunId: new(int64(456)), AiRuntimeTask: &jobs.AiRuntimeTask{Experiment: new("submit-smoke")}}}}
	err = grantExperimentPermissions(t.Context(), w, run, []permission{
		{UserName: new("alice@example.com"), Level: "CAN_MANAGE"},
		{GroupName: new("data-team"), Level: "CAN_VIEW"},
		{ServicePrincipalName: new("training-sp"), Level: "CAN_MANAGE_RUN"},
	})
	require.NoError(t, err)

	// Job levels are mapped onto their nearest experiment equivalents.
	assert.JSONEq(t, `{
		"request_object_type": "experiments",
		"request_object_id": "42",
		"access_control_list": [
			{"user_name": "alice@example.com", "permission_level": "CAN_MANAGE"},
			{"group_name": "data-team", "permission_level": "CAN_READ"},
			{"service_principal_name": "training-sp", "permission_level": "CAN_EDIT"}
		]
	}`, requestBody)
}

func TestResolveSubmittedExperimentIDTimesOut(t *testing.T) {
	server := testserver.New(t)
	t.Cleanup(server.Close)

	// get-output has no experiment id yet.
	server.Handle("GET", "/api/2.2/jobs/runs/get-output", func(req testserver.Request) any {
		return map[string]any{}
	})

	w, err := sdkclients.New(t.Context(), &config.Config{Host: server.URL, Token: "token"})
	require.NoError(t, err)

	// An already-expired context gives up on the first miss without polling.
	ctx, cancel := context.WithTimeout(t.Context(), time.Nanosecond)
	defer cancel()
	run := &jobs.GetRunResponse{Tasks: []jobs.RunTask{{RunId: new(int64(456))}}}
	assert.Empty(t, resolveSubmittedExperimentID(ctx, w, run))
}

// TestApplySubmittedPermissionsAiRuntimeTask covers the real submit path: the run
// reports an ai_runtime_task (not a gen_ai_compute_task), so the experiment id
// comes from runs/get-output and both job and experiment permissions are granted.
func TestApplySubmittedPermissionsAiRuntimeTask(t *testing.T) {
	server := testserver.New(t)
	t.Cleanup(server.Close)

	server.Handle("GET", "/api/2.2/jobs/runs/get", func(req testserver.Request) any {
		return map[string]any{
			"run_id": 555,
			"job_id": 123,
			"tasks": []map[string]any{{
				"run_id":          456,
				"ai_runtime_task": map[string]any{"experiment": "submit-smoke"},
			}},
		}
	})
	server.Handle("GET", "/api/2.2/jobs/runs/get-output", func(req testserver.Request) any {
		return map[string]any{"ai_runtime_task_output": map[string]any{"mlflow_experiment_id": "42"}}
	})
	var jobBody, experimentBody string
	server.Handle("PATCH", "/api/2.0/permissions/jobs/123", func(req testserver.Request) any {
		jobBody = string(req.Body)
		return map[string]any{}
	})
	server.Handle("PATCH", "/api/2.0/permissions/experiments/42", func(req testserver.Request) any {
		experimentBody = string(req.Body)
		return map[string]any{}
	})

	w, err := sdkclients.New(t.Context(), &config.Config{Host: server.URL, Token: "token"})
	require.NoError(t, err)
	applySubmittedPermissions(t.Context(), w, 555, []permission{
		{UserName: new("alice@example.com"), Level: "CAN_MANAGE"},
	})

	assert.JSONEq(t, `{"request_object_type":"jobs","request_object_id":"123","access_control_list":[{"user_name":"alice@example.com","permission_level":"CAN_MANAGE"}]}`, jobBody)
	assert.JSONEq(t, `{"request_object_type":"experiments","request_object_id":"42","access_control_list":[{"user_name":"alice@example.com","permission_level":"CAN_MANAGE"}]}`, experimentBody)
}
