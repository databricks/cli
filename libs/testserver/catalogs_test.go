package testserver

import (
	"encoding/json"
	"reflect"
	"strings"
	"testing"

	"github.com/databricks/databricks-sdk-go/service/catalog"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// createCatalogRequest sets every field CreateCatalog accepts. Tests below assert
// the fake echoes all of them back and that the request stays exhaustive.
const createCatalogRequest = `{
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
}`

// jsonFieldNames returns the wire names of the fields typ serializes.
func jsonFieldNames(typ reflect.Type) []string {
	var names []string
	for field := range typ.Fields() {
		name, _, _ := strings.Cut(field.Tag.Get("json"), ",")
		if name == "" || name == "-" {
			continue
		}
		names = append(names, name)
	}
	return names
}

func TestCatalogsCreate_EchoesEveryAcceptedField(t *testing.T) {
	workspace := NewFakeWorkspace("http://test", "dbapi123")

	response := workspace.CatalogsCreate(Request{Body: []byte(createCatalogRequest)})
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

// CatalogsCreate builds its response field by field, so a field the SDK adds to
// CreateCatalog is silently dropped until someone extends the literal. Fail here
// instead, where the message points at the field, rather than in a bundle test
// that reports drift on a catalog it just created.
func TestCatalogsCreate_RequestCoversEveryAcceptedField(t *testing.T) {
	var sent map[string]any
	require.NoError(t, json.Unmarshal([]byte(createCatalogRequest), &sent))

	for _, name := range jsonFieldNames(reflect.TypeFor[catalog.CreateCatalog]()) {
		assert.Contains(t, sent, name)
	}
}

func TestCatalogsUpdate_AppliesUpdatableFields(t *testing.T) {
	workspace := NewFakeWorkspace("http://test", "dbapi123")
	require.Equal(t, 0, workspace.CatalogsCreate(Request{Body: []byte(createCatalogRequest)}).StatusCode)

	// UpdateCatalog also accepts enable_predictive_optimization and isolation_mode,
	// which DABs deliberately leaves unset (see dresources.ResourceCatalog.DoUpdate),
	// so the fake does not model them.
	response := workspace.CatalogsUpdate(Request{Body: []byte(`{
		"comment": "updated",
		"custom_max_retention_hours": 72,
		"managed_encryption_settings": {"customer_managed_key_id": "key-2"},
		"options": {"opt": "3"},
		"properties": {"prop": "4"}
	}`)}, "my_catalog")
	assert.Equal(t, 0, response.StatusCode)

	info, ok := response.Body.(catalog.CatalogInfo)
	require.True(t, ok)

	assert.Equal(t, "updated", info.Comment)
	assert.Equal(t, int64(72), info.CustomMaxRetentionHours)
	require.NotNil(t, info.ManagedEncryptionSettings)
	assert.Equal(t, "key-2", info.ManagedEncryptionSettings.CustomerManagedKeyId)
	assert.Equal(t, map[string]string{"opt": "3"}, info.Options)
	assert.Equal(t, map[string]string{"prop": "4"}, info.Properties)
}
