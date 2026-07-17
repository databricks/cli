package aircmd

import (
	"net/http"
	"net/http/httptest"
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
