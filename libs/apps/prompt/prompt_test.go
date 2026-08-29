package prompt

import (
	"context"
	"errors"
	"os"
	"path/filepath"
	"testing"
	"time"

	"github.com/databricks/cli/libs/apps/manifest"
	"github.com/databricks/cli/libs/cmdio"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestValidateProjectName(t *testing.T) {
	tests := []struct {
		name        string
		projectName string
		expectError bool
		errorMsg    string
	}{
		{
			name:        "valid simple name",
			projectName: "my-app",
			expectError: false,
		},
		{
			name:        "valid name with numbers",
			projectName: "app123",
			expectError: false,
		},
		{
			name:        "valid name with hyphens",
			projectName: "my-cool-app",
			expectError: false,
		},
		{
			name:        "empty name",
			projectName: "",
			expectError: true,
			errorMsg:    "required",
		},
		{
			name:        "name too long",
			projectName: "this-is-a-very-long-app-name-that-exceeds",
			expectError: true,
			errorMsg:    "too long",
		},
		{
			name:        "name at max length (26 chars)",
			projectName: "abcdefghijklmnopqrstuvwxyz",
			expectError: false,
		},
		{
			name:        "name starts with number",
			projectName: "123app",
			expectError: true,
			errorMsg:    "must start with a letter",
		},
		{
			name:        "name starts with hyphen",
			projectName: "-myapp",
			expectError: true,
			errorMsg:    "must start with a letter",
		},
		{
			name:        "name with uppercase",
			projectName: "MyApp",
			expectError: true,
			errorMsg:    "lowercase",
		},
		{
			name:        "name with underscore",
			projectName: "my_app",
			expectError: true,
			errorMsg:    "lowercase letters, numbers, or hyphens",
		},
		{
			name:        "name with spaces",
			projectName: "my app",
			expectError: true,
			errorMsg:    "lowercase letters, numbers, or hyphens",
		},
		{
			name:        "name with special characters",
			projectName: "my@app!",
			expectError: true,
			errorMsg:    "lowercase letters, numbers, or hyphens",
		},
		{
			name:        "in-place sentinel",
			projectName: InPlaceName,
			expectError: false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			err := ValidateProjectName(tt.projectName)
			if tt.expectError {
				require.Error(t, err)
				assert.Contains(t, err.Error(), tt.errorMsg)
			} else {
				assert.NoError(t, err)
			}
		})
	}
}

func TestRunWithSpinnerCtx(t *testing.T) {
	t.Run("successful action", func(t *testing.T) {
		ctx := cmdio.MockDiscard(t.Context())
		executed := false

		err := RunWithSpinnerCtx(ctx, "Testing...", func() error {
			executed = true
			return nil
		})

		assert.NoError(t, err)
		assert.True(t, executed)
	})

	t.Run("action returns error", func(t *testing.T) {
		ctx := cmdio.MockDiscard(t.Context())
		expectedErr := errors.New("action failed")

		err := RunWithSpinnerCtx(ctx, "Testing...", func() error {
			return expectedErr
		})

		assert.Equal(t, expectedErr, err)
	})

	t.Run("context cancelled", func(t *testing.T) {
		ctx, cancel := context.WithCancel(cmdio.MockDiscard(t.Context()))
		actionStarted := make(chan struct{})
		actionDone := make(chan struct{})

		go func() {
			_ = RunWithSpinnerCtx(ctx, "Testing...", func() error {
				close(actionStarted)
				time.Sleep(100 * time.Millisecond)
				close(actionDone)
				return nil
			})
		}()

		// Wait for action to start
		<-actionStarted
		// Cancel context
		cancel()
		// Wait for action to complete (spinner should wait)
		<-actionDone
	})

	t.Run("action panics - recovered", func(t *testing.T) {
		ctx := cmdio.MockDiscard(t.Context())

		err := RunWithSpinnerCtx(ctx, "Testing...", func() error {
			panic("test panic")
		})

		require.Error(t, err)
		assert.Contains(t, err.Error(), "action panicked")
		assert.Contains(t, err.Error(), "test panic")
	})
}

func TestRunModeConstants(t *testing.T) {
	assert.Equal(t, RunModeNone, RunMode("none"))
	assert.Equal(t, RunModeDev, RunMode("dev"))
	assert.Equal(t, RunModeDevRemote, RunMode("dev-remote"))
}

