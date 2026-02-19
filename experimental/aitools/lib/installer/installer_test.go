package installer

import (
	"context"
	"os"
	"path/filepath"
	"testing"

	"github.com/databricks/cli/libs/cmdio"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestBackupThirdPartySkillDestDoesNotExist(t *testing.T) {
	ctx := cmdio.MockDiscard(context.Background())
	destDir := filepath.Join(t.TempDir(), "nonexistent")

	err := backupThirdPartySkill(ctx, destDir, "/canonical", "databricks", "Test Agent")
	assert.NoError(t, err)
}

func TestBackupThirdPartySkillSymlinkToCanonical(t *testing.T) {
	ctx := cmdio.MockDiscard(context.Background())
	tmp := t.TempDir()

	canonicalDir := filepath.Join(tmp, "canonical", "databricks")
	require.NoError(t, os.MkdirAll(canonicalDir, 0o755))
	require.NoError(t, os.WriteFile(filepath.Join(canonicalDir, "skill.md"), []byte("ok"), 0o644))

	destDir := filepath.Join(tmp, "agent", "skills", "databricks")
	require.NoError(t, os.MkdirAll(filepath.Dir(destDir), 0o755))
	require.NoError(t, os.Symlink(canonicalDir, destDir))

	err := backupThirdPartySkill(ctx, destDir, canonicalDir, "databricks", "Test Agent")
	assert.NoError(t, err)

	// Symlink should still be in place.
	target, err := os.Readlink(destDir)
	require.NoError(t, err)
	assert.Equal(t, canonicalDir, target)
}

func TestBackupThirdPartySkillRegularDir(t *testing.T) {
	ctx := cmdio.MockDiscard(context.Background())
	tmp := t.TempDir()

	destDir := filepath.Join(tmp, "agent", "skills", "databricks")
	require.NoError(t, os.MkdirAll(destDir, 0o755))
	require.NoError(t, os.WriteFile(filepath.Join(destDir, "custom.md"), []byte("custom"), 0o644))

	err := backupThirdPartySkill(ctx, destDir, "/some/canonical", "databricks", "Test Agent")
	require.NoError(t, err)

	// destDir should no longer exist.
	_, err = os.Stat(destDir)
	assert.True(t, os.IsNotExist(err))

	// Backup should contain the original file.
	matches, err := filepath.Glob(filepath.Join(os.TempDir(), "databricks-skill-backup-databricks-*", "databricks", "custom.md"))
	require.NoError(t, err)
	require.NotEmpty(t, matches)

	content, err := os.ReadFile(matches[0])
	require.NoError(t, err)
	assert.Equal(t, "custom", string(content))

	// Clean up backup.
	require.NoError(t, os.RemoveAll(filepath.Dir(filepath.Dir(matches[0]))))
}

func TestBackupThirdPartySkillSymlinkToOtherTarget(t *testing.T) {
	ctx := cmdio.MockDiscard(context.Background())
	tmp := t.TempDir()

	otherDir := filepath.Join(tmp, "other", "databricks")
	require.NoError(t, os.MkdirAll(otherDir, 0o755))
	require.NoError(t, os.WriteFile(filepath.Join(otherDir, "other.md"), []byte("other"), 0o644))

	destDir := filepath.Join(tmp, "agent", "skills", "databricks")
	require.NoError(t, os.MkdirAll(filepath.Dir(destDir), 0o755))
	require.NoError(t, os.Symlink(otherDir, destDir))

	canonicalDir := filepath.Join(tmp, "canonical", "databricks")

	err := backupThirdPartySkill(ctx, destDir, canonicalDir, "databricks", "Test Agent")
	require.NoError(t, err)

	// destDir (the symlink) should no longer exist.
	_, err = os.Lstat(destDir)
	assert.True(t, os.IsNotExist(err))

	// Original target should be untouched.
	content, err := os.ReadFile(filepath.Join(otherDir, "other.md"))
	require.NoError(t, err)
	assert.Equal(t, "other", string(content))
}

func TestBackupThirdPartySkillRegularFile(t *testing.T) {
	ctx := cmdio.MockDiscard(context.Background())
	tmp := t.TempDir()

	// Edge case: destDir is a file, not a directory.
	destDir := filepath.Join(tmp, "agent", "skills", "databricks")
	require.NoError(t, os.MkdirAll(filepath.Dir(destDir), 0o755))
	require.NoError(t, os.WriteFile(destDir, []byte("file"), 0o644))

	err := backupThirdPartySkill(ctx, destDir, "/some/canonical", "databricks", "Test Agent")
	require.NoError(t, err)

	_, err = os.Stat(destDir)
	assert.True(t, os.IsNotExist(err))
}
