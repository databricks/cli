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
	"github.com/databricks/cli/libs/shellquote"
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

// bindingTestResolved returns path with its symlinks resolved, which is how Bind names an existing
// config file. Temp directories are symlinks on some systems, for example macOS.
func bindingTestResolved(t *testing.T, path string) string {
	t.Helper()
	resolved, err := filepath.EvalSymlinks(path)
	require.NoError(t, err)
	return resolved
}

func TestBindProfileAbsent(t *testing.T) {
	path := bindingTestResolved(t, bindingTestConfig(t, bindingTestProfileContents(bindingTestToken)))

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
			want: fmt.Sprintf(`MCP server github-gone is bound to Databricks profile "gone", which no longer exists in %s. Recreate it with: DATABRICKS_CONFIG_FILE=%s databricks auth login --host https://adb-123.test --profile gone or remove the server with: databricks aitools mcp remove --names github-gone`, filepath.ToSlash(path), shellquote.BashArg(filepath.ToSlash(path))),
		},
		{
			name:  "without state entry",
			state: &mcp.State{},
			want:  fmt.Sprintf(`MCP server system.ai.github needs Databricks profile "gone", which does not exist in %s. Set it up with: databricks auth login --profile gone`, filepath.ToSlash(path)),
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

// TestBindConfigFileAbsent checks that an entry pinned to a config file that no longer exists reports
// the profile as gone from that file.
func TestBindConfigFileAbsent(t *testing.T) {
	bindingTestConfig(t, bindingTestProfileContents(bindingTestToken))
	path := filepath.Join(t.TempDir(), "absent")
	t.Setenv("DATABRICKS_CONFIG_FILE", path)

	_, _, err := mcp.Bind(t.Context(), mcp.BindOptions{
		Profile: bindingTestProfile,
		Service: bindingTestService,
		State:   bindingTestState(path, bindingTestHost),
		Stderr:  &bytes.Buffer{},
	})
	requireBindError(t, err, mcp.BindProfileNotFound)
	assert.Contains(t, err.Error(), "which no longer exists in "+filepath.ToSlash(path)+". Recreate")
}

func TestBindHostChanged(t *testing.T) {
	tests := []struct {
		name  string
		entry string
		want  string
	}{
		{
			name:  "host",
			entry: "https://adb-999.test",
			want:  "MCP server github-logfood was set up for https://adb-999.test, but profile \"logfood\" now points at https://adb-123.test. Run: databricks aitools mcp remove --names github-logfood and add it again",
		},
		{
			name:  "host with a query naming another workspace on the same host",
			entry: "https://adb-999.test/?o=111",
			want:  "MCP server github-logfood was set up for https://adb-999.test, but profile \"logfood\" now points at https://adb-123.test. Run: databricks aitools mcp remove --names github-logfood and add it again",
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			cfgPath := bindingTestConfig(t, bindingTestProfileContents(bindingTestToken))

			_, _, err := mcp.Bind(t.Context(), mcp.BindOptions{
				Profile: bindingTestProfile,
				Service: bindingTestService,
				State:   bindingTestState(cfgPath, tt.entry),
				Stderr:  &bytes.Buffer{},
			})
			requireBindError(t, err, mcp.BindProfileHostChanged)
			assert.Equal(t, tt.want, err.Error())
		})
	}
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
	const want = "MCP server github-logfood was set up for workspace 111, but profile \"logfood\" now points at workspace 222. Run: databricks aitools mcp remove --names github-logfood and add it again"
	tests := []struct {
		name        string
		entryHost   string
		entryWS     string
		profileLine string
	}{
		{
			name:        "workspace_id in the profile",
			entryHost:   bindingTestHost,
			entryWS:     "111",
			profileLine: "workspace_id = 222\n",
		},
		{
			name:        "workspace in the recorded host query",
			entryHost:   bindingTestHost + "/?o=111",
			profileLine: "workspace_id = 222\n",
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			cfgPath := bindingTestConfig(t, fmt.Sprintf("[%s]\nhost = %s\n%sauth_type = pat\ntoken = %s\n", bindingTestProfile, bindingTestHost, tt.profileLine, bindingTestToken))
			state := bindingTestState(cfgPath, tt.entryHost)
			state.Servers[0].WorkspaceID = tt.entryWS

			_, _, err := mcp.Bind(t.Context(), mcp.BindOptions{
				Profile: bindingTestProfile,
				Service: bindingTestService,
				State:   state,
				Stderr:  &bytes.Buffer{},
			})
			requireBindError(t, err, mcp.BindProfileHostChanged)
			assert.Equal(t, want, err.Error())
		})
	}
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
			assert.Equal(t, "MCP server github-logfood was set up for workspace 111, but profile \"logfood\" now points at workspace 222. Run: databricks aitools mcp remove --names github-logfood and add it again", err.Error())
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
// TestBindUnpinnedEntryFailsForOtherConfigFile checks that an unpinned entry, which names the default
// file, stops a bind through another file. The profile there points at another host, and the bind must
// not switch the server to it silently.
func TestBindUnpinnedEntryFailsForOtherConfigFile(t *testing.T) {
	defaultPath := bindingTestDefaultConfig(t, bindingTestProfileContents(bindingTestToken))
	other := bindingTestWriteConfig(t, "[logfood]\nhost = https://adb-456.test\nauth_type = pat\ntoken = dapi-other\n")
	t.Setenv("DATABRICKS_CONFIG_FILE", other)
	stderr := &bytes.Buffer{}

	_, _, err := mcp.Bind(t.Context(), mcp.BindOptions{
		Profile: bindingTestProfile,
		Service: bindingTestService,
		State:   bindingTestState("", bindingTestHost),
		Stderr:  stderr,
	})
	requireBindError(t, err, mcp.BindProfileHostChanged)
	assert.Contains(t, err.Error(), "was set up with config file "+filepath.ToSlash(bindingTestResolved(t, defaultPath)))
	assert.Empty(t, stderr.String())
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

// TestBindWorkspaceFollowsSDKResolution checks that the workspace a profile names is the one the SDK
// resolves: workspace_id first, then the ?w= query, then a numeric ?o= or ?workspace_id=. A workspace
// named "none" is not compared.
func TestBindWorkspaceFollowsSDKResolution(t *testing.T) {
	tests := []struct {
		name        string
		workspaceID string // the profile's workspace_id, "" when it has none
		query       string // the host query, with its leading "?"
		recorded    string
		// wantWorkspace is the workspace the drift names. It is "" when no drift is expected.
		wantWorkspace string
	}{
		{name: "w wins over o", query: "?o=111&w=222", recorded: "111", wantWorkspace: "222"},
		{name: "w wins over o when recorded", query: "?o=111&w=222", recorded: "222"},
		{name: "non-numeric o is ignored when w names", query: "?o=abc&w=222", recorded: "222"},
		{name: "non-numeric o names none", query: "?o=abc", recorded: "111"},
		{name: "workspace_id wins over the query", workspaceID: "333", query: "?o=222", recorded: "222", wantWorkspace: "333"},
		{name: "workspace_id wins when recorded", workspaceID: "333", query: "?o=222", recorded: "333"},
		{name: "none in workspace_id names none", workspaceID: "none", query: "?o=222", recorded: "111"},
		{name: "none in o names none", query: "?o=none", recorded: "111"},
		{name: "none in w names none", query: "?w=none", recorded: "111"},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			contents := fmt.Sprintf("[%s]\nhost = %s/%s\n", bindingTestProfile, bindingTestHost, tt.query)
			if tt.workspaceID != "" {
				contents += "workspace_id = " + tt.workspaceID + "\n"
			}
			contents += "auth_type = pat\ntoken = " + bindingTestToken + "\n"
			cfgPath := bindingTestConfig(t, contents)
			state := bindingTestState(cfgPath, bindingTestHost)
			state.Servers[0].WorkspaceID = tt.recorded

			_, entry, err := mcp.Bind(t.Context(), mcp.BindOptions{
				Profile: bindingTestProfile,
				Service: bindingTestService,
				State:   state,
				Stderr:  &bytes.Buffer{},
			})
			if tt.wantWorkspace == "" {
				require.NoError(t, err)
				assert.NotNil(t, entry)
				return
			}
			requireBindError(t, err, mcp.BindProfileHostChanged)
			assert.Contains(t, err.Error(), "now points at workspace "+tt.wantWorkspace+".")
		})
	}
}

