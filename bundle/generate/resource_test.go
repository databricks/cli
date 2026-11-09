package generate

import (
	"testing"

	"github.com/databricks/cli/bundle/config/resources"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestConvertResourceToValueDropsEmptyFieldsAndSortsKeys(t *testing.T) {
	c := resources.Catalog{
		StorageRoot: "s3://bucket",
		Name:        "main",
		Comment:     "primary",
	}

	v, err := ConvertResourceToValue(&c)
	require.NoError(t, err)

	m, ok := v.AsMap()
	require.True(t, ok)
	// The YAML encoder orders entries by line, so the lines encode alphabetical order.
	// BaseResource, ID, and Grants are zero and must not appear.
	lines := make(map[string]int)
	for _, p := range m.Pairs() {
		lines[p.Key.MustString()] = p.Value.Location().Line
	}
	assert.Equal(t, map[string]int{"comment": 1, "name": 2, "storage_root": 3}, lines)
}
