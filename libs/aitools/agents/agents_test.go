package agents

import (
	"path/filepath"
	"testing"

	"github.com/databricks/cli/libs/env"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestAgentsPseudoAgentResolvesDirs(t *testing.T) {
	a := ByName(NameAgents)
	require.NotNil(t, a)

	// Files-only destination: no CLI binary and no plugin.
	assert.Empty(t, a.Binary)
	assert.Nil(t, a.Plugin)
	assert.True(t, a.SupportsProjectScope)

	home := t.TempDir()
	ctx := env.Set(t.Context(), "HOME", home)
	// USERPROFILE drives env.UserHomeDir on Windows.
	ctx = env.Set(ctx, "USERPROFILE", home)

	globalDir, err := a.SkillsDir(ctx)
	require.NoError(t, err)
	assert.Equal(t, filepath.Join(home, ".agents", "skills"), globalDir)

	projectDir := a.ProjectSkillsDir("/repo")
	assert.Equal(t, filepath.Join("/repo", ".agents", "skills"), projectDir)
}
