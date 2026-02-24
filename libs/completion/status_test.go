package completion

import (
	"os"
	"path/filepath"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestStatusNotInstalled(t *testing.T) {
	home := t.TempDir()
	// Override HOMEBREW_PREFIX so the real system Homebrew isn't detected.
	t.Setenv("HOMEBREW_PREFIX", t.TempDir())

	result, err := Status(Zsh, home)
	require.NoError(t, err)
	assert.False(t, result.Installed)
	assert.Empty(t, result.Method)
	assert.Equal(t, filepath.Join(home, ".zshrc"), result.FilePath)
}

func TestStatusInstalledViaMarker(t *testing.T) {
	home := t.TempDir()
	rcPath := filepath.Join(home, ".zshrc")
	require.NoError(t, os.WriteFile(rcPath, []byte(ShimContent(Zsh)), 0o644))

	result, err := Status(Zsh, home)
	require.NoError(t, err)
	assert.True(t, result.Installed)
	assert.Equal(t, "marker", result.Method)
}

func TestStatusFishFileExists(t *testing.T) {
	home := t.TempDir()
	fishPath := filepath.Join(home, ".config", "fish", "completions", "databricks.fish")
	require.NoError(t, os.MkdirAll(filepath.Dir(fishPath), 0o755))
	// Write a file without our markers (simulating package manager install).
	require.NoError(t, os.WriteFile(fishPath, []byte("# package manager completions\n"), 0o644))

	result, err := Status(Fish, home)
	require.NoError(t, err)
	assert.True(t, result.Installed)
	assert.Equal(t, "file", result.Method)
}

func TestStatusFishWithMarker(t *testing.T) {
	home := t.TempDir()
	fishPath := filepath.Join(home, ".config", "fish", "completions", "databricks.fish")
	require.NoError(t, os.MkdirAll(filepath.Dir(fishPath), 0o755))
	require.NoError(t, os.WriteFile(fishPath, []byte(ShimContent(Fish)), 0o644))

	result, err := Status(Fish, home)
	require.NoError(t, err)
	assert.True(t, result.Installed)
	assert.Equal(t, "marker", result.Method)
}

func TestStatusHomebrewZsh(t *testing.T) {
	home := t.TempDir()
	brewPrefix := t.TempDir()

	// Create a fake brew binary so detection works.
	require.NoError(t, os.MkdirAll(filepath.Join(brewPrefix, "bin"), 0o755))
	require.NoError(t, os.WriteFile(filepath.Join(brewPrefix, "bin", "brew"), nil, 0o755))

	// Create the homebrew completion file.
	completionDir := filepath.Join(brewPrefix, "share", "zsh", "site-functions")
	require.NoError(t, os.MkdirAll(completionDir, 0o755))
	require.NoError(t, os.WriteFile(filepath.Join(completionDir, "_databricks"), []byte("#compdef databricks\n"), 0o644))

	t.Setenv("HOMEBREW_PREFIX", brewPrefix)

	result, err := Status(Zsh, home)
	require.NoError(t, err)
	assert.True(t, result.Installed)
	assert.Equal(t, "homebrew", result.Method)
}

func TestStatusMarkerTakesPrecedenceOverHomebrew(t *testing.T) {
	home := t.TempDir()
	brewPrefix := t.TempDir()

	// Set up homebrew.
	require.NoError(t, os.MkdirAll(filepath.Join(brewPrefix, "bin"), 0o755))
	require.NoError(t, os.WriteFile(filepath.Join(brewPrefix, "bin", "brew"), nil, 0o755))
	completionDir := filepath.Join(brewPrefix, "share", "zsh", "site-functions")
	require.NoError(t, os.MkdirAll(completionDir, 0o755))
	require.NoError(t, os.WriteFile(filepath.Join(completionDir, "_databricks"), nil, 0o644))
	t.Setenv("HOMEBREW_PREFIX", brewPrefix)

	// Also install via marker.
	rcPath := filepath.Join(home, ".zshrc")
	require.NoError(t, os.WriteFile(rcPath, []byte(ShimContent(Zsh)), 0o644))

	result, err := Status(Zsh, home)
	require.NoError(t, err)
	assert.True(t, result.Installed)
	assert.Equal(t, "marker", result.Method)
}

func TestStatusBash(t *testing.T) {
	home := t.TempDir()
	filePath := TargetFilePath(Bash, home)
	require.NoError(t, os.WriteFile(filePath, []byte(ShimContent(Bash)), 0o644))

	result, err := Status(Bash, home)
	require.NoError(t, err)
	assert.True(t, result.Installed)
	assert.Equal(t, "marker", result.Method)
}

func TestStatusPowerShell(t *testing.T) {
	home := t.TempDir()
	filePath := TargetFilePath(PowerShell, home)
	require.NoError(t, os.MkdirAll(filepath.Dir(filePath), 0o755))
	require.NoError(t, os.WriteFile(filePath, []byte(ShimContent(PowerShell)), 0o644))

	result, err := Status(PowerShell, home)
	require.NoError(t, err)
	assert.True(t, result.Installed)
	assert.Equal(t, "marker", result.Method)
}
