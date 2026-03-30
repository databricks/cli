package terraform

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"os"
	"path/filepath"
	"strings"

	"github.com/databricks/cli/bundle"
	"github.com/databricks/cli/bundle/statemgmt/resourcestate"
	tfjson "github.com/hashicorp/terraform-json"
)

type (
	ResourceState        = resourcestate.ResourceState
	ExportedResourcesMap = resourcestate.ExportedResourcesMap
)

// Partial representation of the Terraform state file format.
// We are only interested global version and serial numbers,
// plus resource types, names, modes, IDs, and ETags (for dashboards).
type resourcesState struct {
	Version   int             `json:"version"`
	Resources []stateResource `json:"resources"`
}

const SupportedStateVersion = 4

type stateResource struct {
	Type      string                  `json:"type"`
	Name      string                  `json:"name"`
	Mode      tfjson.ResourceMode     `json:"mode"`
	Instances []stateResourceInstance `json:"instances"`
}

type stateResourceInstance struct {
	Attributes stateInstanceAttributes `json:"attributes"`
}

type stateInstanceAttributes struct {
	ID string `json:"id"`

	// Some resources such as Apps do not have an ID, so we use the name instead.
	// We need this for cases when such resource is removed from bundle config but
	// exists in the workspace still so we can correctly display its summary.
	Name string `json:"name,omitempty"`
	ETag string `json:"etag,omitempty"`
}

// Returns a mapping resourceKey -> stateInstanceAttributes
func parseResourcesState(ctx context.Context, path string) (ExportedResourcesMap, error) {
	rawState, err := os.ReadFile(path)
	if err != nil {
		if errors.Is(err, os.ErrNotExist) {
			return nil, nil
		}
		return nil, err
	}
	var state resourcesState
	err = json.Unmarshal(rawState, &state)
	if err != nil {
		return nil, err
	}

	if state.Version != SupportedStateVersion {
		return nil, fmt.Errorf("unsupported deployment state version: %d. Try re-deploying the bundle", state.Version)
	}

	result := make(ExportedResourcesMap)

	for _, resource := range state.Resources {
		if resource.Mode != tfjson.ManagedResourceMode {
			continue
		}
		for _, instance := range resource.Instances {
			var resourceKey string
			var resourceState ResourceState

			if resource.Type == "databricks_secret_acl" {
				// Multiple databricks_secret_acl resources map to a single .permissions entry.
				// The ID is resolved to the scope name in the post-processing step below.
				resourceKey = convertSecretAclResourceNameToKey(resource.Name)
				resourceState = ResourceState{ID: instance.Attributes.ID}
			} else {
				groupName, ok := TerraformToGroupName[resource.Type]
				if !ok {
					continue
				}

				switch groupName {
				case "apps", "secret_scopes", "database_instances", "database_catalogs", "synced_database_tables", "postgres_projects", "postgres_branches", "postgres_endpoints":
					resourceKey = "resources." + groupName + "." + resource.Name
					resourceState = ResourceState{ID: instance.Attributes.Name}
				case "dashboards":
					resourceKey = "resources." + groupName + "." + resource.Name
					resourceState = ResourceState{ID: instance.Attributes.ID, ETag: instance.Attributes.ETag}
				case "permissions":
					resourceKey = convertPermissionsResourceNameToKey(resource.Name)
					resourceState = ResourceState{ID: instance.Attributes.ID}
				case "grants":
					resourceKey = convertGrantsResourceNameToKey(resource.Name)
					resourceState = ResourceState{ID: instance.Attributes.ID}
				default:
					resourceKey = "resources." + groupName + "." + resource.Name
					resourceState = ResourceState{ID: instance.Attributes.ID}
				}
			}

			if resourceKey == "" {
				return nil, fmt.Errorf("cannot calculate resource key for type=%q name=%q id=%q", resource.Type, resource.Name, instance.Attributes.ID)
			}

			result[resourceKey] = resourceState
		}
	}

	// Ensure every secret scope has a .permissions entry. The direct engine manages
	// permissions as a sub-resource (SecretScopeFixups adds MANAGE for the current user).
	// For scopes with databricks_secret_acl in state, resolve the ACL ID to the scope name.
	// For scopes without ACLs, create a .permissions entry with the scope name as ID.
	for key, entry := range result {
		if !strings.HasPrefix(key, "resources.secret_scopes.") || strings.Contains(key, ".permissions") {
			continue
		}
		permKey := key + ".permissions"
		// Use scope name as ID (the direct engine expects it).
		// This overwrites the ACL compound ID if ACLs were present.
		result[permKey] = ResourceState{ID: entry.ID}
	}

	return result, nil
}

// Returns a mapping resourceKey -> stateInstanceAttributes
func ParseResourcesState(ctx context.Context, b *bundle.Bundle) (ExportedResourcesMap, error) {
	cacheDir, err := Dir(ctx, b)
	if err != nil {
		return nil, err
	}
	filename, _ := b.StateFilenameTerraform(ctx)
	return parseResourcesState(ctx, filepath.Join(cacheDir, filename))
}
