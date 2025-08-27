package bundle

import (
	"context"
	"io/fs"
	"os"
	"path/filepath"
	"testing"

	"github.com/databricks/cli/bundle/config"
	"github.com/databricks/cli/bundle/config/resources"
	"github.com/databricks/cli/bundle/env"
	"github.com/databricks/cli/internal/testutil"
	"github.com/databricks/cli/libs/diag"
	"github.com/databricks/cli/libs/dyn"
	"github.com/databricks/cli/libs/logdiag"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// Test wrapper functions to make MustLoad/TryLoad tests more readable

// mustLoad calls MustLoad and returns the bundle and collected diagnostics
func mustLoad(t *testing.T) (*Bundle, []diag.Diagnostic) {
	ctx := logdiag.InitContext(context.Background())
	logdiag.SetCollect(ctx, true)
	b := MustLoad(ctx)
	diags := logdiag.FlushCollected(ctx)
	return b, diags
}

// tryLoad calls TryLoad and returns the bundle and collected diagnostics
func tryLoad(t *testing.T) (*Bundle, []diag.Diagnostic) {
	ctx := logdiag.InitContext(context.Background())
	logdiag.SetCollect(ctx, true)
	b := TryLoad(ctx)
	diags := logdiag.FlushCollected(ctx)
	return b, diags
}

func TestLoadNotExists(t *testing.T) {
	b, err := Load(context.Background(), "/doesntexist")
	assert.ErrorIs(t, err, fs.ErrNotExist)
	assert.Nil(t, b)
}

func TestLoadExists(t *testing.T) {
	b, err := Load(context.Background(), "./tests/basic")
	assert.NoError(t, err)
	assert.NotNil(t, b)
}

func TestBundleLocalStateDir(t *testing.T) {
	ctx := context.Background()
	projectDir := t.TempDir()
	f1, err := os.Create(filepath.Join(projectDir, "databricks.yml"))
	require.NoError(t, err)
	f1.Close()

	bundle, err := Load(ctx, projectDir)
	require.NoError(t, err)

	// Artificially set target.
	// This is otherwise done by [mutators.SelectTarget].
	bundle.Config.Bundle.Target = "default"

	// unset env variable in case it's set
	t.Setenv("DATABRICKS_BUNDLE_TMP", "")

	cacheDir, err := bundle.LocalStateDir(ctx)

	// format is <CWD>/.databricks/bundle/<target>
	assert.NoError(t, err)
	assert.Equal(t, filepath.Join(projectDir, ".databricks", "bundle", "default"), cacheDir)
}

func TestBundleLocalStateDirOverride(t *testing.T) {
	ctx := context.Background()
	projectDir := t.TempDir()
	bundleTmpDir := t.TempDir()
	f1, err := os.Create(filepath.Join(projectDir, "databricks.yml"))
	require.NoError(t, err)
	f1.Close()

	bundle, err := Load(ctx, projectDir)
	require.NoError(t, err)

	// Artificially set target.
	// This is otherwise done by [mutators.SelectTarget].
	bundle.Config.Bundle.Target = "default"

	// now we expect to use 'bundleTmpDir' instead of CWD/.databricks/bundle
	t.Setenv("DATABRICKS_BUNDLE_TMP", bundleTmpDir)

	cacheDir, err := bundle.LocalStateDir(ctx)

	// format is <DATABRICKS_BUNDLE_TMP>/<target>
	assert.NoError(t, err)
	assert.Equal(t, filepath.Join(bundleTmpDir, "default"), cacheDir)
}

func TestBundleMustLoadSuccess(t *testing.T) {
	t.Setenv(env.RootVariable, "./tests/basic")
	b, diags := mustLoad(t)
	require.NotNil(t, b)
	assert.Empty(t, diags, "expected no diagnostics")
	assert.Equal(t, "tests/basic", filepath.ToSlash(b.BundleRootPath))
}

func TestBundleMustLoadFailureWithEnv(t *testing.T) {
	t.Setenv(env.RootVariable, "./tests/doesntexist")
	b, diags := mustLoad(t)
	require.Nil(t, b)
	require.Len(t, diags, 1, "expected diagnostics")
	assert.Contains(t, diags[0].Summary, "invalid bundle root")
	assert.Equal(t, diag.Error, diags[0].Severity)
}

func TestBundleMustLoadFailureIfNotFound(t *testing.T) {
	testutil.Chdir(t, t.TempDir())
	b, diags := mustLoad(t)
	require.Nil(t, b)
	require.Len(t, diags, 1, "expected diagnostics")
	assert.Contains(t, diags[0].Summary, "unable to locate bundle root")
	assert.Equal(t, diag.Error, diags[0].Severity)
}

func TestBundleTryLoadSuccess(t *testing.T) {
	t.Setenv(env.RootVariable, "./tests/basic")
	b, diags := tryLoad(t)
	require.NotNil(t, b)
	assert.Empty(t, diags, "expected no diagnostics")
	assert.Equal(t, "tests/basic", filepath.ToSlash(b.BundleRootPath))
}

func TestBundleTryLoadFailureWithEnv(t *testing.T) {
	t.Setenv(env.RootVariable, "./tests/doesntexist")
	b, diags := tryLoad(t)
	require.Nil(t, b)
	require.Len(t, diags, 1, "expected diagnostics")
	assert.Contains(t, diags[0].Summary, "invalid bundle root")
	assert.Equal(t, diag.Error, diags[0].Severity)
}

func TestBundleTryLoadOkIfNotFound(t *testing.T) {
	testutil.Chdir(t, t.TempDir())
	b, diags := tryLoad(t)
	assert.Nil(t, b)
	assert.Empty(t, diags, "expected no diagnostics")
}

func TestBundleGetResourceConfigJobsPointer(t *testing.T) {
	rootCfg := config.Root{
		Resources: config.Resources{
			Jobs: map[string]*resources.Job{
				"my_job": {
					// Empty job config is sufficient for this test.
				},
			},
		},
	}

	// Initialize the dynamic representation so GetResourceConfig can query it.
	require.NoError(t, rootCfg.Mutate(func(v dyn.Value) (dyn.Value, error) { return v, nil }))

	b := &Bundle{Config: rootCfg}

	res, ok := b.GetResourceConfig("jobs", "my_job")
	require.True(t, ok, "expected to find jobs.my_job in config")

	_, isJob := res.(*resources.Job)
	assert.True(t, isJob, "expected *resources.Job, got %T", res)

	res, ok = b.GetResourceConfig("jobs", "not_found")
	require.False(t, ok)
	require.Nil(t, res)

	res, ok = b.GetResourceConfig("not_found", "my_job")
	require.False(t, ok)
	require.Nil(t, res)
}
