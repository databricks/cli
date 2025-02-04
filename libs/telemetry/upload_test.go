package telemetry

import (
	"encoding/json"
	"net/http"
	"os"
	"path/filepath"
	"testing"

	"github.com/databricks/cli/internal/testutil"
	"github.com/databricks/cli/libs/telemetry/protos"
	"github.com/databricks/cli/libs/testserver"
	"github.com/databricks/databricks-sdk-go/apierr"
	"github.com/google/uuid"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestTelemetryUpload(t *testing.T) {
	server := testserver.New(t)
	t.Cleanup(server.Close)

	count := 0
	server.Handle("POST /telemetry-ext", func(req *http.Request) (resp any, err error) {
		count++
		if count == 1 {
			return ResponseBody{
				NumProtoSuccess: 1,
			}, nil
		}
		if count == 2 {
			return ResponseBody{
				NumProtoSuccess: 2,
			}, nil
		}
		return nil, apierr.NotFound("not found")
	})

	t.Setenv("DATABRICKS_HOST", server.URL)
	t.Setenv("DATABRICKS_TOKEN", "token")

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

	fd, err := os.OpenFile(filepath.Join(tmpDir, "stdin"), os.O_RDONLY, 0o644)
	require.NoError(t, err)

	// Redirect stdin to the file containing the telemetry logs.
	old := os.Stdin
	os.Stdin = fd
	t.Cleanup(func() {
		fd.Close()
		os.Stdin = old
	})

	resp, err := Upload()
	require.NoError(t, err)
	assert.Equal(t, int64(2), resp.NumProtoSuccess)
	assert.Equal(t, 2, count)
}
