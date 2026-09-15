package agents

import (
	"os"
	"path/filepath"
	"runtime"
	"testing"

	"github.com/databricks/cli/libs/env"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestSkillAgentRegistryPaths(t *testing.T) {
	home := t.TempDir()
	cwd := t.TempDir()
	ctx := env.WithUserHomeDir(t.Context(), home)
	ctx = env.Set(ctx, "XDG_CONFIG_HOME", filepath.Join(home, ".config"))
	ctx = env.Set(ctx, "APPDATA", "")

	gooseGlobalDir := filepath.Join(home, ".config", "goose", "skills")
	if runtime.GOOS == "windows" {
		gooseGlobalDir = filepath.Join(home, "AppData", "Roaming", "Block", "goose", "config", "skills")
	}

	tests := []struct {
		name        string
		binary      string
		displayName string
		globalDir   string
		projectDir  string
	}{
		{NamePi, "pi", "Pi", filepath.Join(home, ".pi", "agent", "skills"), filepath.Join(cwd, ".pi", "skills")},
		{NameGemini, "gemini", "Gemini CLI", filepath.Join(home, ".gemini", "skills"), filepath.Join(cwd, ".gemini", "skills")},
		{NameGoose, "goose", "Goose", gooseGlobalDir, filepath.Join(cwd, ".goose", "skills")},
	}

	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			a := ByName(tc.name)
			require.NotNil(t, a)
			assert.Equal(t, tc.binary, a.Binary)
			assert.Equal(t, tc.displayName, a.DisplayName)
			assert.True(t, a.SupportsProjectScope)
			assert.Nil(t, a.Plugin)

			globalDir, err := a.SkillsDir(ctx)
			require.NoError(t, err)
			assert.Equal(t, tc.globalDir, globalDir)
			assert.Equal(t, tc.projectDir, a.ProjectSkillsDir(cwd))
		})
	}
}

func TestDetectProjectInstalled(t *testing.T) {
	cwd := t.TempDir()
	for _, name := range []string{NamePi, NameGemini, NameGoose} {
		dir := filepath.Join(ByName(name).ProjectSkillsDir(cwd), "databricks-core")
		require.NoError(t, os.MkdirAll(dir, 0o755))
	}
	// A non-databricks skill must not count as a Databricks install.
	require.NoError(t, os.MkdirAll(filepath.Join(cwd, ".pi", "skills", "other-skill"), 0o755))

	var names []string
	for _, a := range DetectProjectInstalled(cwd) {
		names = append(names, a.Name)
	}
	assert.ElementsMatch(t, []string{NamePi, NameGemini, NameGoose}, names)
}
