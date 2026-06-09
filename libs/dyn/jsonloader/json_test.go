package jsonloader

import (
	"testing"

	"github.com/databricks/cli/libs/dyn/convert"
	"github.com/databricks/databricks-sdk-go/service/jobs"
	"github.com/stretchr/testify/assert"
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
	v, err := LoadJSON([]byte(jsonData), "(inline)")
	assert.NoError(t, err)

	var r jobs.ResetJob
	err = convert.ToTyped(&r, v)
	assert.NoError(t, err)
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
	assert.ErrorContains(t, err, "error decoding JSON at (inline):6:16: invalid character ',' after object key")
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
	assert.ErrorContains(t, err, "error decoding JSON at path/to/file.json:6:28: invalid character ']' looking for beginning of value")
}

const eofData = `
{
    "job_id": 123,
    "new_settings": {
        "name": "xxx",`

func TestJsonLoaderEOF(t *testing.T) {
	_, err := LoadJSON([]byte(eofData), "path/to/file.json")
	assert.ErrorContains(t, err, "unexpected end of JSON input")
}

const mapWithNoBraces = `
"job_id": 123,
"new_settings": {
    "name": "xxx",
    "wrong": "xxx",
}
`

func TestJsonMapWithoutBraces(t *testing.T) {
	_, err := LoadJSON([]byte(mapWithNoBraces), "path/to/file.json")
	assert.ErrorContains(t, err, "error decoding JSON at")
}

const validInline = `["job_id", 123]`

func TestJsonValidInline(t *testing.T) {
	_, err := LoadJSON([]byte(validInline), "path/to/file.json")
	assert.NoError(t, err)
}

func TestJsonLoaderNumbers(t *testing.T) {
	for _, tc := range []struct {
		input    string
		expected any
	}{
		{`123`, int64(123)},
		{`-1`, int64(-1)},
		// Above 2^53: would come back as 123456789012345680 if parsed as float64.
		{`123456789012345678`, int64(123456789012345678)},
		{`-123456789012345678`, int64(-123456789012345678)},
		{`9223372036854775807`, int64(9223372036854775807)},
		// A fraction or exponent keeps the value a float.
		{`2.0`, 2.0},
		{`2.5`, 2.5},
		{`1e3`, 1000.0},
		// Integer literals that overflow int64 fall back to float64.
		{`18446744073709551615`, 1.8446744073709552e+19},
	} {
		v, err := LoadJSON([]byte(tc.input), "(inline)")
		assert.NoError(t, err, tc.input)
		assert.Equal(t, tc.expected, v.AsAny(), tc.input)
	}
}

func TestJsonLoaderNumberOutOfRange(t *testing.T) {
	_, err := LoadJSON([]byte(`1e400`), "(inline)")
	assert.ErrorContains(t, err, "value out of range")
}

const mixedNumbersData = `
{
    "job_id": 123456789012345678,
    "new_settings": {
        "name": "xxx",
        "timeout_seconds": 100
    }
}
`

func TestJsonLoaderMixedNumbersToTyped(t *testing.T) {
	v, err := LoadJSON([]byte(mixedNumbersData), "(inline)")
	require.NoError(t, err)

	var r jobs.ResetJob
	err = convert.ToTyped(&r, v)
	require.NoError(t, err)
	assert.Equal(t, int64(123456789012345678), r.JobId)
	assert.Equal(t, 100, r.NewSettings.TimeoutSeconds)
}
