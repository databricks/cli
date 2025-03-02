package telemetry

import (
	"context"
	"encoding/json"
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/databricks/cli/internal/testutil"
	"github.com/databricks/cli/libs/telemetry/protos"
	"github.com/databricks/cli/libs/testserver"
	"github.com/databricks/databricks-sdk-go/apierr"
	"github.com/google/uuid"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func configureStdin(t *testing.T) {
	logs := []protos.FrontendLog{
		{
			FrontendLogEventID: uuid.New().String(),
			Entry: protos.FrontendLogEntry{
				DatabricksCliLog: protos.DatabricksCliLog{
					CliTestEvent: &protos.CliTestEvent{Name: protos.DummyCliEnumValue1},
				},
			},
		},
		{
			FrontendLogEventID: uuid.New().String(),
			Entry: protos.FrontendLogEntry{
				DatabricksCliLog: protos.DatabricksCliLog{
					CliTestEvent: &protos.CliTestEvent{Name: protos.DummyCliEnumValue2},
				},
			},
		},
	}

	processIn := UploadConfig{
		Logs: logs,
	}

	b, err := json.Marshal(processIn)
	require.NoError(t, err)

	tmpDir := t.TempDir()
	testutil.WriteFile(t, filepath.Join(tmpDir, "stdin"), string(b))

	f, err := os.OpenFile(filepath.Join(tmpDir, "stdin"), os.O_RDONLY, 0o644)
	require.NoError(t, err)

	// Redirect stdin to the file containing the telemetry logs.
	old := os.Stdin
	os.Stdin = f
	t.Cleanup(func() {
		f.Close()
		os.Stdin = old
	})
}

func TestTelemetryUploadRetriesOnPartialSuccess(t *testing.T) {
	server := testserver.New(t)
	t.Cleanup(server.Close)

	count := 0
	server.Handle("POST", "/telemetry-ext", func(req testserver.Request) any {
		count++
		if count == 1 {
			return ResponseBody{
				NumProtoSuccess: 1,
			}
		}
		if count == 2 {
			return ResponseBody{
				NumProtoSuccess: 2,
			}
		}
		return nil
	})

	t.Setenv("DATABRICKS_HOST", server.URL)
	t.Setenv("DATABRICKS_TOKEN", "token")

	configureStdin(t)

	resp, err := Upload(context.Background())
	require.NoError(t, err)
	assert.Equal(t, int64(2), resp.NumProtoSuccess)
	assert.Equal(t, 2, count)
}

func uploadRetriesFor(t *testing.T, statusCode int) {
	server := testserver.New(t)
	t.Cleanup(server.Close)

	count := 0
	server.Handle("POST", "/telemetry-ext", func(req testserver.Request) any {
		count++
		if count == 1 {
			return testserver.Response{
				StatusCode: statusCode,
				Body: apierr.APIError{
					StatusCode: statusCode,
					Message:    "Some error",
				},
			}
		}
		if count == 2 {
			return ResponseBody{
				NumProtoSuccess: 2,
			}
		}
		return nil
	})

	t.Setenv("DATABRICKS_HOST", server.URL)
	t.Setenv("DATABRICKS_TOKEN", "token")

	configureStdin(t)

	resp, err := Upload(context.Background())
	require.NoError(t, err)
	assert.Equal(t, int64(2), resp.NumProtoSuccess)
	assert.Equal(t, 2, count)
}

func TestTelemetryUploadRetriesForStatusCodes(t *testing.T) {
	// These retries happen in the CLI itself since the SDK does not automatically
	// retry for 5xx errors.
	uploadRetriesFor(t, 500)
	uploadRetriesFor(t, 504)

	// These retries happen on the SDK layer.
	// ref: https://github.com/databricks/databricks-sdk-go/blob/cdb28002afacb8b762348534a4c4040a9f19c24b/apierr/errors.go#L91
	uploadRetriesFor(t, 503)
	uploadRetriesFor(t, 429)
}

func TestTelemetryUploadMaxRetries(t *testing.T) {
	server := testserver.New(t)
	t.Cleanup(server.Close)
	count := 0

	server.Handle("POST", "/telemetry-ext", func(req testserver.Request) any {
		count++
		return ResponseBody{
			NumProtoSuccess: 1,
		}
	})

	t.Setenv("DATABRICKS_HOST", server.URL)
	t.Setenv("DATABRICKS_TOKEN", "token")

	configureStdin(t)

	_, err := Upload(context.Background())
	assert.EqualError(t, err, "upload did not succeed after three attempts. err: <nil>. response body: &telemetry.ResponseBody{Errors:[]telemetry.LogError(nil), NumProtoSuccess:1}")
	assert.Equal(t, 3, count)
}

func TestReadFiles(t *testing.T) {
	raw := `{
	"logs": [
		{
			"frontend_log_event_id": "1",
			"entry": {
				"databricks_cli_log": {
					"cli_test_event": {
						"name": "DummyCliEnumValue1"
					}
				}
			}
		},
		{
			"frontend_log_event_id": "2",
			"entry": {
				"databricks_cli_log": {
					"cli_test_event": {
						"name": "DummyCliEnumValue2"
					}
				}
			}
		}
	]
}`

	r := strings.NewReader(raw)
	logs, err := readLogs(r)
	require.NoError(t, err)

	assert.Equal(t, []string{
		`{"frontend_log_event_id":"1","entry":{"databricks_cli_log":{"cli_test_event":{"name":"DummyCliEnumValue1"}}}}`,
		`{"frontend_log_event_id":"2","entry":{"databricks_cli_log":{"cli_test_event":{"name":"DummyCliEnumValue2"}}}}`,
	}, logs)
}
