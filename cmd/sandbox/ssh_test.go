package sandbox

import (
	"io"
	"net/http"
	"net/http/httptest"
	"os"
	"path/filepath"
	"runtime"
	"strconv"
	"strings"
	"testing"

	sshtunnel "github.com/databricks/cli/experimental/ssh/sandbox"
	"github.com/databricks/cli/libs/cmdctx"
	"github.com/databricks/cli/libs/env"
	"github.com/databricks/databricks-sdk-go"
	"github.com/databricks/databricks-sdk-go/config"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestBuildSSHArgsBaseFlags(t *testing.T) {
	identity := &sshtunnel.Identity{KeyPath: "/keys/id", KnownHostsPath: "/keys/known hosts", HostKeyAlias: "sandbox-host"}
	got := buildSSHArgs(identity, "databricks sandbox ssh id --proxy", nil)
	assert.Equal(t, "sandbox-agent@sandbox-host", got[len(got)-1])
	assert.Contains(t, got, "StrictHostKeyChecking=yes")
	assert.Contains(t, got, `UserKnownHostsFile="/keys/known hosts"`)
	assert.Contains(t, got, "ProxyCommand=databricks sandbox ssh id --proxy")
}

func TestSSHClientOptionsPreserveProfileAndTarget(t *testing.T) {
	for _, test := range []struct {
		name    string
		flags   []string
		profile string
		target  string
	}{
		{name: "explicit profile and target", flags: []string{"--profile", "work", "--target", "dev"}, profile: "work", target: "dev"},
		{name: "implicit profile"},
		{name: "deprecated environment flag", flags: []string{"--environment", "dev"}, target: "dev"},
		{name: "target takes precedence", flags: []string{"--target", "dev", "--environment", "prod"}, target: "dev"},
	} {
		t.Run(test.name, func(t *testing.T) {
			cmd := newSSHCommand()
			cmd.SetContext(env.Set(t.Context(), "DATABRICKS_TOKEN", "environment-token"))
			cmd.Flags().String("profile", "", "Databricks profile")
			cmd.Flags().String("target", "", "Bundle target")
			cmd.Flags().String("environment", "", "Deprecated bundle target")
			require.NoError(t, cmd.ParseFlags(test.flags))
			cfg := &config.Config{Host: "https://workspace.test", WorkspaceID: "123", Profile: "work"}
			identity := &sshtunnel.Identity{KeyPath: "/keys/id", KnownHostsPath: "/keys/known-hosts", HostKeyAlias: "sandbox-scope"}
			options, err := sshClientOptions(cmd, cfg, "sandboxes/test", identity, "/releases", []string{"uname", "-a"})
			require.NoError(t, err)
			var proxyCommand string
			for _, arg := range options.Args {
				if strings.HasPrefix(arg, "ProxyCommand=") {
					proxyCommand = arg
				}
			}
			assert.Contains(t, proxyCommand, "sandbox ssh sandboxes/test --proxy")
			assert.Contains(t, proxyCommand, "--ssh-identity sandbox-scope")
			assert.Contains(t, proxyCommand, "--releases-dir /releases")
			if test.profile == "" {
				assert.NotContains(t, proxyCommand, "--profile")
			} else {
				assert.Contains(t, proxyCommand, "--profile "+test.profile)
			}
			if test.target == "" {
				assert.NotContains(t, proxyCommand, "--target")
			} else {
				assert.Contains(t, proxyCommand, "--target "+test.target)
			}
			assert.Equal(t, []string{"sandbox-agent@sandbox-scope", "uname", "-a"}, options.Args[len(options.Args)-3:])
			assert.Contains(t, options.Env, "DATABRICKS_HOST=https://workspace.test")
			assert.Contains(t, options.Env, "DATABRICKS_WORKSPACE_ID=123")
			assert.Contains(t, options.Env, "DATABRICKS_CONFIG_PROFILE=work")
			assert.Contains(t, options.Env, "DATABRICKS_TOKEN=environment-token")
			assert.Contains(t, options.Args, "ConnectTimeout=900")
		})
	}
}

func TestSSHProcessEnvironmentPreservesResolvedWorkspaceAndCredentials(t *testing.T) {
	ctx := env.Set(t.Context(), "DATABRICKS_HOST", "https://old-workspace.test")
	ctx = env.Set(ctx, "DATABRICKS_TOKEN", "environment-token")
	ctx = env.Set(ctx, "DATABRICKS_WORKSPACE_ID", "old-workspace")
	configuration := &config.Config{Host: "https://selected-workspace.test", WorkspaceID: "123", AccountID: "selected-account", Profile: "work", ConfigFile: "/profiles/databrickscfg"}
	actual := map[string]string{}
	for _, variable := range sshProcessEnvironment(ctx, configuration) {
		key, value, _ := strings.Cut(variable, "=")
		actual[key] = value
	}
	assert.Equal(t, configuration.Host, actual["DATABRICKS_HOST"])
	assert.Equal(t, "123", actual["DATABRICKS_WORKSPACE_ID"])
	assert.Equal(t, "selected-account", actual["DATABRICKS_ACCOUNT_ID"])
	assert.Equal(t, "work", actual["DATABRICKS_CONFIG_PROFILE"])
	assert.Equal(t, configuration.ConfigFile, actual["DATABRICKS_CONFIG_FILE"])
	assert.Equal(t, "environment-token", actual["DATABRICKS_TOKEN"])
}

func TestSSHProxyLoadsSavedConfigFile(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		_, _ = io.WriteString(w, `{}`)
	}))
	defer server.Close()
	filename := filepath.Join(t.TempDir(), "selected config")
	require.NoError(t, os.WriteFile(filename, []byte("[work]\nhost = "+server.URL+"\ntoken = saved-token\n"), 0o600))
	t.Setenv("DATABRICKS_CONFIG_FILE", filepath.Join(t.TempDir(), "missing"))
	previous := os.Getenv("DATABRICKS_CONFIG_FILE")
	cmd := newSSHCommand()
	cmd.SetContext(t.Context())
	cmd.Flags().String("profile", "", "Databricks profile")
	require.NoError(t, cmd.ParseFlags([]string{"--proxy", "--profile", "work", "--ssh-config-file", filename}))
	require.NoError(t, cmd.PreRunE(cmd, nil))
	cfg := cmdctx.WorkspaceClient(cmd.Context()).Config
	assert.Equal(t, filename, cfg.ConfigFile)
	assert.Equal(t, server.URL, cfg.Host)
	assert.Equal(t, "saved-token", cfg.Token)
	assert.Equal(t, previous, os.Getenv("DATABRICKS_CONFIG_FILE"))

	cmd = newSSHCommand()
	require.NoError(t, cmd.ParseFlags([]string{"--ssh-config-file", filename}))
	require.ErrorContains(t, cmd.PreRunE(cmd, nil), "requires --proxy")
}

