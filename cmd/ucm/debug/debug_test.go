package debug_test

import (
	"bytes"
	"context"
	"encoding/json"
	"os"
	"path/filepath"
	"runtime"
	"strings"
	"testing"

	cmdUcm "github.com/databricks/cli/cmd/ucm"
	"github.com/databricks/cli/cmd/ucm/debug"
	"github.com/databricks/cli/cmd/ucm/utils"
	"github.com/databricks/cli/libs/cmdio"
	"github.com/databricks/cli/libs/flags"
	"github.com/databricks/cli/ucm"
	"github.com/databricks/cli/ucm/config"
	"github.com/databricks/cli/ucm/deploy"
	"github.com/databricks/cli/ucm/deploy/direct"
	ucmtf "github.com/databricks/cli/ucm/deploy/terraform"
	"github.com/databricks/databricks-sdk-go/experimental/mocks"
	"github.com/databricks/databricks-sdk-go/service/iam"
	"github.com/spf13/cobra"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// runDebug invokes `<subcmd> <extra>` on the debug group rooted at workDir.
// Returns captured stdout+stderr exactly like cmd/ucm/helpers_test.go's
// runVerb — kept local because the debug subpackage can't import the `ucm`
// test helpers (cycle: ucm -> debug).
//
// Mirrors the runVerbInDir shape: build a tiny cobra root that owns the
// persistent --output flag, install cmdio on the context, then let the
// production code run logdiag.InitContext. No pre-init here so ProcessUcm
// (called by the states subcommand) does not panic on double-init.
func runDebug(t *testing.T, workDir string, args ...string) (string, string, error) {
	t.Helper()
	t.Chdir(workDir)
	setupDebugTestEnv(t)
	installDebugTestHook(t)

	rootCmd := &cobra.Command{Use: "test-root", SilenceUsage: true, SilenceErrors: true}
	output := flags.OutputText
	rootCmd.PersistentFlags().VarP(&output, "output", "o", "output type: text or json")
	rootCmd.AddGroup(&cobra.Group{ID: "development", Title: "Development"})

	// Build the full ucm subtree so the persistent --var / --target / --profile
	// flags ProcessUcm reads off cmd.Flags() are wired. The standalone
	// debug.New() output omits them.
	ucmCmd := cmdUcm.New()
	rootCmd.AddCommand(ucmCmd)
	stripHooks(ucmCmd)

	out := &bytes.Buffer{}
	errOut := &bytes.Buffer{}
	rootCmd.SetOut(out)
	rootCmd.SetErr(errOut)
	rootCmd.SetArgs(append([]string{"ucm", "debug"}, args...))

	cmdIO := cmdio.NewIO(t.Context(), flags.OutputText, nil, out, errOut, "", "")
	rootCmd.SetContext(cmdio.InContext(t.Context(), cmdIO))

	err := rootCmd.Execute()
	return out.String(), errOut.String(), err
}

// installDebugTestHook installs a utils.TestProcessHook for the duration of
// the test so the debug states subcommand (which calls ProcessUcm) finds a
// pre-seeded CurrentUser + mock WorkspaceClient. Mirrors the helper in
// cmd/ucm/helpers_test.go but lives here to avoid the import cycle.
func installDebugTestHook(t *testing.T) {
	t.Helper()
	prev := utils.TestProcessHook
	utils.TestProcessHook = func(ctx context.Context, u *ucm.Ucm) {
		if prev != nil {
			prev(ctx, u)
		}
		if u == nil {
			return
		}
		if u.CurrentUser == nil {
			u.CurrentUser = &config.User{
				ShortName: "test-user",
				User:      &iam.User{UserName: "test-user@example.com"},
			}
		}
		m := mocks.NewMockWorkspaceClient(t)
		u.SetWorkspaceClient(m.WorkspaceClient)
	}
	t.Cleanup(func() { utils.TestProcessHook = prev })
}

// setupDebugTestEnv mirrors helpers_test.setupTestEnvironment for the debug
// subpackage. Kept local so cmd/ucm/debug doesn't have to import test code
// from cmd/ucm.
func setupDebugTestEnv(t *testing.T) {
	t.Helper()
	tempHomeDir := t.TempDir()
	homeEnvVar := "HOME"
	if runtime.GOOS == "windows" {
		homeEnvVar = "USERPROFILE"
	}
	t.Setenv("DATABRICKS_CONFIG_FILE", filepath.Join(tempHomeDir, "missing-databrickscfg"))
	t.Setenv(homeEnvVar, tempHomeDir)
	t.Setenv("DATABRICKS_TOKEN", "test-token")
	t.Setenv("DATABRICKS_AUTH_TYPE", "pat")
	t.Setenv("DATABRICKS_CONFIG_PROFILE", "")
	t.Setenv("DATABRICKS_HOST", "")
	t.Setenv("DATABRICKS_METADATA_SERVICE_URL", "")
	if runtime.GOOS == "windows" {
		t.Setenv("PATH", `C:\Windows\System32`)
	} else {
		t.Setenv("PATH", "/usr/bin:/bin")
	}
}

// stripHooks mirrors cmd/ucm/helpers_test.go.stripAuthHooks so tests don't
// hit the workspace-client auth path for verbs that (today) don't need it.
func stripHooks(cmd *cobra.Command) {
	cmd.PersistentPreRunE = nil
	cmd.PersistentPreRun = nil
	cmd.PreRunE = nil
	cmd.PreRun = nil
	for _, sub := range cmd.Commands() {
		stripHooks(sub)
	}
}

func TestDebug_Hidden(t *testing.T) {
	cmd := debug.New()
	assert.True(t, cmd.Hidden, "debug group must be Hidden to match cmd/bundle/debug")
}

func TestDebug_HasSubcommands(t *testing.T) {
	cmd := debug.New()
	names := map[string]bool{}
	for _, sub := range cmd.Commands() {
		names[sub.Name()] = true
	}
	assert.True(t, names["terraform"], "debug group must wire the terraform subcommand")
	assert.True(t, names["states"], "debug group must wire the states subcommand")
}

func TestDebug_Terraform_PrintsVersionsText(t *testing.T) {
	stdout, stderr, err := runDebug(t, t.TempDir(), "terraform")
	t.Logf("stdout=%q stderr=%q", stdout, stderr)

	require.NoError(t, err)
	assert.Contains(t, stdout, ucmtf.ProviderVersion, "text output must include the pinned provider version")
	assert.Contains(t, stdout, "Databricks Terraform Provider version", "template header must be rendered")
	assert.Contains(t, stdout, "DATABRICKS_TF_EXEC_PATH", "air-gap env-var instructions must be rendered")
}

func TestDebug_Terraform_JSON(t *testing.T) {
	// Manually create a root cobra so the `-o json` persistent flag is wired
	// like it is under `databricks ucm debug terraform -o json` in production.
	// The flag must be a *flags.Output for root.OutputType's type assertion
	// to succeed — a plain string would panic.
	rootCmd := &cobra.Command{Use: "root"}
	output := flags.OutputJSON
	rootCmd.PersistentFlags().VarP(&output, "output", "o", "output type: text or json")
	rootCmd.AddCommand(debug.NewTerraformCommand())

	var out bytes.Buffer
	rootCmd.SetOut(&out)
	rootCmd.SetErr(&out)
	rootCmd.SetArgs([]string{"terraform", "-o", "json"})
	rootCmd.SetContext(context.Background())

	require.NoError(t, rootCmd.Execute())

	var payload struct {
		Terraform struct {
			Version         string `json:"version"`
			ProviderVersion string `json:"providerVersion"`
			ProviderSource  string `json:"providerSource"`
			ProviderHost    string `json:"providerHost"`
		} `json:"terraform"`
	}
	require.NoError(t, json.Unmarshal(out.Bytes(), &payload))
	assert.Equal(t, ucmtf.ProviderVersion, payload.Terraform.ProviderVersion)
	assert.Equal(t, ucmtf.ProviderSource, payload.Terraform.ProviderSource)
	assert.NotEmpty(t, payload.Terraform.Version)
	assert.NotEmpty(t, payload.Terraform.ProviderHost)
}

// writeUcmFixture seeds a minimal ucm.yml in dir so ProcessUcm can load and
// select the default target. Matches the valid-fixture shape but is local to
// the debug tests so they don't cross-reference cmd/ucm/testdata.
func writeUcmFixture(t *testing.T, dir string) {
	t.Helper()
	body := `ucm:
  name: debug-states-fixture

workspace:
  host: https://example.cloud.databricks.com

resources:
  catalogs:
    c:
      name: c
`
	require.NoError(t, os.WriteFile(filepath.Join(dir, "ucm.yml"), []byte(body), 0o644))
}

// seedStateDir creates .databricks/ucm/<target>/ with the three files the
// states command scans for. Sizes are distinct so assertions can key on them.
func seedStateDir(t *testing.T, root, target string) {
	t.Helper()
	dir := filepath.Join(root, filepath.FromSlash(deploy.LocalCacheDir), target)
	require.NoError(t, os.MkdirAll(filepath.Join(dir, "terraform"), 0o755))
	require.NoError(t, os.WriteFile(
		filepath.Join(dir, deploy.UcmStateFileName),
		[]byte(`{"version":1,"seq":3}`),
		0o644,
	))
	require.NoError(t, os.WriteFile(
		filepath.Join(dir, "terraform", deploy.TfStateFileName),
		[]byte(`{"version":4,"resources":[]}`),
		0o644,
	))
	require.NoError(t, os.WriteFile(
		filepath.Join(dir, direct.StateFileName),
		[]byte(`{"version":1}`),
		0o644,
	))
}

func TestDebug_States_ListsSeededFiles(t *testing.T) {
	work := t.TempDir()
	writeUcmFixture(t, work)
	// The fixture declares no explicit target, so LoadDefaultTarget
	// synthesises the "default" target — mirror summary_test.go's convention.
	seedStateDir(t, work, "default")

	stdout, stderr, err := runDebug(t, work, "states")
	t.Logf("stdout=%q stderr=%q", stdout, stderr)

	require.NoError(t, err)
	// Each of the three seeded files must appear, referenced by its basename.
	assert.Contains(t, stdout, deploy.UcmStateFileName)
	assert.Contains(t, stdout, deploy.TfStateFileName)
	assert.Contains(t, stdout, direct.StateFileName)
	// Forward-slashes only — matches the style guide and keeps output stable
	// across OSes. Reject any backslash sneaking into the rendered path.
	for _, line := range strings.Split(strings.TrimSpace(stdout), "\n") {
		assert.NotContains(t, line, `\`, "state paths must be forward-slashed")
	}
}

func TestDebug_States_NoStateFilesPlaceholder(t *testing.T) {
	work := t.TempDir()
	writeUcmFixture(t, work)

	stdout, stderr, err := runDebug(t, work, "states")
	t.Logf("stdout=%q stderr=%q", stdout, stderr)

	require.NoError(t, err)
	assert.Contains(t, stdout, "No state files found.")
}

func TestDebug_States_ForcePullFlagExists(t *testing.T) {
	// Flag is wired but intentionally a no-op pending #57. Guard the wiring
	// so the next engineer doesn't accidentally drop it when implementing
	// the real pull path.
	cmd := debug.NewStatesCommand()
	require.NotNil(t, cmd.Flag("force-pull"), "force-pull flag must stay wired (TODO #57)")
}
