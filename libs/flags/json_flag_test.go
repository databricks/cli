package flags

import (
	"os"
	"path"
	"testing"

	"github.com/databricks/cli/libs/diag"
	"github.com/databricks/cli/libs/dyn"
	"github.com/databricks/databricks-sdk-go/service/jobs"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestJsonFlagEmpty(t *testing.T) {
	var body JsonFlag

	var request any
	diags := body.Unmarshal(&request)

	assert.Equal(t, "JSON (0 bytes)", body.String())
	assert.NoError(t, diags.Error())
	assert.Empty(t, diags)
	assert.Nil(t, request)
}

func TestJsonFlagInline(t *testing.T) {
	var body JsonFlag

	err := body.Set(`{"foo": "bar"}`)
	assert.NoError(t, err)

	var request any
	diags := body.Unmarshal(&request)
	assert.NoError(t, diags.Error())
	assert.Empty(t, diags)

	assert.Equal(t, "JSON (14 bytes)", body.String())
	assert.Equal(t, map[string]any{"foo": "bar"}, request)
}

func TestJsonFlagError(t *testing.T) {
	var body JsonFlag

	err := body.Set(`{"foo":`)
	assert.NoError(t, err)

	var request any
	diags := body.Unmarshal(&request)
	assert.EqualError(t, diags.Error(), "error decoding JSON at (inline):1:8: unexpected end of JSON input")
	assert.Equal(t, "JSON (7 bytes)", body.String())
}

func TestJsonFlagFile(t *testing.T) {
	var body JsonFlag
	var request any

	var fpath string
	payload := []byte(`{"foo": "bar"}`)

	{
		f, err := os.Create(path.Join(t.TempDir(), "file"))
		require.NoError(t, err)
		_, err = f.Write(payload)
		require.NoError(t, err)
		f.Close()
		fpath = f.Name()
	}

	err := body.Set("@" + fpath)
	require.NoError(t, err)

	diags := body.Unmarshal(&request)
	assert.NoError(t, diags.Error())
	assert.Empty(t, diags)

	assert.Equal(t, map[string]any{"foo": "bar"}, request)
}

const jsonData = `
{
    "job_id": 123,
    "new_settings": {
        "name": "new job",
        "email_notifications": {
            "on_start": [],
            "on_success": [],
            "on_failure": []
        },
        "notification_settings": {
            "no_alert_for_canceled_runs": false
        },
        "timeout_seconds": 0,
        "max_concurrent_runs": 1,
        "tasks": [
            {
                "task_key": "new task",
                "email_notifications": {},
                "notification_settings": {},
                "timeout_seconds": 0,
                "max_retries": 0,
                "min_retry_interval_millis": 0,
                "retry_on_timeout": "true"
            }
        ]
    }
}
`

func TestJsonUnmarshalForRequest(t *testing.T) {
	var body JsonFlag

	var r jobs.ResetJob
	err := body.Set(jsonData)
	require.NoError(t, err)

	diags := body.Unmarshal(&r)
	assert.NoError(t, diags.Error())
	assert.Empty(t, diags)

	assert.Equal(t, int64(123), r.JobId)
	assert.Equal(t, "new job", r.NewSettings.Name)
	assert.Equal(t, 0, r.NewSettings.TimeoutSeconds)
	assert.Equal(t, 1, r.NewSettings.MaxConcurrentRuns)
	assert.Len(t, r.NewSettings.Tasks, 1)
	assert.Equal(t, "new task", r.NewSettings.Tasks[0].TaskKey)
	assert.Equal(t, 0, r.NewSettings.Tasks[0].TimeoutSeconds)
	assert.Equal(t, 0, r.NewSettings.Tasks[0].MaxRetries)
	assert.Equal(t, 0, r.NewSettings.Tasks[0].MinRetryIntervalMillis)
	assert.True(t, r.NewSettings.Tasks[0].RetryOnTimeout)
}

const incorrectJsonData = `{
    "job_id": 123,
    "settings": {
        "name": "new job",
        "email_notifications": {
            "on_start": [],
            "on_success": [],
            "on_failure": []
        },
        "notification_settings": {
            "no_alert_for_skipped_runs": true,
            "no_alert_for_canceled_runs": true
        },
        "timeout_seconds": {},
        "max_concurrent_runs": {},
        "tasks": [
            {
                "task_key": "new task",
                "email_notifications": {},
                "notification_settings": {},
                "timeout_seconds": 0,
                "max_retries": 0,
                "min_retry_interval_millis": 0,
                "retry_on_timeout": "true"
            }
        ]
    }
}
`

func TestJsonUnmarshalRequestMismatch(t *testing.T) {
	var body JsonFlag

	var r jobs.ResetJob
	err := body.Set(incorrectJsonData)
	require.NoError(t, err)

	diags := body.Unmarshal(&r)
	assert.NoError(t, diags.Error())
	assert.NotEmpty(t, diags)

	assert.Contains(t, diags, diag.Diagnostic{
		Severity: diag.Warning,
		Summary:  "unknown field: settings",
		Locations: []dyn.Location{
			{
				File:   "(inline)",
				Line:   3,
				Column: 6,
			},
		},
		Paths: []dyn.Path{{}},
	})
}

const wrontTypeJsonData = `{
    "job_id": 123,
    "new_settings": {
        "name": "new job",
        "email_notifications": {
            "on_start": [],
            "on_success": [],
            "on_failure": []
        },
        "notification_settings": {
            "no_alert_for_skipped_runs": true,
            "no_alert_for_canceled_runs": true
        },
        "timeout_seconds": "wrong_type",
        "max_concurrent_runs": {},
        "tasks": [
            {
                "task_key": "new task",
                "email_notifications": {},
                "notification_settings": {},
                "timeout_seconds": 0,
                "max_retries": 0,
                "min_retry_interval_millis": 0,
                "retry_on_timeout": "true"
            }
        ]
    }
}
`

func TestJsonUnmarshalWrongTypeReportsCorrectLocation(t *testing.T) {
	var body JsonFlag

	var r jobs.ResetJob
	err := body.Set(`{
    "job_id": [1, 2, 3]
}
`)
	require.NoError(t, err)

	diags := body.Unmarshal(&r)
	assert.NoError(t, diags.Error())
	assert.NotEmpty(t, diags)

	assert.Contains(t, diags, diag.Diagnostic{
		Severity: diag.Warning,
		Summary:  "expected int, found sequence",
		Locations: []dyn.Location{
			{
				File:   "(inline)",
				Line:   2,
				Column: 15,
			},
		},
		Paths: []dyn.Path{dyn.NewPath(dyn.Key("job_id"))},
	})
}

func TestJsonUnmarshalArrayInsteadOfIntReportsCorrectLocation(t *testing.T) {
	var body JsonFlag

	var r jobs.ResetJob
	err := body.Set(wrontTypeJsonData)
	require.NoError(t, err)

	diags := body.Unmarshal(&r)
	assert.NoError(t, diags.Error())
	assert.NotEmpty(t, diags)

	assert.Contains(t, diags, diag.Diagnostic{
		Severity: diag.Warning,
		Summary:  "cannot parse \"wrong_type\" as an integer",
		Locations: []dyn.Location{
			{
				File:   "(inline)",
				Line:   14,
				Column: 40,
			},
		},
		Paths: []dyn.Path{dyn.NewPath(dyn.Key("new_settings"), dyn.Key("timeout_seconds"))},
	})
}

func TestJsonUnmarshalForRequestWithForceSendFields(t *testing.T) {
	var body JsonFlag

	var r jobs.ResetJob
	err := body.Set(jsonData)
	require.NoError(t, err)

	diags := body.Unmarshal(&r)
	assert.NoError(t, diags.Error())
	assert.Empty(t, diags)

	assert.False(t, r.NewSettings.NotificationSettings.NoAlertForSkippedRuns)
	assert.False(t, r.NewSettings.NotificationSettings.NoAlertForCanceledRuns)
	assert.NotContains(t, r.NewSettings.NotificationSettings.ForceSendFields, "NoAlertForSkippedRuns")
	assert.Contains(t, r.NewSettings.NotificationSettings.ForceSendFields, "NoAlertForCanceledRuns")
}
