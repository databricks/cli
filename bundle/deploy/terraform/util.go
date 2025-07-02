package terraform

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"os"
	"path/filepath"

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

// Returns a mapping group -> name -> stateInstanceAttributes
func ParseResourcesState(ctx context.Context, b *bundle.Bundle) (ExportedResourcesMap, error) {
	cacheDir, err := Dir(ctx, b)
	if err != nil {
		return nil, err
	}
	rawState, err := os.ReadFile(filepath.Join(cacheDir, b.StateFilename()))
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
			groupName, ok := TerraformToGroupName[resource.Type]
			if !ok {
				// permissions, grants, secret_acls
				continue
			}

			group, present := result[groupName]
			if !present {
				group = make(map[string]ResourceState)
				result[groupName] = group
			}

			switch groupName {
			case "apps":
				group[resource.Name] = ResourceState{ID: instance.Attributes.Name}
			case "secret_scopes":
				group[resource.Name] = ResourceState{ID: instance.Attributes.Name}
			case "dashboards":
				group[resource.Name] = ResourceState{ID: instance.Attributes.ID, ETag: instance.Attributes.ETag}
			default:
				group[resource.Name] = ResourceState{ID: instance.Attributes.ID}
			}
		}
	}

	return result, nil
}
