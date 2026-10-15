package testserver

import (
	"encoding/json"
	"reflect"
	"strings"
	"testing"

	catalogs "github.com/databricks/cli/libs/sdktypes/uc/catalogs/v1"
	"github.com/databricks/cli/libs/utils"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestCatalogsCreate_RejectsEmptyName(t *testing.T) {
	workspace := NewFakeWorkspace("http://test", "dbapi123")

	response := workspace.CatalogsCreate(Request{Body: []byte(`{"name": ""}`)})
	assert.Equal(t, 400, response.StatusCode)

	// A stored-but-unreadable catalog under the empty key is the original bug. Asserted
	// before the require below so it is still reported when the rejection is missing.
	assert.Empty(t, workspace.Catalogs)

	body, ok := response.Body.(map[string]string)
	require.True(t, ok)
	assert.Equal(t, "INVALID_PARAMETER_VALUE", body["error_code"])
	assert.Contains(t, body["message"], "is not a valid name")
}

func TestCatalogsCreate_AllowsNonEmptyName(t *testing.T) {
	workspace := NewFakeWorkspace("http://test", "dbapi123")

	response := workspace.CatalogsCreate(Request{Body: []byte(`{"name": "my_catalog"}`)})
	// StatusCode 0 gets converted to 200 by normalizeResponse in the server
	require.Equal(t, 0, response.StatusCode)

	// Read back through the same helper the GET route uses: a mis-keyed store must fail here.
	getResponse := MapGet(workspace, workspace.Catalogs, "my_catalog")
	require.Equal(t, 0, getResponse.StatusCode)

	body, ok := getResponse.Body.(catalogs.CatalogInfo)
	require.True(t, ok)
	assert.Equal(t, "my_catalog", utils.Deref(body.Name))
	assert.Equal(t, "my_catalog", utils.Deref(body.FullName))
}

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

	info, ok := response.Body.(catalogs.CatalogInfo)
	require.True(t, ok)

	assert.Equal(t, "my_catalog", utils.Deref(info.Name))
	assert.Equal(t, "c", utils.Deref(info.Comment))
	assert.Equal(t, "my_connection", utils.Deref(info.ConnectionName))
	assert.Equal(t, int64(48), utils.Deref(info.CustomMaxRetentionHours))
	require.NotNil(t, info.ManagedEncryptionSettings)
	assert.Equal(t, "key-1", utils.Deref(info.ManagedEncryptionSettings.CustomerManagedKeyId))
	assert.Equal(t, map[string]string{"opt": "1"}, info.Options)
	assert.Equal(t, map[string]string{"prop": "2"}, info.Properties)
	assert.Equal(t, "my_provider", utils.Deref(info.ProviderName))
	assert.Equal(t, "my_share", utils.Deref(info.ShareName))
	assert.Equal(t, "s3://bucket/root", utils.Deref(info.StorageRoot))
}

// catalogCreateServerFields are CreateCatalogRequest fields the server computes
// or DABs never sends: sdk-go models the create request as a full CatalogInfo.
var catalogCreateServerFields = map[string]bool{
	"owner":                                  true,
	"enable_predictive_optimization":         true,
	"catalog_type":                           true,
	"metastore_id":                           true,
	"created_at":                             true,
	"created_by":                             true,
	"updated_at":                             true,
	"updated_by":                             true,
	"storage_location":                       true,
	"isolation_mode":                         true,
	"effective_predictive_optimization_flag": true,
	"browse_only":                            true,
	"provisioning_info":                      true,
	"full_name":                              true,
	"securable_type":                         true,
}

// CatalogsCreate builds its response field by field, so a field the SDK adds to
// CreateCatalog is silently dropped until someone extends the literal. Fail here
// instead, where the message points at the field, rather than in a bundle test
// that reports drift on a catalog it just created.
func TestCatalogsCreate_RequestCoversEveryAcceptedField(t *testing.T) {
	var sent map[string]any
	require.NoError(t, json.Unmarshal([]byte(createCatalogRequest), &sent))

	for _, name := range jsonFieldNames(reflect.TypeFor[catalogs.CreateCatalogRequest]()) {
		if catalogCreateServerFields[name] {
			continue
		}
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

	info, ok := response.Body.(catalogs.CatalogInfo)
	require.True(t, ok)

	assert.Equal(t, "updated", utils.Deref(info.Comment))
	assert.Equal(t, int64(72), utils.Deref(info.CustomMaxRetentionHours))
	require.NotNil(t, info.ManagedEncryptionSettings)
	assert.Equal(t, "key-2", utils.Deref(info.ManagedEncryptionSettings.CustomerManagedKeyId))
	assert.Equal(t, map[string]string{"opt": "3"}, info.Options)
	assert.Equal(t, map[string]string{"prop": "4"}, info.Properties)
}
