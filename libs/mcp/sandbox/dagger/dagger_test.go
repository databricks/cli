package dagger

import (
	"context"
	"os"
	"strings"
	"testing"
	"time"
)

// TestMain handles setup and teardown for all tests in this package.
func TestMain(m *testing.M) {
	code := m.Run()

	// Cleanup global client after all tests
	CloseGlobalClient()

	os.Exit(code)
}

// resetGlobalClient closes and resets the global client for test isolation.
// Call this in tests that need a fresh client.
func resetGlobalClient() {
	clientMu.Lock()
	defer clientMu.Unlock()

	if globalClient != nil {
		globalClient.Close()
		globalClient = nil
	}
}

func TestNewDaggerSandbox(t *testing.T) {
	if testing.Short() {
		t.Skip("skipping Dagger integration test")
	}

	t.Cleanup(resetGlobalClient)

	ctx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
	defer cancel()

	tests := []struct {
		name    string
		cfg     Config
		wantErr bool
	}{
		{
			name:    "default config",
			cfg:     Config{},
			wantErr: false,
		},
		{
			name: "custom image",
			cfg: Config{
				Image:          "alpine:latest",
				ExecuteTimeout: 300,
			},
			wantErr: false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			sb, err := NewDaggerSandbox(ctx, tt.cfg)
			if (err != nil) != tt.wantErr {
				t.Errorf("NewDaggerSandbox() error = %v, wantErr %v", err, tt.wantErr)
				return
			}
			if !tt.wantErr && sb != nil {
				defer sb.Close()

				if tt.cfg.Image == "" && sb.image != "node:20-alpine" {
					t.Errorf("expected default image 'node:20-alpine', got %s", sb.image)
				}
				if tt.cfg.ExecuteTimeout == 0 && sb.executeTimeout != 600 {
					t.Errorf("expected default timeout 600, got %d", sb.executeTimeout)
				}
			}
		})
	}
}

func TestDaggerSandbox_Exec(t *testing.T) {
	if testing.Short() {
		t.Skip("skipping Dagger integration test")
	}

	t.Cleanup(resetGlobalClient)

	ctx, cancel := context.WithTimeout(context.Background(), 60*time.Second)
	defer cancel()

	sb, err := NewDaggerSandbox(ctx, Config{})
	if err != nil {
		t.Fatalf("failed to create sandbox: %v", err)
	}
	defer sb.Close()

	tests := []struct {
		name         string
		command      string
		wantExitCode int
		wantStdout   string
		wantErr      bool
	}{
		{
			name:         "simple echo",
			command:      "echo 'hello world'",
			wantExitCode: 0,
			wantStdout:   "hello world",
			wantErr:      false,
		},
		{
			name:         "print working directory",
			command:      "pwd",
			wantExitCode: 0,
			wantStdout:   "/workspace",
			wantErr:      false,
		},
		{
			name:         "exit code non-zero",
			command:      "exit 1",
			wantExitCode: 1,
			wantErr:      true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result, err := sb.Exec(ctx, tt.command)
			if (err != nil) != tt.wantErr {
				t.Errorf("Exec() error = %v, wantErr %v", err, tt.wantErr)
				return
			}

			if result == nil {
				t.Fatal("expected result, got nil")
			}

			if result.ExitCode != tt.wantExitCode {
				t.Errorf("expected exit code %d, got %d", tt.wantExitCode, result.ExitCode)
			}

			if tt.wantStdout != "" && !strings.Contains(result.Stdout, tt.wantStdout) {
				t.Errorf("expected stdout to contain %q, got: %s", tt.wantStdout, result.Stdout)
			}
		})
	}
}

