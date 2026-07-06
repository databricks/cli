package installer

import (
	"context"
	"crypto/sha256"
	"encoding/hex"
	"io/fs"
	"os"
	"path/filepath"
	"testing"

	"github.com/databricks/cli/libs/aitools/agents"
	"github.com/databricks/cli/libs/cmdio"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func sha(content string) string {
	sum := sha256.Sum256([]byte(content))
	return hex.EncodeToString(sum[:])
}

func TestRemoveLegacyRawSkills(t *testing.T) {
	home := t.TempDir()
	t.Setenv("HOME", home)
	t.Setenv("USERPROFILE", home)
	ctx := cmdio.MockDiscard(t.Context())

	baseDir, err := GlobalSkillsDir(ctx)
	require.NoError(t, err)
	require.NoError(t, os.MkdirAll(filepath.Join(baseDir, "alpha"), 0o755))
	require.NoError(t, os.WriteFile(filepath.Join(baseDir, "alpha", "SKILL.md"), []byte("alpha"), 0o644))

	agentDir := filepath.Join(home, ".claude", "skills")
	require.NoError(t, os.MkdirAll(agentDir, 0o755))

	// alpha: a symlink into our canonical dir -> removed.
	require.NoError(t, os.Symlink(filepath.Join(baseDir, "alpha"), filepath.Join(agentDir, "alpha")))
	// beta: a copy whose file matches the recorded checksum -> removed.
	require.NoError(t, os.MkdirAll(filepath.Join(agentDir, "beta"), 0o755))
	require.NoError(t, os.WriteFile(filepath.Join(agentDir, "beta", "SKILL.md"), []byte("beta-content"), 0o644))
	// gamma: recorded but the on-disk file differs (user edited) -> kept.
	require.NoError(t, os.MkdirAll(filepath.Join(agentDir, "gamma"), 0o755))
	require.NoError(t, os.WriteFile(filepath.Join(agentDir, "gamma", "SKILL.md"), []byte("user edited"), 0o644))
	// thirdparty: no recorded provenance -> kept.
	require.NoError(t, os.MkdirAll(filepath.Join(agentDir, "thirdparty"), 0o755))
	require.NoError(t, os.WriteFile(filepath.Join(agentDir, "thirdparty", "SKILL.md"), []byte("tp"), 0o644))
	// delta: recorded file matches but the user added an extra file -> kept.
	require.NoError(t, os.MkdirAll(filepath.Join(agentDir, "delta"), 0o755))
	require.NoError(t, os.WriteFile(filepath.Join(agentDir, "delta", "SKILL.md"), []byte("delta-content"), 0o644))
	require.NoError(t, os.WriteFile(filepath.Join(agentDir, "delta", "extra.md"), []byte("mine"), 0o644))

	require.NoError(t, SaveState(baseDir, &InstallState{
		SchemaVersion: schemaVersionV2,
		Files: map[string]FileRecord{
			"alpha/SKILL.md": {SHA256: sha("alpha")},
			"beta/SKILL.md":  {SHA256: sha("beta-content")},
			"gamma/SKILL.md": {SHA256: sha("gamma-original")},
			"delta/SKILL.md": {SHA256: sha("delta-content")},
		},
	}))

	agent := &agents.Agent{
		Name:        agents.NameClaudeCode,
		DisplayName: "Claude Code",
		ConfigDir:   func(_ context.Context) (string, error) { return filepath.Join(home, ".claude"), nil },
	}

	require.NoError(t, RemoveLegacyRawSkills(ctx, agent, ScopeGlobal))

	assertGone(t, filepath.Join(agentDir, "alpha"))
	assertGone(t, filepath.Join(agentDir, "beta"))
	assertExists(t, filepath.Join(agentDir, "gamma"))
	assertExists(t, filepath.Join(agentDir, "thirdparty"))
	assertExists(t, filepath.Join(agentDir, "delta"))
}

func TestHasManagedRawSkillsForAgent(t *testing.T) {
	home := t.TempDir()
	t.Setenv("HOME", home)
	t.Setenv("USERPROFILE", home)
	ctx := cmdio.MockDiscard(t.Context())

	baseDir, err := GlobalSkillsDir(ctx)
	require.NoError(t, err)
	require.NoError(t, os.MkdirAll(filepath.Join(baseDir, "alpha"), 0o755))
	require.NoError(t, os.WriteFile(filepath.Join(baseDir, "alpha", "SKILL.md"), []byte("alpha"), 0o644))

	agentDir := filepath.Join(home, ".claude", "skills")
	require.NoError(t, os.MkdirAll(agentDir, 0o755))
	require.NoError(t, os.Symlink(filepath.Join(baseDir, "alpha"), filepath.Join(agentDir, "alpha")))
	require.NoError(t, os.MkdirAll(filepath.Join(agentDir, "thirdparty"), 0o755))
	require.NoError(t, os.WriteFile(filepath.Join(agentDir, "thirdparty", "SKILL.md"), []byte("mine"), 0o644))

	require.NoError(t, SaveState(baseDir, &InstallState{
		SchemaVersion: schemaVersionV2,
		Files: map[string]FileRecord{
			"alpha/SKILL.md": {SHA256: sha("alpha")},
		},
	}))

	agent := &agents.Agent{
		Name:        agents.NameClaudeCode,
		DisplayName: "Claude Code",
		ConfigDir:   func(_ context.Context) (string, error) { return filepath.Join(home, ".claude"), nil },
	}

	has, err := HasManagedRawSkillsForAgent(ctx, agent, ScopeGlobal)
	require.NoError(t, err)
	assert.True(t, has)

	require.NoError(t, RemoveLegacyRawSkills(ctx, agent, ScopeGlobal))
	has, err = HasManagedRawSkillsForAgent(ctx, agent, ScopeGlobal)
	require.NoError(t, err)
	assert.False(t, has)
}

func assertGone(t *testing.T, path string) {
	t.Helper()
	_, err := os.Lstat(path)
	assert.ErrorIs(t, err, fs.ErrNotExist, "%s should be removed", path)
}

func assertExists(t *testing.T, path string) {
	t.Helper()
	_, err := os.Lstat(path)
	assert.NoError(t, err, "%s should be kept", path)
}
