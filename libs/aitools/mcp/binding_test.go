package mcp_test

import (
	"bytes"
	"errors"
	"fmt"
	"io"
	"os"
	"path/filepath"
	"testing"

	"github.com/databricks/cli/libs/aitools/mcp"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

const (
	bindingTestService = "system.ai.github"
	bindingTestProfile = "logfood"
	bindingTestHost    = "https://adb-123.test"
	bindingTestToken   = "dapi-profile-token"
)

// bindingTestConfig writes a .databrickscfg with contents, points DATABRICKS_CONFIG_FILE at it,
// and sets steering variables that name another workspace. Bind must ignore those.
func bindingTestConfig(t *testing.T, contents string) string {
	t.Helper()
	path := filepath.Join(t.TempDir(), ".databrickscfg")
	require.NoError(t, os.WriteFile(path, []byte(contents), 0o600))
	t.Setenv("DATABRICKS_CONFIG_FILE", path)
	t.Setenv("DATABRICKS_HOST", "https://adb-999.test")
	t.Setenv("DATABRICKS_TOKEN", "dapi-env-token")
	t.Setenv("DATABRICKS_CONFIG_PROFILE", "other")
	t.Setenv("DATABRICKS_AUTH_TYPE", "basic")
	return path
}

func bindingTestProfileContents(token string) string {
	return fmt.Sprintf(`[%s]
host = %s
auth_type = pat
token = %s
`, bindingTestProfile, bindingTestHost, token)
}

func bindingTestState(host string) *mcp.State {
	return &mcp.State{Servers: []mcp.Entry{{
		Name:    "github-logfood",
		Service: bindingTestService,
		Profile: bindingTestProfile,
		Host:    host,
	}}}
}

func TestBindSucceedsWithStateEntry(t *testing.T) {
	bindingTestConfig(t, bindingTestProfileContents(bindingTestToken))
	stderr := &bytes.Buffer{}

	cfg, entry, err := mcp.Bind(t.Context(), mcp.BindOptions{
		Profile: bindingTestProfile,
		Service: bindingTestService,
		State:   bindingTestState(bindingTestHost),
		Stderr:  stderr,
	})
	require.NoError(t, err)
	assert.Equal(t, bindingTestProfile, cfg.Profile)
	assert.Equal(t, bindingTestHost, cfg.Host)
	assert.Equal(t, bindingTestToken, cfg.Token)
	require.NotNil(t, entry)
	assert.Equal(t, "github-logfood", entry.Name)
	assert.Empty(t, stderr.String())
}

func TestBindPrintsNoticeWithoutStateEntry(t *testing.T) {
	bindingTestConfig(t, bindingTestProfileContents(bindingTestToken))
	stderr := &bytes.Buffer{}

	cfg, entry, err := mcp.Bind(t.Context(), mcp.BindOptions{
		Profile: bindingTestProfile,
		Service: bindingTestService,
		State:   &mcp.State{},
		Stderr:  stderr,
	})
	require.NoError(t, err)
	assert.Nil(t, entry)
	assert.Equal(t, bindingTestHost, cfg.Host)
	assert.Equal(t, "No install record for system.ai.github on profile \"logfood\"; using its current host https://adb-123.test.\n", stderr.String())
}

func TestBindProfileAbsent(t *testing.T) {
	path := bindingTestConfig(t, bindingTestProfileContents(bindingTestToken))

	tests := []struct {
		name  string
		state *mcp.State
		want  string
	}{
		{
			name: "with state entry",
			state: &mcp.State{Servers: []mcp.Entry{{
				Name:    "github-gone",
				Service: bindingTestService,
				Profile: "gone",
				Host:    bindingTestHost,
			}}},
			want: fmt.Sprintf(`MCP server github-gone is bound to Databricks profile "gone", which no longer exists in %s. Recreate it with: databricks auth login --host https://adb-123.test --profile gone   or remove the server with: databricks aitools mcp remove --profile gone`, path),
		},
		{
			name:  "without state entry",
			state: &mcp.State{},
			want:  fmt.Sprintf(`MCP server system.ai.github is bound to Databricks profile "gone", which no longer exists in %s. Recreate it with: databricks auth login --profile gone   or remove the server with: databricks aitools mcp remove --profile gone`, path),
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			_, _, err := mcp.Bind(t.Context(), mcp.BindOptions{
				Profile: "gone",
				Service: bindingTestService,
				State:   tt.state,
				Stderr:  &bytes.Buffer{},
			})
			requireBindError(t, err, mcp.BindProfileNotFound)
			assert.Equal(t, tt.want, err.Error())
		})
	}
}

