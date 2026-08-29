//go:build windows

package installer

import (
	"os"
	"path/filepath"
	"syscall"
	"testing"

	"github.com/databricks/cli/libs/cmdio"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestBackupThirdPartySkillWindowsNotSameDeviceFallback(t *testing.T) {
	ctx := cmdio.MockDiscard(t.Context())
	tmp := t.TempDir()
	skillName := "databricks-windows-cross-device"

	cleanupBackups := func() {
		matches, _ := filepath.Glob(filepath.Join(os.TempDir(), "databricks-skill-backup-"+skillName+"-*"))
		for _, match := range matches {
			_ = os.RemoveAll(match)
		}
	}
	cleanupBackups()
	t.Cleanup(cleanupBackups)

	destDir := filepath.Join(tmp, "agent", "skills", skillName)
	require.NoError(t, os.MkdirAll(destDir, 0o755))
	require.NoError(t, os.WriteFile(filepath.Join(destDir, "custom.md"), []byte("custom"), 0o644))

	orig := renamePathFn
	t.Cleanup(func() { renamePathFn = orig })
	renameCalled := false
	renamePathFn = func(oldpath, newpath string) error {
		if oldpath == destDir {
			renameCalled = true
			return &os.LinkError{Op: "rename", Old: oldpath, New: newpath, Err: syscall.Errno(17)}
		}
		return os.Rename(oldpath, newpath)
	}

	err := backupThirdPartySkill(ctx, destDir, "/some/canonical", skillName, "Test Agent")
	require.NoError(t, err)
	assert.True(t, renameCalled)

	_, err = os.Stat(destDir)
	assert.ErrorIs(t, err, os.ErrNotExist)

	matches, err := filepath.Glob(filepath.Join(os.TempDir(), "databricks-skill-backup-"+skillName+"-*", skillName, "custom.md"))
	require.NoError(t, err)
	require.Len(t, matches, 1)

	content, err := os.ReadFile(matches[0])
	require.NoError(t, err)
	assert.Equal(t, "custom", string(content))
}
