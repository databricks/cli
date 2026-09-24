package statemgmt

import (
	"os"
	"path/filepath"
	"testing"

	"github.com/stretchr/testify/require"
)

func TestInstallLocalDirectStateFailure(t *testing.T) {
	t.Run("directory creation", func(t *testing.T) {
		root := t.TempDir()
		blocked := filepath.Join(root, "blocked")
		require.NoError(t, os.WriteFile(blocked, nil, 0o600))
		localDirectPath := filepath.Join(blocked, "resources.json")

		err := installLocalDirectState(filepath.Join(root, "resources.migrating.json"), localDirectPath)
		require.ErrorIs(t, err, errLocalDirectStateInstallation)
	})

	t.Run("rename", func(t *testing.T) {
		root := t.TempDir()
		tempStatePath := filepath.Join(root, "resources.migrating.json")
		localDirectPath := filepath.Join(root, "resources.json")
		require.NoError(t, os.WriteFile(tempStatePath, nil, 0o600))
		require.NoError(t, os.Mkdir(localDirectPath, 0o700))

		err := installLocalDirectState(tempStatePath, localDirectPath)
		require.ErrorIs(t, err, errLocalDirectStateInstallation)
	})
}
