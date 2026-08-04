package testserver

import (
	"testing"

	"github.com/databricks/databricks-sdk-go/service/catalog"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestCatalogsCreate_EchoesEveryAcceptedField(t *testing.T) {
	workspace := NewFakeWorkspace("http://test", "dbapi123")

	response := workspace.CatalogsCreate(Request{Body: []byte(`{
		"name": "my_catalog",
		"comment": "c",
		"connection_name": "my_connection",
		"custom_max_retention_hours": 48,
		"managed_encryption_settings": {"customer_managed_key_id": "key-1"},
		"options": {"opt": "1"},
		"properties": {"prop": "2"},
		"provider_name": "my_provider",
		"share_name": "my_share",
		"storage_root": "s3://bucket/root"
	}`)})
	assert.Equal(t, 0, response.StatusCode)

	info, ok := response.Body.(catalog.CatalogInfo)
	require.True(t, ok)

	assert.Equal(t, "my_catalog", info.Name)
	assert.Equal(t, "c", info.Comment)
	assert.Equal(t, "my_connection", info.ConnectionName)
	assert.Equal(t, int64(48), info.CustomMaxRetentionHours)
	require.NotNil(t, info.ManagedEncryptionSettings)
	assert.Equal(t, "key-1", info.ManagedEncryptionSettings.CustomerManagedKeyId)
	assert.Equal(t, map[string]string{"opt": "1"}, info.Options)
	assert.Equal(t, map[string]string{"prop": "2"}, info.Properties)
	assert.Equal(t, "my_provider", info.ProviderName)
	assert.Equal(t, "my_share", info.ShareName)
	assert.Equal(t, "s3://bucket/root", info.StorageRoot)
}
