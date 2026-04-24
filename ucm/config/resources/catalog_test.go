package resources

import (
	"net/url"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestCatalogInitializeURL(t *testing.T) {
	base, err := url.Parse("https://mycompany.databricks.com")
	require.NoError(t, err)

	c := &Catalog{Name: "my_catalog"}
	c.InitializeURL(*base)

	assert.Equal(t, "https://mycompany.databricks.com/explore/data/my_catalog", c.URL)
}

func TestCatalogInitializeURLSkipsWhenNameEmpty(t *testing.T) {
	base, err := url.Parse("https://mycompany.databricks.com")
	require.NoError(t, err)

	c := &Catalog{}
	c.InitializeURL(*base)

	assert.Empty(t, c.URL)
}
