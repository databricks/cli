package migrate

import (
	"context"
	"encoding/json"
	"errors"
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

// ETagFor returns the "etag" attribute for a bundle resource, or "" if absent.
// Reads directly from the raw JSON without full path translation.
func (a TFStateAttrs) ETagFor(group, name string) string {
	tfType, ok := terraform.GroupToTerraformName[group]
	if !ok {
		return ""
	}
	raw, ok := a[tfType][name]
	if !ok {
		return ""
	}
	var v struct {
		Etag string `json:"etag,omitempty"`
	}
	if err := json.Unmarshal(raw, &v); err != nil {
		return ""
	}
	return v.Etag
}

// TFState holds everything parsed from a single terraform state file read.
type TFState struct {
	// Attrs maps (tfResourceType → resourceName → raw JSON attributes).
	Attrs TFStateAttrs
	// IDs maps bundle resource key → resource ID.
	IDs map[string]string

	// Lineage and Serial are the top-level state metadata used to seed the direct state.
	Lineage string
	Serial  int
}

// ParseTFStateFull reads the terraform state file once and returns all parsed data.
// Returns nil without error when the file does not exist (first deploy with no resources).
func ParseTFStateFull(ctx context.Context, path string) (*TFState, error) {
	raw, err := os.ReadFile(path)
	if err != nil {
		if errors.Is(err, os.ErrNotExist) {
			return nil, nil
		}
		return nil, err
	}

	// Parse once: lineage/serial live at the top level alongside the resources array,
	// so a single unmarshal captures everything needed for both attrs and IDs.
	var parsed rawTFState
	if err := json.Unmarshal(raw, &parsed); err != nil {
		return nil, err
	}

	attrs := parseTFStateAttrsFromRaw(&parsed)

	exportedIDs, err := terraform.ParseResourcesStateFromBytes(ctx, raw)
	if err != nil {
		return nil, err
	}
	ids := make(map[string]string, len(exportedIDs))
	for k, v := range exportedIDs {
		ids[k] = v.ID
	}
	return &TFState{Attrs: attrs, IDs: ids, Lineage: parsed.Lineage, Serial: parsed.Serial}, nil
}

// rawTFState is the on-disk terraform state format; it captures everything we need in one parse.
type rawTFState struct {
	Version   int    `json:"version"`
	Lineage   string `json:"lineage"`
	Serial    int    `json:"serial"`
	Resources []struct {
		Type      string              `json:"type"`
		Name      string              `json:"name"`
		Mode      tfjson.ResourceMode `json:"mode"`
		Instances []struct {
			Attributes json.RawMessage `json:"attributes"`
		} `json:"instances"`
	} `json:"resources"`
}

func parseTFStateAttrsFromRaw(s *rawTFState) TFStateAttrs {
	result := make(TFStateAttrs)
	for _, r := range s.Resources {
		if r.Mode != tfjson.ManagedResourceMode || len(r.Instances) == 0 {
			continue
		}
		if result[r.Type] == nil {
			result[r.Type] = make(map[string]json.RawMessage)
		}
		result[r.Type][r.Name] = r.Instances[0].Attributes
	}
	return result
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

	// Apply state-only field aliases for single-segment fields whose DABs name differs from TF state name.
	if aliases, ok := tfStateFieldAliases[group]; ok {
		if head, ok := fieldPath.StringKey(); ok {
			if tfName, ok := aliases[head]; ok {
				aliasPath := structpath.NewStringKey(nil, tfName)
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
