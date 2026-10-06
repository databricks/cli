package validation_test

import (
	"bytes"
	"log/slog"
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/databricks/cli/internal/testutil"
	"github.com/databricks/cli/libs/apps/packagejson"
	"github.com/databricks/cli/libs/apps/validation"
	"github.com/databricks/cli/libs/log"
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
	for _, lockfile := range []string{"package-lock.json", "npm-shrinkwrap.json", "pnpm-lock.yaml"} {
		t.Run(lockfile, func(t *testing.T) {
			dir := t.TempDir()
			require.NoError(t, os.Mkdir(filepath.Join(dir, lockfile), 0o755))
			_, err := validation.DetectPackageManager(t.Context(), dir)
			require.EqualError(t, err, "lockfile "+lockfile+" must be a regular file")
		})
	}
}

func TestDetectPackageManagerDeclaration(t *testing.T) {
	tests := []struct {
		name      string
		manifest  string
		lockfiles []string
		want      string
		warnings  []string
	}{
		{name: "npm without lockfile", manifest: `{"packageManager":"npm@10.9.2"}`, want: "npm"},
		{name: "pnpm without lockfile", manifest: `{"packageManager":"pnpm@10.30.3"}`, want: "pnpm"},
		{name: "name without version", manifest: `{"packageManager":"pnpm"}`, want: "pnpm"},
		{name: "version remains native", manifest: `{"packageManager":"pnpm@invalid-version"}`, want: "pnpm"},
		{
			name:      "npm overrides pnpm lockfile",
			manifest:  `{"packageManager":"npm@10.9.2"}`,
			lockfiles: []string{"pnpm-lock.yaml"},
			want:      "npm",
			warnings:  []string{"using npm from package.json and ignoring pnpm-lock.yaml"},
		},
		{
			name:      "pnpm overrides npm lockfiles",
			manifest:  `{"packageManager":"pnpm@10.30.3"}`,
			lockfiles: []string{"package-lock.json", "npm-shrinkwrap.json", "pnpm-lock.yaml"},
			want:      "pnpm",
			warnings: []string{
				"using pnpm from package.json and ignoring package-lock.json",
				"using pnpm from package.json and ignoring npm-shrinkwrap.json",
			},
		},
		{name: "matching lockfile", manifest: `{"packageManager":"pnpm@10.30.3"}`, lockfiles: []string{"pnpm-lock.yaml"}, want: "pnpm"},
		{name: "missing field", manifest: `{}`, lockfiles: []string{"pnpm-lock.yaml"}, want: "pnpm"},
		{name: "empty field", manifest: `{"packageManager":""}`, lockfiles: []string{"pnpm-lock.yaml"}, want: "pnpm"},
		{name: "null field", manifest: `{"packageManager":null}`, lockfiles: []string{"pnpm-lock.yaml"}, want: "pnpm"},
		{name: "exact field name", manifest: `{"PackageManager":"pnpm@10.30.3"}`, want: "npm"},
		{name: "devEngines is not used", manifest: `{"devEngines":{"packageManager":{"name":"pnpm"}}}`, want: "npm"},
		{name: "scripts are not interpreted", manifest: `{"packageManager":"pnpm@10.30.3","scripts":{"typecheck":null,"//":["a comment"]}}`, want: "pnpm"},
		{name: "BOM", manifest: "\xef\xbb\xbf" + `{"packageManager":"pnpm@10.30.3"}`, want: "pnpm"},
		{
			name:      "unsupported declaration uses lockfiles",
			manifest:  `{"packageManager":"yarn@4.0.0"}`,
			lockfiles: []string{"pnpm-lock.yaml"},
			want:      "pnpm",
			warnings:  []string{"ignoring unsupported packageManager"},
		},
		{
			name:     "unsupported declaration uses npm default",
			manifest: `{"packageManager":"bun@1.0.0"}`,
			want:     "npm",
			warnings: []string{"ignoring unsupported packageManager"},
		},
		{
			name:     "exact manager name",
			manifest: `{"packageManager":"PNPM@10.30.3"}`,
			want:     "npm",
			warnings: []string{"ignoring unsupported packageManager"},
		},
		{
			name:      "non-string declaration uses lockfiles",
			manifest:  `{"packageManager":{"name":"npm"}}`,
			lockfiles: []string{"pnpm-lock.yaml"},
			want:      "pnpm",
			warnings:  []string{"ignoring invalid packageManager"},
		},
		{name: "malformed manifest uses npm default", manifest: `{"packageManager":"pnpm@10.30.3",}`, want: "npm"},
		{name: "malformed manifest uses lockfiles", manifest: `{"packageManager":"npm@10.9.2",}`, lockfiles: []string{"pnpm-lock.yaml"}, want: "pnpm"},
		{name: "null manifest stays native", manifest: `null`, lockfiles: []string{"pnpm-lock.yaml"}, want: "pnpm"},
		{name: "non-object manifest stays native", manifest: `[]`, lockfiles: []string{"pnpm-lock.yaml"}, want: "pnpm"},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			dir := t.TempDir()
			testutil.WriteFile(t, filepath.Join(dir, packagejson.FileName), tt.manifest)
			for _, lockfile := range tt.lockfiles {
				testutil.Touch(t, dir, lockfile)
			}
			var warnings bytes.Buffer
			ctx := log.NewContext(t.Context(), slog.New(slog.NewTextHandler(&warnings, nil)))
			manager, err := validation.DetectPackageManager(ctx, dir)
			require.NoError(t, err)
			assert.Equal(t, tt.want, manager)
			assert.Equal(t, len(tt.warnings), strings.Count(warnings.String(), "level=WARN"), warnings.String())
			for _, warning := range tt.warnings {
				assert.Contains(t, warnings.String(), warning)
			}
		})
	}
}

