package phases

import (
	"os"
	"path/filepath"
	"testing"

	"github.com/databricks/cli/bundle"
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
