package dynloc

import (
	"bytes"
	"os"
	"path/filepath"
	"testing"

	"github.com/databricks/cli/libs/dyn"
	assert "github.com/databricks/cli/libs/dyn/dynassert"
	"github.com/databricks/cli/libs/dyn/merge"
	"github.com/databricks/cli/libs/dyn/yamlloader"
	"github.com/stretchr/testify/require"
)

func load(t *testing.T, path string) dyn.Value {
	matches, err := filepath.Glob(path + "/*.yml")
	require.NoError(t, err)
	require.NotEmpty(t, matches)

	// Load all files.
	vout := dyn.NilValue
	for _, match := range matches {
		buf, err := os.ReadFile(match)
		require.NoError(t, err)

		v, err := yamlloader.LoadYAML(match, bytes.NewBuffer(buf))
		require.NoError(t, err)

		vout, err = merge.Merge(vout, v)
		require.NoError(t, err)
	}

	return vout
}

func TestLocations_Default(t *testing.T) {
	v := load(t, "testdata/default")
	locs, err := Build(v)
	require.NoError(t, err)
	assert.Equal(t, 1, locs.Version)
	assert.Equal(t, []string{"testdata/default/a.yml", "testdata/default/b.yml"}, locs.Files)
	assert.Equal(t, map[string][][]int{
		"a":   {{0, 2, 3}},
		"a.b": {{0, 2, 6}},
		"b":   {{1, 2, 3}},
		"b.c": {{1, 2, 6}},
	}, locs.Locations)
}

func TestLocations_DefaultWithBasePath(t *testing.T) {
	v := load(t, "testdata/default")
	locs, err := Build(v, WithBasePath("testdata/default"))
	require.NoError(t, err)
	assert.Equal(t, 1, locs.Version)
	assert.Equal(t, []string{"a.yml", "b.yml"}, locs.Files)
	assert.Equal(t, map[string][][]int{
		"a":   {{0, 2, 3}},
		"a.b": {{0, 2, 6}},
		"b":   {{1, 2, 3}},
		"b.c": {{1, 2, 6}},
	}, locs.Locations)
}

func TestLocations_Override(t *testing.T) {
	v := load(t, "testdata/override")
	locs, err := Build(v)
	require.NoError(t, err)
	assert.Equal(t, 1, locs.Version)
	assert.Equal(t, []string{"testdata/override/a.yml", "testdata/override/b.yml"}, locs.Files)

	// Note: specific ordering of locations is described in [merge.Merge].
	assert.Equal(t, map[string][][]int{
		"a": {
			{0, 2, 3},
			{1, 2, 3},
		},
		"a.b": {
			{1, 2, 6},
			{0, 2, 6},
		},
	}, locs.Locations)
}

func TestLocations_MaxDepth(t *testing.T) {
	v := load(t, "testdata/depth")

	var locs Locations
	var err error

	// Test with no max depth.
	locs, err = Build(v)
	require.NoError(t, err)
	assert.Len(t, locs.Locations, 5)

	// Test with max depth and see that the number of locations is reduced.
	locs, err = Build(v, WithMaxDepth(3))
	require.NoError(t, err)
	assert.Len(t, locs.Locations, 3)
}
