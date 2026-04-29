package config

import (
	"fmt"
	"reflect"

	"github.com/databricks/cli/libs/dyn"
	"github.com/databricks/cli/libs/dyn/convert"
)

// GetNodeAndType and returns parent resource node and type of the resource in direct backend.
// Examples:
//
//	"resources.catalogs.foo.name" -> ("resources.catalogs.foo", "catalogs")
//	"resources.catalogs.foo.grants[0].principal -> ("resources.catalogs.foo.grants", "catalogs.grants")
func GetNodeAndType(path dyn.Path) (dyn.Path, string) {
	if len(path) < 3 {
		return nil, ""
	}

	if path[0].Key() != "resources" {
		return nil, ""
	}

	if len(path) >= 4 {
		if path[3].Key() == "permissions" || path[3].Key() == "grants" {
			return path[:4], path[1].Key() + "." + path[3].Key()
		}
	}

	return path[:3], path[1].Key()
}

// GetResourceTypeFromKey extracts the resource group from a resource path.
// For example, "resources.catalogs.foo" returns "catalogs".
// Returns empty string if the path is not in the expected format.
func GetResourceTypeFromKey(path string) string {
	dp, err := dyn.NewPathFromString(path)
	if err != nil {
		return ""
	}
	_, rType := GetNodeAndType(dp)
	return rType
}

// GetResourceConfig returns the configuration object for a given resource path.
// The path should be in the format "resources.group.name" (e.g., "resources.catalogs.foo").
// The returned value is a pointer to the concrete struct that represents that resource type.
// When the path is invalid or resource is not found, the second return value is false.
func (r *Root) GetResourceConfig(path string) (any, error) {
	dynPath, err := dyn.NewPathFromString(path)
	if err != nil {
		return nil, err
	}

	// Extract and validate the resource group from the path
	node, resourceType := GetNodeAndType(dynPath)
	if resourceType == "" {
		return nil, fmt.Errorf("path does not correspond to resource: %q", path)
	}

	// Resolve the Go type that represents a single resource in this group.
	typ, ok := ResourcesTypes[resourceType]
	if !ok {
		return nil, fmt.Errorf("no such resource type in the config: %q", resourceType)
	}

	// Fetch the raw value from the dynamic representation of the bundle config.
	v, err := dyn.GetByPath(r.Value(), node)
	if err != nil {
		if dyn.IsNoSuchKeyError(err) {
			return nil, nil
		}
		return nil, fmt.Errorf("cannot fetch config for %s: %w", node, err)
	}

	typedConfigPtr := reflect.New(typ)

	err = convert.ToTyped(typedConfigPtr.Interface(), v)
	if err != nil {
		return nil, fmt.Errorf("cannot convert config to %s: %w", typ.String(), err)
	}

	return typedConfigPtr.Interface(), nil
}
