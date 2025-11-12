package local

import (
	"os"
	"path/filepath"
	"testing"
)

func TestValidatePath(t *testing.T) {
	// Create a temp directory for testing
	baseDir := t.TempDir()

	tests := []struct {
		name     string
		userPath string
		wantErr  bool
		errMsg   string
		setup    func() error
	}{
		{
			name:     "simple relative path",
			userPath: "test.txt",
			wantErr:  false,
		},
		{
			name:     "nested relative path",
			userPath: "subdir/test.txt",
			wantErr:  false,
		},
		{
			name:     "path with dots",
			userPath: "subdir/./test.txt",
			wantErr:  false,
		},
		{
			name:     "path traversal attempt",
			userPath: "../outside.txt",
			wantErr:  true,
			errMsg:   "outside base directory",
		},
		{
			name:     "absolute path traversal",
			userPath: "/../etc/passwd",
			wantErr:  true,
			errMsg:   "absolute paths not allowed",
		},
		{
			name:     "symlink escape attempt",
			userPath: "symlink/test.txt",
			wantErr:  true,
			errMsg:   "outside base directory",
			setup: func() error {
				// Create a symlink pointing outside baseDir
				linkPath := filepath.Join(baseDir, "symlink")
				return os.Symlink("/tmp", linkPath)
			},
		},
		{
			name:     "valid symlink within base",
			userPath: "goodlink/test.txt",
			wantErr:  false,
			setup: func() error {
				// Create a target directory
				targetDir := filepath.Join(baseDir, "target")
				if err := os.MkdirAll(targetDir, 0755); err != nil {
					return err
				}
				// Create a symlink to it
				linkPath := filepath.Join(baseDir, "goodlink")
				return os.Symlink(targetDir, linkPath)
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			// Run setup if provided
			if tt.setup != nil {
				if err := tt.setup(); err != nil {
					t.Fatalf("setup failed: %v", err)
				}
			}

			result, err := ValidatePath(baseDir, tt.userPath)

			if tt.wantErr {
				if err == nil {
					t.Errorf("ValidatePath() expected error containing %q, got nil", tt.errMsg)
				} else if tt.errMsg != "" && !contains(err.Error(), tt.errMsg) {
					t.Errorf("ValidatePath() error = %v, want error containing %q", err, tt.errMsg)
				}
			} else {
				if err != nil {
					t.Errorf("ValidatePath() unexpected error: %v", err)
				}
				// Verify result is absolute
				if !filepath.IsAbs(result) {
					t.Errorf("ValidatePath() result %q is not absolute", result)
				}
				// Verify result starts with baseDir
				absBase, _ := filepath.Abs(baseDir)
				if !contains(result, absBase) {
					t.Errorf("ValidatePath() result %q does not start with base %q", result, absBase)
				}
			}
		})
	}
}

func TestRelativePath(t *testing.T) {
	tests := []struct {
		name       string
		baseDir    string
		targetPath string
		want       string
		wantErr    bool
	}{
		{
			name:       "simple relative path",
			baseDir:    "/tmp/base",
			targetPath: "/tmp/base/file.txt",
			want:       "file.txt",
			wantErr:    false,
		},
		{
			name:       "nested relative path",
			baseDir:    "/tmp/base",
			targetPath: "/tmp/base/sub/dir/file.txt",
			want:       "sub/dir/file.txt",
			wantErr:    false,
		},
		{
			name:       "outside base directory",
			baseDir:    "/tmp/base",
			targetPath: "/tmp/other/file.txt",
			wantErr:    true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got, err := RelativePath(tt.baseDir, tt.targetPath)
			if tt.wantErr {
				if err == nil {
					t.Errorf("RelativePath() expected error, got nil")
				}
			} else {
				if err != nil {
					t.Errorf("RelativePath() unexpected error: %v", err)
				}
				if got != tt.want {
					t.Errorf("RelativePath() = %q, want %q", got, tt.want)
				}
			}
		})
	}
}

func TestMustValidatePath(t *testing.T) {
	baseDir := t.TempDir()

	// Should not panic for valid path
	result := MustValidatePath(baseDir, "test.txt")
	if result == "" {
		t.Error("MustValidatePath() returned empty string")
	}

	// Should panic for invalid path
	defer func() {
		if r := recover(); r == nil {
			t.Error("MustValidatePath() should panic for invalid path")
		}
	}()
	MustValidatePath(baseDir, "../outside.txt")
}

// Helper function to check if a string contains a substring
func contains(s, substr string) bool {
	return len(s) >= len(substr) && (s == substr || len(substr) == 0 ||
		(len(s) > 0 && len(substr) > 0 && someContains(s, substr)))
}

func someContains(s, substr string) bool {
	for i := 0; i <= len(s)-len(substr); i++ {
		if s[i:i+len(substr)] == substr {
			return true
		}
	}
	return false
}
