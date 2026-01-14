package pathutil

import (
	"os"
	"path/filepath"
	"strings"
	"testing"
)

func TestValidatePath(t *testing.T) {
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
				linkPath := filepath.Join(baseDir, "symlink")
				return os.Symlink("/tmp", linkPath)
			},
		},
		{
			name:     "valid symlink within base",
			userPath: "goodlink/test.txt",
			wantErr:  false,
			setup: func() error {
				targetDir := filepath.Join(baseDir, "target")
				if err := os.MkdirAll(targetDir, 0o755); err != nil {
					return err
				}
				linkPath := filepath.Join(baseDir, "goodlink")
				return os.Symlink(targetDir, linkPath)
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if tt.setup != nil {
				if err := tt.setup(); err != nil {
					t.Fatalf("setup failed: %v", err)
				}
			}

			result, err := ValidatePath(baseDir, tt.userPath)

			if tt.wantErr {
				if err == nil {
					t.Errorf("ValidatePath() expected error containing %q, got nil", tt.errMsg)
				} else if tt.errMsg != "" && !strings.Contains(err.Error(), tt.errMsg) {
					t.Errorf("ValidatePath() error = %v, want error containing %q", err, tt.errMsg)
				}
			} else {
				if err != nil {
					t.Errorf("ValidatePath() unexpected error: %v", err)
				}
				if !filepath.IsAbs(result) {
					t.Errorf("ValidatePath() result %q is not absolute", result)
				}
				absBase, _ := filepath.Abs(baseDir)
				if !strings.Contains(result, absBase) {
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

	result := MustValidatePath(baseDir, "test.txt")
	if result == "" {
		t.Error("MustValidatePath() returned empty string")
	}

	defer func() {
		if r := recover(); r == nil {
			t.Error("MustValidatePath() should panic for invalid path")
		}
	}()
	MustValidatePath(baseDir, "../outside.txt")
}
