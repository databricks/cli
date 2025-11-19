package fileutil

import (
	"os"
	"path/filepath"
	"testing"
)

func TestAtomicWriteFile(t *testing.T) {
	t.Run("write file successfully", func(t *testing.T) {
		tempDir := t.TempDir()
		filePath := filepath.Join(tempDir, "test.txt")
		content := []byte("test content")

		err := AtomicWriteFile(filePath, content, 0o644)
		if err != nil {
			t.Fatalf("AtomicWriteFile() error = %v", err)
		}

		got, err := os.ReadFile(filePath)
		if err != nil {
			t.Fatalf("failed to read file: %v", err)
		}

		if string(got) != string(content) {
			t.Errorf("file content = %q, want %q", string(got), string(content))
		}

		info, err := os.Stat(filePath)
		if err != nil {
			t.Fatalf("failed to stat file: %v", err)
		}

		if info.Mode().Perm() != 0o644 {
			t.Errorf("file mode = %o, want %o", info.Mode().Perm(), 0o644)
		}
	})

	t.Run("create parent directories", func(t *testing.T) {
		tempDir := t.TempDir()
		filePath := filepath.Join(tempDir, "a", "b", "c", "test.txt")
		content := []byte("nested content")

		err := AtomicWriteFile(filePath, content, 0o644)
		if err != nil {
			t.Fatalf("AtomicWriteFile() error = %v", err)
		}

		got, err := os.ReadFile(filePath)
		if err != nil {
			t.Fatalf("failed to read file: %v", err)
		}

		if string(got) != string(content) {
			t.Errorf("file content = %q, want %q", string(got), string(content))
		}
	})

	t.Run("overwrite existing file", func(t *testing.T) {
		tempDir := t.TempDir()
		filePath := filepath.Join(tempDir, "test.txt")

		err := os.WriteFile(filePath, []byte("old content"), 0o644)
		if err != nil {
			t.Fatalf("failed to create initial file: %v", err)
		}

		newContent := []byte("new content")
		err = AtomicWriteFile(filePath, newContent, 0o644)
		if err != nil {
			t.Fatalf("AtomicWriteFile() error = %v", err)
		}

		got, err := os.ReadFile(filePath)
		if err != nil {
			t.Fatalf("failed to read file: %v", err)
		}

		if string(got) != string(newContent) {
			t.Errorf("file content = %q, want %q", string(got), string(newContent))
		}
	})

	t.Run("cleanup temp file on rename failure", func(t *testing.T) {
		tempDir := t.TempDir()
		filePath := filepath.Join(tempDir, "readonly", "test.txt")

		if err := os.MkdirAll(filepath.Dir(filePath), 0o755); err != nil {
			t.Fatalf("failed to create directory: %v", err)
		}

		if err := os.WriteFile(filePath, []byte("locked"), 0o644); err != nil {
			t.Fatalf("failed to create file: %v", err)
		}

		if err := os.Chmod(filepath.Dir(filePath), 0o555); err != nil {
			t.Fatalf("failed to set read-only: %v", err)
		}
		defer func() {
			_ = os.Chmod(filepath.Dir(filePath), 0o755)
		}()

		err := AtomicWriteFile(filePath, []byte("new content"), 0o644)
		if err == nil {
			t.Fatal("expected error, got nil")
		}

		tempPath := filePath + ".tmp"
		if _, err := os.Stat(tempPath); !os.IsNotExist(err) {
			t.Errorf("temp file should be cleaned up, but exists")
		}
	})

	t.Run("preserve file permissions", func(t *testing.T) {
		tempDir := t.TempDir()
		filePath := filepath.Join(tempDir, "test.txt")
		content := []byte("test content")

		err := AtomicWriteFile(filePath, content, 0o600)
		if err != nil {
			t.Fatalf("AtomicWriteFile() error = %v", err)
		}

		info, err := os.Stat(filePath)
		if err != nil {
			t.Fatalf("failed to stat file: %v", err)
		}

		if info.Mode().Perm() != 0o600 {
			t.Errorf("file mode = %o, want %o", info.Mode().Perm(), 0o600)
		}
	})
}
