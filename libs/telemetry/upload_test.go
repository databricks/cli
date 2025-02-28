package telemetry

import (
	"context"
	"encoding/json"
	"os"
	"path/filepath"
	"testing"

	"github.com/databricks/cli/internal/testutil"
	"github.com/databricks/cli/libs/telemetry/protos"
	"github.com/databricks/cli/libs/testserver"
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

func TestTelemetryUploadRetries(t *testing.T) {
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

func TestTelemetryUploadCanceled(t *testing.T) {
	server := testserver.New(t)
	t.Cleanup(server.Close)

	ctx, cancel := context.WithCancel(context.Background())
	cancel()

	configureStdin(t)
	_, err := Upload(ctx)

	// Since the context is already cancelled, upload should fail immediately
	// with a timeout error.
	assert.ErrorContains(t, err, "Failed to flush telemetry log due to timeout")
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
	assert.EqualError(t, err, "Failed to upload all telemetry logs after 4 tries. Only 1/2 logs uploaded")
	assert.Equal(t, 4, count)
}
