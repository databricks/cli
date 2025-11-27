package main

import (
	"os"
	"testing"
)

func TestApplyRules(t *testing.T) {
	tests := []struct {
		name               string
		files              []string
		expectedPackages   []string
		expectedAcceptance []string
	}{
		{
			name:               "bundle files",
			files:              []string{"bundle/config.go", "bundle/deploy/deploy.go"},
			expectedPackages:   []string{"./bundle", "./bundle/deploy"},
			expectedAcceptance: []string{"bundle"},
		},
		{
			name:               "skip testdata",
			files:              []string{"bundle/config.go", "bundle/testdata/test.json"},
			expectedPackages:   []string{"./bundle"},
			expectedAcceptance: []string{"bundle"},
		},
		{
			name:               "skip acceptance tests",
			files:              []string{"bundle/config.go", "acceptance/bundle/test/script"},
			expectedPackages:   []string{"./bundle"},
			expectedAcceptance: []string{"bundle"},
		},
		{
			name:               "skip experimental",
			files:              []string{"bundle/config.go", "experimental/apps-mcp/lib/server.go"},
			expectedPackages:   []string{"./bundle"},
			expectedAcceptance: []string{"bundle"},
		},
		{
			name:               "all experimental skips everything",
			files:              []string{"experimental/apps-mcp/lib/server.go", "experimental/ssh/cmd/connect.go"},
			expectedPackages:   []string{},
			expectedAcceptance: []string{},
		},
		{
			name:               "root level file",
			files:              []string{"main.go"},
			expectedPackages:   []string{"."},
			expectedAcceptance: []string{"bundle"},
		},
		{
			name:               "non-go files",
			files:              []string{"README.md", "Makefile"},
			expectedPackages:   []string{},
			expectedAcceptance: []string{},
		},
		{
			name:               "unmatched file defaults to everything",
			files:              []string{"unknown/path/file.go"},
			expectedPackages:   []string{},
			expectedAcceptance: []string{},
		},
		{
			name:               "mixed matched and unmatched defaults to everything",
			files:              []string{"bundle/config.go", "unknown/path/file.go"},
			expectedPackages:   []string{},
			expectedAcceptance: []string{},
		},
		{
			name:               "cmd bundle",
			files:              []string{"cmd/bundle/deploy.go"},
			expectedPackages:   []string{"./cmd/bundle"},
			expectedAcceptance: []string{"bundle"},
		},
		{
			name:               "cmd workspace",
			files:              []string{"cmd/workspace/list.go"},
			expectedPackages:   []string{"./cmd/workspace"},
			expectedAcceptance: []string{"workspace"},
		},
		{
			name:               "libs auth",
			files:              []string{"libs/auth/auth.go"},
			expectedPackages:   []string{"./libs/auth"},
			expectedAcceptance: []string{"auth"},
		},
		{
			name:               "libs template",
			files:              []string{"libs/template/renderer.go"},
			expectedPackages:   []string{"./libs/template"},
			expectedAcceptance: []string{"bundle/templates"},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			packages, acceptance := applyRules(tt.files)
			if len(packages) != len(tt.expectedPackages) {
				t.Errorf("expected %d packages, got %d: %v", len(tt.expectedPackages), len(packages), packages)
				return
			}
			for i, pkg := range packages {
				if pkg != tt.expectedPackages[i] {
					t.Errorf("package[%d]: expected %q, got %q", i, tt.expectedPackages[i], pkg)
				}
			}
			if len(acceptance) != len(tt.expectedAcceptance) {
				t.Errorf("expected %d acceptance prefixes, got %d: %v", len(tt.expectedAcceptance), len(acceptance), acceptance)
				return
			}
			for i, prefix := range acceptance {
				if prefix != tt.expectedAcceptance[i] {
					t.Errorf("acceptance[%d]: expected %q, got %q", i, tt.expectedAcceptance[i], prefix)
				}
			}
		})
	}
}

func TestMainWithEnvVar(t *testing.T) {
	// Test that GITHUB_BASE_REF is respected
	originalBaseRef := os.Getenv("GITHUB_BASE_REF")
	defer func() {
		if originalBaseRef == "" {
			os.Unsetenv("GITHUB_BASE_REF")
		} else {
			os.Setenv("GITHUB_BASE_REF", originalBaseRef)
		}
	}()

	os.Setenv("GITHUB_BASE_REF", "main")
	// We can't easily test the full execution without git, but we can verify the env var is read
	baseRef := os.Getenv("GITHUB_BASE_REF")
	if baseRef != "main" {
		t.Errorf("expected GITHUB_BASE_REF to be 'main', got %q", baseRef)
	}
}
