package ucm

import (
	"os"
	"path/filepath"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// runInit invokes `databricks ucm init <args...>` inside a temp dir that
// doubles as the template's output parent. Returns stdout, stderr, and the
// directory the project was materialized into. Wraps runVerbInDir so init
// shares the auth + cmdio + cobra-root setup with the rest of the verb tests.
func runInit(t *testing.T, args ...string) (string, string, string, error) {
	t.Helper()

	workDir := t.TempDir()
	stdout, stderr, err := runVerbInDir(t, workDir, append([]string{"init"}, args...)...)
	return stdout, stderr, workDir, err
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
	stdout, stderr, err := runVerbInDir(t, projectDir, "validate")
	require.NoError(t, err, "stdout=%q stderr=%q", stdout, stderr)
	assert.Contains(t, stdout, "Validation OK!")
}
