package local

import (
	"context"
	"os"
	"path/filepath"
	"strings"
	"sync"
	"testing"
)

func TestNewLocalSandbox(t *testing.T) {
	baseDir := t.TempDir()

	sb, err := NewLocalSandbox(baseDir)
	if err != nil {
		t.Fatalf("NewLocalSandbox() error = %v", err)
	}
	if sb == nil {
		t.Fatal("NewLocalSandbox() returned nil sandbox")
	}

	// Verify base directory was created
	if _, err := os.Stat(baseDir); os.IsNotExist(err) {
		t.Error("NewLocalSandbox() did not create base directory")
	}
}

func TestLocalSandbox_WriteFile(t *testing.T) {
	tests := []struct {
		name    string
		path    string
		content string
		wantErr bool
		errMsg  string
	}{
		{
			name:    "simple file",
			path:    "test.txt",
			content: "hello world",
			wantErr: false,
		},
		{
			name:    "nested path",
			path:    "subdir/test.txt",
			content: "nested content",
			wantErr: false,
		},
		{
			name:    "deeply nested path",
			path:    "a/b/c/d/test.txt",
			content: "deep content",
			wantErr: false,
		},
		{
			name:    "path traversal attempt",
			path:    "../outside.txt",
			content: "malicious",
			wantErr: true,
			errMsg:  "outside base directory",
		},
		{
			name:    "absolute path attempt",
			path:    "/etc/passwd",
			content: "malicious",
			wantErr: true,
			errMsg:  "absolute paths not allowed",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			baseDir := t.TempDir()
			sb, err := NewLocalSandbox(baseDir)
			if err != nil {
				t.Fatalf("NewLocalSandbox() error = %v", err)
			}

			ctx := context.Background()
			err = sb.WriteFile(ctx, tt.path, tt.content)

			if tt.wantErr {
				if err == nil {
					t.Errorf("WriteFile() expected error, got nil")
				} else if tt.errMsg != "" && !strings.Contains(err.Error(), tt.errMsg) {
					t.Errorf("WriteFile() error = %v, want error containing %q", err, tt.errMsg)
				}
			} else {
				if err != nil {
					t.Errorf("WriteFile() unexpected error: %v", err)
				}
				// Verify file was written
				content, err := sb.ReadFile(ctx, tt.path)
				if err != nil {
					t.Errorf("ReadFile() error after WriteFile: %v", err)
				}
				if content != tt.content {
					t.Errorf("ReadFile() = %q, want %q", content, tt.content)
				}
			}
		})
	}
}

func TestLocalSandbox_ReadFile(t *testing.T) {
	baseDir := t.TempDir()
	sb, err := NewLocalSandbox(baseDir)
	if err != nil {
		t.Fatalf("NewLocalSandbox() error = %v", err)
	}

	ctx := context.Background()

	// Write a test file
	testContent := "test content"
	err = sb.WriteFile(ctx, "test.txt", testContent)
	if err != nil {
		t.Fatalf("WriteFile() error = %v", err)
	}

	// Read it back
	content, err := sb.ReadFile(ctx, "test.txt")
	if err != nil {
		t.Errorf("ReadFile() error = %v", err)
	}
	if content != testContent {
		t.Errorf("ReadFile() = %q, want %q", content, testContent)
	}

	// Try to read non-existent file
	_, err = sb.ReadFile(ctx, "nonexistent.txt")
	if err == nil {
		t.Error("ReadFile() expected error for non-existent file, got nil")
	}
}

func TestLocalSandbox_WriteFiles(t *testing.T) {
	baseDir := t.TempDir()
	sb, err := NewLocalSandbox(baseDir)
	if err != nil {
		t.Fatalf("NewLocalSandbox() error = %v", err)
	}

	ctx := context.Background()

	files := map[string]string{
		"file1.txt":       "content 1",
		"dir/file2.txt":   "content 2",
		"dir/file3.txt":   "content 3",
		"other/file4.txt": "content 4",
	}

	err = sb.WriteFiles(ctx, files)
	if err != nil {
		t.Errorf("WriteFiles() error = %v", err)
	}

	// Verify all files were written
	for path, expectedContent := range files {
		content, err := sb.ReadFile(ctx, path)
		if err != nil {
			t.Errorf("ReadFile(%q) error = %v", path, err)
		}
		if content != expectedContent {
			t.Errorf("ReadFile(%q) = %q, want %q", path, content, expectedContent)
		}
	}
}

func TestLocalSandbox_DeleteFile(t *testing.T) {
	baseDir := t.TempDir()
	sb, err := NewLocalSandbox(baseDir)
	if err != nil {
		t.Fatalf("NewLocalSandbox() error = %v", err)
	}

	ctx := context.Background()

	// Write a file
	err = sb.WriteFile(ctx, "test.txt", "content")
	if err != nil {
		t.Fatalf("WriteFile() error = %v", err)
	}

	// Delete it
	err = sb.DeleteFile(ctx, "test.txt")
	if err != nil {
		t.Errorf("DeleteFile() error = %v", err)
	}

	// Verify it's gone
	_, err = sb.ReadFile(ctx, "test.txt")
	if err == nil {
		t.Error("ReadFile() expected error after deletion, got nil")
	}
}