func TestApplyResolvedValues(t *testing.T) {
	t.Run("maps resolver names to manifest field names", func(t *testing.T) {
		r := manifest.Resource{
			ResourceKey: "postgres",
			Fields: map[string]manifest.ResourceField{
				"branch":       {Description: "branch path"},
				"database":     {Description: "database name"},
				"host":         {Resolve: "postgres:host"},
				"databaseName": {Resolve: "postgres:databaseName"},
				"endpointPath": {Resolve: "postgres:endpointPath"},
				"port":         {Value: "5432"},
			},
		}

		resolvedValues := map[string]string{
			"postgres:host":         "my-host.example.com",
			"postgres:databaseName": "my_db",
			"postgres:endpointPath": "projects/p1/branches/b1/endpoints/e1",
		}

		result := map[string]string{
			"postgres.branch":   "projects/p1/branches/b1",
			"postgres.database": "projects/p1/branches/b1/databases/d1",
		}

		applyResolvedValues(r, resolvedValues, result)

		assert.Equal(t, map[string]string{
			"postgres.branch":       "projects/p1/branches/b1",
			"postgres.database":     "projects/p1/branches/b1/databases/d1",
			"postgres.host":         "my-host.example.com",
			"postgres.databaseName": "my_db",
			"postgres.endpointPath": "projects/p1/branches/b1/endpoints/e1",
		}, result)
	})

	t.Run("renamed fields still map via resolver", func(t *testing.T) {
		r := manifest.Resource{
			ResourceKey: "postgres",
			Fields: map[string]manifest.ResourceField{
				"pg_host":     {Resolve: "postgres:host"},
				"pg_database": {Resolve: "postgres:databaseName"},
				"pg_endpoint": {Resolve: "postgres:endpointPath"},
			},
		}

		resolvedValues := map[string]string{
			"postgres:host":         "host.example.com",
			"postgres:databaseName": "testdb",
			"postgres:endpointPath": "projects/p1/branches/b1/endpoints/e1",
		}

		result := map[string]string{}
		applyResolvedValues(r, resolvedValues, result)

		assert.Equal(t, map[string]string{
			"postgres.pg_host":     "host.example.com",
			"postgres.pg_database": "testdb",
			"postgres.pg_endpoint": "projects/p1/branches/b1/endpoints/e1",
		}, result)
	})

	t.Run("skips fields without resolve", func(t *testing.T) {
		r := manifest.Resource{
			ResourceKey: "postgres",
			Fields: map[string]manifest.ResourceField{
				"branch": {Description: "no resolve"},
				"host":   {Resolve: "postgres:host"},
				"port":   {Value: "5432"},
			},
		}

		resolvedValues := map[string]string{
			"postgres:host": "my-host",
		}

		result := map[string]string{}
		applyResolvedValues(r, resolvedValues, result)

		assert.Equal(t, map[string]string{
			"postgres.host": "my-host",
		}, result)
	})

	t.Run("skips resolve values not in resolvedValues map", func(t *testing.T) {
		r := manifest.Resource{
			ResourceKey: "postgres",
			Fields: map[string]manifest.ResourceField{
				"host":    {Resolve: "postgres:host"},
				"unknown": {Resolve: "postgres:unknownResolver"},
			},
		}

		resolvedValues := map[string]string{
			"postgres:host": "my-host",
		}

		result := map[string]string{}
		applyResolvedValues(r, resolvedValues, result)

		assert.Equal(t, map[string]string{
			"postgres.host": "my-host",
		}, result)
	})
}

func TestVolumePathToSecurableName(t *testing.T) {
	tests := []struct {
		input string
		want  string
	}{
		{"/Volumes/catalog/schema/vol", "catalog.schema.vol"},
		{"/Volumes/my-cat/my-schema/my-vol", "my-cat.my-schema.my-vol"},
		{"catalog.schema.vol", "catalog.schema.vol"},
		{"/Volumes/a/b/c", "a.b.c"},
	}
	for _, tt := range tests {
		t.Run(tt.input, func(t *testing.T) {
			assert.Equal(t, tt.want, volumePathToSecurableName(tt.input))
		})
	}
}

func TestMaxAppNameLength(t *testing.T) {
	// Verify the constant is set correctly
	assert.Equal(t, 30, MaxAppNameLength)
	assert.Equal(t, "dev-", DevTargetPrefix)

	// Max allowed name length should be 30 - 4 ("dev-") = 26
	maxAllowed := MaxAppNameLength - len(DevTargetPrefix)
	assert.Equal(t, 26, maxAllowed)

	// Test at boundary
	validName := "abcdefghijklmnopqrstuvwxyz" // 26 chars
	assert.Len(t, validName, 26)
	assert.NoError(t, ValidateProjectName(validName))

	// Test over boundary
	invalidName := "abcdefghijklmnopqrstuvwxyz1" // 27 chars
	assert.Len(t, invalidName, 27)
	assert.Error(t, ValidateProjectName(invalidName))
}

