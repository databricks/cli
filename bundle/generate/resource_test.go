package generate

import (
	"testing"

	"github.com/databricks/cli/bundle/config/resources"
	"github.com/databricks/cli/libs/dyn"
	"github.com/databricks/databricks-sdk-go/service/catalog"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestFromResponseKeepsOnlyConfigFields(t *testing.T) {
	info := catalog.CatalogInfo{
		Name:        "main",
		Comment:     "primary",
		Owner:       "someone@example.com",
		CatalogType: catalog.CatalogTypeManagedCatalog,
		CreatedAt:   1234,
		Properties:  map[string]string{"k": "v"},
	}

	c, err := FromResponse[resources.Catalog](info)
	require.NoError(t, err)

	assert.Equal(t, "main", c.Name)
	assert.Equal(t, "primary", c.Comment)
	assert.Equal(t, map[string]string{"k": "v"}, c.Properties)
	// Owner is not part of CreateCatalog; CatalogType and CreatedAt are read-only.
	assert.Empty(t, c.ID)
}

func TestConvertResourceToValueDropsManagedAndSkippedFields(t *testing.T) {
	c := resources.Catalog{
		ID: "abc",
		CreateCatalog: catalog.CreateCatalog{
			Name:        "main",
			Comment:     "primary",
			StorageRoot: "s3://bucket",
		},
	}
	c.URL = "https://example.com/catalogs/abc"

	v, err := ConvertResourceToValue(c, "storage_root")
	require.NoError(t, err)

	m, ok := v.AsMap()
	require.True(t, ok)
	assert.ElementsMatch(t, []string{"name", "comment"}, keys(m))
}

func TestConvertResourceToValueOrdersKeysAlphabetically(t *testing.T) {
	c := resources.Catalog{
		CreateCatalog: catalog.CreateCatalog{
			StorageRoot: "s3://bucket",
			Name:        "main",
			Comment:     "primary",
		},
	}

	v, err := ConvertResourceToValue(c)
	require.NoError(t, err)

	m, ok := v.AsMap()
	require.True(t, ok)
	var got []string
	var lines []int
	for _, p := range m.Pairs() {
		got = append(got, p.Key.MustString())
		lines = append(lines, p.Value.Location().Line)
	}
	assert.Equal(t, []string{"comment", "name", "storage_root"}, got)
	assert.Equal(t, []int{1, 2, 3}, lines)
}

func keys(m dyn.Mapping) []string {
	var out []string
	for _, k := range m.Keys() {
		out = append(out, k.MustString())
	}
	return out
}
