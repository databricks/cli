package detector_test

import (
	"context"
	"os"
	"path/filepath"
	"testing"

	"github.com/databricks/cli/experimental/apps-mcp/lib/detector"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestDetectorRegistry_EmptyDir(t *testing.T) {
	dir := t.TempDir()
	ctx := context.Background()

	registry := detector.NewRegistry()
	detected := registry.Detect(ctx, dir)

	assert.False(t, detected.InProject)
	assert.Empty(t, detected.TargetTypes)
	assert.Empty(t, detected.Template)
}

func TestDetectorRegistry_BundleWithApps(t *testing.T) {
	dir := t.TempDir()
	ctx := context.Background()

	bundleYml := `bundle:
  name: my-app
resources:
  apps:
    my_app: {}
`
	require.NoError(t, os.WriteFile(filepath.Join(dir, "databricks.yml"), []byte(bundleYml), 0o644))

	registry := detector.NewRegistry()
	detected := registry.Detect(ctx, dir)

	assert.True(t, detected.InProject)
	assert.Equal(t, []string{"apps"}, detected.TargetTypes)
	assert.Equal(t, "my-app", detected.BundleInfo.Name)
}

func TestDetectorRegistry_BundleWithJobs(t *testing.T) {
	dir := t.TempDir()
	ctx := context.Background()

	bundleYml := `bundle:
  name: my-job
resources:
  jobs:
    daily_job: {}
`
	require.NoError(t, os.WriteFile(filepath.Join(dir, "databricks.yml"), []byte(bundleYml), 0o644))

	registry := detector.NewRegistry()
	detected := registry.Detect(ctx, dir)

	assert.True(t, detected.InProject)
	assert.Equal(t, []string{"jobs"}, detected.TargetTypes)
	assert.Equal(t, "my-job", detected.BundleInfo.Name)
}

func TestDetectorRegistry_CombinedBundle(t *testing.T) {
	dir := t.TempDir()
	ctx := context.Background()

	bundleYml := `bundle:
  name: my-project
resources:
  apps:
    my_app: {}
  jobs:
    daily_job: {}
`
	require.NoError(t, os.WriteFile(filepath.Join(dir, "databricks.yml"), []byte(bundleYml), 0o644))

	registry := detector.NewRegistry()
	detected := registry.Detect(ctx, dir)

	assert.True(t, detected.InProject)
	assert.Contains(t, detected.TargetTypes, "apps")
	assert.Contains(t, detected.TargetTypes, "jobs")
	assert.Equal(t, "my-project", detected.BundleInfo.Name)
}

func TestDetectorRegistry_AppkitTemplate(t *testing.T) {
	dir := t.TempDir()
	ctx := context.Background()

	// bundle + package.json with appkit marker
	bundleYml := `bundle:
  name: my-app
resources:
  apps:
    my_app: {}
`
	packageJson := `{
  "name": "my-app",
  "dependencies": {
    "@databricks/sql": "^1.0.0"
  }
}`
	require.NoError(t, os.WriteFile(filepath.Join(dir, "databricks.yml"), []byte(bundleYml), 0o644))
	require.NoError(t, os.WriteFile(filepath.Join(dir, "package.json"), []byte(packageJson), 0o644))

	registry := detector.NewRegistry()
	detected := registry.Detect(ctx, dir)

	assert.True(t, detected.InProject)
	assert.Equal(t, []string{"apps"}, detected.TargetTypes)
	assert.Equal(t, "appkit-typescript", detected.Template)
}