// TestBindMatchesEntryWhenHomeIsSymlink checks that an entry pinned as "~/..." matches the config file
// named by its real path when HOME is a symlink to the directory that holds the file.
func TestBindMatchesEntryWhenHomeIsSymlink(t *testing.T) {
	cfgPath := bindingTestConfig(t, bindingTestProfileContents(bindingTestToken))
	link := filepath.Join(t.TempDir(), "home")
	if err := os.Symlink(filepath.Dir(cfgPath), link); err != nil {
		t.Skipf("cannot create symlink: %v", err)
	}
	t.Setenv(env.HomeEnvVar(), link)

	_, entry, err := mcp.Bind(t.Context(), mcp.BindOptions{
		Profile: bindingTestProfile,
		Service: bindingTestService,
		State:   bindingTestState("~/.databrickscfg", bindingTestHost),
		Stderr:  &bytes.Buffer{},
	})
	require.NoError(t, err)
	assert.NotNil(t, entry)
}

// TestBindMatchesEntryWhenDotDotFollowsSymlink checks that a config path with ".." after a symlink names
// the file the OS reads through the link's target, which is also the entry's resolved path.
func TestBindMatchesEntryWhenDotDotFollowsSymlink(t *testing.T) {
	bindingTestConfig(t, bindingTestProfileContents(bindingTestToken))
	base := t.TempDir()
	cfgPath := filepath.Join(base, "a", ".databrickscfg")
	require.NoError(t, os.MkdirAll(filepath.Join(base, "a", "b"), 0o755))
	require.NoError(t, os.WriteFile(cfgPath, []byte(bindingTestProfileContents(bindingTestToken)), 0o600))
	link := filepath.Join(base, "link")
	if err := os.Symlink(filepath.Join(base, "a", "b"), link); err != nil {
		t.Skipf("cannot create symlink: %v", err)
	}
	// filepath.Join would clean the ".." away, so the path is joined by hand.
	sep := string(filepath.Separator)
	t.Setenv("DATABRICKS_CONFIG_FILE", link+sep+".."+sep+".databrickscfg")

	_, entry, err := mcp.Bind(t.Context(), mcp.BindOptions{
		Profile: bindingTestProfile,
		Service: bindingTestService,
		State:   bindingTestState(bindingTestResolved(t, cfgPath), bindingTestHost),
		Stderr:  &bytes.Buffer{},
	})
	require.NoError(t, err)
	assert.NotNil(t, entry)
}

