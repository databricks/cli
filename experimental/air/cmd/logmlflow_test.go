package aircmd

import (
	"bytes"
	"net/http"
	"net/http/httptest"
	"os"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestConstructLogPath(t *testing.T) {
	assert.Equal(t, "logs/node_0", constructLogPath(0, 0, false))
	assert.Equal(t, "logs/node_3", constructLogPath(3, 2, false))
	assert.Equal(t, "logs/attempt_2/node_3", constructLogPath(3, 2, true))
}

func TestChunkFileName(t *testing.T) {
	assert.Equal(t, "logs-0.chunk.txt", chunkFileName(0))
	assert.Equal(t, "logs-7.chunk.txt", chunkFileName(7))
}

func TestChunkFilePattern(t *testing.T) {
	m := chunkFilePattern.FindStringSubmatch("logs-12.chunk.txt")
	require.NotNil(t, m)
	assert.Equal(t, "12", m[1])

	assert.Nil(t, chunkFilePattern.FindStringSubmatch("logs-12.chunk.txt.bak"))
	assert.Nil(t, chunkFilePattern.FindStringSubmatch("node_0"))
}

// artifactListServer serves a fixed artifacts/list response for any path.
func artifactListServer(t *testing.T, body string) *httptest.Server {
	t.Helper()
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.URL.Path == "/api/2.0/mlflow/artifacts/list" {
			_, _ = w.Write([]byte(body))
			return
		}
		_, _ = w.Write([]byte(`{}`))
	}))
	t.Cleanup(srv.Close)
	return srv
}

func TestListLogChunksSortsAndFiltersByIndex(t *testing.T) {
	// Out-of-order chunks plus a non-chunk file; result is ascending, chunk-only.
	srv := artifactListServer(t, `{"files": [
		{"path": "logs/node_0/logs-2.chunk.txt"},
		{"path": "logs/node_0/other.txt"},
		{"path": "logs/node_0/logs-0.chunk.txt"},
		{"path": "logs/node_0/logs-1.chunk.txt"}
	]}`)
	w := newTestWorkspaceClient(t, srv.URL)

	chunks, err := listLogChunks(t.Context(), w, "run1", "logs/node_0")
	require.NoError(t, err)
	require.Len(t, chunks, 3)
	assert.Equal(t, 0, chunks[0].index)
	assert.Equal(t, 1, chunks[1].index)
	assert.Equal(t, 2, chunks[2].index)
	assert.Equal(t, "logs/node_0/logs-0.chunk.txt", chunks[0].path)
}

func TestDiscoverAttemptPrefix(t *testing.T) {
	// Old format: a bare logs/node_N dir means no attempt prefix.
	old := artifactListServer(t, `{"files": [{"path": "logs/node_0", "is_dir": true}]}`)
	got, err := discoverAttemptPrefix(t.Context(), newTestWorkspaceClient(t, old.URL), "run1", 0)
	require.NoError(t, err)
	assert.False(t, got)

	// New format: a logs/attempt_N entry and no bare node dir.
	newFmt := artifactListServer(t, `{"files": [{"path": "logs/attempt_0", "is_dir": true}]}`)
	got, err = discoverAttemptPrefix(t.Context(), newTestWorkspaceClient(t, newFmt.URL), "run1", 0)
	require.NoError(t, err)
	assert.True(t, got)
}

// noMLflowServer serves a run with no resolvable MLflow run id, so the fallback
// finds no logs.
func noMLflowServer(t *testing.T) *httptest.Server {
	t.Helper()
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		switch r.URL.Path {
		case "/api/2.2/jobs/runs/get":
			_, _ = w.Write([]byte(`{"run_id": 5, "state": {"life_cycle_state": "TERMINATED", "result_state": "SUCCESS"}, "tasks": [{"run_id": 456}]}`))
		case "/api/2.2/jobs/runs/get-output":
			_, _ = w.Write([]byte(`{}`))
		default:
			_, _ = w.Write([]byte(`{}`))
		}
	}))
	t.Cleanup(srv.Close)
	return srv
}

func TestMLflowFallbackNoLogsReflectsRunOutcome(t *testing.T) {
	srv := noMLflowServer(t)
	w := newTestWorkspaceClient(t, srv.URL)

	// A SUCCESS run with no logs still reports success (exit 0), matching the
	// Bricklens path, rather than failing just because no logs exist.
	success, err := mlflowLogFallback(t.Context(), w, &bytes.Buffer{},
		logRequest{runID: 5}, logRunStatus{lifeCycleState: "TERMINATED", resultState: "SUCCESS"})
	require.NoError(t, err)
	assert.True(t, success)

	// A FAILED run with no logs reports failure (exit 1).
	success, err = mlflowLogFallback(t.Context(), w, &bytes.Buffer{},
		logRequest{runID: 5}, logRunStatus{lifeCycleState: "TERMINATED", resultState: "FAILED"})
	require.NoError(t, err)
	assert.False(t, success)
}

func TestDownloadArtifactSendsUnbracketedPath(t *testing.T) {
	var gotPath string
	var base string
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		switch r.URL.Path {
		case "/api/2.0/mlflow/artifacts/credentials-for-read":
			gotPath = r.URL.Query().Get("path")
			_, _ = w.Write([]byte(`{"credential_infos": [{"signed_uri": "` + base + `/presigned"}]}`))
		case "/presigned":
			_, _ = w.Write([]byte("bytes\n"))
		default:
			_, _ = w.Write([]byte(`{}`))
		}
	}))
	base = srv.URL
	t.Cleanup(srv.Close)

	local, err := downloadArtifact(t.Context(), newTestWorkspaceClient(t, srv.URL), "run1", "logs/node_0/logs-0.chunk.txt")
	require.NoError(t, err)
	t.Cleanup(func() { os.Remove(local) })

	// The backend signs whatever path it is given and returns 200 even for a
	// bracketed one, so only the download 404s. Assert on the path sent.
	assert.Equal(t, "logs/node_0/logs-0.chunk.txt", gotPath)
}
