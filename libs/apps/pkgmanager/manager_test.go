package pkgmanager_test

import (
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
	assert.Equal(t, "npm", m.WorkspaceConfigName)
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
