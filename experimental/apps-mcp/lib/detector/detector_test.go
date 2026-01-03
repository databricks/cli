package detector_test

import (
	"context"
	"os"
	"path/filepath"
	"slices"
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

func TestDetectorRegistry_EmptyBundle(t *testing.T) {
	dir := t.TempDir()
	ctx := context.Background()

	bundleYml := `bundle:
  name: empty-project
`
	require.NoError(t, os.WriteFile(filepath.Join(dir, "databricks.yml"), []byte(bundleYml), 0o644))

	registry := detector.NewRegistry()
	detected := registry.Detect(ctx, dir)

	assert.True(t, detected.InProject)
	assert.Equal(t, []string{"bundle"}, detected.TargetTypes)
	assert.Equal(t, "empty-project", detected.BundleInfo.Name)
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
	assert.Equal(t, []string{"jobs", "bundle"}, detected.TargetTypes)
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

func TestDetectorRegistry_AppsWithOtherResources(t *testing.T) {
	testCases := []struct {
		name          string
		bundleYml     string
		expectBundle  bool
		expectAppOnly bool
	}{
		{
			name: "app_only",
			bundleYml: `bundle:
  name: test
resources:
  apps:
    my_app: {}
`,
			expectBundle:  false,
			expectAppOnly: true,
		},
		{
			name: "apps_with_jobs",
			bundleYml: `bundle:
  name: test
resources:
  apps:
    my_app: {}
  jobs:
    my_job: {}
`,
			expectBundle:  true,
			expectAppOnly: false,
		},
		{
			name: "apps_with_pipelines",
			bundleYml: `bundle:
  name: test
resources:
  apps:
    my_app: {}
  pipelines:
    my_pipeline: {}
`,
			expectBundle:  true,
			expectAppOnly: false,
		},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			dir := t.TempDir()
			ctx := context.Background()

			require.NoError(t, os.WriteFile(filepath.Join(dir, "databricks.yml"), []byte(tc.bundleYml), 0o644))

			registry := detector.NewRegistry()
			detected := registry.Detect(ctx, dir)

			assert.True(t, detected.InProject)
			assert.Contains(t, detected.TargetTypes, "apps")

			if tc.expectBundle {
				assert.Contains(t, detected.TargetTypes, "bundle", "should include 'bundle' for apps + other resources")
			} else {
				assert.NotContains(t, detected.TargetTypes, "bundle", "should not include 'bundle' for app-only")
			}

			isAppOnly := slices.Contains(detected.TargetTypes, "apps") && len(detected.TargetTypes) == 1
			assert.Equal(t, tc.expectAppOnly, isAppOnly)
		})
	}
}
