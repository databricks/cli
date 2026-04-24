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

	c := &Catalog{Name: "my_catalog", ID: "my_catalog"}
	c.InitializeURL(*base)

	assert.Equal(t, "https://mycompany.databricks.com/explore/data/my_catalog", c.URL)
}

func TestCatalogInitializeURLSkipsWhenIDEmpty(t *testing.T) {
	base, err := url.Parse("https://mycompany.databricks.com")
	require.NoError(t, err)

	// Name is set but ID is not — a declared-but-not-deployed catalog.
	c := &Catalog{Name: "my_catalog"}
	c.InitializeURL(*base)

	assert.Empty(t, c.URL)
}
