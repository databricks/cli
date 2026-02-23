package completion

import (
	"os"
	"path/filepath"
	"runtime"
	"strings"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestInstallFreshZsh(t *testing.T) {
	home := t.TempDir()

	filePath, alreadyInstalled, err := Install(Zsh, home)
	require.NoError(t, err)
	assert.False(t, alreadyInstalled)
	assert.Equal(t, filepath.Join(home, ".zshrc"), filePath)

	content, err := os.ReadFile(filePath)
	require.NoError(t, err)
	assert.Contains(t, string(content), BeginMarker)
	assert.Contains(t, string(content), EndMarker)
	assert.Contains(t, string(content), `eval "$(databricks completion zsh)"`)
}

func TestInstallIdempotent(t *testing.T) {
	home := t.TempDir()

	_, _, err := Install(Zsh, home)
	require.NoError(t, err)

	filePath, alreadyInstalled, err := Install(Zsh, home)
	require.NoError(t, err)
	assert.True(t, alreadyInstalled)

	content, err := os.ReadFile(filePath)
	require.NoError(t, err)
	assert.Equal(t, 1, strings.Count(string(content), BeginMarker))
}

func TestInstallAppendsToExistingFile(t *testing.T) {
	home := t.TempDir()
	rcPath := filepath.Join(home, ".zshrc")
	require.NoError(t, os.WriteFile(rcPath, []byte("# existing config\n"), 0o644))

	_, _, err := Install(Zsh, home)
	require.NoError(t, err)

	content, err := os.ReadFile(rcPath)
	require.NoError(t, err)
	assert.True(t, strings.HasPrefix(string(content), "# existing config\n"))
	assert.Contains(t, string(content), BeginMarker)
}

func TestInstallAddsNewlineIfMissing(t *testing.T) {
	home := t.TempDir()
	rcPath := filepath.Join(home, ".zshrc")
	require.NoError(t, os.WriteFile(rcPath, []byte("# no trailing newline"), 0o644))

	_, _, err := Install(Zsh, home)
	require.NoError(t, err)

	content, err := os.ReadFile(rcPath)
	require.NoError(t, err)
	assert.Contains(t, string(content), "# no trailing newline\n"+BeginMarker)
}

func TestInstallPreservesPermissions(t *testing.T) {
	home := t.TempDir()
	rcPath := filepath.Join(home, ".zshrc")
	require.NoError(t, os.WriteFile(rcPath, []byte(""), 0o600))

	_, _, err := Install(Zsh, home)
	require.NoError(t, err)

	info, err := os.Stat(rcPath)
	require.NoError(t, err)

	if runtime.GOOS != "windows" {
		assert.Equal(t, os.FileMode(0o600), info.Mode().Perm())
	} else {
		// Windows has different permission semantics; verify file remains writable.
		err = os.WriteFile(rcPath, []byte("# writable"), 0o600)
		assert.NoError(t, err)
	}
}

func TestInstallFish(t *testing.T) {
	home := t.TempDir()

	filePath, alreadyInstalled, err := Install(Fish, home)
	require.NoError(t, err)
	assert.False(t, alreadyInstalled)
	assert.Equal(t, filepath.Join(home, ".config", "fish", "completions", "databricks.fish"), filePath)

	content, err := os.ReadFile(filePath)
	require.NoError(t, err)
	assert.Contains(t, string(content), "databricks completion fish | source")
}

func TestInstallFishForeignFilePreserved(t *testing.T) {
	home := t.TempDir()
	filePath := filepath.Join(home, ".config", "fish", "completions", "databricks.fish")
	require.NoError(t, os.MkdirAll(filepath.Dir(filePath), 0o755))

	original := "# fish completion from package manager\n"
	require.NoError(t, os.WriteFile(filePath, []byte(original), 0o644))

	gotPath, alreadyInstalled, err := Install(Fish, home)
	require.NoError(t, err)
	assert.True(t, alreadyInstalled)
	assert.Equal(t, filePath, gotPath)

	content, err := os.ReadFile(filePath)
	require.NoError(t, err)
	assert.Equal(t, original, string(content))
}

func TestInstallFishIdempotent(t *testing.T) {
	home := t.TempDir()

	_, _, err := Install(Fish, home)
	require.NoError(t, err)

	_, alreadyInstalled, err := Install(Fish, home)
	require.NoError(t, err)
	assert.True(t, alreadyInstalled)
}

func TestInstallFishCreatesDirectory(t *testing.T) {
	home := t.TempDir()
	fishDir := filepath.Join(home, ".config", "fish", "completions")

	_, err := os.Stat(fishDir)
	assert.True(t, os.IsNotExist(err))

	_, _, err = Install(Fish, home)
	require.NoError(t, err)

	_, err = os.Stat(fishDir)
	assert.NoError(t, err)
}

func TestInstallPowerShellCreatesDirectory(t *testing.T) {
	home := t.TempDir()

	filePath, _, err := Install(PowerShell, home)
	require.NoError(t, err)

	_, err = os.Stat(filepath.Dir(filePath))
	assert.NoError(t, err)

	content, err := os.ReadFile(filePath)
	require.NoError(t, err)
	assert.Contains(t, string(content), "databricks completion powershell | Out-String | Invoke-Expression")
}

func TestInstallBashShimContent(t *testing.T) {
	home := t.TempDir()

	_, _, err := Install(Bash, home)
	require.NoError(t, err)

	filePath := TargetFilePath(Bash, home)
	content, err := os.ReadFile(filePath)
	require.NoError(t, err)
	assert.Contains(t, string(content), `eval "$(databricks completion bash)"`)
}

func TestInstallEmptyFile(t *testing.T) {
	home := t.TempDir()
	rcPath := filepath.Join(home, ".zshrc")
	require.NoError(t, os.WriteFile(rcPath, []byte(""), 0o644))

	_, _, err := Install(Zsh, home)
	require.NoError(t, err)

	content, err := os.ReadFile(rcPath)
	require.NoError(t, err)
	assert.Contains(t, string(content), BeginMarker)
}
