package validation_test

import (
	"os"
	"path/filepath"
	"testing"

	"github.com/databricks/cli/internal/testutil"
	"github.com/databricks/cli/libs/apps/validation"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestDetectPackageManager(t *testing.T) {
	tests := []struct {
		name      string
		lockfiles []string
		want      string
		wantError string
	}{
		{name: "no lockfile", want: "npm"},
		{name: "npm", lockfiles: []string{"package-lock.json"}, want: "npm"},
		{name: "npm shrinkwrap", lockfiles: []string{"npm-shrinkwrap.json"}, want: "npm"},
		{name: "pnpm", lockfiles: []string{"pnpm-lock.yaml"}, want: "pnpm"},
		{name: "npm aliases", lockfiles: []string{"package-lock.json", "npm-shrinkwrap.json"}, want: "npm"},
		{
			name:      "unsupported yarn",
			lockfiles: []string{"yarn.lock"},
			wantError: "yarn is not supported for apps validation (found yarn.lock); use npm or pnpm",
		},
		{
			name:      "unsupported bun",
			lockfiles: []string{"bun.lock"},
			wantError: "bun is not supported for apps validation (found bun.lock); use npm or pnpm",
		},
		{
			name:      "unsupported bun binary",
			lockfiles: []string{"bun.lockb"},
			wantError: "bun is not supported for apps validation (found bun.lockb); use npm or pnpm",
		},
		{
			name:      "unsupported manager alongside npm",
			lockfiles: []string{"package-lock.json", "yarn.lock"},
			wantError: "yarn is not supported for apps validation (found yarn.lock); use npm or pnpm",
		},
		{
			name:      "conflicting managers",
			lockfiles: []string{"pnpm-lock.yaml", "package-lock.json"},
			wantError: "conflicting package manager lockfiles: package-lock.json, pnpm-lock.yaml; keep lockfiles for only one package manager",
		},
		{
			name:      "conflict includes all lockfiles",
			lockfiles: []string{"pnpm-lock.yaml", "package-lock.json", "npm-shrinkwrap.json"},
			wantError: "conflicting package manager lockfiles: package-lock.json, npm-shrinkwrap.json, pnpm-lock.yaml; keep lockfiles for only one package manager",
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			dir := t.TempDir()
			for _, lockfile := range tt.lockfiles {
				testutil.Touch(t, dir, lockfile)
			}
			manager, err := validation.DetectPackageManager(dir)
			if tt.wantError != "" {
				require.EqualError(t, err, tt.wantError)
				assert.Empty(t, manager)
				return
			}
			require.NoError(t, err)
			assert.Equal(t, tt.want, manager)
		})
	}
}

func TestDetectPackageManagerInvalidLockfile(t *testing.T) {
	dir := t.TempDir()
	require.NoError(t, os.Mkdir(filepath.Join(dir, "pnpm-lock.yaml"), 0o755))
	_, err := validation.DetectPackageManager(dir)
	require.EqualError(t, err, "lockfile pnpm-lock.yaml must be a regular file")
}

func TestDetectPackageManagerProjectDirectory(t *testing.T) {
	dir := t.TempDir()
	testutil.Touch(t, dir, "pnpm-lock.yaml")
	projectDir := filepath.Join(dir, "app")
	testutil.Touch(t, projectDir, "package-lock.json")
	manager, err := validation.DetectPackageManager(projectDir)
	require.NoError(t, err)
	assert.Equal(t, "npm", manager)
}