func TestDaggerSandbox_WriteReadFile(t *testing.T) {
	if testing.Short() {
		t.Skip("skipping Dagger integration test")
	}

	t.Cleanup(resetGlobalClient)

	ctx, cancel := context.WithTimeout(context.Background(), 60*time.Second)
	defer cancel()

	sb, err := NewDaggerSandbox(ctx, Config{})
	if err != nil {
		t.Fatalf("failed to create sandbox: %v", err)
	}
	defer sb.Close()

	tests := []struct {
		name    string
		path    string
		content string
	}{
		{
			name:    "simple file",
			path:    "test.txt",
			content: "test content",
		},
		{
			name:    "file in subdirectory",
			path:    "subdir/nested.txt",
			content: "nested content",
		},
		{
			name:    "file with special characters",
			path:    "special.txt",
			content: "content with\nnewlines\tand\ttabs",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if err := sb.WriteFile(ctx, tt.path, tt.content); err != nil {
				t.Fatalf("WriteFile() error = %v", err)
			}

			read, err := sb.ReadFile(ctx, tt.path)
			if err != nil {
				t.Fatalf("ReadFile() error = %v", err)
			}

			if read != tt.content {
				t.Errorf("expected content %q, got %q", tt.content, read)
			}
		})
	}
}

func TestDaggerSandbox_WriteFiles_BulkOperation(t *testing.T) {
	if testing.Short() {
		t.Skip("skipping Dagger integration test")
	}

	t.Cleanup(resetGlobalClient)

	ctx, cancel := context.WithTimeout(context.Background(), 60*time.Second)
	defer cancel()

	sb, err := NewDaggerSandbox(ctx, Config{})
	if err != nil {
		t.Fatalf("failed to create sandbox: %v", err)
	}
	defer sb.Close()

	files := map[string]string{
		"file1.txt":     "content1",
		"file2.txt":     "content2",
		"dir/file3.txt": "content3",
	}

	if err := sb.WriteFiles(ctx, files); err != nil {
		t.Fatalf("WriteFiles() error = %v", err)
	}

	for path, expected := range files {
		content, err := sb.ReadFile(ctx, path)
		if err != nil {
			t.Errorf("failed to read %s: %v", path, err)
			continue
		}
		if content != expected {
			t.Errorf("%s: expected %q, got %q", path, expected, content)
		}
	}
}

func TestDaggerSandbox_DeleteFile(t *testing.T) {
	if testing.Short() {
		t.Skip("skipping Dagger integration test")
	}

	t.Cleanup(resetGlobalClient)

	ctx, cancel := context.WithTimeout(context.Background(), 60*time.Second)
	defer cancel()

	sb, err := NewDaggerSandbox(ctx, Config{})
	if err != nil {
		t.Fatalf("failed to create sandbox: %v", err)
	}
	defer sb.Close()

	path := "test.txt"
	content := "test content"

	if err := sb.WriteFile(ctx, path, content); err != nil {
		t.Fatalf("WriteFile() error = %v", err)
	}

	if err := sb.DeleteFile(ctx, path); err != nil {
		t.Fatalf("DeleteFile() error = %v", err)
	}

	_, err = sb.ReadFile(ctx, path)
	if err == nil {
		t.Error("expected error when reading deleted file, got nil")
	}
}

func TestDaggerSandbox_ListDirectory(t *testing.T) {
	if testing.Short() {
		t.Skip("skipping Dagger integration test")
	}

	t.Cleanup(resetGlobalClient)

	ctx, cancel := context.WithTimeout(context.Background(), 60*time.Second)
	defer cancel()

	sb, err := NewDaggerSandbox(ctx, Config{})
	if err != nil {
		t.Fatalf("failed to create sandbox: %v", err)
	}
	defer sb.Close()

	files := map[string]string{
		"file1.txt": "content1",
		"file2.txt": "content2",
		"file3.txt": "content3",
	}

	if err := sb.WriteFiles(ctx, files); err != nil {
		t.Fatalf("WriteFiles() error = %v", err)
	}

	entries, err := sb.ListDirectory(ctx, ".")
	if err != nil {
		t.Fatalf("ListDirectory() error = %v", err)
	}

	if len(entries) != len(files) {
		t.Errorf("expected %d entries, got %d", len(files), len(entries))
	}

	for _, entry := range entries {
		if _, ok := files[entry]; !ok {
			t.Errorf("unexpected entry: %s", entry)
		}
	}
}

