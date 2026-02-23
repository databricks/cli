package completion

import (
	"os"
	"path/filepath"
	"runtime"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestUninstallRemovesBlock(t *testing.T) {
	home := t.TempDir()
	rcPath := filepath.Join(home, ".zshrc")
	content := "# before\n" + ShimContent(Zsh) + "# after\n"
	require.NoError(t, os.WriteFile(rcPath, []byte(content), 0o644))

	filePath, wasInstalled, err := Uninstall(Zsh, home)
	require.NoError(t, err)
	assert.True(t, wasInstalled)
	assert.Equal(t, rcPath, filePath)

	result, err := os.ReadFile(rcPath)
	require.NoError(t, err)
	// Assert exact content to verify line boundaries are preserved.
	assert.Equal(t, "# before\n# after\n", string(result))
}

func TestUninstallNotInstalled(t *testing.T) {
	home := t.TempDir()
	rcPath := filepath.Join(home, ".zshrc")
	require.NoError(t, os.WriteFile(rcPath, []byte("# no completion here\n"), 0o644))

	_, wasInstalled, err := Uninstall(Zsh, home)
	require.NoError(t, err)
	assert.False(t, wasInstalled)
}

func TestUninstallFileDoesNotExist(t *testing.T) {
	home := t.TempDir()

	_, wasInstalled, err := Uninstall(Zsh, home)
	require.NoError(t, err)
	assert.False(t, wasInstalled)
}

func TestUninstallCorruptedMissingEnd(t *testing.T) {
	home := t.TempDir()
	rcPath := filepath.Join(home, ".zshrc")
	content := "# before\n" + BeginMarker + "\neval something\n"
	require.NoError(t, os.WriteFile(rcPath, []byte(content), 0o644))

	_, _, err := Uninstall(Zsh, home)
	require.Error(t, err)
	assert.ErrorContains(t, err, "corrupted completion block")
	assert.ErrorContains(t, err, "missing end marker")
	assert.ErrorContains(t, err, "line 2")

	// Verify file is unchanged.
	result, readErr := os.ReadFile(rcPath)
	require.NoError(t, readErr)
	assert.Equal(t, content, string(result))
}

func TestUninstallCollapsesDoubleBlankLines(t *testing.T) {
	home := t.TempDir()
	rcPath := filepath.Join(home, ".zshrc")
	content := "# before\n\n" + ShimContent(Zsh) + "\n# after\n"
	require.NoError(t, os.WriteFile(rcPath, []byte(content), 0o644))

	_, _, err := Uninstall(Zsh, home)
	require.NoError(t, err)

	result, err := os.ReadFile(rcPath)
	require.NoError(t, err)
	assert.NotContains(t, string(result), "\n\n\n")
}

func TestUninstallPreservesPermissions(t *testing.T) {
	home := t.TempDir()
	rcPath := filepath.Join(home, ".zshrc")
	require.NoError(t, os.WriteFile(rcPath, []byte(ShimContent(Zsh)), 0o600))

	_, _, err := Uninstall(Zsh, home)
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

func TestUninstallFish(t *testing.T) {
	home := t.TempDir()
	fishPath := filepath.Join(home, ".config", "fish", "completions", "databricks.fish")
	require.NoError(t, os.MkdirAll(filepath.Dir(fishPath), 0o755))
	// Write content that includes our marker (simulating a CLI-managed file).
	require.NoError(t, os.WriteFile(fishPath, []byte(ShimContent(Fish)), 0o644))

	filePath, wasInstalled, err := Uninstall(Fish, home)
	require.NoError(t, err)
	assert.True(t, wasInstalled)
	assert.Equal(t, fishPath, filePath)

	_, err = os.Stat(fishPath)
	assert.True(t, os.IsNotExist(err))
}

func TestUninstallFishForeignFile(t *testing.T) {
	home := t.TempDir()
	fishPath := filepath.Join(home, ".config", "fish", "completions", "databricks.fish")
	require.NoError(t, os.MkdirAll(filepath.Dir(fishPath), 0o755))
	// Write content without our marker (e.g. installed by a package manager).
	require.NoError(t, os.WriteFile(fishPath, []byte("# fish completions from homebrew\n"), 0o644))

	_, wasInstalled, err := Uninstall(Fish, home)
	require.NoError(t, err)
	assert.False(t, wasInstalled)

	// File must be preserved.
	_, err = os.Stat(fishPath)
	assert.NoError(t, err)
}

func TestUninstallFishNotPresent(t *testing.T) {
	home := t.TempDir()

	_, wasInstalled, err := Uninstall(Fish, home)
	require.NoError(t, err)
	assert.False(t, wasInstalled)
}

func TestInstallThenUninstallRoundTrip(t *testing.T) {
	home := t.TempDir()
	rcPath := filepath.Join(home, ".zshrc")
	original := "# my zsh config\nexport FOO=bar\n"
	require.NoError(t, os.WriteFile(rcPath, []byte(original), 0o644))

	_, _, err := Install(Zsh, home)
	require.NoError(t, err)

	content, err := os.ReadFile(rcPath)
	require.NoError(t, err)
	assert.Contains(t, string(content), BeginMarker)

	_, _, err = Uninstall(Zsh, home)
	require.NoError(t, err)

	result, err := os.ReadFile(rcPath)
	require.NoError(t, err)
	assert.Equal(t, original, string(result))
}
