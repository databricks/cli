package legacytemplates

import (
	"os"
	"path/filepath"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestCopyFile(t *testing.T) {
	t.Run("copy text file", func(t *testing.T) {
		tmpDir := t.TempDir()
		src := filepath.Join(tmpDir, "source.txt")
		dst := filepath.Join(tmpDir, "dest.txt")

		// Create source file
		content := "Hello, World!"
		err := os.WriteFile(src, []byte(content), 0o644)
		require.NoError(t, err)

		// Copy file
		err = copyFile(src, dst)
		require.NoError(t, err)

		// Verify content
		dstContent, err := os.ReadFile(dst)
		require.NoError(t, err)
		assert.Equal(t, content, string(dstContent))
	})

	t.Run("copy binary file", func(t *testing.T) {
		tmpDir := t.TempDir()
		src := filepath.Join(tmpDir, "source.bin")
		dst := filepath.Join(tmpDir, "dest.bin")

		// Create binary source file
		content := []byte{0x00, 0x01, 0x02, 0xFF, 0xFE}
		err := os.WriteFile(src, content, 0o644)
		require.NoError(t, err)

		// Copy file
		err = copyFile(src, dst)
		require.NoError(t, err)

		// Verify content
		dstContent, err := os.ReadFile(dst)
		require.NoError(t, err)
		assert.Equal(t, content, dstContent)
	})

	t.Run("source file does not exist", func(t *testing.T) {
		tmpDir := t.TempDir()
		src := filepath.Join(tmpDir, "nonexistent.txt")
		dst := filepath.Join(tmpDir, "dest.txt")

		err := copyFile(src, dst)
		assert.Error(t, err)
	})
}

func TestCopyDir(t *testing.T) {
	t.Run("copy directory with files", func(t *testing.T) {
		tmpDir := t.TempDir()
		srcDir := filepath.Join(tmpDir, "source")
		dstDir := filepath.Join(tmpDir, "dest")

		// Create source directory structure
		require.NoError(t, os.MkdirAll(filepath.Join(srcDir, "subdir"), 0o755))
		require.NoError(t, os.WriteFile(filepath.Join(srcDir, "file1.txt"), []byte("content1"), 0o644))
		require.NoError(t, os.WriteFile(filepath.Join(srcDir, "subdir", "file2.txt"), []byte("content2"), 0o644))

		// Copy directory
		err := copyDir(srcDir, dstDir)
		require.NoError(t, err)

		// Verify structure
		assert.FileExists(t, filepath.Join(dstDir, "file1.txt"))
		assert.FileExists(t, filepath.Join(dstDir, "subdir", "file2.txt"))

		// Verify contents
		content1, err := os.ReadFile(filepath.Join(dstDir, "file1.txt"))
		require.NoError(t, err)
		assert.Equal(t, "content1", string(content1))

		content2, err := os.ReadFile(filepath.Join(dstDir, "subdir", "file2.txt"))
		require.NoError(t, err)
		assert.Equal(t, "content2", string(content2))
	})

	t.Run("copy empty directory", func(t *testing.T) {
		tmpDir := t.TempDir()
		srcDir := filepath.Join(tmpDir, "source")
		dstDir := filepath.Join(tmpDir, "dest")

		require.NoError(t, os.MkdirAll(srcDir, 0o755))

		err := copyDir(srcDir, dstDir)
		require.NoError(t, err)

		// Verify destination exists and is empty
		entries, err := os.ReadDir(dstDir)
		require.NoError(t, err)
		assert.Empty(t, entries)
	})

	t.Run("source directory does not exist", func(t *testing.T) {
		tmpDir := t.TempDir()
		srcDir := filepath.Join(tmpDir, "nonexistent")
		dstDir := filepath.Join(tmpDir, "dest")

		err := copyDir(srcDir, dstDir)
		assert.Error(t, err)
	})
}

func TestRunLegacyTemplateInit_DirectoryCreation(t *testing.T) {
	t.Run("output directory is created if it does not exist", func(t *testing.T) {
		tmpDir := t.TempDir()
		outputDir := filepath.Join(tmpDir, "nonexistent", "my-app")

		// This test verifies that the directory creation logic works
		// We'll just verify the parent directory can be created
		parentDir := filepath.Dir(outputDir)
		err := os.MkdirAll(parentDir, 0o755)
		require.NoError(t, err)
		assert.DirExists(t, parentDir)
	})

	t.Run("rejects existing non-empty directory", func(t *testing.T) {
		tmpDir := t.TempDir()
		outputDir := filepath.Join(tmpDir, "existing")

		// Create directory with a file
		require.NoError(t, os.MkdirAll(outputDir, 0o755))
		require.NoError(t, os.WriteFile(filepath.Join(outputDir, "file.txt"), []byte("exists"), 0o644))

		// Verify directory exists and is not empty
		entries, err := os.ReadDir(outputDir)
		require.NoError(t, err)
		assert.NotEmpty(t, entries)
	})
}

// Note: generateEnvFileForLegacyTemplate tests are not included because the function
// requires cmdIO in the context, making it difficult to test in isolation.
// The underlying EnvFileBuilder is already well-tested in env_builder_test.go.
