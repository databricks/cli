package ucm_test

import (
	"bytes"
	"context"
	"os"
	"path/filepath"
	"testing"

	cmdUcm "github.com/databricks/cli/cmd/ucm"
	"github.com/databricks/cli/libs/cmdctx"
	"github.com/databricks/cli/libs/cmdio"
	"github.com/databricks/cli/libs/logdiag"
	"github.com/databricks/cli/libs/telemetry"
	"github.com/databricks/databricks-sdk-go/config"
	"github.com/databricks/databricks-sdk-go/experimental/mocks"
	"github.com/spf13/cobra"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// runInit invokes `databricks ucm init <args...>` inside a temp dir that
// doubles as the template's output parent. Returns stdout, stderr, and the
// directory the project was materialized into.
func runInit(t *testing.T, args ...string) (string, string, string, error) {
	t.Helper()

	workDir := t.TempDir()
	prev, err := os.Getwd()
	require.NoError(t, err)
	require.NoError(t, os.Chdir(workDir))
	t.Cleanup(func() { _ = os.Chdir(prev) })

	cmd := cmdUcm.New()
	// init's PreRunE resolves a real workspace client; tests inject a mock via
	// the context instead so they don't depend on ~/.databrickscfg.
	stripInitAuthHook(cmd)
	var out, errOut bytes.Buffer
	cmd.SetOut(&out)
	cmd.SetErr(&errOut)
	cmd.SetArgs(append([]string{"init"}, args...))

	ctx, diagOut := cmdio.NewTestContextWithStderr(context.Background())
	ctx = logdiag.InitContext(ctx)
	ctx = telemetry.WithNewLogger(ctx)
	logdiag.SetRoot(ctx, workDir)

	m := mocks.NewMockWorkspaceClient(t)
	m.WorkspaceClient.Config = &config.Config{Host: "https://example.cloud.databricks.com"}
	ctx = cmdctx.SetWorkspaceClient(ctx, m.WorkspaceClient)

	cmd.SetContext(ctx)

	err = cmd.Execute()
	return out.String(), diagOut.String() + errOut.String(), workDir, err
}

func stripInitAuthHook(cmd *cobra.Command) {
	for _, sub := range cmd.Commands() {
		if sub.Name() == "init" {
			sub.PreRunE = nil
			sub.PreRun = nil
		}
	}
}

func TestCmd_Init_Default_MaterializesValidProject(t *testing.T) {
	_, _, workDir, err := runInit(t, "default")
	require.NoError(t, err)

	projectDir := filepath.Join(workDir, "my_ucm_project")
	info, err := os.Stat(projectDir)
	require.NoError(t, err)
	assert.True(t, info.IsDir())

	ucmYml := filepath.Join(projectDir, "ucm.yml")
	data, err := os.ReadFile(ucmYml)
	require.NoError(t, err)
	assert.Contains(t, string(data), "name: my_ucm_project")
	assert.Contains(t, string(data), "team_alpha")
	assert.Contains(t, string(data), "bronze")

	_, err = os.Stat(filepath.Join(projectDir, "README.md"))
	require.NoError(t, err)
}

func TestCmd_Init_Brownfield_EmitsStub(t *testing.T) {
	_, _, workDir, err := runInit(t, "brownfield")
	require.NoError(t, err)

	ucmYml := filepath.Join(workDir, "my_ucm_project", "ucm.yml")
	data, err := os.ReadFile(ucmYml)
	require.NoError(t, err)
	assert.Contains(t, string(data), "databricks ucm generate")
}

func TestCmd_Init_Multienv_DeclaresTargets(t *testing.T) {
	_, _, workDir, err := runInit(t, "multienv")
	require.NoError(t, err)

	ucmYml := filepath.Join(workDir, "my_ucm_project", "ucm.yml")
	data, err := os.ReadFile(ucmYml)
	require.NoError(t, err)
	assert.Contains(t, string(data), "targets:")
	assert.Contains(t, string(data), "dev:")
	assert.Contains(t, string(data), "staging:")
	assert.Contains(t, string(data), "prod:")
}

func TestCmd_Init_UnknownTemplate_ReturnsError(t *testing.T) {
	_, _, _, err := runInit(t, "no-such-template")
	require.Error(t, err)
}

func TestCmd_Init_TagAndBranchConflict(t *testing.T) {
	_, _, _, err := runInit(t, "default", "--tag", "v1", "--branch", "main")
	require.Error(t, err)
	assert.Contains(t, err.Error(), "only one of --tag or --branch")
}

// TestCmd_Init_Default_ValidatesCleanly verifies the rendered project passes
// `ucm validate` — the happy path contract the starter template must keep.
func TestCmd_Init_Default_ValidatesCleanly(t *testing.T) {
	_, _, workDir, err := runInit(t, "default")
	require.NoError(t, err)

	projectDir := filepath.Join(workDir, "my_ucm_project")

	prev, err := os.Getwd()
	require.NoError(t, err)
	require.NoError(t, os.Chdir(projectDir))
	t.Cleanup(func() { _ = os.Chdir(prev) })

	cmd := cmdUcm.New()
	var out, errOut bytes.Buffer
	cmd.SetOut(&out)
	cmd.SetErr(&errOut)
	cmd.SetArgs([]string{"validate"})

	ctx, diagOut := cmdio.NewTestContextWithStderr(context.Background())
	ctx = logdiag.InitContext(ctx)
	logdiag.SetRoot(ctx, projectDir)
	cmd.SetContext(ctx)

	err = cmd.Execute()
	require.NoError(t, err, "stdout=%q stderr=%q", out.String(), diagOut.String()+errOut.String())
	assert.Contains(t, out.String(), "Validation OK!")
}
