package mcp_test

import (
	"bytes"
	"errors"
	"fmt"
	"io"
	"net/http"
	"net/http/httptest"
	"os"
	"path/filepath"
	"testing"

	"github.com/databricks/cli/libs/aitools/mcp"
	"github.com/databricks/cli/libs/env"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

const (
	bindingTestService = "system.ai.github"
	bindingTestProfile = "logfood"
	bindingTestHost    = "https://adb-123.test"
	bindingTestToken   = "dapi-profile-token"
)

// bindingTestWriteConfig writes a .databrickscfg with contents and returns its path.
func bindingTestWriteConfig(t *testing.T, contents string) string {
	t.Helper()
	path := filepath.Join(t.TempDir(), ".databrickscfg")
	require.NoError(t, os.WriteFile(path, []byte(contents), 0o600))
	return path
}

// bindingTestConfig writes a .databrickscfg, points DATABRICKS_CONFIG_FILE at it, and sets
// steering variables that name another workspace. Bind must ignore those. The home directory
// is a separate temp dir so the config path is never abbreviated to "~" in messages.
func bindingTestConfig(t *testing.T, contents string) string {
	t.Helper()
	path := bindingTestWriteConfig(t, contents)
	t.Setenv(env.HomeEnvVar(), t.TempDir())
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

// bindingTestState returns a state with one entry pinned to configFile, the file the test wrote.
func bindingTestState(configFile, host string) *mcp.State {
	return &mcp.State{Servers: []mcp.Entry{{
		Name:       "github-logfood",
		Service:    bindingTestService,
		Profile:    bindingTestProfile,
		Host:       host,
		ConfigFile: configFile,
	}}}
}

func TestBindSucceedsWithStateEntry(t *testing.T) {
	cfgPath := bindingTestConfig(t, bindingTestProfileContents(bindingTestToken))
	stderr := &bytes.Buffer{}

	cfg, entry, err := mcp.Bind(t.Context(), mcp.BindOptions{
		Profile: bindingTestProfile,
		Service: bindingTestService,
		State:   bindingTestState(cfgPath, bindingTestHost),
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
	assert.Equal(t, "No install record for system.ai.github on profile \"logfood\"; using its current host https://adb-123.test\n", stderr.String())
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
				Name:       "github-gone",
				Service:    bindingTestService,
				Profile:    "gone",
				Host:       bindingTestHost,
				ConfigFile: path,
			}}},
			want: fmt.Sprintf(`MCP server github-gone is bound to Databricks profile "gone", which no longer exists in %s. Recreate it with: databricks auth login --host https://adb-123.test --profile gone   or remove the server with: databricks aitools mcp remove --profile gone`, filepath.ToSlash(path)),
		},
		{
			name:  "without state entry",
			state: &mcp.State{},
			want:  fmt.Sprintf(`MCP server system.ai.github is bound to Databricks profile "gone", which no longer exists in %s. Recreate it with: databricks auth login --profile gone   or remove the server with: databricks aitools mcp remove --profile gone`, filepath.ToSlash(path)),
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
	cfgPath := bindingTestConfig(t, bindingTestProfileContents(bindingTestToken))
	path := filepath.Join(t.TempDir(), "absent")
	t.Setenv("DATABRICKS_CONFIG_FILE", path)

	_, _, err := mcp.Bind(t.Context(), mcp.BindOptions{
		Profile: bindingTestProfile,
		Service: bindingTestService,
		State:   bindingTestState(cfgPath, bindingTestHost),
		Stderr:  &bytes.Buffer{},
	})
	requireBindError(t, err, mcp.BindProfileNotFound)
	assert.Contains(t, err.Error(), "which no longer exists in "+filepath.ToSlash(path)+". Recreate")
}

func TestBindHostChanged(t *testing.T) {
	cfgPath := bindingTestConfig(t, bindingTestProfileContents(bindingTestToken))

	_, _, err := mcp.Bind(t.Context(), mcp.BindOptions{
		Profile: bindingTestProfile,
		Service: bindingTestService,
		State:   bindingTestState(cfgPath, "https://adb-999.test"),
		Stderr:  &bytes.Buffer{},
	})
	requireBindError(t, err, mcp.BindProfileHostChanged)
	assert.Equal(t, "MCP server github-logfood was set up for https://adb-999.test, but profile \"logfood\" now points at https://adb-123.test. Run: databricks aitools mcp remove --profile logfood and add it again", err.Error())
}

// TestBindHostChangedDoesNotContactNewHost checks that a re-pointed profile is rejected before
// any request reaches the new host.
func TestBindHostChangedDoesNotContactNewHost(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		t.Errorf("unexpected request to the re-pointed host: %s %s", r.Method, r.URL)
		w.WriteHeader(http.StatusInternalServerError)
	}))
	t.Cleanup(server.Close)
	cfgPath := bindingTestConfig(t, fmt.Sprintf("[%s]\nhost = %s\nauth_type = pat\ntoken = %s\n", bindingTestProfile, server.URL, bindingTestToken))

	_, _, err := mcp.Bind(t.Context(), mcp.BindOptions{
		Profile: bindingTestProfile,
		Service: bindingTestService,
		State:   bindingTestState(cfgPath, bindingTestHost),
		Stderr:  &bytes.Buffer{},
	})
	requireBindError(t, err, mcp.BindProfileHostChanged)
}