// TestBindSkipsEntryWithoutHomeWhenAbsoluteEntryMatches checks that an entry whose "~" path cannot be
// resolved, because HOME is unset, does not stop the match against an entry with an absolute path.
func TestBindSkipsEntryWithoutHomeWhenAbsoluteEntryMatches(t *testing.T) {
	cfgPath := bindingTestConfig(t, bindingTestProfileContents(bindingTestToken))
	t.Setenv(env.HomeEnvVar(), "")
	state := &mcp.State{Servers: []mcp.Entry{
		{Name: "github-home", Service: bindingTestService, Profile: bindingTestProfile, Host: "https://adb-999.test", ConfigFile: "~/.databrickscfg"},
		{Name: "github-absolute", Service: bindingTestService, Profile: bindingTestProfile, Host: bindingTestHost, ConfigFile: cfgPath},
	}}

	_, entry, err := mcp.Bind(t.Context(), mcp.BindOptions{
		Profile: bindingTestProfile,
		Service: bindingTestService,
		State:   state,
		Stderr:  &bytes.Buffer{},
	})
	require.NoError(t, err)
	require.NotNil(t, entry)
	assert.Equal(t, "github-absolute", entry.Name)
}

// TestBindFailsWhenUnresolvableEntryMightMatch checks that an entry whose "~" path cannot be resolved
// fails the bind when no other entry matches, since the bind cannot rule that entry out.
func TestBindFailsWhenUnresolvableEntryMightMatch(t *testing.T) {
	bindingTestConfig(t, bindingTestProfileContents(bindingTestToken))
	t.Setenv(env.HomeEnvVar(), "")
	state := &mcp.State{Servers: []mcp.Entry{
		{Name: "github-home", Service: bindingTestService, Profile: bindingTestProfile, Host: bindingTestHost, ConfigFile: "~/.databrickscfg"},
	}}

	_, _, err := mcp.Bind(t.Context(), mcp.BindOptions{
		Profile: bindingTestProfile,
		Service: bindingTestService,
		State:   state,
		Stderr:  &bytes.Buffer{},
	})
	requireBindError(t, err, mcp.BindUncategorized)
	assert.Contains(t, err.Error(), "MCP server github-home: cannot find homedir: $HOME is not set")
}