func TestBuildSSHArgsQuoting(t *testing.T) {
	for _, tc := range []struct {
		name      string
		extraArgs []string
		// expected is what we expect at the end of args, after the destination.
		expected []string
	}{
		{
			name:      "no extras",
			extraArgs: nil,
			expected:  nil,
		},
		{
			name:      "single safe word — passed through",
			extraArgs: []string{"uname"},
			expected:  []string{"uname"},
		},
		{
			name:      "single string with spaces — passed through so the remote shell can split it",
			extraArgs: []string{"echo hello-from-string"},
			expected:  []string{"echo hello-from-string"},
		},
		{
			name:      "single string with a pipe — passed through to the remote shell",
			extraArgs: []string{"cat /etc/os-release | head -3"},
			expected:  []string{"cat /etc/os-release | head -3"},
		},
		{
			name:      "multi safe words — no quoting",
			extraArgs: []string{"ls", "-la", "/tmp"},
			expected:  []string{"ls", "-la", "/tmp"},
		},
		{
			name: "bash -c '<cmd>' — third arg gets quoted",
			// The whole point: without the quoting, the remote shell
			// re-splits "bash -c echo hi" and bash's -c eats just "echo".
			extraArgs: []string{"bash", "-c", "echo hi"},
			expected:  []string{"bash", "-c", "'echo hi'"},
		},
		{
			name:      "arg with single quote — escaped, not lost",
			extraArgs: []string{"echo", "it's fine"},
			expected:  []string{"echo", `'it'\''s fine'`},
		},
		{
			name:      "glob pattern — quoted so the remote (not the local) shell expands it",
			extraArgs: []string{"find", ".", "-name", "*.txt"},
			expected:  []string{"find", ".", "-name", "'*.txt'"},
		},
		{
			name:      "ssh -o key=val style — gets quoted because of '='; harmless given current arg ordering",
			extraArgs: []string{"-o", "StrictHostKeyChecking=no"},
			expected:  []string{"-o", "'StrictHostKeyChecking=no'"},
		},
		{
			name:      "empty arg gets explicit empty quotes (not lost)",
			extraArgs: []string{"sh", "-c", ""},
			expected:  []string{"sh", "-c", "''"},
		},
	} {
		t.Run(tc.name, func(t *testing.T) {
			got := buildSSHArgs(&sshtunnel.Identity{HostKeyAlias: "host"}, "proxy", tc.extraArgs)
			// Trim everything up to and including the destination so we
			// can assert just on the appended extras.
			dest := "sandbox-agent@host"
			cut := -1
			for i, a := range got {
				if a == dest {
					cut = i + 1
					break
				}
			}
			assert.GreaterOrEqual(t, cut, 0, "destination not found in args")
			tail := got[cut:]
			if tc.expected == nil {
				assert.Empty(t, tail)
			} else {
				assert.Equal(t, tc.expected, tail)
			}
		})
	}
}

