package flags

import (
	"fmt"
	"os"
	"path"
	"testing"

	"github.com/databricks/databricks-sdk-go/service/jobs"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestJsonFlagEmpty(t *testing.T) {
	var body JsonFlag

	var request any
	err := body.Unmarshal(&request)

	assert.Equal(t, "JSON (0 bytes)", body.String())
	assert.NoError(t, err)
	assert.Nil(t, request)
}

func TestJsonFlagInline(t *testing.T) {
	var body JsonFlag

	err := body.Set(`{"foo": "bar"}`)
	assert.NoError(t, err)

	var request any
	err = body.Unmarshal(&request)
	assert.NoError(t, err)

	assert.Equal(t, "JSON (14 bytes)", body.String())
	assert.Equal(t, map[string]any{"foo": "bar"}, request)
}

func TestJsonFlagError(t *testing.T) {
	var body JsonFlag

	err := body.Set(`{"foo":`)
	assert.NoError(t, err)

	var request any
	err = body.Unmarshal(&request)
	assert.EqualError(t, err, "unexpected end of JSON input")
	assert.Equal(t, "JSON (7 bytes)", body.String())
}

func TestJsonFlagFile(t *testing.T) {
	var body JsonFlag
	var request any

	var fpath string
	var payload = []byte(`{"hello": "world"}`)

	{
		f, err := os.Create(path.Join(t.TempDir(), "file"))
		require.NoError(t, err)
		f.Write(payload)
		f.Close()
		fpath = f.Name()
	}

	err := body.Set(fmt.Sprintf("@%s", fpath))
	require.NoError(t, err)

	err = body.Unmarshal(&request)
	require.NoError(t, err)

	assert.Equal(t, map[string]interface{}{"hello": "world"}, request)
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
            "no_alert_for_skipped_runs": true,
            "no_alert_for_canceled_runs": true
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

	err = body.Unmarshal(&r)
	require.NoError(t, err)

	assert.Equal(t, int64(123), r.JobId)
	assert.Equal(t, "new job", r.NewSettings.Name)
	assert.Equal(t, 0, r.NewSettings.TimeoutSeconds)
	assert.Equal(t, 1, r.NewSettings.MaxConcurrentRuns)
	assert.Equal(t, 1, len(r.NewSettings.Tasks))
	assert.Equal(t, "new task", r.NewSettings.Tasks[0].TaskKey)
	assert.Equal(t, 0, r.NewSettings.Tasks[0].TimeoutSeconds)
	assert.Equal(t, 0, r.NewSettings.Tasks[0].MaxRetries)
	assert.Equal(t, 0, r.NewSettings.Tasks[0].MinRetryIntervalMillis)
	assert.Equal(t, true, r.NewSettings.Tasks[0].RetryOnTimeout)
}

const incorrectJsonData = `
{
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

	err = body.Unmarshal(&r)
	require.ErrorContains(t, err, `json input error:
- unknown field: settings`)
}
