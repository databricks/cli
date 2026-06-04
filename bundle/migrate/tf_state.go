package migrate

import (
	"encoding/json"
	"fmt"
	"os"

	"github.com/databricks/cli/bundle/deploy/terraform"
	"github.com/databricks/cli/bundle/terraform_dabs_map"
	"github.com/databricks/cli/libs/structs/structpath"
	tfjson "github.com/hashicorp/terraform-json"
)

// TFStateAttrs maps (tfResourceType → resourceName → raw JSON attributes).
type TFStateAttrs map[string]map[string]json.RawMessage

// ParseTFStateAttrs parses the terraform state file returning full attribute JSON per resource.
func ParseTFStateAttrs(path string) (TFStateAttrs, error) {
	raw, err := os.ReadFile(path)
	if err != nil {
		return nil, err
	}

	var state struct {
		Version   int `json:"version"`
		Resources []struct {
			Type      string              `json:"type"`
			Name      string              `json:"name"`
			Mode      tfjson.ResourceMode `json:"mode"`
			Instances []struct {
				Attributes json.RawMessage `json:"attributes"`
			} `json:"instances"`
		} `json:"resources"`
	}

	if err := json.Unmarshal(raw, &state); err != nil {
		return nil, err
	}

	result := make(TFStateAttrs)
	for _, r := range state.Resources {
		if r.Mode != tfjson.ManagedResourceMode || len(r.Instances) == 0 {
			continue
		}
		if result[r.Type] == nil {
			result[r.Type] = make(map[string]json.RawMessage)
		}
		result[r.Type][r.Name] = r.Instances[0].Attributes
	}
	return result, nil
}

// LookupTFField looks up a field from TF state attributes for a bundle resource.
// group is the DABs group (e.g. "pipelines"), name is the resource name.
// fieldPath is the path to the field (may be in DABs or TF naming; both handled by DABsPathToTerraform).
func LookupTFField(state TFStateAttrs, group, name string, fieldPath *structpath.PathNode) (any, error) {
	tfType, ok := terraform.GroupToTerraformName[group]
	if !ok {
		return nil, fmt.Errorf("unknown resource group %q", group)
	}

	// Translate field path to TF naming.
	// DABsPathToTerraform handles both DABs names (renames) and TF names (pass-through for unknowns).
	// Returns error for known DABs-only fields that have no TF equivalent.
	tfFieldPath, err := terraform_dabs_map.DABsPathToTerraform(group, fieldPath)
	if err != nil {
		return nil, err
	}

	attrsJSON, ok := state[tfType][name]
	if !ok {
		return nil, fmt.Errorf("%s.%s not found in TF state", tfType, name)
	}

	// Unmarshal into map[string]any to handle TF list-blocks: in TF state, single-block
	// fields are stored as single-element arrays [{"field": "value"}], not as plain objects.
	// Navigating via map avoids the json.Unmarshal type mismatch between []T in JSON and
	// struct-typed schema fields.
	var attrs map[string]any
	if err := json.Unmarshal(attrsJSON, &attrs); err != nil {
		return nil, fmt.Errorf("cannot parse TF state for %s.%s: %w", tfType, name, err)
	}

	return navigateTFState(attrs, tfFieldPath)
}

// navigateTFState walks the TF state map using the given path.
// TF stores single-block fields as single-element arrays ([{…}]).  When a string-key
// step encounters a []any, it auto-descends into element [0] so callers can use plain
// paths like "continuous.pause_status" even though TF stores them as [{"pause_status":…}].
func navigateTFState(data map[string]any, path *structpath.PathNode) (any, error) {
	var current any = data
	for _, node := range path.AsSlice() {
		if current == nil {
			return nil, nil
		}

		if key, ok := node.StringKey(); ok {
			// Auto-unwrap TF list-blocks: if the current value is a single-element
			// array and the next step wants a map key, descend into element 0.
			if arr, isArr := current.([]any); isArr {
				if len(arr) == 0 {
					return nil, nil
				}
				current = arr[0]
			}
			m, ok := current.(map[string]any)
			if !ok {
				return nil, fmt.Errorf("expected map at %q, got %T", key, current)
			}
			val, ok := m[key]
			if !ok {
				return nil, fmt.Errorf("%q: key not found", key)
			}
			current = val
		} else if idx, ok := node.Index(); ok {
			switch v := current.(type) {
			case []any:
				if idx < 0 || idx >= len(v) {
					return nil, fmt.Errorf("index %d out of range (len %d)", idx, len(v))
				}
				current = v[idx]
			default:
				// TF [0] on a non-slice (already unwrapped) is a no-op.
				if idx == 0 {
					continue
				}
				return nil, fmt.Errorf("index %d: not a slice (%T)", idx, current)
			}
		}
	}
	return current, nil
}