func TestSSHProxyCommandQuotesArguments(t *testing.T) {
	command, err := sshProxyCommand("/path with spaces/databricks", "sandboxes/test", sshProxyOptions{Profile: "work profile", ReleasesDir: "/release dir"})
	require.NoError(t, err)
	if runtime.GOOS == "windows" {
		assert.Equal(t, `"/path with spaces/databricks" sandbox ssh sandboxes/test --proxy --profile "work profile" --releases-dir "/release dir"`, command)
	} else {
		assert.Equal(t, "'/path with spaces/databricks' sandbox ssh sandboxes/test --proxy --profile 'work profile' --releases-dir '/release dir'", command)
	}
	percent, percentErr := sshProxyCommand("databricks", "sandboxes/test", sshProxyOptions{Profile: "a%h"})
	if runtime.GOOS == "windows" {
		require.ErrorContains(t, percentErr, "unsupported Windows SSH proxy argument")
	} else {
		require.NoError(t, percentErr)
		assert.Equal(t, "databricks sandbox ssh sandboxes/test --proxy --profile 'a%%h'", percent)
	}
	_, err = sshProxyCommand("databricks", "sandboxes/test", sshProxyOptions{Profile: "bad\nprofile"})
	require.ErrorContains(t, err, "control characters")
	scoped, err := sshProxyCommand("databricks", "sandboxes/test", sshProxyOptions{Profile: "work", IdentityScope: "sandbox-scope", Target: "dev"})
	require.NoError(t, err)
	assert.Equal(t, "databricks sandbox ssh sandboxes/test --proxy --profile work --ssh-identity sandbox-scope --target dev", scoped)
	saved, err := sshProxyCommand("databricks", "sandboxes/test", sshProxyOptions{Profile: "work", ConfigFile: "/profiles/custom config"})
	require.NoError(t, err)
	if runtime.GOOS == "windows" {
		assert.Equal(t, `databricks sandbox ssh sandboxes/test --proxy --profile work --ssh-config-file "/profiles/custom config"`, saved)
	} else {
		assert.Equal(t, "databricks sandbox ssh sandboxes/test --proxy --profile work --ssh-config-file '/profiles/custom config'", saved)
	}
}

func TestSSHProxyCommandDispatchAndIdentityMismatch(t *testing.T) {
	for _, mismatch := range []bool{false, true} {
		t.Run(strconv.FormatBool(mismatch), func(t *testing.T) {
			ctx := env.WithUserHomeDir(t.Context(), t.TempDir())
			var paths []string
			endpoint := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
				paths = append(paths, r.URL.Path)
				switch r.URL.Path {
				case "/.well-known/databricks-config":
					_, _ = io.WriteString(w, `{}`)
				case "/api/2.0/preview/scim/v2/Me":
					_, _ = io.WriteString(w, `{"id":"caller"}`)
				default:
					_, _ = io.WriteString(w, `{"status":{"state":"SANDBOX_STATE_UNSPECIFIED"}}`)
				}
			}))
			defer endpoint.Close()
			w, err := databricks.NewWorkspaceClient(&databricks.Config{Host: endpoint.URL, Token: "test-token", WorkspaceID: "123"})
			require.NoError(t, err)
			cmd := newSSHCommand()
			cmd.PreRunE = nil
			args := []string{"test", "--proxy"}
			if mismatch {
				args = append(args, "--ssh-identity", "different-workspace")
			}
			cmd.SetArgs(args)
			cmd.SetOut(io.Discard)
			cmd.SetErr(io.Discard)
			err = cmd.ExecuteContext(cmdctx.SetWorkspaceClient(ctx, w))
			if mismatch {
				require.ErrorContains(t, err, "workspace or caller changed")
			} else {
				require.ErrorContains(t, err, "unsupported state")
			}
			assert.Contains(t, paths, "/api/2.0/sandboxes/test")
			assert.Contains(t, paths, "/api/2.0/preview/scim/v2/Me")
		})
	}
}

