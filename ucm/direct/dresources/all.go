package dresources

import (
	"fmt"

	"github.com/databricks/databricks-sdk-go"
)

// SupportedResources maps a resource type identifier to a typed-nil pointer
// of the resource implementation. Mirrors bundle/direct/dresources/all.go.
// More entries land as each ucm resource is migrated to the
// embedded-SDK-type pattern.
var SupportedResources = map[string]any{
	"catalogs":            (*ResourceCatalog)(nil),
	"schemas":             (*ResourceSchema)(nil),
	"volumes":             (*ResourceVolume)(nil),
	"connections":         (*ResourceConnection)(nil),
	"external_locations":  (*ResourceExternalLocation)(nil),
	"storage_credentials": (*ResourceStorageCredential)(nil),

	// Grants
	"catalogs.grants":            (*ResourceGrants)(nil),
	"schemas.grants":             (*ResourceGrants)(nil),
	"volumes.grants":             (*ResourceGrants)(nil),
	"external_locations.grants":  (*ResourceGrants)(nil),
	"storage_credentials.grants": (*ResourceGrants)(nil),
}

func InitAll(client *databricks.WorkspaceClient) (map[string]*Adapter, error) {
	result := make(map[string]*Adapter)
	for resourceType, resource := range SupportedResources {
		adapter, err := NewAdapter(resource, resourceType, client)
		if err != nil {
			return nil, fmt.Errorf("%s: %w", resourceType, err)
		}
		result[resourceType] = adapter
	}
	return result, nil
}