func TestBindWorkspaceChanged(t *testing.T) {
	cfgPath := bindingTestConfig(t, fmt.Sprintf("[%s]\nhost = %s\nworkspace_id = 222\nauth_type = pat\ntoken = %s\n", bindingTestProfile, bindingTestHost, bindingTestToken))
	state := bindingTestState(cfgPath, bindingTestHost)
	state.Servers[0].WorkspaceID = "111"

	_, _, err := mcp.Bind(t.Context(), mcp.BindOptions{
		Profile: bindingTestProfile,
		Service: bindingTestService,
		State:   state,
		Stderr:  &bytes.Buffer{},
	})
	requireBindError(t, err, mcp.BindProfileHostChanged)
	assert.Equal(t, "MCP server github-logfood was set up for workspace 111, but profile \"logfood\" now points at workspace 222. Run: databricks aitools mcp remove --profile logfood and add it again", err.Error())
}

func TestBindWorkspaceNotComparedWhenProfileNamesNone(t *testing.T) {
	cfgPath := bindingTestConfig(t, fmt.Sprintf("[%s]\nhost = %s\nworkspace_id = none\nauth_type = pat\ntoken = %s\n", bindingTestProfile, bindingTestHost, bindingTestToken))
	state := bindingTestState(cfgPath, bindingTestHost)
	state.Servers[0].WorkspaceID = "111"

	_, entry, err := mcp.Bind(t.Context(), mcp.BindOptions{
		Profile: bindingTestProfile,
		Service: bindingTestService,
		State:   state,
		Stderr:  &bytes.Buffer{},
	})
	require.NoError(t, err)
	assert.NotNil(t, entry)
}

func TestBindHostUnchangedWhenOnlyTrailingSlashDiffers(t *testing.T) {
	cfgPath := bindingTestConfig(t, bindingTestProfileContents(bindingTestToken))

	_, entry, err := mcp.Bind(t.Context(), mcp.BindOptions{
		Profile: bindingTestProfile,
		Service: bindingTestService,
		State:   bindingTestState(cfgPath, "https://adb-123.test/"),
		Stderr:  &bytes.Buffer{},
	})
	require.NoError(t, err)
	assert.NotNil(t, entry)
}

// TestBindNeverUsesEnvTokenForProfileWithoutToken checks that a profile without a token does not
// pick up DATABRICKS_TOKEN from the environment, which names another workspace here.
func TestBindNeverUsesEnvTokenForProfileWithoutToken(t *testing.T) {
	cfgPath := bindingTestConfig(t, bindingTestProfileContents(""))

	_, _, err := mcp.Bind(t.Context(), mcp.BindOptions{
		Profile: bindingTestProfile,
		Service: bindingTestService,
		State:   bindingTestState(cfgPath, bindingTestHost),
		Stderr:  &bytes.Buffer{},
	})
	requireBindError(t, err, mcp.BindAuthFailed)
	assert.Contains(t, err.Error(), "\nRun: databricks auth login --profile logfood")
	assert.NotContains(t, err.Error(), "dapi-env-token")
	assert.Error(t, errors.Unwrap(err))
}