func TestDaggerSandbox_SetWorkdir(t *testing.T) {
	if testing.Short() {
		t.Skip("skipping Dagger integration test")
	}

	t.Cleanup(resetGlobalClient)

	ctx, cancel := context.WithTimeout(context.Background(), 60*time.Second)
	defer cancel()

	sb, err := NewDaggerSandbox(ctx, Config{})
	if err != nil {
		t.Fatalf("failed to create sandbox: %v", err)
	}
	defer sb.Close()

	newWorkdir := "/tmp"
	if err := sb.SetWorkdir(ctx, newWorkdir); err != nil {
		t.Fatalf("SetWorkdir() error = %v", err)
	}

	result, err := sb.Exec(ctx, "pwd")
	if err != nil {
		t.Fatalf("Exec() error = %v", err)
	}

	if !strings.Contains(result.Stdout, newWorkdir) {
		t.Errorf("expected working directory %s, got: %s", newWorkdir, result.Stdout)
	}
}

func TestDaggerSandbox_ExportDirectory(t *testing.T) {
	if testing.Short() {
		t.Skip("skipping Dagger integration test")
	}

	t.Cleanup(resetGlobalClient)

	ctx, cancel := context.WithTimeout(context.Background(), 60*time.Second)
	defer cancel()

	sb, err := NewDaggerSandbox(ctx, Config{})
	if err != nil {
		t.Fatalf("failed to create sandbox: %v", err)
	}
	defer sb.Close()

	files := map[string]string{
		"export/file1.txt": "content1",
		"export/file2.txt": "content2",
	}

	if err := sb.WriteFiles(ctx, files); err != nil {
		t.Fatalf("WriteFiles() error = %v", err)
	}

	tmpDir := t.TempDir()
	exportedPath, err := sb.ExportDirectory(ctx, "export", tmpDir+"/exported")
	if err != nil {
		t.Fatalf("ExportDirectory() error = %v", err)
	}

	if exportedPath == "" {
		t.Error("expected non-empty exported path")
	}
}

func TestDaggerSandbox_RefreshFromHost(t *testing.T) {
	if testing.Short() {
		t.Skip("skipping Dagger integration test")
	}

	t.Cleanup(resetGlobalClient)

	ctx, cancel := context.WithTimeout(context.Background(), 60*time.Second)
	defer cancel()

	sb, err := NewDaggerSandbox(ctx, Config{})
	if err != nil {
		t.Fatalf("failed to create sandbox: %v", err)
	}
	defer sb.Close()

	tmpDir := t.TempDir()
	testFile := tmpDir + "/test.txt"
	testContent := "host content"

	if err := writeHostFile(testFile, testContent); err != nil {
		t.Fatalf("failed to write host file: %v", err)
	}

	if err := sb.RefreshFromHost(ctx, tmpDir, "/imported"); err != nil {
		t.Fatalf("RefreshFromHost() error = %v", err)
	}

	content, err := sb.ReadFile(ctx, "/imported/test.txt")
	if err != nil {
		t.Fatalf("ReadFile() error = %v", err)
	}

	if content != testContent {
		t.Errorf("expected content %q, got %q", testContent, content)
	}
}

func TestDaggerSandbox_Close(t *testing.T) {
	if testing.Short() {
		t.Skip("skipping Dagger integration test")
	}

	t.Cleanup(resetGlobalClient)

	ctx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
	defer cancel()

	sb, err := NewDaggerSandbox(ctx, Config{})
	if err != nil {
		t.Fatalf("failed to create sandbox: %v", err)
	}

	if err := sb.Close(); err != nil {
		t.Errorf("Close() error = %v", err)
	}

	if err := sb.Close(); err != nil {
		t.Errorf("second Close() error = %v", err)
	}
}

func TestDaggerSandbox_Fork(t *testing.T) {
	if testing.Short() {
		t.Skip("skipping Dagger integration test")
	}

	t.Cleanup(resetGlobalClient)

	ctx, cancel := context.WithTimeout(context.Background(), 180*time.Second)
	defer cancel()

	original, err := NewDaggerSandbox(ctx, Config{})
	if err != nil {
		t.Fatalf("failed to create sandbox: %v", err)
	}
	defer original.Close()

	if err := original.WriteFile(ctx, "base.txt", "base content"); err != nil {
		t.Fatalf("write to original failed: %v", err)
	}

	forked := original.Fork()
	defer forked.Close()

	content, err := forked.ReadFile(ctx, "base.txt")
	if err != nil {
		t.Fatalf("read from fork failed: %v", err)
	}

	if content != "base content" {
		t.Errorf("expected 'base content', got %q", content)
	}

	if err := forked.WriteFile(ctx, "forked.txt", "forked content"); err != nil {
		t.Fatalf("write to fork failed: %v", err)
	}

	_, err = original.ReadFile(ctx, "forked.txt")
	if err != nil {
		t.Log("original does not have forked file (expected due to Dagger immutability)")
	}
}

