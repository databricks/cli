package jsonloader

import (
	"testing"

	"github.com/databricks/cli/libs/dyn/convert"
	"github.com/databricks/cli/libs/dyn/dynassert"
	"github.com/databricks/databricks-sdk-go/service/jobs"
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
	v, err := LoadJSON([]byte(jsonData), "(inline)")
	dynassert.NoError(t, err)

	var r jobs.ResetJob
	err = convert.ToTyped(&r, v)
	dynassert.NoError(t, err)
}

const malformedMap = `
{
    "job_id": 123,
    "new_settings": {
        "name": "xxx",
        "wrong",
    }
}
`

func TestJsonLoaderMalformedMap(t *testing.T) {
	_, err := LoadJSON([]byte(malformedMap), "(inline)")
	dynassert.ErrorContains(t, err, "error decoding JSON at (inline):6:16: invalid character ',' after object key")
}

const malformedArray = `
{
    "job_id": 123,
    "new_settings": {
        "name": "xxx",
        "tasks": [1, "asd",]
    }
}`

func TestJsonLoaderMalformedArray(t *testing.T) {
	_, err := LoadJSON([]byte(malformedArray), "path/to/file.json")
	dynassert.ErrorContains(t, err, "error decoding JSON at path/to/file.json:6:28: invalid character ']' looking for beginning of value")
}

const eofData = `
{
    "job_id": 123,
    "new_settings": {
        "name": "xxx",`

func TestJsonLoaderEOF(t *testing.T) {
	_, err := LoadJSON([]byte(eofData), "path/to/file.json")
	dynassert.ErrorContains(t, err, "unexpected end of JSON input")
}
