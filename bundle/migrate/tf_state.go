package migrate

import (
	"context"
	"encoding/json"
	"fmt"
	"os"

	"github.com/databricks/cli/bundle/deploy/terraform"
	"github.com/databricks/cli/bundle/terraform_dabs_map"
	"github.com/databricks/cli/libs/structs/structpath"
	tfjson "github.com/hashicorp/terraform-json"
)

// tfStateFieldAliases maps DABs group → DABs field name → TF state field name for
// cases where a DABs state-computed field has a different name in TF state.
// These fields are not captured by DABsToTerraformRenameMap because they are
// state-only (not part of the bundle config struct).
var tfStateFieldAliases = map[string]map[string]string{
	// models.model_id is the numeric model ID; TF stores it as registered_model_id.
	"models": {"model_id": "registered_model_id"},
}

// TFStateAttrs maps (tfResourceType → resourceName → raw JSON attributes).
type TFStateAttrs map[string]map[string]json.RawMessage

// TFStateMeta holds the top-level metadata from a terraform state file.
type TFStateMeta struct {
	Lineage string
	Serial  int
}

// ParseTFStateFull reads the terraform state file once and returns the full
// attribute map, the resource ID map, and the state metadata (lineage/serial).
// Avoids reading and unmarshaling the file multiple times.
func ParseTFStateFull(ctx context.Context, path string) (TFStateAttrs, terraform.ExportedResourcesMap, TFStateMeta, error) {
	raw, err := os.ReadFile(path)
	if err != nil {
		return nil, nil, TFStateMeta{}, err
	}

	var meta struct {
		Lineage string `json:"lineage"`
		Serial  int    `json:"serial"`
	}
	if err := json.Unmarshal(raw, &meta); err != nil {
		return nil, nil, TFStateMeta{}, err
	}

	attrs, err := parseTFStateAttrsFromBytes(raw)
	if err != nil {
		return nil, nil, TFStateMeta{}, err
	}
	ids, err := terraform.ParseResourcesStateFromBytes(ctx, raw)
	if err != nil {
		return nil, nil, TFStateMeta{}, err
	}
	return attrs, ids, TFStateMeta{Lineage: meta.Lineage, Serial: meta.Serial}, nil
}

// ParseTFStateAttrs parses the terraform state file returning full attribute JSON per resource.
func ParseTFStateAttrs(path string) (TFStateAttrs, error) {
	raw, err := os.ReadFile(path)
	if err != nil {
		return nil, err
	}
	return parseTFStateAttrsFromBytes(raw)
}

func parseTFStateAttrsFromBytes(raw []byte) (TFStateAttrs, error) {
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

	value, err := navigateTFState(attrs, tfFieldPath)
	if err == nil {
		return value, nil
	}

	// Some DABs fields are top-level in TF state but DABsPathToTerraform added a
	// wrapper prefix (e.g. "spec" for postgres resources). When the wrapped path
	// fails, retry with the original unwrapped path.
	if _, hasWrapper := terraform_dabs_map.DABsToTerraformWrappers[group]; hasWrapper {
		if v, e := navigateTFState(attrs, fieldPath); e == nil {
			return v, nil
		}
	}

	// Apply state-only field aliases for fields whose DABs name differs from TF state name.
	if aliases, ok := tfStateFieldAliases[group]; ok {
		// Replace the first path segment if it matches a known alias.
		if head, ok := fieldPath.StringKey(); ok {
			if tfName, ok := aliases[head]; ok {
				aliasPath := structpath.NewStringKey(nil, tfName)
				if rest := fieldPath.SkipPrefix(1); rest != nil {
					_ = rest // navigate through the alias root
				}
				// Translate aliased path with full DABsToTerraform for the renamed field.
				if aliasFieldPath, e := terraform_dabs_map.DABsPathToTerraform(group, aliasPath); e == nil {
					if v, e := navigateTFState(attrs, aliasFieldPath); e == nil {
						return v, nil
					}
				}
			}
		}
	}

	return nil, err
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