// TestBindReadsConfigFileFromContext checks that the config file named in the context, not the
// process environment, is used for both the profile lookup and authentication.
func TestBindReadsConfigFileFromContext(t *testing.T) {
	bindingTestConfig(t, bindingTestProfileContents(bindingTestToken))
	path := bindingTestWriteConfig(t, bindingTestProfileContents(bindingTestToken))
	t.Setenv("DATABRICKS_CONFIG_FILE", filepath.Join(t.TempDir(), "absent"))
	ctx := env.Set(t.Context(), "DATABRICKS_CONFIG_FILE", path)

	cfg, _, err := mcp.Bind(ctx, mcp.BindOptions{
		Profile: bindingTestProfile,
		Service: bindingTestService,
		State:   bindingTestState(path, bindingTestHost),
		Stderr:  &bytes.Buffer{},
	})
	require.NoError(t, err)
	assert.Equal(t, bindingTestToken, cfg.Token)
}

// TestBindMatchesEntryByConfigFile checks that an entry pinned to another config file does not
// bind a profile of the same name in the current file.
func TestBindMatchesEntryByConfigFile(t *testing.T) {
	bindingTestConfig(t, bindingTestProfileContents(bindingTestToken))
	current := bindingTestWriteConfig(t, "[logfood]\nhost = https://adb-456.test\nauth_type = pat\ntoken = dapi-current\n")
	ctx := env.Set(t.Context(), "DATABRICKS_CONFIG_FILE", current)
	state := &mcp.State{Servers: []mcp.Entry{
		{Name: "other", Service: bindingTestService, Profile: bindingTestProfile, Host: bindingTestHost, ConfigFile: "/elsewhere/.databrickscfg"},
		{Name: "current", Service: bindingTestService, Profile: bindingTestProfile, Host: "https://adb-456.test", ConfigFile: current},
	}}

	cfg, entry, err := mcp.Bind(ctx, mcp.BindOptions{
		Profile: bindingTestProfile,
		Service: bindingTestService,
		State:   state,
		Stderr:  &bytes.Buffer{},
	})
	require.NoError(t, err)
	require.NotNil(t, entry)
	assert.Equal(t, "current", entry.Name)
	assert.Equal(t, "dapi-current", cfg.Token)
}

func TestBindTokenCannotBeMinted(t *testing.T) {
	cfgPath := bindingTestConfig(t, bindingTestProfileContents(""))
	t.Setenv("DATABRICKS_TOKEN", "")

	_, _, err := mcp.Bind(t.Context(), mcp.BindOptions{
		Profile: bindingTestProfile,
		Service: bindingTestService,
		State:   bindingTestState(cfgPath, bindingTestHost),
		Stderr:  &bytes.Buffer{},
	})
	requireBindError(t, err, mcp.BindAuthFailed)
	assert.Contains(t, err.Error(), "\nRun: databricks auth login --profile logfood")
	assert.Error(t, errors.Unwrap(err))
}