func TestLocalSandbox_ListDirectory(t *testing.T) {
	baseDir := t.TempDir()
	sb, err := NewLocalSandbox(baseDir)
	if err != nil {
		t.Fatalf("NewLocalSandbox() error = %v", err)
	}

	ctx := context.Background()

	// Create some files
	files := map[string]string{
		"file1.txt":     "content1",
		"file2.txt":     "content2",
		"dir/file3.txt": "content3",
	}
	err = sb.WriteFiles(ctx, files)
	if err != nil {
		t.Fatalf("WriteFiles() error = %v", err)
	}

	// List root directory
	entries, err := sb.ListDirectory(ctx, ".")
	if err != nil {
		t.Errorf("ListDirectory() error = %v", err)
	}

	// Should contain file1.txt, file2.txt, and dir
	expected := []string{"dir", "file1.txt", "file2.txt"}
	if len(entries) != len(expected) {
		t.Errorf("ListDirectory() returned %d entries, want %d", len(entries), len(expected))
	}

	for i, want := range expected {
		if i >= len(entries) || entries[i] != want {
			t.Errorf("ListDirectory()[%d] = %q, want %q", i, entries[i], want)
		}
	}
}

func TestLocalSandbox_SetWorkdir(t *testing.T) {
	baseDir := t.TempDir()
	sb, err := NewLocalSandbox(baseDir)
	if err != nil {
		t.Fatalf("NewLocalSandbox() error = %v", err)
	}

	ctx := context.Background()

	// Create a subdirectory
	subdir := "subdir"
	err = os.MkdirAll(filepath.Join(baseDir, subdir), 0755)
	if err != nil {
		t.Fatalf("MkdirAll() error = %v", err)
	}

	// Set working directory
	err = sb.SetWorkdir(ctx, subdir)
	if err != nil {
		t.Errorf("SetWorkdir() error = %v", err)
	}

	// Verify workdir was set (by checking internal state)
	if sb.workDir != subdir {
		t.Errorf("SetWorkdir() workDir = %q, want %q", sb.workDir, subdir)
	}

	// Try to set to non-existent directory
	err = sb.SetWorkdir(ctx, "nonexistent")
	if err == nil {
		t.Error("SetWorkdir() expected error for non-existent directory, got nil")
	}
}

func TestLocalSandbox_Exec(t *testing.T) {
	baseDir := t.TempDir()
	sb, err := NewLocalSandbox(baseDir)
	if err != nil {
		t.Fatalf("NewLocalSandbox() error = %v", err)
	}

	ctx := context.Background()

	tests := []struct {
		name       string
		command    string
		wantExit   int
		wantStdout string
		wantStderr string
		wantErr    bool
	}{
		{
			name:       "simple echo",
			command:    "echo hello",
			wantExit:   0,
			wantStdout: "hello",
			wantErr:    false,
		},
		{
			name:     "command with exit code",
			command:  "exit 42",
			wantExit: 42,
			wantErr:  false,
		},
		{
			name:       "stderr output",
			command:    "echo error >&2",
			wantExit:   0,
			wantStderr: "error",
			wantErr:    false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result, err := sb.Exec(ctx, tt.command)

			if tt.wantErr {
				if err == nil {
					t.Error("Exec() expected error, got nil")
				}
				return
			}

			if err != nil {
				t.Errorf("Exec() unexpected error: %v", err)
				return
			}

			if result.ExitCode != tt.wantExit {
				t.Errorf("Exec() ExitCode = %d, want %d", result.ExitCode, tt.wantExit)
			}

			if tt.wantStdout != "" && !strings.Contains(result.Stdout, tt.wantStdout) {
				t.Errorf("Exec() Stdout = %q, want to contain %q", result.Stdout, tt.wantStdout)
			}

			if tt.wantStderr != "" && !strings.Contains(result.Stderr, tt.wantStderr) {
				t.Errorf("Exec() Stderr = %q, want to contain %q", result.Stderr, tt.wantStderr)
			}
		})
	}
}

func TestLocalSandbox_PathTraversal(t *testing.T) {
	baseDir := t.TempDir()
	sb, err := NewLocalSandbox(baseDir)
	if err != nil {
		t.Fatalf("NewLocalSandbox() error = %v", err)
	}

	ctx := context.Background()

	// Try various path traversal attacks
	attacks := []string{
		"../../../etc/passwd",
		"subdir/../../etc/passwd",
		"./../outside.txt",
	}

	for _, attack := range attacks {
		t.Run("attack:"+attack, func(t *testing.T) {
			err := sb.WriteFile(ctx, attack, "malicious")
			if err == nil {
				t.Errorf("WriteFile(%q) expected error, got nil", attack)
				return
			}
			if !strings.Contains(err.Error(), "outside base directory") {
				t.Errorf("WriteFile(%q) error = %v, want 'outside base directory'", attack, err)
			}
		})
	}
}

