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
	}{
		{name: "no lockfile", want: "npm"},
		{name: "npm", lockfiles: []string{"package-lock.json"}, want: "npm"},
		{name: "npm shrinkwrap", lockfiles: []string{"npm-shrinkwrap.json"}, want: "npm"},
		{name: "pnpm", lockfiles: []string{"pnpm-lock.yaml"}, want: "pnpm"},
		{name: "npm aliases", lockfiles: []string{"package-lock.json", "npm-shrinkwrap.json"}, want: "npm"},
		// Unsupported and conflicting lockfiles are ignored (with a warning), never fatal.
		{name: "unsupported yarn", lockfiles: []string{"yarn.lock"}, want: "npm"},
		{name: "unsupported bun", lockfiles: []string{"bun.lock"}, want: "npm"},
		{name: "unsupported bun binary", lockfiles: []string{"bun.lockb"}, want: "npm"},
		{name: "unsupported manager alongside pnpm", lockfiles: []string{"pnpm-lock.yaml", "yarn.lock"}, want: "pnpm"},
		{name: "conflicting managers prefer npm", lockfiles: []string{"pnpm-lock.yaml", "package-lock.json"}, want: "npm"},
		{name: "conflict with npm aliases", lockfiles: []string{"pnpm-lock.yaml", "package-lock.json", "npm-shrinkwrap.json"}, want: "npm"},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			dir := t.TempDir()
			for _, lockfile := range tt.lockfiles {
				testutil.Touch(t, dir, lockfile)
			}
			manager, err := validation.DetectPackageManager(t.Context(), dir)
			require.NoError(t, err)
			assert.Equal(t, tt.want, manager)
		})
	}
}

func TestDetectPackageManagerInvalidLockfile(t *testing.T) {
	dir := t.TempDir()
	require.NoError(t, os.Mkdir(filepath.Join(dir, "pnpm-lock.yaml"), 0o755))
	_, err := validation.DetectPackageManager(t.Context(), dir)
	require.EqualError(t, err, "lockfile pnpm-lock.yaml must be a regular file")
}

func TestDetectPackageManagerProjectDirectory(t *testing.T) {
	dir := t.TempDir()
	testutil.Touch(t, dir, "pnpm-lock.yaml")
	projectDir := filepath.Join(dir, "app")
	testutil.Touch(t, projectDir, "package-lock.json")
	manager, err := validation.DetectPackageManager(t.Context(), projectDir)
	require.NoError(t, err)
	assert.Equal(t, "npm", manager)
}