func TestBindConfigFileAbsent(t *testing.T) {
	bindingTestConfig(t, bindingTestProfileContents(bindingTestToken))
	path := filepath.Join(t.TempDir(), "absent")
	t.Setenv("DATABRICKS_CONFIG_FILE", path)

	_, _, err := mcp.Bind(t.Context(), mcp.BindOptions{
		Profile: bindingTestProfile,
		Service: bindingTestService,
		State:   bindingTestState(bindingTestHost),
		Stderr:  &bytes.Buffer{},
	})
	requireBindError(t, err, mcp.BindProfileNotFound)
	assert.Contains(t, err.Error(), "which no longer exists in "+path+".")
}

func TestBindHostChanged(t *testing.T) {
	bindingTestConfig(t, bindingTestProfileContents(bindingTestToken))

	_, _, err := mcp.Bind(t.Context(), mcp.BindOptions{
		Profile: bindingTestProfile,
		Service: bindingTestService,
		State:   bindingTestState("https://adb-999.test"),
		Stderr:  &bytes.Buffer{},
	})
	requireBindError(t, err, mcp.BindProfileHostChanged)
	assert.Equal(t, "MCP server github-logfood was set up for https://adb-999.test, but profile \"logfood\" now points at https://adb-123.test. Run: databricks aitools mcp remove --profile logfood and add it again.", err.Error())
}

func TestBindHostUnchangedWhenOnlyTrailingSlashDiffers(t *testing.T) {
	bindingTestConfig(t, bindingTestProfileContents(bindingTestToken))

	_, entry, err := mcp.Bind(t.Context(), mcp.BindOptions{
		Profile: bindingTestProfile,
		Service: bindingTestService,
		State:   bindingTestState("https://adb-123.test/"),
		Stderr:  &bytes.Buffer{},
	})
	require.NoError(t, err)
	assert.NotNil(t, entry)
}

func TestBindTokenCannotBeMinted(t *testing.T) {
	bindingTestConfig(t, bindingTestProfileContents(""))
	t.Setenv("DATABRICKS_TOKEN", "")

	_, _, err := mcp.Bind(t.Context(), mcp.BindOptions{
		Profile: bindingTestProfile,
		Service: bindingTestService,
		State:   bindingTestState(bindingTestHost),
		Stderr:  &bytes.Buffer{},
	})
	requireBindError(t, err, mcp.BindAuthFailed)
	assert.Contains(t, err.Error(), "\nRun: databricks auth login --profile logfood")
	assert.Error(t, errors.Unwrap(err))
}

// TestBindDoesNotReadStdin feeds a message on stdin and checks Bind leaves it unread.
func TestBindDoesNotReadStdin(t *testing.T) {
	bindingTestConfig(t, bindingTestProfileContents(bindingTestToken))
	r, w, err := os.Pipe()
	require.NoError(t, err)
	t.Cleanup(func() { _ = r.Close() })
	const message = "{\"jsonrpc\":\"2.0\",\"id\":1,\"method\":\"initialize\"}\n"
	_, err = w.WriteString(message)
	require.NoError(t, err)
	require.NoError(t, w.Close())
	stdin := os.Stdin
	os.Stdin = r
	t.Cleanup(func() { os.Stdin = stdin })

	_, _, err = mcp.Bind(t.Context(), mcp.BindOptions{
		Profile: bindingTestProfile,
		Service: bindingTestService,
		State:   bindingTestState(bindingTestHost),
		Stderr:  &bytes.Buffer{},
	})
	require.NoError(t, err)

	rest, err := io.ReadAll(r)
	require.NoError(t, err)
	assert.Equal(t, message, string(rest))
}

func requireBindError(t *testing.T, err error, want mcp.BindCategory) {
	t.Helper()
	require.Error(t, err)
	var bindErr *mcp.BindError
	require.ErrorAs(t, err, &bindErr)
	assert.Equal(t, want, bindErr.Category)
	assert.Equal(t, 2, bindErr.ExitCode())
}
