package pkgmanager_test

import (
	"os"
	"path/filepath"
	"testing"

	"github.com/databricks/cli/libs/apps/pkgmanager"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestResolvePnpm(t *testing.T) {
	m, err := pkgmanager.Resolve("pnpm")
	require.NoError(t, err)
	assert.Equal(t, "pnpm", m.Name)
	assert.Equal(t, "pnpm install --frozen-lockfile", m.InstallCommand)
	assert.Equal(t, "pnpm-lock.yaml", m.LockfileName)
	assert.Equal(t, "pnpm-workspace.yaml", m.WorkspaceConfigName)
}

func TestResolveNpm(t *testing.T) {
	m, err := pkgmanager.Resolve("npm")
	require.NoError(t, err)
	assert.Equal(t, "npm", m.Name)
	assert.Equal(t, "npm ci", m.InstallCommand)
	assert.Equal(t, "package-lock.json", m.LockfileName)
	assert.Empty(t, m.WorkspaceConfigName)
}

func TestResolveUnknown(t *testing.T) {
	_, err := pkgmanager.Resolve("unknown")
	require.Error(t, err)
	assert.Equal(t, `unknown package manager "unknown" (allowed: npm, pnpm)`, err.Error())
}

func TestDefault(t *testing.T) {
	m := pkgmanager.Default()
	assert.Equal(t, "pnpm", m.Name)
	assert.Equal(t, "pnpm install --frozen-lockfile", m.InstallCommand)
}

func TestPrunePnpm(t *testing.T) {
	dir := t.TempDir()

	// Create all PM-specific artifacts
	for _, file := range []string{"pnpm-lock.yaml", "package-lock.json", "pnpm-workspace.yaml", "package.json"} {
		require.NoError(t, os.WriteFile(filepath.Join(dir, file), []byte("content"), 0o644))
	}

	m, err := pkgmanager.Resolve("pnpm")
	require.NoError(t, err)
	require.NoError(t, m.Prune(dir))

	// pnpm should keep its lockfile and workspace config
	assert.FileExists(t, filepath.Join(dir, "pnpm-lock.yaml"))
	assert.FileExists(t, filepath.Join(dir, "pnpm-workspace.yaml"))
	assert.FileExists(t, filepath.Join(dir, "package.json"))

	// pnpm should remove npm lockfile
	assert.NoFileExists(t, filepath.Join(dir, "package-lock.json"))
}

func TestPruneNpm(t *testing.T) {
	dir := t.TempDir()

	// Create all PM-specific artifacts
	for _, file := range []string{"pnpm-lock.yaml", "package-lock.json", "pnpm-workspace.yaml", "package.json"} {
		require.NoError(t, os.WriteFile(filepath.Join(dir, file), []byte("content"), 0o644))
	}

	m, err := pkgmanager.Resolve("npm")
	require.NoError(t, err)
	require.NoError(t, m.Prune(dir))

	// npm should keep its lockfile only
	assert.FileExists(t, filepath.Join(dir, "package-lock.json"))
	assert.FileExists(t, filepath.Join(dir, "package.json"))

	// npm should remove pnpm lockfile and workspace config
	assert.NoFileExists(t, filepath.Join(dir, "pnpm-lock.yaml"))
	assert.NoFileExists(t, filepath.Join(dir, "pnpm-workspace.yaml"))
}

func TestPruneIdempotent(t *testing.T) {
	dir := t.TempDir()

	// Only create pnpm files
	require.NoError(t, os.WriteFile(filepath.Join(dir, "pnpm-lock.yaml"), []byte("content"), 0o644))
	require.NoError(t, os.WriteFile(filepath.Join(dir, "package.json"), []byte("content"), 0o644))

	m, err := pkgmanager.Resolve("pnpm")
	require.NoError(t, err)

	// Prune should succeed even when npm artifacts don't exist
	require.NoError(t, m.Prune(dir))
	require.NoError(t, m.Prune(dir))

	// pnpm artifacts should remain
	assert.FileExists(t, filepath.Join(dir, "pnpm-lock.yaml"))
	assert.FileExists(t, filepath.Join(dir, "package.json"))
}
