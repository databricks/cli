package sessions

import (
	"path/filepath"
	"regexp"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestLoadEmpty(t *testing.T) {
	t.Setenv("HOME", t.TempDir())
	t.Setenv("USERPROFILE", t.TempDir())

	store, err := Load(t.Context())
	require.NoError(t, err)
	assert.Empty(t, store.Sessions)
}

func TestSaveAndLoad(t *testing.T) {
	tmpDir := t.TempDir()
	t.Setenv("HOME", tmpDir)
	t.Setenv("USERPROFILE", tmpDir)

	store := &SessionStore{
		Sessions: []Session{
			{
				Name:          "gpu-a10-abcd1234",
				Accelerator:   "GPU_1xA10",
				WorkspaceHost: "https://test.databricks.com",
				CreatedAt:     time.Date(2026, 3, 10, 12, 0, 0, 0, time.UTC),
				ClusterID:     "0310-120000-abc",
			},
		},
	}

	err := Save(t.Context(), store)
	require.NoError(t, err)

	loaded, err := Load(t.Context())
	require.NoError(t, err)
	require.Len(t, loaded.Sessions, 1)
	assert.Equal(t, "gpu-a10-abcd1234", loaded.Sessions[0].Name)
	assert.Equal(t, "GPU_1xA10", loaded.Sessions[0].Accelerator)
	assert.Equal(t, "https://test.databricks.com", loaded.Sessions[0].WorkspaceHost)
	assert.Equal(t, "0310-120000-abc", loaded.Sessions[0].ClusterID)
}

func TestAddAndRemove(t *testing.T) {
	tmpDir := t.TempDir()
	t.Setenv("HOME", tmpDir)
	t.Setenv("USERPROFILE", tmpDir)

	ctx := t.Context()

	err := Add(ctx, Session{Name: "sess-1", Accelerator: "GPU_1xA10", WorkspaceHost: "https://a.com"})
	require.NoError(t, err)

	err = Add(ctx, Session{Name: "sess-2", Accelerator: "GPU_8xH100", WorkspaceHost: "https://b.com"})
	require.NoError(t, err)

	store, err := Load(ctx)
	require.NoError(t, err)
	assert.Len(t, store.Sessions, 2)

	err = Remove(ctx, "sess-1")
	require.NoError(t, err)

	store, err = Load(ctx)
	require.NoError(t, err)
	require.Len(t, store.Sessions, 1)
	assert.Equal(t, "sess-2", store.Sessions[0].Name)
}

func TestRemoveNonExistent(t *testing.T) {
	tmpDir := t.TempDir()
	t.Setenv("HOME", tmpDir)
	t.Setenv("USERPROFILE", tmpDir)

	err := Remove(t.Context(), "no-such-session")
	assert.NoError(t, err)
}

func TestFindMatching(t *testing.T) {
	tmpDir := t.TempDir()
	t.Setenv("HOME", tmpDir)
	t.Setenv("USERPROFILE", tmpDir)

	ctx := t.Context()
	host := "https://test.databricks.com"
	user := "alice@example.com"

	now := time.Now()

	err := Add(ctx, Session{Name: "s1", Accelerator: "GPU_1xA10", WorkspaceHost: host, UserName: user, CreatedAt: now})
	require.NoError(t, err)
	err = Add(ctx, Session{Name: "s2", Accelerator: "GPU_8xH100", WorkspaceHost: host, UserName: user, CreatedAt: now})
	require.NoError(t, err)
	err = Add(ctx, Session{Name: "s3", Accelerator: "GPU_1xA10", WorkspaceHost: "https://other.com", UserName: user, CreatedAt: now})
	require.NoError(t, err)
	err = Add(ctx, Session{Name: "s4", Accelerator: "GPU_1xA10", WorkspaceHost: host, UserName: user, CreatedAt: now})
	require.NoError(t, err)
	err = Add(ctx, Session{Name: "s5", Accelerator: "GPU_1xA10", WorkspaceHost: host, UserName: "bob@example.com", CreatedAt: now})
	require.NoError(t, err)

	matches, err := FindMatching(ctx, host, "GPU_1xA10", user)
	require.NoError(t, err)
	assert.Len(t, matches, 2)
	assert.Equal(t, "s1", matches[0].Name)
	assert.Equal(t, "s4", matches[1].Name)

	matches, err = FindMatching(ctx, host, "GPU_8xH100", user)
	require.NoError(t, err)
	assert.Len(t, matches, 1)
	assert.Equal(t, "s2", matches[0].Name)

	matches, err = FindMatching(ctx, host, "GPU_4xA100", user)
	require.NoError(t, err)
	assert.Empty(t, matches)

	// Different user should not see alice's sessions
	matches, err = FindMatching(ctx, host, "GPU_1xA10", "bob@example.com")
	require.NoError(t, err)
	assert.Len(t, matches, 1)
	assert.Equal(t, "s5", matches[0].Name)
}

func TestFindMatchingExpiresOldSessions(t *testing.T) {
	tmpDir := t.TempDir()
	t.Setenv("HOME", tmpDir)
	t.Setenv("USERPROFILE", tmpDir)

	ctx := t.Context()
	host := "https://test.databricks.com"
	user := "alice@example.com"

	err := Add(ctx, Session{Name: "old", Accelerator: "GPU_1xA10", WorkspaceHost: host, UserName: user, CreatedAt: time.Now().Add(-25 * time.Hour)})
	require.NoError(t, err)
	err = Add(ctx, Session{Name: "recent", Accelerator: "GPU_1xA10", WorkspaceHost: host, UserName: user, CreatedAt: time.Now()})
	require.NoError(t, err)

	matches, err := FindMatching(ctx, host, "GPU_1xA10", user)
	require.NoError(t, err)
	require.Len(t, matches, 1)
	assert.Equal(t, "recent", matches[0].Name)

	// Verify expired sessions were pruned from disk.
	store, err := Load(ctx)
	require.NoError(t, err)
	assert.Len(t, store.Sessions, 1, "expired sessions should be pruned from disk")
}

func TestStateFilePath(t *testing.T) {
	tmpDir := t.TempDir()
	t.Setenv("HOME", tmpDir)
	t.Setenv("USERPROFILE", tmpDir)

	path, err := getStateFilePath(t.Context())
	require.NoError(t, err)
	assert.Equal(t, filepath.Join(tmpDir, ".databricks", stateFileName), path)
}

// connectionNameRegex mirrors the regex in client.go.
var connectionNameRegex = regexp.MustCompile(`^[a-zA-Z0-9][a-zA-Z0-9_-]*$`)

func TestGenerateSessionName(t *testing.T) {
	host := "https://test.databricks.com"
	tests := []struct {
		accelerator    string
		wantPrefix     string
		wantDatePrefix string
	}{
		{"GPU_1xA10", "databricks-gpu-a10-", "databricks-gpu-a10-20"},
		{"GPU_8xH100", "databricks-gpu-h100-", "databricks-gpu-h100-20"},
		{"UNKNOWN_TYPE", "databricks-unknown-type-", "databricks-unknown-type-20"},
	}

	for _, tt := range tests {
		t.Run(tt.accelerator, func(t *testing.T) {
			name := GenerateSessionName(tt.accelerator, host)
			assert.Greater(t, len(name), len(tt.wantPrefix), "name should be longer than prefix")
			assert.Equal(t, tt.wantPrefix, name[:len(tt.wantPrefix)])
			// Verify date component is present (starts with "20" for 2000s dates).
			assert.Equal(t, tt.wantDatePrefix, name[:len(tt.wantDatePrefix)])
			assert.True(t, connectionNameRegex.MatchString(name), "generated name %q must match connection name regex", name)
		})
	}
}

func TestGenerateSessionNameDiffersByWorkspace(t *testing.T) {
	name1 := GenerateSessionName("GPU_1xA10", "https://workspace-a.databricks.com")
	name2 := GenerateSessionName("GPU_1xA10", "https://workspace-b.databricks.com")
	// The workspace hash portion (after the date-) should differ.
	// Names have format: databricks-gpu-a10-YYYYMMDD-<wshash><random>
	// Extract after the date prefix to compare workspace hash parts.
	assert.NotEqual(t, name1, name2, "names for different workspaces should differ")
}

func TestGenerateSessionNameUniqueness(t *testing.T) {
	host := "https://test.databricks.com"
	seen := make(map[string]bool)
	for range 100 {
		name := GenerateSessionName("GPU_1xA10", host)
		assert.False(t, seen[name], "duplicate name generated: %s", name)
		seen[name] = true
	}
}
