package resources

import (
	"net/url"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestExternalLocationInitializeURL(t *testing.T) {
	base, err := url.Parse("https://mycompany.databricks.com")
	require.NoError(t, err)

	e := &ExternalLocation{Name: "my_loc", Url: "s3://bucket/path"}
	e.InitializeURL(*base)

	assert.Equal(t, "https://mycompany.databricks.com/explore/external-locations/my_loc", e.URL)
	// Url (cloud storage path) is a separate field and must not be touched.
	assert.Equal(t, "s3://bucket/path", e.Url)
}

func TestExternalLocationInitializeURLSkipsWhenNameEmpty(t *testing.T) {
	base, err := url.Parse("https://mycompany.databricks.com")
	require.NoError(t, err)

	e := &ExternalLocation{}
	e.InitializeURL(*base)

	assert.Empty(t, e.URL)
}