func TestLocalSandbox_SymlinkEscape(t *testing.T) {
	baseDir := t.TempDir()
	sb, err := NewLocalSandbox(baseDir)
	if err != nil {
		t.Fatalf("NewLocalSandbox() error = %v", err)
	}

	ctx := context.Background()

	// Create a symlink pointing outside the base directory
	linkPath := filepath.Join(baseDir, "escape")
	err = os.Symlink("/tmp", linkPath)
	if err != nil {
		t.Fatalf("Symlink() error = %v", err)
	}

	// Try to write through the symlink
	err = sb.WriteFile(ctx, "escape/malicious.txt", "content")
	if err == nil {
		t.Error("WriteFile() through escape symlink expected error, got nil")
	}
}

func TestLocalSandbox_Concurrent(t *testing.T) {
	baseDir := t.TempDir()
	sb, err := NewLocalSandbox(baseDir)
	if err != nil {
		t.Fatalf("NewLocalSandbox() error = %v", err)
	}

	ctx := context.Background()

	// Run multiple operations concurrently
	const numGoroutines = 10
	const numOpsPerGoroutine = 100

	var wg sync.WaitGroup
	wg.Add(numGoroutines)

	for i := 0; i < numGoroutines; i++ {
		go func(id int) {
			defer wg.Done()

			for j := 0; j < numOpsPerGoroutine; j++ {
				path := filepath.Join("concurrent", string(rune('a'+id)), "test.txt")
				content := string(rune('0' + j%10))

				// Write
				if err := sb.WriteFile(ctx, path, content); err != nil {
					t.Errorf("Concurrent WriteFile() error: %v", err)
					return
				}

				// Read
				if _, err := sb.ReadFile(ctx, path); err != nil {
					t.Errorf("Concurrent ReadFile() error: %v", err)
					return
				}
			}
		}(i)
	}

	wg.Wait()
}

func TestLocalSandbox_ExportDirectory(t *testing.T) {
	baseDir := t.TempDir()
	sb, err := NewLocalSandbox(baseDir)
	if err != nil {
		t.Fatalf("NewLocalSandbox() error = %v", err)
	}

	ctx := context.Background()

	// Create some files
	files := map[string]string{
		"export/file1.txt": "content1",
		"export/file2.txt": "content2",
	}
	err = sb.WriteFiles(ctx, files)
	if err != nil {
		t.Fatalf("WriteFiles() error = %v", err)
	}

	// Export directory
	hostPath := t.TempDir()
	exportedPath, err := sb.ExportDirectory(ctx, "export", hostPath)
	if err != nil {
		t.Errorf("ExportDirectory() error = %v", err)
	}

	// Verify files were exported
	for relPath := range files {
		baseName := filepath.Base(relPath)
		exportedFile := filepath.Join(exportedPath, baseName)
		if _, err := os.Stat(exportedFile); os.IsNotExist(err) {
			t.Errorf("ExportDirectory() did not export %s", baseName)
		}
	}
}

func TestLocalSandbox_RefreshFromHost(t *testing.T) {
	baseDir := t.TempDir()
	sb, err := NewLocalSandbox(baseDir)
	if err != nil {
		t.Fatalf("NewLocalSandbox() error = %v", err)
	}

	ctx := context.Background()

	// Create a host directory with files
	hostDir := t.TempDir()
	hostFile := filepath.Join(hostDir, "host-file.txt")
	err = os.WriteFile(hostFile, []byte("host content"), 0644)
	if err != nil {
		t.Fatalf("WriteFile() error = %v", err)
	}

	// Refresh from host
	err = sb.RefreshFromHost(ctx, hostDir, "imported")
	if err != nil {
		t.Errorf("RefreshFromHost() error = %v", err)
	}

	// Verify file was imported
	content, err := sb.ReadFile(ctx, "imported/host-file.txt")
	if err != nil {
		t.Errorf("ReadFile() after RefreshFromHost error = %v", err)
	}
	if content != "host content" {
		t.Errorf("ReadFile() = %q, want %q", content, "host content")
	}
}

func TestLocalSandbox_Close(t *testing.T) {
	baseDir := t.TempDir()
	sb, err := NewLocalSandbox(baseDir)
	if err != nil {
		t.Fatalf("NewLocalSandbox() error = %v", err)
	}

	// Close should not error
	err = sb.Close()
	if err != nil {
		t.Errorf("Close() error = %v", err)
	}
}