func TestRenderStabilityTier(t *testing.T) {
	tests := []struct {
		tier       string
		wantEmpty  bool
		wantSubstr string
	}{
		{"", true, ""},
		{"beta", false, "(beta)"},
		{"alpha", false, "(alpha)"},
	}
	for _, tc := range tests {
		t.Run(tc.tier, func(t *testing.T) {
			got := RenderStabilityTier(tc.tier)
			if tc.wantEmpty {
				assert.Empty(t, got)
				return
			}
			assert.Contains(t, got, tc.wantSubstr)
		})
	}
}

// mkDirNamed creates a child directory with the given name under t.TempDir()
// and returns its absolute path. Used to control filepath.Base(absCwd) when
// exercising in-place name derivation.
func mkDirNamed(t *testing.T, name string) string {
	t.Helper()
	dir := filepath.Join(t.TempDir(), name)
	require.NoError(t, os.MkdirAll(dir, 0o755))
	return dir
}

func TestDeriveInPlaceAppName(t *testing.T) {
	t.Run("valid basename", func(t *testing.T) {
		dir := mkDirNamed(t, "my-app")
		got, err := DeriveInPlaceAppName(dir)
		require.NoError(t, err)
		assert.Equal(t, "my-app", got)
	})

	t.Run("uppercase basename rejected", func(t *testing.T) {
		dir := mkDirNamed(t, "MyApp")
		_, err := DeriveInPlaceAppName(dir)
		require.Error(t, err)
		assert.Contains(t, err.Error(), "MyApp")
		assert.Contains(t, err.Error(), "rename the directory")
	})

	t.Run("underscore basename rejected", func(t *testing.T) {
		dir := mkDirNamed(t, "my_app")
		_, err := DeriveInPlaceAppName(dir)
		require.Error(t, err)
		assert.Contains(t, err.Error(), "my_app")
	})

	t.Run("leading digit basename rejected", func(t *testing.T) {
		dir := mkDirNamed(t, "1app")
		_, err := DeriveInPlaceAppName(dir)
		require.Error(t, err)
		assert.Contains(t, err.Error(), "1app")
	})

	t.Run("too long basename rejected", func(t *testing.T) {
		dir := mkDirNamed(t, "this-is-far-too-long-for-the-app-name-limit")
		_, err := DeriveInPlaceAppName(dir)
		require.Error(t, err)
	})
}

func TestCheckInPlaceDirectory(t *testing.T) {
	t.Run("empty directory is OK", func(t *testing.T) {
		dir := t.TempDir()
		assert.NoError(t, CheckInPlaceDirectory(dir))
	})

	t.Run("dotgit only is OK", func(t *testing.T) {
		dir := t.TempDir()
		require.NoError(t, os.MkdirAll(filepath.Join(dir, ".git"), 0o755))
		assert.NoError(t, CheckInPlaceDirectory(dir))
	})

	t.Run("pre-existing gitignore is rejected", func(t *testing.T) {
		// .gitignore is intentionally NOT allow-listed: the template ships
		// _gitignore that renames to .gitignore, which would silently
		// overwrite the user's file.
		dir := t.TempDir()
		require.NoError(t, os.MkdirAll(filepath.Join(dir, ".git"), 0o755))
		require.NoError(t, os.WriteFile(filepath.Join(dir, ".gitignore"), []byte("node_modules\n"), 0o644))
		err := CheckInPlaceDirectory(dir)
		require.Error(t, err)
		assert.Contains(t, err.Error(), "not empty")
	})

	t.Run("unexpected file is rejected", func(t *testing.T) {
		dir := t.TempDir()
		require.NoError(t, os.WriteFile(filepath.Join(dir, "README.md"), []byte("hi"), 0o644))
		err := CheckInPlaceDirectory(dir)
		require.Error(t, err)
		assert.Contains(t, err.Error(), "not empty")
		assert.Contains(t, err.Error(), "apps init --name .")
		// Concise wording: we deliberately do not enumerate offending files.
		assert.NotContains(t, err.Error(), "README.md")
	})

	t.Run("mix of allowed and disallowed is rejected", func(t *testing.T) {
		dir := t.TempDir()
		require.NoError(t, os.MkdirAll(filepath.Join(dir, ".git"), 0o755))
		require.NoError(t, os.WriteFile(filepath.Join(dir, "stray.txt"), []byte(""), 0o644))
		err := CheckInPlaceDirectory(dir)
		require.Error(t, err)
		assert.Contains(t, err.Error(), "not empty")
	})

	t.Run("symlink named .git is rejected", func(t *testing.T) {
		// A symlink masquerading as .git would let os.WriteFile follow the
		// link if any allow-listed name later became a write target.
		dir := t.TempDir()
		target := filepath.Join(t.TempDir(), "elsewhere")
		require.NoError(t, os.MkdirAll(target, 0o755))
		require.NoError(t, os.Symlink(target, filepath.Join(dir, ".git")))
		err := CheckInPlaceDirectory(dir)
		require.Error(t, err)
		assert.Contains(t, err.Error(), "not empty")
	})

	t.Run("error message shows absolute path", func(t *testing.T) {
		dir := t.TempDir()
		require.NoError(t, os.WriteFile(filepath.Join(dir, "stray.txt"), []byte(""), 0o644))
		err := CheckInPlaceDirectory(dir)
		require.Error(t, err)
		assert.Contains(t, err.Error(), dir)
	})

	t.Run("missing directory returns error", func(t *testing.T) {
		err := CheckInPlaceDirectory(filepath.Join(t.TempDir(), "nope"))
		require.Error(t, err)
	})
}

