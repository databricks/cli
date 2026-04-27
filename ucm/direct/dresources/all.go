package dresources

import (
	"fmt"

	"github.com/databricks/databricks-sdk-go"
)

// SupportedResources maps a resource type identifier to a typed-nil pointer
// of the resource implementation. UCM-specific resource handlers will be
// added here as each resource is migrated to the embedded-SDK-type pattern
// (mirroring bundle/direct/dresources/all.go).
//
// Sub-resource handlers (".permissions", ".grants") follow the same convention
// as bundle/direct/dresources but are gated on the parent resource being
// migrated first.
var SupportedResources = map[string]any{}

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