// TestBindProfileWithoutHost checks that a profile section with no host is reported as having no host,
// not as missing from the file.
func TestBindProfileWithoutHost(t *testing.T) {
	path := bindingTestConfig(t, fmt.Sprintf("[%s]\nauth_type = pat\ntoken = %s\n", bindingTestProfile, bindingTestToken))

	_, _, err := mcp.Bind(t.Context(), mcp.BindOptions{
		Profile: bindingTestProfile,
		Service: bindingTestService,
		State:   bindingTestState(path, bindingTestHost),
		Stderr:  &bytes.Buffer{},
	})
	requireBindError(t, err, mcp.BindProfileNotFound)
	resolved := filepath.ToSlash(bindingTestResolved(t, path))
	assert.Equal(t, fmt.Sprintf(`MCP server github-logfood is bound to Databricks profile "logfood", which has no host in %s. Recreate it with: DATABRICKS_CONFIG_FILE=%s databricks auth login --host https://adb-123.test --profile logfood or remove the server with: databricks aitools mcp remove --names github-logfood`, resolved, shellquote.BashArg(resolved)), err.Error())
}

// TestBindHostMetadataFailureIsNotAuthFailure checks that a host whose metadata cannot be fetched does
// not fail the bind. The SDK logs that failure and goes on, and a profile with a token still binds.
func TestBindHostMetadataFailureIsNotAuthFailure(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		http.NotFound(w, r)
	}))
	t.Cleanup(server.Close)
	cfgPath := bindingTestConfig(t, fmt.Sprintf("[%s]\nhost = %s\nauth_type = pat\ntoken = %s\n", bindingTestProfile, server.URL, bindingTestToken))

	cfg, entry, err := mcp.Bind(t.Context(), mcp.BindOptions{
		Profile: bindingTestProfile,
		Service: bindingTestService,
		State:   bindingTestState(cfgPath, server.URL),
		Stderr:  &bytes.Buffer{},
	})
	require.NoError(t, err)
	assert.NotNil(t, entry)
	assert.Equal(t, bindingTestToken, cfg.Token)
}

// TestBindConfigErrorIsUncategorized checks that a configuration error from the SDK, here an
// environment value that does not parse, is reported as uncategorized, not as an authentication
// failure, so the message does not suggest logging in.
func TestBindConfigErrorIsUncategorized(t *testing.T) {
	cfgPath := bindingTestConfig(t, bindingTestProfileContents(bindingTestToken))
	t.Setenv("DATABRICKS_RATE_LIMIT", "soon")

	_, _, err := mcp.Bind(t.Context(), mcp.BindOptions{
		Profile: bindingTestProfile,
		Service: bindingTestService,
		State:   bindingTestState(cfgPath, bindingTestHost),
		Stderr:  &bytes.Buffer{},
	})
	requireBindError(t, err, mcp.BindUncategorized)
	assert.NotContains(t, err.Error(), "Run:")
	assert.NotContains(t, err.Error(), "auth login")
}

// TestBindEntryInOtherConfigFileFailsClosed checks that an entry for the service and profile, pinned to
// another config file, stops the bind when the file in use names no entry. Binding there would be a new
// host with no record of the switch.
func TestBindEntryInOtherConfigFileFailsClosed(t *testing.T) {
	recordedPath := bindingTestWriteConfig(t, bindingTestProfileContents(bindingTestToken))
	bindingTestConfig(t, bindingTestProfileContents(bindingTestToken))
	stderr := &bytes.Buffer{}

	_, _, err := mcp.Bind(t.Context(), mcp.BindOptions{
		Profile: bindingTestProfile,
		Service: bindingTestService,
		State:   bindingTestState(recordedPath, bindingTestHost),
		Stderr:  stderr,
	})
	requireBindError(t, err, mcp.BindProfileHostChanged)
	assert.Contains(t, err.Error(), "was set up with config file "+filepath.ToSlash(bindingTestResolved(t, recordedPath)))
	assert.Empty(t, stderr.String())
}

// TestBindEntryForOtherProfileKeepsNotice checks that an entry for a different profile does not stop
// a bind to a profile that has no entry, which still gets the notice.
func TestBindEntryForOtherProfileKeepsNotice(t *testing.T) {
	bindingTestConfig(t, bindingTestProfileContents(bindingTestToken))
	state := bindingTestState(bindingTestWriteConfig(t, ""), bindingTestHost)
	state.Servers[0].Profile = "other"
	stderr := &bytes.Buffer{}

	_, entry, err := mcp.Bind(t.Context(), mcp.BindOptions{
		Profile: bindingTestProfile,
		Service: bindingTestService,
		State:   state,
		Stderr:  stderr,
	})
	require.NoError(t, err)
	assert.Nil(t, entry)
	assert.Contains(t, stderr.String(), "No install record for system.ai.github")
}

