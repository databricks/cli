package installer_test

import (
	"bytes"
	"encoding/json"
	"io"
	"net/http"
	"net/http/httptest"
	"os"
	"os/exec"
	"path/filepath"
	"runtime"
	"strings"
	"testing"

	"github.com/databricks/cli/cmd/aitools"
	"github.com/databricks/cli/internal/build"
	"github.com/databricks/cli/libs/aitools/installer"
	"github.com/databricks/cli/libs/clicompat"
	"github.com/databricks/cli/libs/cmdio"
	"github.com/databricks/cli/libs/flags"
	"github.com/databricks/cli/libs/process"
	"github.com/spf13/cobra"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

type releaseTransport func(*http.Request) (*http.Response, error)

func (f releaseTransport) RoundTrip(req *http.Request) (*http.Response, error) {
	return f(req)
}

func TestCommandsWithRateLimitedSkillsRelease(t *testing.T) {
	for _, command := range []string{"list", "install"} {
		t.Run(command, func(t *testing.T) {
			homeDir := t.TempDir()
			t.Setenv("HOME", homeDir)
			t.Setenv("USERPROFILE", homeDir)
			t.Setenv("XDG_CONFIG_HOME", filepath.Join(homeDir, ".config"))
			t.Setenv("DATABRICKS_SKILLS_REF", "")
			t.Setenv("DATABRICKS_FORCE_EMBEDDED_COMPAT", "false")
			t.Setenv("DATABRICKS_CACHE_ENABLED", "true")
			t.Setenv("DATABRICKS_CACHE_DIR", homeDir)
			require.NoError(t, os.WriteFile(filepath.Join(homeDir, "compat-manifest.json"), []byte(`{"1.22.0":{"appkit":"0.84.0","skills":"latest"}}`), 0o600))
			originalVersion := build.GetInfo().Version
			build.SetBuildVersion("1.22.0")
			t.Cleanup(func() { build.SetBuildVersion(originalVersion) })
			embedded, err := clicompat.ResolveEmbeddedAgentSkillsVersion()
			require.NoError(t, err)

			server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
				assert.Equal(t, "/v"+embedded+"/manifest.json", r.URL.Path)
				_, _ = io.WriteString(w, `{"version":"1","skills":{"test-skill":{"version":"1.0.0","files":["SKILL.md"]}}}`)
			}))
			t.Cleanup(server.Close)
			t.Setenv("DATABRICKS_SKILLS_BASE_URL", server.URL)
			client := &http.Client{}
			lookups := 0
			client.Transport = releaseTransport(func(req *http.Request) (*http.Response, error) {
				if req.URL.Host == "api.github.com" {
					assert.Equal(t, "/repos/databricks/databricks-agent-skills/releases/latest", req.URL.Path)
					lookups++
					return &http.Response{StatusCode: http.StatusForbidden, Body: io.NopCloser(strings.NewReader(`{"message":"API rate limit exceeded"}`)), Header: make(http.Header)}, nil
				}
				return http.DefaultTransport.RoundTrip(req)
			})
			t.Cleanup(installer.SetHTTPClientForTest(client))
			ctx, _ := cmdio.NewTestContextWithStderr(t.Context())
			ctx, stub := process.WithStub(ctx)
			stub.WithCallback(func(*exec.Cmd) error { return nil })

			var cmd *cobra.Command
			switch command {
			case "list":
				cmd = aitools.NewListCmd()
				cmd.SetArgs([]string{"--output", "json"})
			case "install":
				// The process stub handles the agent CLI. A file on PATH is enough
				// for detection and works on Windows without a shell script.
				binary := "claude"
				if runtime.GOOS == "windows" {
					binary += ".exe"
				}
				require.NoError(t, os.WriteFile(filepath.Join(homeDir, binary), nil, 0o755))
				t.Setenv("PATH", homeDir)
				cmd = aitools.NewInstallCmd()
				cmd.SetArgs([]string{"--agents", "claude-code", "--scope", "global", "--output", "json"})
			}
			outputType := flags.OutputText
			cmd.PersistentFlags().VarP(&outputType, "output", "o", "output type")
			var stdout bytes.Buffer
			cmd.SetOut(&stdout)
			cmd.SetContext(ctx)
			require.NoError(t, cmd.Execute())
			assert.Equal(t, 1, lookups, "a failing release lookup must use the embedded version")
			var output map[string]any
			require.NoError(t, json.Unmarshal(stdout.Bytes(), &output))
			switch command {
			case "list":
				assert.Equal(t, embedded, output["release"])
				assert.NotEmpty(t, output["skills"])
				assert.NotEmpty(t, output["agents"])
			case "install":
				assert.Contains(t, strings.Join(stub.Commands(), "\n"), "plugin install databricks@claude-plugins-official --scope user")
				state, err := installer.LoadState(filepath.Join(homeDir, ".databricks", "aitools", "skills"))
				require.NoError(t, err)
				require.NotNil(t, state)
				assert.Contains(t, state.Plugins, "claude-code")
			}
		})
	}
}
