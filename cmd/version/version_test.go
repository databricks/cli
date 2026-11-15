package version

import (
	"bytes"
	"os"
	"path/filepath"
	"testing"

	"github.com/databricks/cli/internal/testutil"
	"github.com/databricks/cli/libs/aitools/agents"
	"github.com/databricks/cli/libs/aitools/installer"
	"github.com/databricks/cli/libs/cmdio"
	"github.com/databricks/cli/libs/dbr"
	"github.com/databricks/cli/libs/flags"
	"github.com/databricks/cli/libs/versioncheck"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestUpdateCheckTemplate(t *testing.T) {
	tests := []struct {
		name   string
		result versioncheck.Result
		want   string
	}{
		{
			name: "update available with upgrade command",
			result: versioncheck.Result{
				CurrentVersion:  "0.240.0",
				LatestVersion:   "0.245.0",
				UpdateAvailable: true,
				InstallMethod:   versioncheck.InstallHomebrew,
				UpgradeCommand:  "brew upgrade databricks",
			},
			want: `Databricks CLI v0.240.0
A new version is available: 0.245.0
To upgrade, run:
  brew upgrade databricks
`,
		},
		{
			name: "update available without known install method",
			result: versioncheck.Result{
				CurrentVersion:  "0.240.0",
				LatestVersion:   "0.245.0",
				UpdateAvailable: true,
				InstallMethod:   versioncheck.InstallUnknown,
			},
			want: `Databricks CLI v0.240.0
A new version is available: 0.245.0
Download the latest release: https://github.com/databricks/cli/releases
`,
		},
		{
			name: "up to date",
			result: versioncheck.Result{
				CurrentVersion:  "0.245.0",
				LatestVersion:   "0.245.0",
				UpdateAvailable: false,
			},
			want: `Databricks CLI v0.245.0
You're on the latest version.
`,
		},
		{
			name: "development build",
			result: versioncheck.Result{
				CurrentVersion:   "0.0.0-dev+abc123",
				DevelopmentBuild: true,
			},
			want: `Databricks CLI v0.0.0-dev+abc123
This is a development build; skipping the update check.
`,
		},
		{
			name: "check failed",
			result: versioncheck.Result{
				CurrentVersion: "0.240.0",
				CheckFailed:    true,
			},
			want: `Databricks CLI v0.240.0
Could not reach GitHub to check for a newer version. See https://github.com/databricks/cli/releases for the latest release.
`,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			ctx, out := cmdio.NewTestContextWithStdout(t.Context())
			err := cmdio.RenderWithTemplate(ctx, tt.result, "", updateCheckTemplate)
			require.NoError(t, err)
			assert.Equal(t, tt.want, out.String())
		})
	}
}

func TestRenderUpdateCheckAIToolsTip(t *testing.T) {
	upToDate := &versioncheck.Result{CurrentVersion: "0.245.0", LatestVersion: "0.245.0"}
	devBuild := &versioncheck.Result{CurrentVersion: "0.0.0-dev+abc123", DevelopmentBuild: true}
	upToDateText := "Databricks CLI v0.245.0\nYou're on the latest version.\n"
	tip := "Tip: coding agents detected without Databricks skills. Run 'databricks aitools install' to install them.\n"

	tests := []struct {
		name                string
		result              *versioncheck.Result
		output              flags.Output
		recordedCodexPlugin bool
		wantStdout          string
		wantStderr          string
	}{
		{
			name:       "text output prints the tip on stderr",
			result:     upToDate,
			output:     flags.OutputText,
			wantStdout: upToDateText,
			wantStderr: tip,
		},
		{
			name:                "codex plugin recorded in the install state",
			result:              upToDate,
			output:              flags.OutputText,
			recordedCodexPlugin: true,
			wantStdout:          upToDateText,
		},
		{
			name:       "development build prints the tip too",
			result:     devBuild,
			output:     flags.OutputText,
			wantStdout: "Databricks CLI v0.0.0-dev+abc123\nThis is a development build; skipping the update check.\n",
			wantStderr: tip,
		},
		{
			name:       "json output",
			result:     upToDate,
			output:     flags.OutputJSON,
			wantStdout: "{\n  \"current_version\": \"0.245.0\",\n  \"latest_version\": \"0.245.0\",\n  \"update_available\": false\n}\n",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			home := t.TempDir()
			testutil.SetAgentEnv(t, "HOME", home, "USERPROFILE", home)
			t.Chdir(t.TempDir())
			// Codex CLI is detected by its config dir; a plugin install through
			// `databricks aitools install` is recorded only in the install state.
			require.NoError(t, os.MkdirAll(filepath.Join(home, ".codex"), 0o755))
			if tt.recordedCodexPlugin {
				state := &installer.InstallState{
					Plugins: map[string]installer.PluginRecord{
						"codex": {Plugin: "databricks", Version: "0.2.6"},
					},
				}
				require.NoError(t, installer.SaveState(filepath.Join(home, agents.CanonicalSkillsDir), state))
			}

			var stdout, stderr bytes.Buffer
			ctx := dbr.MockRuntime(t.Context(), dbr.Environment{})
			ctx = cmdio.InContext(ctx, cmdio.NewIO(ctx, tt.output, nil, &stdout, &stderr, "", ""))
			err := renderUpdateCheck(ctx, tt.result, tt.output)
			require.NoError(t, err)
			assert.Equal(t, tt.wantStdout, stdout.String())
			assert.Equal(t, tt.wantStderr, stderr.String())
		})
	}
}