func TestDaggerSandbox_WithEnv(t *testing.T) {
	if testing.Short() {
		t.Skip("skipping Dagger integration test")
	}

	t.Cleanup(resetGlobalClient)

	ctx, cancel := context.WithTimeout(context.Background(), 60*time.Second)
	defer cancel()

	sb, err := NewDaggerSandbox(ctx, Config{})
	if err != nil {
		t.Fatalf("failed to create sandbox: %v", err)
	}
	defer sb.Close()

	sb.WithEnv("TEST_VAR", "test_value")

	result, err := sb.Exec(ctx, "echo $TEST_VAR")
	if err != nil {
		t.Fatalf("exec failed: %v", err)
	}

	if !strings.Contains(result.Stdout, "test_value") {
		t.Errorf("expected 'test_value' in stdout, got: %s", result.Stdout)
	}
}

func TestDaggerSandbox_WithEnvs(t *testing.T) {
	if testing.Short() {
		t.Skip("skipping Dagger integration test")
	}

	t.Cleanup(resetGlobalClient)

	ctx, cancel := context.WithTimeout(context.Background(), 60*time.Second)
	defer cancel()

	sb, err := NewDaggerSandbox(ctx, Config{})
	if err != nil {
		t.Fatalf("failed to create sandbox: %v", err)
	}
	defer sb.Close()

	envs := map[string]string{
		"VAR1": "value1",
		"VAR2": "value2",
		"VAR3": "value3",
	}

	sb.WithEnvs(envs)

	for key, expected := range envs {
		result, err := sb.Exec(ctx, "echo $"+key)
		if err != nil {
			t.Fatalf("exec failed for %s: %v", key, err)
		}

		if !strings.Contains(result.Stdout, expected) {
			t.Errorf("expected %q in stdout for %s, got: %s", expected, key, result.Stdout)
		}
	}
}

func TestDaggerSandbox_ExecWithTimeout(t *testing.T) {
	if testing.Short() {
		t.Skip("skipping Dagger integration test")
	}

	t.Cleanup(resetGlobalClient)

	ctx := context.Background()

	t.Run("timeout exceeded", func(t *testing.T) {
		sb, err := NewDaggerSandbox(ctx, Config{ExecuteTimeout: 2})
		if err != nil {
			t.Fatalf("failed to create sandbox: %v", err)
		}
		defer sb.Close()

		_, err = sb.ExecWithTimeout(ctx, "sleep 5")
		if err == nil {
			t.Error("expected timeout error")
		}

		if !strings.Contains(err.Error(), "deadline") && !strings.Contains(err.Error(), "timeout") && !strings.Contains(err.Error(), "canceled") {
			t.Errorf("expected timeout/deadline/canceled error, got: %v", err)
		}
	})

	t.Run("quick command succeeds", func(t *testing.T) {
		sb, err := NewDaggerSandbox(ctx, Config{ExecuteTimeout: 10})
		if err != nil {
			t.Fatalf("failed to create sandbox: %v", err)
		}
		defer sb.Close()

		result, err := sb.ExecWithTimeout(ctx, "echo 'quick command'")
		if err != nil {
			t.Fatalf("expected success for quick command: %v", err)
		}

		if !strings.Contains(result.Stdout, "quick command") {
			t.Errorf("expected 'quick command' in stdout, got: %s", result.Stdout)
		}
	})
}

func writeHostFile(path, content string) error {
	f, err := os.Create(path)
	if err != nil {
		return err
	}
	defer f.Close()

	_, err = f.WriteString(content)
	return err
}
