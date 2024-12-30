package terraform

import (
	"context"
	"encoding/json"
	"errors"
	"os"
	"path/filepath"

	"github.com/databricks/cli/bundle"
	tfjson "github.com/hashicorp/terraform-json"
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

func ParseResourcesState(ctx context.Context, b *bundle.Bundle) (*resourcesState, error) {
	cacheDir, err := Dir(ctx, b)
	if err != nil {
		return nil, err
	}
	rawState, err := os.ReadFile(filepath.Join(cacheDir, TerraformStateFileName))
	if err != nil {
		if errors.Is(err, os.ErrNotExist) {
			return &resourcesState{Version: SupportedStateVersion}, nil
		}
		return nil, err
	}
	var state resourcesState
	err = json.Unmarshal(rawState, &state)
	return &state, err
}
