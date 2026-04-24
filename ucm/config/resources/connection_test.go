package resources

import (
	"net/url"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestConnectionInitializeURL(t *testing.T) {
	base, err := url.Parse("https://mycompany.databricks.com")
	require.NoError(t, err)

	c := &Connection{Name: "my_conn", ConnectionType: "POSTGRESQL"}
	c.InitializeURL(*base)

	assert.Equal(t, "https://mycompany.databricks.com/explore/connections/my_conn", c.URL)
}

func TestConnectionInitializeURLSkipsWhenNameEmpty(t *testing.T) {
	base, err := url.Parse("https://mycompany.databricks.com")
	require.NoError(t, err)

	c := &Connection{}
	c.InitializeURL(*base)

	assert.Empty(t, c.URL)
}