func TestSSHCommandRejectsExtraSandboxArguments(t *testing.T) {
	cmd := newSSHCommand()
	// Argument validation runs before credentials are resolved by RunE.
	cmd.PreRunE = nil
	cmd.SetArgs([]string{"first", "second"})
	require.ErrorContains(t, cmd.ExecuteContext(t.Context()), "at most one sandbox")
	cmd = newSSHCommand()
	cmd.PreRunE = nil
	cmd.SetArgs([]string{"first", "--proxy", "--", "echo"})
	require.ErrorContains(t, cmd.ExecuteContext(t.Context()), "--proxy does not accept SSH arguments")
}

func TestResolveSSHName(t *testing.T) {
	ctx := env.WithUserHomeDir(t.Context(), t.TempDir())
	calls := 0
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.URL.Path == "/.well-known/databricks-config" {
			_, _ = io.WriteString(w, `{}`)
			return
		}
		calls++
		if strings.HasSuffix(r.URL.Path, "missing") {
			w.WriteHeader(http.StatusNotFound)
			_, _ = io.WriteString(w, `{"error_code":"NOT_FOUND","message":"missing"}`)
			return
		}
		_, _ = io.WriteString(w, `{"name":"sandboxes/test","status":{"state":"SANDBOX_STATE_RUNNING"}}`)
	}))
	defer server.Close()
	w, err := databricks.NewWorkspaceClient(&databricks.Config{Host: server.URL, Token: "test-token"})
	require.NoError(t, err)
	require.NoError(t, setDefault(ctx, server.URL, "missing"))
	_, err = resolveSSHName(ctx, w, "")
	require.ErrorContains(t, err, "unavailable through the Sandbox API")
	assert.Equal(t, "missing", getDefault(ctx, server.URL))
	_, err = resolveSSHName(ctx, w, "a/b")
	require.ErrorContains(t, err, "invalid sandbox ID")
	name, err := resolveSSHName(ctx, w, "sandboxes/test")
	require.NoError(t, err)
	assert.Equal(t, "sandboxes/test", name)
	assert.Equal(t, 2, calls)
}

func TestPrepareSSHIdentityUsesWorkspaceAndCaller(t *testing.T) {
	ctx := env.WithUserHomeDir(t.Context(), t.TempDir())
	failure := false
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.URL.Path == "/.well-known/databricks-config" {
			_, _ = io.WriteString(w, `{}`)
			return
		}
		assert.Equal(t, "/api/2.0/preview/scim/v2/Me", r.URL.Path)
		if failure {
			http.Error(w, `{"error_code":"PERMISSION_DENIED","message":"denied"}`, http.StatusForbidden)
			return
		}
		_, _ = io.WriteString(w, `{"id":"caller"}`)
	}))
	defer server.Close()
	w, err := databricks.NewWorkspaceClient(&databricks.Config{Host: server.URL, Token: "test-token", WorkspaceID: "123"})
	require.NoError(t, err)
	got, err := prepareSSHIdentity(ctx, w, "sandboxes/test")
	require.NoError(t, err)
	expected, err := sshtunnel.LocalIdentity(ctx, server.URL+"/123", "caller", "sandboxes/test")
	require.NoError(t, err)
	assert.Equal(t, expected.KeyPath, got.KeyPath)
	failure = true
	_, err = prepareSSHIdentity(ctx, w, "sandboxes/test")
	require.ErrorContains(t, err, "resolve SSH caller")
}
