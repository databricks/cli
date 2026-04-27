package resources

import (
	"github.com/databricks/databricks-sdk-go/service/catalog"
	"net/url"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestSchemaInitializeURL(t *testing.T) {
	base, err := url.Parse("https://mycompany.databricks.com")
	require.NoError(t, err)

	s := &Schema{CreateSchema: catalog.CreateSchema{Name: "my_schema", CatalogName: "my_catalog"}, ID: "my_catalog.my_schema"}
	s.InitializeURL(*base)

	assert.Equal(t, "https://mycompany.databricks.com/explore/data/my_catalog/my_schema", s.URL)
}

func TestSchemaInitializeURLSkipsWhenIDEmpty(t *testing.T) {
	base, err := url.Parse("https://mycompany.databricks.com")
	require.NoError(t, err)

	// Declared but not deployed: URL must not be set.
	s := &Schema{CreateSchema: catalog.CreateSchema{Name: "my_schema", CatalogName: "my_catalog"}}
	s.InitializeURL(*base)

	assert.Empty(t, s.URL)
}
