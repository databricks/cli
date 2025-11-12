package local

import (
	"context"
	"fmt"
	"testing"
)

func BenchmarkLocalSandbox_WriteFile(b *testing.B) {
	baseDir := b.TempDir()
	sb, err := NewLocalSandbox(baseDir)
	if err != nil {
		b.Fatalf("NewLocalSandbox() error = %v", err)
	}
	defer sb.Close()

	ctx := context.Background()
	content := "benchmark content"

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		path := fmt.Sprintf("bench_%d.txt", i)
		err := sb.WriteFile(ctx, path, content)
		if err != nil {
			b.Fatalf("WriteFile() error = %v", err)
		}
	}
}

func BenchmarkLocalSandbox_ReadFile(b *testing.B) {
	baseDir := b.TempDir()
	sb, err := NewLocalSandbox(baseDir)
	if err != nil {
		b.Fatalf("NewLocalSandbox() error = %v", err)
	}
	defer sb.Close()

	ctx := context.Background()
	content := "benchmark content"

	// Write a test file
	err = sb.WriteFile(ctx, "bench.txt", content)
	if err != nil {
		b.Fatalf("WriteFile() error = %v", err)
	}

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		_, err := sb.ReadFile(ctx, "bench.txt")
		if err != nil {
			b.Fatalf("ReadFile() error = %v", err)
		}
	}
}

func BenchmarkLocalSandbox_WriteFiles(b *testing.B) {
	baseDir := b.TempDir()
	sb, err := NewLocalSandbox(baseDir)
	if err != nil {
		b.Fatalf("NewLocalSandbox() error = %v", err)
	}
	defer sb.Close()

	ctx := context.Background()

	// Prepare files map
	files := map[string]string{
		"file1.txt": "content 1",
		"file2.txt": "content 2",
		"file3.txt": "content 3",
		"file4.txt": "content 4",
		"file5.txt": "content 5",
	}

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		// Modify paths to avoid conflicts
		benchFiles := make(map[string]string, len(files))
		for path, content := range files {
			benchFiles[fmt.Sprintf("bench_%d_%s", i, path)] = content
		}

		err := sb.WriteFiles(ctx, benchFiles)
		if err != nil {
			b.Fatalf("WriteFiles() error = %v", err)
		}
	}
}

func BenchmarkLocalSandbox_ListDirectory(b *testing.B) {
	baseDir := b.TempDir()
	sb, err := NewLocalSandbox(baseDir)
	if err != nil {
		b.Fatalf("NewLocalSandbox() error = %v", err)
	}
	defer sb.Close()

	ctx := context.Background()

	// Create some files
	for i := 0; i < 100; i++ {
		path := fmt.Sprintf("file_%d.txt", i)
		err := sb.WriteFile(ctx, path, "content")
		if err != nil {
			b.Fatalf("WriteFile() error = %v", err)
		}
	}

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		_, err := sb.ListDirectory(ctx, ".")
		if err != nil {
			b.Fatalf("ListDirectory() error = %v", err)
		}
	}
}

func BenchmarkLocalSandbox_Exec(b *testing.B) {
	baseDir := b.TempDir()
	sb, err := NewLocalSandbox(baseDir)
	if err != nil {
		b.Fatalf("NewLocalSandbox() error = %v", err)
	}
	defer sb.Close()

	ctx := context.Background()

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		_, err := sb.Exec(ctx, "echo benchmark")
		if err != nil {
			b.Fatalf("Exec() error = %v", err)
		}
	}
}

func BenchmarkLocalSandbox_ExecComplex(b *testing.B) {
	baseDir := b.TempDir()
	sb, err := NewLocalSandbox(baseDir)
	if err != nil {
		b.Fatalf("NewLocalSandbox() error = %v", err)
	}
	defer sb.Close()

	ctx := context.Background()

	// Create a test file
	err = sb.WriteFile(ctx, "input.txt", "line 1\nline 2\nline 3\n")
	if err != nil {
		b.Fatalf("WriteFile() error = %v", err)
	}

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		_, err := sb.Exec(ctx, "grep 'line' input.txt | wc -l")
		if err != nil {
			b.Fatalf("Exec() error = %v", err)
		}
	}
}

func BenchmarkValidatePath(b *testing.B) {
	baseDir := b.TempDir()

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		_, err := ValidatePath(baseDir, "test/path/file.txt")
		if err != nil {
			b.Fatalf("ValidatePath() error = %v", err)
		}
	}
}

func BenchmarkValidatePath_Existing(b *testing.B) {
	baseDir := b.TempDir()

	// Create an existing file
	sb, err := NewLocalSandbox(baseDir)
	if err != nil {
		b.Fatalf("NewLocalSandbox() error = %v", err)
	}
	defer sb.Close()

	ctx := context.Background()
	err = sb.WriteFile(ctx, "existing.txt", "content")
	if err != nil {
		b.Fatalf("WriteFile() error = %v", err)
	}

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		_, err := ValidatePath(baseDir, "existing.txt")
		if err != nil {
			b.Fatalf("ValidatePath() error = %v", err)
		}
	}
}