// TestBindProfileWithEmptyHost checks that a profile whose host is set to nothing is reported as having
// no host, not as a profile whose host changed to "".
func TestBindProfileWithEmptyHost(t *testing.T) {
	cfgPath := bindingTestConfig(t, fmt.Sprintf("[%s]\nhost =\nauth_type = pat\ntoken = %s\n", bindingTestProfile, bindingTestToken))

	_, _, err := mcp.Bind(t.Context(), mcp.BindOptions{
		Profile: bindingTestProfile,
		Service: bindingTestService,
		State:   bindingTestState(cfgPath, bindingTestHost),
		Stderr:  &bytes.Buffer{},
	})
	requireBindError(t, err, mcp.BindProfileNotFound)
	assert.Contains(t, err.Error(), "which has no host in")
}

// TestBindRecreateHintQuotesValues checks that the recreate hint quotes a profile name that needs it,
// and names the pinned config file it was set up with.
func TestBindRecreateHintQuotesValues(t *testing.T) {
	cfgPath := bindingTestConfig(t, bindingTestProfileContents(bindingTestToken))
	state := &mcp.State{Servers: []mcp.Entry{{
		Name:       "github-logfood",
		Service:    bindingTestService,
		Profile:    "my profile",
		Host:       bindingTestHost,
		ConfigFile: cfgPath,
	}}}

	_, _, err := mcp.Bind(t.Context(), mcp.BindOptions{
		Profile: "my profile",
		Service: bindingTestService,
		State:   state,
		Stderr:  &bytes.Buffer{},
	})
	requireBindError(t, err, mcp.BindProfileNotFound)
	want := fmt.Sprintf(`MCP server github-logfood is bound to Databricks profile "my profile", which no longer exists in %s. Recreate it with: DATABRICKS_CONFIG_FILE=%s databricks auth login --host https://adb-123.test --profile 'my profile' or remove the server with: databricks aitools mcp remove --names github-logfood`,
		filepath.ToSlash(bindingTestResolved(t, cfgPath)), shellquote.BashArg(filepath.ToSlash(bindingTestResolved(t, cfgPath))))
	assert.Equal(t, want, err.Error())
}

// TestCheckBindingMatchesBind checks that CheckBinding refuses the same drift as Bind, before any request.
// bindingTestCheck runs CheckBinding for the test service and profile, with the config file resolved as the caller resolves it.
func bindingTestCheck(t *testing.T, state *mcp.State) (*mcp.Entry, error) {
	t.Helper()
	configFile, err := mcp.ResolveConfigFile(t.Context(), env.Get(t.Context(), "DATABRICKS_CONFIG_FILE"))
	require.NoError(t, err)
	return mcp.CheckBinding(t.Context(), state, bindingTestService, bindingTestProfile, configFile)
}

func TestCheckBindingMatchesBind(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		t.Errorf("unexpected request to the profile's host: %s %s", r.Method, r.URL)
		w.WriteHeader(http.StatusInternalServerError)
	}))
	t.Cleanup(server.Close)
	cfgPath := bindingTestConfig(t, fmt.Sprintf("[%s]\nhost = %s\nauth_type = pat\ntoken = %s\n", bindingTestProfile, server.URL, bindingTestToken))
	state := bindingTestState(cfgPath, bindingTestHost)

	_, err := bindingTestCheck(t, state)
	requireBindError(t, err, mcp.BindProfileHostChanged)
	assert.Equal(t, "MCP server github-logfood was set up for https://adb-123.test, but profile \"logfood\" now points at "+server.URL+". Run: databricks aitools mcp remove --names github-logfood and add it again", err.Error())
}

func TestCheckBindingPassesWhenEntryMatchesOrIsAbsent(t *testing.T) {
	cfgPath := bindingTestConfig(t, bindingTestProfileContents(bindingTestToken))
	entry, err := bindingTestCheck(t, bindingTestState(cfgPath, bindingTestHost))
	require.NoError(t, err)
	require.NotNil(t, entry, "a matching entry is returned")
	assert.Equal(t, "github-logfood", entry.Name)

	entry, err = bindingTestCheck(t, &mcp.State{})
	require.NoError(t, err)
	assert.Nil(t, entry)
}

