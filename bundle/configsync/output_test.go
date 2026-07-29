package configsync

import (
	"errors"
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/databricks/cli/bundle"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestSaveFiles_Success(t *testing.T) {
	ctx := t.Context()

	tmpDir := t.TempDir()

	yamlPath := filepath.Join(tmpDir, "subdir", "databricks.yml")
	require.NoError(t, os.MkdirAll(filepath.Dir(yamlPath), 0o755))
	require.NoError(t, os.WriteFile(yamlPath, []byte("original content"), 0o640))
	modifiedContent := `resources:
  jobs:
    test_job:
      name: "Updated Job"
      timeout_seconds: 7200
`

	files := []FileChange{
		{
			Path:            yamlPath,
			OriginalContent: "original content",
			ModifiedContent: modifiedContent,
		},
	}

	err := SaveFiles(ctx, &bundle.Bundle{}, files)
	require.NoError(t, err)

	_, err = os.Stat(yamlPath)
	require.NoError(t, err)

	content, err := os.ReadFile(yamlPath)
	require.NoError(t, err)
	assert.Equal(t, modifiedContent, string(content))

	_, err = os.Stat(filepath.Dir(yamlPath))
	require.NoError(t, err)
}

func TestSaveFiles_MultipleFiles(t *testing.T) {
	ctx := t.Context()

	tmpDir := t.TempDir()

	file1Path := filepath.Join(tmpDir, "file1.yml")
	file2Path := filepath.Join(tmpDir, "subdir", "file2.yml")
	require.NoError(t, os.WriteFile(file1Path, []byte("original 1"), 0o644))
	require.NoError(t, os.MkdirAll(filepath.Dir(file2Path), 0o755))
	require.NoError(t, os.WriteFile(file2Path, []byte("original 2"), 0o644))
	content1 := "content for file 1"
	content2 := "content for file 2"

	files := []FileChange{
		{
			Path:            file1Path,
			OriginalContent: "original 1",
			ModifiedContent: content1,
		},
		{
			Path:            file2Path,
			OriginalContent: "original 2",
			ModifiedContent: content2,
		},
	}

	err := SaveFiles(ctx, &bundle.Bundle{}, files)
	require.NoError(t, err)

	content, err := os.ReadFile(file1Path)
	require.NoError(t, err)
	assert.Equal(t, content1, string(content))

	content, err = os.ReadFile(file2Path)
	require.NoError(t, err)
	assert.Equal(t, content2, string(content))
}

func TestSaveFiles_EmptyList(t *testing.T) {
	ctx := t.Context()

	err := SaveFiles(ctx, &bundle.Bundle{}, []FileChange{})
	require.NoError(t, err)
}

func TestSaveFiles_RejectsStaleSource(t *testing.T) {
	path := filepath.Join(t.TempDir(), "databricks.yml")
	require.NoError(t, os.WriteFile(path, []byte("concurrent edit"), 0o644))

	err := SaveFiles(t.Context(), &bundle.Bundle{}, []FileChange{{
		Path:            path,
		OriginalContent: "original",
		ModifiedContent: "remote edit",
	}})
	require.ErrorContains(t, err, "changed while remote changes were being resolved")
	require.ErrorIs(t, err, ErrSourceChanged)
	require.NotErrorIs(t, err, ErrSourceRecoveryRequired)
	content, readErr := os.ReadFile(path)
	require.NoError(t, readErr)
	assert.Equal(t, "concurrent edit", string(content))
}

func TestSaveFiles_RollsBackEarlierFilesWhenCommitFails(t *testing.T) {
	directory := t.TempDir()
	firstPath := filepath.Join(directory, "first.yml")
	secondPath := filepath.Join(directory, "second.yml")
	require.NoError(t, os.WriteFile(firstPath, []byte("first original"), 0o644))
	require.NoError(t, os.WriteFile(secondPath, []byte("second original"), 0o644))

	failed := false
	rename := func(oldPath, newPath string) error {
		if !failed && newPath == secondPath && strings.Contains(filepath.Base(oldPath), ".config-remote-sync-") {
			failed = true
			return errors.New("injected rename failure")
		}
		return os.Rename(oldPath, newPath)
	}
	err := saveFiles(t.Context(), []FileChange{
		{Path: firstPath, OriginalContent: "first original", ModifiedContent: "first modified"},
		{Path: secondPath, OriginalContent: "second original", ModifiedContent: "second modified"},
	}, rename)
	require.ErrorContains(t, err, "injected rename failure")

	first, readErr := os.ReadFile(firstPath)
	require.NoError(t, readErr)
	second, readErr := os.ReadFile(secondPath)
	require.NoError(t, readErr)
	assert.Equal(t, "first original", string(first))
	assert.Equal(t, "second original", string(second))
}

func TestSaveFiles_PreservesRecoveryFileWhenRollbackFails(t *testing.T) {
	directory := t.TempDir()
	firstPath := filepath.Join(directory, "first.yml")
	secondPath := filepath.Join(directory, "second.yml")
	require.NoError(t, os.WriteFile(firstPath, []byte("first original"), 0o644))
	require.NoError(t, os.WriteFile(secondPath, []byte("second original"), 0o644))

	firstRenames := 0
	rename := func(oldPath, newPath string) error {
		if strings.Contains(filepath.Base(oldPath), ".config-remote-sync-") && newPath == firstPath {
			firstRenames++
			if firstRenames == 2 {
				return errors.New("injected rollback failure")
			}
		}
		if strings.Contains(filepath.Base(oldPath), ".config-remote-sync-") && newPath == secondPath {
			return errors.New("injected commit failure")
		}
		return os.Rename(oldPath, newPath)
	}
	err := saveFiles(t.Context(), []FileChange{
		{Path: firstPath, OriginalContent: "first original", ModifiedContent: "first modified"},
		{Path: secondPath, OriginalContent: "second original", ModifiedContent: "second modified"},
	}, rename)
	require.ErrorContains(t, err, "injected commit failure")
	require.ErrorContains(t, err, "injected rollback failure")
	require.ErrorIs(t, err, ErrSourceRecoveryRequired)

	first, readErr := os.ReadFile(firstPath)
	require.NoError(t, readErr)
	assert.Equal(t, "first modified", string(first))
	second, readErr := os.ReadFile(secondPath)
	require.NoError(t, readErr)
	assert.Equal(t, "second original", string(second))

	entries, readErr := os.ReadDir(directory)
	require.NoError(t, readErr)
	var recoveryFiles []string
	for _, entry := range entries {
		if strings.Contains(entry.Name(), ".config-remote-sync-") {
			recoveryFiles = append(recoveryFiles, filepath.Join(directory, entry.Name()))
		}
	}
	require.Len(t, recoveryFiles, 1)
	recovery, readErr := os.ReadFile(recoveryFiles[0])
	require.NoError(t, readErr)
	assert.Equal(t, "first original", string(recovery))
}
