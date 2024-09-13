package jsonloader

import (
	"testing"

	"github.com/databricks/cli/libs/dyn/convert"
	"github.com/databricks/databricks-sdk-go/service/jobs"
	"github.com/stretchr/testify/require"
)

const jsonData = `
{
    "job_id": 123,
    "new_settings": {
        "name": "xxx",
        "email_notifications": {
            "on_start": [],
            "on_success": [],
            "on_failure": []
        },
        "webhook_notifications": {
            "on_start": [],
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
                "task_key": "xxx",
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

func TestJsonLoader(t *testing.T) {
	v, err := LoadJSON([]byte(jsonData))
	require.NoError(t, err)

	var r jobs.ResetJob
	err = convert.ToTyped(&r, v)
	require.NoError(t, err)
}
