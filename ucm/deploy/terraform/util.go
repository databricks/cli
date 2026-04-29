package terraform

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"os"
	"path/filepath"

	"github.com/databricks/cli/libs/log"
	"github.com/databricks/cli/ucm"
	"github.com/databricks/cli/ucm/statemgmt/resourcestate"
	tfjson "github.com/hashicorp/terraform-json"
)

type (
	ResourceState        = resourcestate.ResourceState
	ExportedResourcesMap = resourcestate.ExportedResourcesMap
)

// Partial representation of the Terraform state file format. Mirrors the
// shape used by bundle/deploy/terraform's parser - only the global version,
// resource type/name/mode and per-instance ID/Name/ETag are needed to
// reconstruct the resource-key map consumed by direct-engine state.
type resourcesState struct {
	Version   int             `json:"version"`
	Resources []stateResource `json:"resources"`
}

// SupportedStateVersion is the on-disk terraform state schema version this
// parser understands. Mirrors bundle's constant of the same name; bumped in
// lockstep when terraform changes its tfstate format.
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

	// Some resources (e.g. apps) do not surface an ID in their tfstate
	// attributes, so the resource name is used as the canonical handle when
	// later reconciling against the workspace.
	Name string `json:"name,omitempty"`
	ETag string `json:"etag,omitempty"`
}

// parseResourcesState reads a terraform.tfstate JSON blob from disk and
// returns a mapping of ucm resource keys (`resources.<group>.<name>`) to
// the IDs that direct-engine state needs to take ownership of those
// resources without re-creating them.
//
// A missing file is not an error - it returns (nil, nil) so callers can
// treat "no prior state" the same as "no resources tracked".
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
			groupName, ok := terraformToGroupName[resource.Type]
			if !ok {
				log.Warnf(ctx, "Unknown Terraform resource type: %s", resource.Type)
				continue
			}

			// ucm groups are flat (`resources.<group>.<name>`); sub-resource
			// conventions like bundle's `secret_acls` / `permissions` /
			// `dashboards` ETag don't apply to ucm's M0 resource set.
			// Should ucm grow those resource types, mirror the bundle
			// switch in bundle/deploy/terraform/util.go.
			resourceKey := "resources." + groupName + "." + resource.Name
			result[resourceKey] = ResourceState{ID: instance.Attributes.ID}
		}
	}

	return result, nil
}

// ParseResourcesState reads the local terraform.tfstate for the current ucm
// target and returns the resource-key -> ID map. Used by the migrate verb
// to seed direct-engine state from a terraform-managed deployment.
//
// Errors if no target has been selected; returns (nil, nil) when there is
// no local tfstate (e.g. before the first deploy).
func ParseResourcesState(ctx context.Context, u *ucm.Ucm) (ExportedResourcesMap, error) {
	workingDir, err := WorkingDir(u)
	if err != nil {
		return nil, err
	}
	return parseResourcesState(ctx, filepath.Join(workingDir, "terraform.tfstate"))
}
