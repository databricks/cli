package deployment

import (
	"os"
	"path/filepath"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestBackupTerraformStateReportsWhetherUndoIsAvailable(t *testing.T) {
	t.Run("backup created", func(t *testing.T) {
		localTerraformPath := filepath.Join(t.TempDir(), "terraform.tfstate")
		require.NoError(t, os.WriteFile(localTerraformPath, []byte("state"), 0o600))

		backupCreated := backupTerraformStateForManualMigration(t.Context(), localTerraformPath)

		require.True(t, backupCreated)
		require.NoFileExists(t, localTerraformPath)
		require.FileExists(t, localTerraformPath+backupSuffix)
	})

	t.Run("backup failed", func(t *testing.T) {
		dir := t.TempDir()
		localTerraformPath := filepath.Join(dir, "terraform.tfstate")
		require.NoError(t, os.Mkdir(localTerraformPath, 0o700))
		require.NoError(t, os.Mkdir(localTerraformPath+backupSuffix, 0o700))

		backupCreated := backupTerraformStateForManualMigration(t.Context(), localTerraformPath)

		require.False(t, backupCreated)
		require.DirExists(t, localTerraformPath)
	})
}

func TestMigrationUndoInstructionsRequireBackup(t *testing.T) {
	assert.Empty(t, migrationUndoInstructions(false, "resources.json", "terraform.tfstate.backup", "terraform.tfstate"))
	assert.Contains(t, migrationUndoInstructions(true, "resources.json", "terraform.tfstate.backup", "terraform.tfstate"), "terraform.tfstate.backup")
}