// TestBindDoesNotReadStdin feeds a message on stdin and checks Bind leaves it unread.
func TestBindDoesNotReadStdin(t *testing.T) {
	cfgPath := bindingTestConfig(t, bindingTestProfileContents(bindingTestToken))
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
		State:   bindingTestState(cfgPath, bindingTestHost),
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

// bindingTestDefaultConfig writes contents to ~/.databrickscfg in a fresh home directory, with
// DATABRICKS_CONFIG_FILE empty, and returns the path of the default file.
func bindingTestDefaultConfig(t *testing.T, contents string) string {
	t.Helper()
	home := t.TempDir()
	t.Setenv(env.HomeEnvVar(), home)
	t.Setenv("DATABRICKS_CONFIG_FILE", "")
	t.Setenv("DATABRICKS_HOST", "https://adb-999.test")
	t.Setenv("DATABRICKS_TOKEN", "dapi-env-token")
	path := filepath.Join(home, ".databrickscfg")
	require.NoError(t, os.WriteFile(path, []byte(contents), 0o600))
	return path
}

// TestBindWorkspaceFromHostQueryChanged checks that a workspace named in the profile's host query
// is compared with the recorded one, as the SDK resolves it.
func TestBindWorkspaceFromHostQueryChanged(t *testing.T) {
	for _, query := range []string{"?o=222", "?w=222", "?workspace_id=222"} {
		t.Run(query, func(t *testing.T) {
			cfgPath := bindingTestConfig(t, fmt.Sprintf("[%s]\nhost = %s/%s\nauth_type = pat\ntoken = %s\n", bindingTestProfile, bindingTestHost, query, bindingTestToken))
			state := bindingTestState(cfgPath, bindingTestHost)
			state.Servers[0].WorkspaceID = "111"

			_, _, err := mcp.Bind(t.Context(), mcp.BindOptions{
				Profile: bindingTestProfile,
				Service: bindingTestService,
				State:   state,
				Stderr:  &bytes.Buffer{},
			})
			requireBindError(t, err, mcp.BindProfileHostChanged)
			assert.Equal(t, "MCP server github-logfood was set up for workspace 111, but profile \"logfood\" now points at workspace 222. Run: databricks aitools mcp remove --profile logfood and add it again", err.Error())
		})
	}
}

// TestBindWorkspaceFromHostQueryNoneIsUnset checks that a host query naming no workspace is not
// compared.
func TestBindWorkspaceFromHostQueryNoneIsUnset(t *testing.T) {
	for _, query := range []string{"?o=none", "?w=none"} {
		t.Run(query, func(t *testing.T) {
			cfgPath := bindingTestConfig(t, fmt.Sprintf("[%s]\nhost = %s/%s\nauth_type = pat\ntoken = %s\n", bindingTestProfile, bindingTestHost, query, bindingTestToken))
			state := bindingTestState(cfgPath, bindingTestHost)
			state.Servers[0].WorkspaceID = "111"

			_, entry, err := mcp.Bind(t.Context(), mcp.BindOptions{
				Profile: bindingTestProfile,
				Service: bindingTestService,
				State:   state,
				Stderr:  &bytes.Buffer{},
			})
			require.NoError(t, err)
			assert.NotNil(t, entry)
		})
	}
}

// TestBindUnpinnedEntryMatchesDefaultConfigFile checks that an entry with no pinned config file
// names the default file, so it matches whether DATABRICKS_CONFIG_FILE is unset or names that file.
func TestBindUnpinnedEntryMatchesDefaultConfigFile(t *testing.T) {
	defaultPath := bindingTestDefaultConfig(t, bindingTestProfileContents(bindingTestToken))
	state := &mcp.State{Servers: []mcp.Entry{{
		Name:    "github-logfood",
		Service: bindingTestService,
		Profile: bindingTestProfile,
		Host:    "https://adb-999.test",
	}}}

	for _, configFile := range []string{"", defaultPath} {
		t.Run("env="+configFile, func(t *testing.T) {
			t.Setenv("DATABRICKS_CONFIG_FILE", configFile)

			_, _, err := mcp.Bind(t.Context(), mcp.BindOptions{
				Profile: bindingTestProfile,
				Service: bindingTestService,
				State:   state,
				Stderr:  &bytes.Buffer{},
			})
			requireBindError(t, err, mcp.BindProfileHostChanged)
		})
	}
}

// TestBindUnpinnedEntryDoesNotMatchOtherConfigFile checks that an entry with no pinned config file
// does not bind a profile of the same name in another config file.
func TestBindUnpinnedEntryDoesNotMatchOtherConfigFile(t *testing.T) {
	bindingTestDefaultConfig(t, bindingTestProfileContents(bindingTestToken))
	other := bindingTestWriteConfig(t, "[logfood]\nhost = https://adb-456.test\nauth_type = pat\ntoken = dapi-other\n")
	t.Setenv("DATABRICKS_CONFIG_FILE", other)
	stderr := &bytes.Buffer{}

	cfg, entry, err := mcp.Bind(t.Context(), mcp.BindOptions{
		Profile: bindingTestProfile,
		Service: bindingTestService,
		State:   bindingTestState("", bindingTestHost),
		Stderr:  stderr,
	})
	require.NoError(t, err)
	assert.Nil(t, entry)
	assert.Equal(t, "https://adb-456.test", cfg.Host)
	assert.Contains(t, stderr.String(), "No install record for system.ai.github on profile \"logfood\"")
}

// TestBindWithoutHomeAndAbsoluteConfigFile checks that a bind needs no home directory when the
// config file is named by an absolute path.
func TestBindWithoutHomeAndAbsoluteConfigFile(t *testing.T) {
	cfgPath := bindingTestConfig(t, bindingTestProfileContents(bindingTestToken))
	t.Setenv(env.HomeEnvVar(), "")

	cfg, entry, err := mcp.Bind(t.Context(), mcp.BindOptions{
		Profile: bindingTestProfile,
		Service: bindingTestService,
		State:   bindingTestState(cfgPath, bindingTestHost),
		Stderr:  &bytes.Buffer{},
	})
	require.NoError(t, err)
	assert.NotNil(t, entry)
	assert.Equal(t, bindingTestToken, cfg.Token)
}
