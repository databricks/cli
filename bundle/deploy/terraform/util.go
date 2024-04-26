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
	Version   *int            `json:"version"`
	Resources []stateResource `json:"resources"`
}

type serialState struct {
	Serial int `json:"serial"`
}

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

func IsLocalStateStale(local io.Reader, remote io.Reader) bool {
	localState, err := loadState(local)
	if err != nil {
		return true
	}

	remoteState, err := loadState(remote)
	if err != nil {
		return false
	}

	return localState.Serial < remoteState.Serial
}

func loadState(input io.Reader) (*serialState, error) {
	content, err := io.ReadAll(input)
	if err != nil {
		return nil, err
	}
	var s serialState
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
	rawState, stateFileErr := os.ReadFile(filepath.Join(cacheDir, TerraformStateFileName))
	if errors.Is(stateFileErr, os.ErrNotExist) {
		return &resourcesState{}, nil
	} else if stateFileErr != nil {
		return nil, stateFileErr
	}
	var state resourcesState
	err = json.Unmarshal(rawState, &state)
	return &state, err
}
