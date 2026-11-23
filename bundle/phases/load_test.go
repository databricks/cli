package phases

import (
	"os"
	"path/filepath"
	"slices"
	"testing"

	"github.com/databricks/cli/bundle"
	"github.com/databricks/cli/bundle/bitmap"
	"github.com/databricks/cli/libs/logdiag"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// Deploy telemetry is logged after a failed load too. The bitmap must not
// describe a configuration that was never selected.
func TestLoadNamedTargetMissingTargetHasNoBitmap(t *testing.T) {
	dir := t.TempDir()
	config := "bundle: {name: x}\ntargets: {dev: {default: true}}\n"
	require.NoError(t, os.WriteFile(filepath.Join(dir, "databricks.yml"), []byte(config), 0o644))

	ctx := logdiag.InitContext(t.Context())
	logdiag.SetCollect(ctx, true)
	b, err := bundle.Load(ctx, dir)
	require.NoError(t, err)

	LoadNamedTarget(ctx, b, "missing")
	assert.True(t, logdiag.HasError(ctx))
	assert.Nil(t, b.Bitmap)

	ctx = logdiag.InitContext(t.Context())
	logdiag.SetCollect(ctx, true)
	b, err = bundle.Load(ctx, dir)
	require.NoError(t, err)

	LoadNamedTarget(ctx, b, "dev")
	assert.False(t, logdiag.HasError(ctx), "%v", logdiag.FlushCollected(ctx))
	assert.NotNil(t, b.Bitmap)
}

// loadBits loads databricks.yml with the given target and returns the schema
// entries of the bits the load phase set.
func loadBits(t *testing.T, config, target string) []string {
	dir := t.TempDir()
	require.NoError(t, os.WriteFile(filepath.Join(dir, "databricks.yml"), []byte(config), 0o644))
	ctx := logdiag.InitContext(t.Context())
	logdiag.SetCollect(ctx, true)
	b, err := bundle.Load(ctx, dir)
	require.NoError(t, err)
	LoadNamedTarget(ctx, b, target)
	require.False(t, logdiag.HasError(ctx), "%v", logdiag.FlushCollected(ctx))
	require.NotNil(t, b.Bitmap)
	bits, err := b.Bitmap.Bits(struct{}{})
	require.NoError(t, err)
	schema := bitmap.EmbeddedSchema()
	var set []string
	for i, on := range bits {
		if on {
			set = append(set, schema[i])
		}
	}
	return set
}

// A target that overrides a task with the same task_key ends up with one task,
// and the bitmap counts that task's value, not the base task's too.
func TestLoadMarksTargetOverrideOfKeyedListOnce(t *testing.T) {
	got := loadBits(t, `
bundle: {name: x}
resources:
  jobs:
    j:
      tasks:
        - task_key: a
          disable_auto_optimization: true
targets:
  dev:
    resources:
      jobs:
        j:
          tasks:
            - task_key: a
              disable_auto_optimization: false
`, "dev")
	field := "resources.jobs.*.tasks[*].disable_auto_optimization"
	assert.Contains(t, got, field+"=false")
	assert.NotContains(t, got, field+"=true")
}

func TestLoadMarksDeprecatedComputeId(t *testing.T) {
	const config = `
bundle: {name: x}
targets:
  dev:
    compute_id: c1
  prod: {}
`
	assert.Contains(t, loadBits(t, config, "dev"), "bundle.compute_id")
	// Only the selected target counts.
	assert.NotContains(t, loadBits(t, config, "prod"), "bundle.compute_id")

	root := loadBits(t, "bundle: {name: x, compute_id: c1}\n", "default")
	assert.Contains(t, root, "bundle.compute_id")
}

// A glob that matches no file leaves no include list after loading; the user
// still wrote one.
func TestLoadMarksIncludeThatMatchesNothing(t *testing.T) {
	got := loadBits(t, "bundle: {name: x}\ninclude: [\"resources/*.yml\"]\n", "default")
	assert.True(t, slices.Contains(got, "include") && slices.Contains(got, "include[*]"), "%v", got)
}