func TestDetectPackageManagerDeclarationInvalidLockfile(t *testing.T) {
	dir := t.TempDir()
	testutil.WriteFile(t, filepath.Join(dir, packagejson.FileName), `{"packageManager":"npm@10.9.2"}`)
	require.NoError(t, os.Mkdir(filepath.Join(dir, "pnpm-lock.yaml"), 0o755))
	_, err := validation.DetectPackageManager(t.Context(), dir)
	require.EqualError(t, err, "lockfile pnpm-lock.yaml must be a regular file")
}

func TestDetectPackageManagerDeclarationReadError(t *testing.T) {
	dir := t.TempDir()
	require.NoError(t, os.Mkdir(filepath.Join(dir, packagejson.FileName), 0o755))
	testutil.Touch(t, dir, "pnpm-lock.yaml")
	manager, err := validation.DetectPackageManager(t.Context(), dir)
	require.NoError(t, err)
	assert.Equal(t, "pnpm", manager)
}

func TestDetectPackageManagerDeclarationProjectDirectory(t *testing.T) {
	dir := t.TempDir()
	testutil.WriteFile(t, filepath.Join(dir, packagejson.FileName), `{"packageManager":"pnpm@10.30.3"}`)
	t.Chdir(dir)
	for _, tt := range []struct {
		name     string
		manifest string
		lockfile string
	}{
		{name: "declared", manifest: `{"packageManager":"npm@10.9.2"}`, lockfile: "pnpm-lock.yaml"},
		{name: "no-parent-inheritance", manifest: `{}`},
	} {
		t.Run(tt.name, func(t *testing.T) {
			testutil.WriteFile(t, filepath.Join(tt.name, packagejson.FileName), tt.manifest)
			if tt.lockfile != "" {
				testutil.Touch(t, tt.name, tt.lockfile)
			}
			manager, err := validation.DetectPackageManager(t.Context(), tt.name)
			require.NoError(t, err)
			assert.Equal(t, "npm", manager)
		})
	}
}

func TestDetectPackageManagerUnsupportedLockfileDirectory(t *testing.T) {
	for _, lockfile := range []string{"yarn.lock", "bun.lock", "bun.lockb"} {
		for _, manager := range []string{"npm", "pnpm"} {
			t.Run(lockfile+"/"+manager, func(t *testing.T) {
				dir := t.TempDir()
				require.NoError(t, os.Mkdir(filepath.Join(dir, lockfile), 0o755))
				if manager == "pnpm" {
					testutil.Touch(t, dir, "pnpm-lock.yaml")
				}
				var warnings bytes.Buffer
				ctx := log.NewContext(t.Context(), slog.New(slog.NewTextHandler(&warnings, nil)))
				actual, err := validation.DetectPackageManager(ctx, dir)
				require.NoError(t, err)
				assert.Equal(t, manager, actual)
				assert.Contains(t, warnings.String(), "ignoring unsupported lockfile "+lockfile)
			})
		}
	}
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
