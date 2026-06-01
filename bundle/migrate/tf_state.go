package migrate

import (
	"encoding/json"
	"fmt"
	"os"
	"reflect"
	"strings"
	"sync"

	"github.com/databricks/cli/bundle/deploy/terraform"
	tfschema "github.com/databricks/cli/bundle/internal/tf/schema"
	"github.com/databricks/cli/bundle/terraform_dabs_map"
	"github.com/databricks/cli/libs/structs/structaccess"
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

// tfSchemaTypeMap maps TF resource type name → schema struct type (via AllResources json tags).
var tfSchemaTypeMap = sync.OnceValue(func() map[string]reflect.Type {
	t := reflect.TypeOf(tfschema.AllResources{})
	m := make(map[string]reflect.Type, t.NumField())
	for i := range t.NumField() {
		f := t.Field(i)
		tag := strings.Split(f.Tag.Get("json"), ",")[0]
		if tag != "" && tag != "-" {
			m[tag] = f.Type
		}
	}
	return m
})

// LookupTFField looks up a field from TF state attributes for a bundle resource.
// group is the DABs group (e.g. "pipelines"), name is the resource name.
// fieldPath is the path to the field (may be in DABs or TF naming; both handled by DABsPathToTerraform).
// Returns (nil, nil) for empty/zero fields, error if the resource or field is not found.
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

	schemaType, ok := tfSchemaTypeMap()[tfType]
	if !ok {
		return nil, fmt.Errorf("no schema type registered for %q", tfType)
	}

	// Unmarshal attributes into a new instance of the schema struct.
	ptr := reflect.New(schemaType)
	if err := json.Unmarshal(attrsJSON, ptr.Interface()); err != nil {
		return nil, fmt.Errorf("cannot parse TF state for %s.%s: %w", tfType, name, err)
	}

	return structaccess.Get(ptr.Interface(), tfFieldPath)
}
