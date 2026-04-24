package resources

import (
	"net/url"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestVolumeInitializeURL(t *testing.T) {
	base, err := url.Parse("https://mycompany.databricks.com")
	require.NoError(t, err)

	v := &Volume{Name: "my_volume", CatalogName: "my_catalog", SchemaName: "my_schema"}
	v.InitializeURL(*base)

	assert.Equal(t, "https://mycompany.databricks.com/explore/data/my_catalog/my_schema/my_volume", v.URL)
}

func TestVolumeInitializeURLSkipsWhenParentMissing(t *testing.T) {
	base, err := url.Parse("https://mycompany.databricks.com")
	require.NoError(t, err)

	v := &Volume{Name: "my_volume"}
	v.InitializeURL(*base)

	assert.Empty(t, v.URL)
}
