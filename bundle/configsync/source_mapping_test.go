package configsync

import (
	"os"
	"path/filepath"
	"testing"

	"github.com/databricks/cli/bundle"
	"github.com/databricks/cli/bundle/config/mutator"
	"github.com/databricks/cli/libs/dyn"
	"github.com/databricks/cli/libs/logdiag"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestSourceMappingReadsFilesWhenResolved(t *testing.T) {
	ctx := logdiag.InitContext(t.Context())
	directory := t.TempDir()
	path := filepath.Join(directory, "databricks.yml")
	loaded := "bundle:\n  name: loaded\n"
	current := "bundle:\n  name: current\n"
	require.NoError(t, os.WriteFile(path, []byte(loaded), 0o644))

	b, err := bundle.Load(ctx, directory)
	require.NoError(t, err)
	mutator.DefaultMutators(ctx, b)
	require.False(t, logdiag.HasError(ctx))
	require.NoError(t, os.WriteFile(path, []byte(current), 0o644))

	sources, err := loadSourceIndex(b)
	require.NoError(t, err)
	name, err := dyn.GetByPath(sources.files[path], dyn.NewPath(dyn.Key("bundle"), dyn.Key("name")))
	require.NoError(t, err)
	assert.Equal(t, "current", name.MustString())
}
