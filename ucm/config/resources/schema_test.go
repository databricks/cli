package resources

import (
	"net/url"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestSchemaInitializeURL(t *testing.T) {
	base, err := url.Parse("https://mycompany.databricks.com")
	require.NoError(t, err)

	s := &Schema{Name: "my_schema", Catalog: "my_catalog"}
	s.InitializeURL(*base)

	assert.Equal(t, "https://mycompany.databricks.com/explore/data/my_catalog/my_schema", s.URL)
}

func TestSchemaInitializeURLSkipsWhenCatalogEmpty(t *testing.T) {
	base, err := url.Parse("https://mycompany.databricks.com")
	require.NoError(t, err)

	s := &Schema{Name: "my_schema"}
	s.InitializeURL(*base)

	assert.Empty(t, s.URL)
}