func TestCheckBindingRefusesWorkspaceChangeOnSameHost(t *testing.T) {
	cfgPath := bindingTestConfig(t, fmt.Sprintf("[%s]\nhost = %s\nworkspace_id = 222\nauth_type = pat\ntoken = %s\n", bindingTestProfile, bindingTestHost, bindingTestToken))
	state := bindingTestState(cfgPath, bindingTestHost)
	state.Servers[0].WorkspaceID = "111"

	_, err := bindingTestCheck(t, state)
	requireBindError(t, err, mcp.BindProfileHostChanged)
}

func TestCheckBindingRefusesEntryPinnedToOtherConfigFile(t *testing.T) {
	cfgPath := bindingTestConfig(t, bindingTestProfileContents(bindingTestToken))
	other := bindingTestWriteConfig(t, bindingTestProfileContents(bindingTestToken))
	require.NotEqual(t, cfgPath, other)

	_, err := bindingTestCheck(t, bindingTestState(other, bindingTestHost))
	requireBindError(t, err, mcp.BindProfileHostChanged)
}

// TestCheckBindingMatchesEntryByConfigFileWhenOtherEntryComesFirst checks that CheckBinding returns the entry of the config file in use, even when an
// entry for the same service and profile in another file comes first in the state.
func TestCheckBindingMatchesEntryByConfigFileWhenOtherEntryComesFirst(t *testing.T) {
	cfgPath := bindingTestConfig(t, bindingTestProfileContents(bindingTestToken))
	other := bindingTestWriteConfig(t, bindingTestProfileContents(bindingTestToken))
	state := &mcp.State{Servers: []mcp.Entry{
		{Name: "github-other", Service: bindingTestService, Profile: bindingTestProfile, Host: bindingTestHost, ConfigFile: other},
		{Name: "github-default", Service: bindingTestService, Profile: bindingTestProfile, Host: bindingTestHost, ConfigFile: cfgPath},
	}}

	entry, err := bindingTestCheck(t, state)
	require.NoError(t, err)
	require.NotNil(t, entry)
	assert.Equal(t, "github-default", entry.Name)
}

// TestCheckBindingChecksProfileWithoutEntry checks that a profile that is missing, or has no host, is refused when no entry
// exists either, with the error Bind returns for it. An account-level profile has no workspace, so it has no host.
func TestCheckBindingChecksProfileWithoutEntry(t *testing.T) {
	tests := []struct {
		name     string
		contents string
		problem  string
	}{
		{name: "missing", contents: "[other]\nhost = https://other.test\n", problem: "which does not exist in "},
		{name: "account profile without host", contents: "[" + bindingTestProfile + "]\naccount_id = 123\n", problem: "which has no host in "},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			bindingTestConfig(t, tt.contents)

			_, err := bindingTestCheck(t, &mcp.State{})
			requireBindError(t, err, mcp.BindProfileNotFound)
			assert.Contains(t, err.Error(), tt.problem)

			_, _, bindErr := mcp.Bind(t.Context(), mcp.BindOptions{
				Profile: bindingTestProfile, Service: bindingTestService, State: &mcp.State{}, Stderr: io.Discard,
			})
			require.Error(t, bindErr)
			assert.Equal(t, bindErr.Error(), err.Error())
		})
	}
}

func TestAuthErrorNamesPinnedConfigFile(t *testing.T) {
	cause := errors.New("Invalid access token.")
	assert.EqualError(t, mcp.AuthError("logfood", "/work/cfg/logfood.cfg", cause),
		"Invalid access token.\nRun: DATABRICKS_CONFIG_FILE=/work/cfg/logfood.cfg databricks auth login --profile logfood")
	assert.EqualError(t, mcp.AuthError("logfood", "", cause), "Invalid access token.\nRun: databricks auth login --profile logfood")
}

// TestConfigFileCommandUsesForwardSlashes checks that a pinned config file is named with forward slashes, so that the
// hints read the same on Windows. A path that filepath builds there has backslashes, which the shell command must not
// carry. On other systems the two forms are the same, so the test guards the Windows run.
func TestConfigFileCommandUsesForwardSlashes(t *testing.T) {
	path := filepath.Join("work", "cfg", "logfood.cfg")
	assert.Equal(t, "DATABRICKS_CONFIG_FILE="+filepath.ToSlash(path)+" databricks auth login --profile logfood",
		mcp.ConfigFileCommand(path, "databricks auth login --profile logfood"))
}
