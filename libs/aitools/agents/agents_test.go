package agents

import (
	"path/filepath"
	"testing"

	"github.com/databricks/cli/libs/env"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestClaudeConfigDir(t *testing.T) {
	home := t.TempDir()
	custom := t.TempDir()
	for _, tc := range []struct {
		name     string
		override string
		want     string
	}{
		{"default", "", filepath.Join(home, ".claude")},
		{"custom", custom, custom},
		{"relative", "custom-claude", "custom-claude"},
	} {
		t.Run(tc.name, func(t *testing.T) {
			ctx := env.WithUserHomeDir(t.Context(), home)
			ctx = env.Set(ctx, "CLAUDE_CONFIG_DIR", tc.override)
			agent := ByName(NameClaudeCode)
			dir, err := agent.ConfigDir(ctx)
			require.NoError(t, err)
			assert.Equal(t, tc.want, dir)
			skills, err := agent.SkillsDir(ctx)
			require.NoError(t, err)
			assert.Equal(t, filepath.Join(tc.want, "skills"), skills)
		})
	}
}

func TestSupportedNamesMatchesRegistry(t *testing.T) {
	names := SupportedNames()
	assert.Len(t, names, len(Registry))
	for i, a := range Registry {
		assert.Equal(t, a.DisplayName, names[i])
	}
}

func TestSkillsOnlyNamesMatchesRegistry(t *testing.T) {
	names := SkillsOnlyNames()
	// Skills-only agents (Plugin nil) are listed; plugin agents are not.
	assert.Contains(t, names, "Pi")
	assert.Contains(t, names, "Gemini CLI")
	assert.Contains(t, names, "Goose")
	assert.NotContains(t, names, "Claude Code")
	for _, a := range Registry {
		if a.Plugin != nil {
			assert.NotContains(t, names, a.DisplayName)
		}
	}
}