func TestShouldOfferInPlace(t *testing.T) {
	t.Run("returns basename when dir is empty and name is valid", func(t *testing.T) {
		dir := mkDirNamed(t, "my-app")
		name, ok := ShouldOfferInPlace(dir)
		assert.True(t, ok)
		assert.Equal(t, "my-app", name)
	})

	t.Run("declines when dir has stray files", func(t *testing.T) {
		dir := mkDirNamed(t, "my-app")
		require.NoError(t, os.WriteFile(filepath.Join(dir, "stray.txt"), []byte(""), 0o644))
		_, ok := ShouldOfferInPlace(dir)
		assert.False(t, ok)
	})

	t.Run("declines when basename is invalid", func(t *testing.T) {
		dir := mkDirNamed(t, "Bad_Name")
		_, ok := ShouldOfferInPlace(dir)
		assert.False(t, ok)
	})

	t.Run("declines when dir does not exist", func(t *testing.T) {
		_, ok := ShouldOfferInPlace(filepath.Join(t.TempDir(), "missing"))
		assert.False(t, ok)
	})
}

func TestValidateProjectNameForPrompt(t *testing.T) {
	t.Run("valid name without outputDir", func(t *testing.T) {
		assert.NoError(t, validateProjectNameForPrompt("my-app", ""))
	})

	t.Run("in-place sentinel without outputDir is accepted", func(t *testing.T) {
		assert.NoError(t, validateProjectNameForPrompt(InPlaceName, ""))
	})

	t.Run("in-place sentinel with outputDir is rejected with sentinel error", func(t *testing.T) {
		err := validateProjectNameForPrompt(InPlaceName, "/some/dir")
		require.Error(t, err)
		assert.ErrorIs(t, err, ErrNameDotWithOutputDir)
	})

	t.Run("invalid name surfaces ValidateProjectName error", func(t *testing.T) {
		err := validateProjectNameForPrompt("My_App", "")
		require.Error(t, err)
		assert.Contains(t, err.Error(), "lowercase letters")
	})
}

func TestPrintSuccessInPlace(t *testing.T) {
	ctx, out := cmdio.NewTestContextWithStderr(t.Context())
	PrintSuccess(ctx, "my-app", "/abs/path/my-app", 12, "npm run dev", true)
	got := out.String()
	assert.Contains(t, got, "Location: /abs/path/my-app")
	assert.Contains(t, got, "Files: 12")
	assert.Contains(t, got, "npm run dev")
	assert.NotContains(t, got, "cd my-app")
}

func TestPrintSuccessNotInPlace(t *testing.T) {
	ctx, out := cmdio.NewTestContextWithStderr(t.Context())
	PrintSuccess(ctx, "my-app", "/abs/path/my-app", 12, "npm run dev", false)
	got := out.String()
	assert.Contains(t, got, "cd my-app")
	assert.Contains(t, got, "npm run dev")
}
