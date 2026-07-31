package aircmd

import (
	"net/http"
	"net/http/httptest"
	"os"
	"path/filepath"
	"testing"

	"github.com/databricks/databricks-sdk-go/service/jobs"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// airRunWithCompute builds a run whose AI runtime task reports the given
// accelerator type and count, for node-count resolution tests.
func airRunWithCompute(accelType string, count int) *jobs.Run {
	return &jobs.Run{
		RunId: 123,
		Tasks: []jobs.RunTask{{
			RunId: 456,
			AiRuntimeTask: &jobs.AiRuntimeTask{
				Deployments: []jobs.DeploymentSpec{{
					Compute: jobs.ComputeSpec{
						AcceleratorType:  jobs.ComputeSpecAcceleratorType(accelType),
						AcceleratorCount: count,
					},
				}},
			},
		}},
	}
}

func TestResolveNodeCount(t *testing.T) {
	tests := []struct {
		accelType string
		count     int
		want      int
	}{
		{"GPU_1xA10", 2, 2},
		{"GPU_1xH100", 4, 4},
		{"GPU_8xH100", 16, 2},
	}
	for _, tt := range tests {
		n, err := resolveNodeCount(airRunWithCompute(tt.accelType, tt.count))
		require.NoError(t, err)
		assert.Equal(t, tt.want, n)
	}

	// A run with no AI runtime compute errors.
	_, err := resolveNodeCount(&jobs.Run{RunId: 1})
	require.Error(t, err)
}

// downloadServer serves the MLflow artifact chain used by the download path:
// artifacts/list (logs dir + per-node chunk), credentials-for-read (a pre-signed
// URL pointing back at itself), and the chunk bytes.
func downloadServer(t *testing.T) *httptest.Server {
	t.Helper()
	var base string
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		switch r.URL.Path {
		case "/api/2.0/mlflow/artifacts/list":
			if r.URL.Query().Get("path") == "logs" {
				_, _ = w.Write([]byte(`{"files": [{"path": "logs/node_0", "is_dir": true}, {"path": "logs/node_1", "is_dir": true}]}`))
			} else {
				_, _ = w.Write([]byte(`{"files": [{"path": "` + r.URL.Query().Get("path") + `/logs-0.chunk.txt", "file_size": 12}]}`))
			}
		case "/api/2.0/mlflow/artifacts/credentials-for-read":
			_, _ = w.Write([]byte(`{"credential_infos": [{"signed_uri": "` + base + `/presigned"}]}`))
		case "/presigned":
			_, _ = w.Write([]byte("line one\nline two\n"))
		default:
			_, _ = w.Write([]byte(`{}`))
		}
	}))
	base = srv.URL
	t.Cleanup(srv.Close)
	return srv
}

func TestDownloadNodeLogWritesConcatenatedChunks(t *testing.T) {
	w := newTestWorkspaceClient(t, downloadServer(t).URL)
	dir := t.TempDir()

	path, err := downloadNodeLog(t.Context(), w, "run1", 0, 0, false, dir)
	require.NoError(t, err)
	require.NotEmpty(t, path)

	got, err := os.ReadFile(path)
	require.NoError(t, err)
	assert.Equal(t, "line one\nline two\n", string(got))
	assert.Equal(t, filepath.Join(dir, "logs", "node_0.log"), path)
}

func TestDownloadAllNodeLogs(t *testing.T) {
	w := newTestWorkspaceClient(t, downloadServer(t).URL)
	dir := t.TempDir()

	nodeLogs, err := downloadAllNodeLogs(t.Context(), w, "run1", dir, []int{0, 1}, -1)
	require.NoError(t, err)
	require.Len(t, nodeLogs, 2)
	assert.FileExists(t, nodeLogs[0])
	assert.FileExists(t, nodeLogs[1])
}
