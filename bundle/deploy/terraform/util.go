package terraform

import (
	"context"
	"encoding/json"
	"errors"
	"io"
	"os"
	"path/filepath"

	"github.com/databricks/cli/bundle"
	tfjson "github.com/hashicorp/terraform-json"
)

// Partial representation of the Terraform state file format.
// We are only interested global version and serial numbers,
// plus resource types, names, modes, and ids.
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
}

func loadState(input io.Reader) (*tfState, error) {
	content, err := io.ReadAll(input)
	if err != nil {
		return nil, err
	}
	var s tfState
	err = json.Unmarshal(content, &s)
	if err != nil {
		return nil, err
	}

	return &s, nil
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
