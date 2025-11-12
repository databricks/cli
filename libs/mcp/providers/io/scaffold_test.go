package io

import (
	"context"
	"os"
	"path/filepath"
	"testing"

	"github.com/databricks/cli/libs/mcp"
	"github.com/databricks/cli/libs/mcp/session"
)

func TestProvider_Scaffold(t *testing.T) {
	tests := []struct {
		name     string
		args     *ScaffoldArgs
		setup    func(*testing.T) string
		wantErr  bool
		validate func(*testing.T, string, *ScaffoldResult)
	}{
		{
			name: "scaffold to empty directory",
			args: &ScaffoldArgs{},
			setup: func(t *testing.T) string {
				return t.TempDir()
			},
			wantErr: false,
			validate: func(t *testing.T, dir string, result *ScaffoldResult) {
				// Check files exist
				if _, err := os.Stat(filepath.Join(dir, "package.json")); os.IsNotExist(err) {
					t.Errorf("package.json does not exist")
				}
				if result.FilesCopied == 0 {
					t.Errorf("Expected files to be copied, got 0")
				}
				if result.TemplateName != "TRPC" {
					t.Errorf("Expected template name TRPC, got %s", result.TemplateName)
				}
			},
		},
		{
			name: "scaffold to non-empty directory without force",
			args: &ScaffoldArgs{
				ForceRewrite: false,
			},
			setup: func(t *testing.T) string {
				dir := t.TempDir()
				// Create existing file
				os.WriteFile(filepath.Join(dir, "existing.txt"), []byte("test"), 0644)
				return dir
			},
			wantErr: true,
		},
		{
			name: "scaffold with force rewrite",
			args: &ScaffoldArgs{
				ForceRewrite: true,
			},
			setup: func(t *testing.T) string {
				dir := t.TempDir()
				os.WriteFile(filepath.Join(dir, "existing.txt"), []byte("test"), 0644)
				return dir
			},
			wantErr: false,
			validate: func(t *testing.T, dir string, result *ScaffoldResult) {
				// Old file should be gone
				if _, err := os.Stat(filepath.Join(dir, "existing.txt")); !os.IsNotExist(err) {
					t.Errorf("existing.txt should not exist")
				}
				// New files exist
				if _, err := os.Stat(filepath.Join(dir, "package.json")); os.IsNotExist(err) {
					t.Errorf("package.json does not exist")
				}
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			dir := tt.setup(t)
			tt.args.WorkDir = dir

			cfg := &mcp.IoConfig{}
			sess := session.NewSession()
			p, err := NewProvider(cfg, sess, slog.Default())
			if err != nil {
				t.Fatalf("Failed to create provider: %v", err)
			}

			result, err := p.Scaffold(context.Background(), tt.args)
			if (err != nil) != tt.wantErr {
				t.Errorf("Scaffold() error = %v, wantErr %v", err, tt.wantErr)
				return
			}
			if !tt.wantErr && tt.validate != nil {
				tt.validate(t, dir, result)
			}
		})
	}
}
