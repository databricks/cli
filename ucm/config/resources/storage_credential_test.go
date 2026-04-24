package resources

import (
	"net/url"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestStorageCredentialInitializeURL(t *testing.T) {
	base, err := url.Parse("https://mycompany.databricks.com")
	require.NoError(t, err)

	s := &StorageCredential{Name: "my_cred", ID: "my_cred"}
	s.InitializeURL(*base)

	assert.Equal(t, "https://mycompany.databricks.com/explore/storage-credentials/my_cred", s.URL)
}

func TestStorageCredentialInitializeURLSkipsWhenIDEmpty(t *testing.T) {
	base, err := url.Parse("https://mycompany.databricks.com")
	require.NoError(t, err)

	s := &StorageCredential{Name: "my_cred"}
	s.InitializeURL(*base)

	assert.Empty(t, s.URL)
}
