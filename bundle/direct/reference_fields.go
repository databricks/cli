package direct

import (
	"fmt"
	"slices"
	"strings"

	"github.com/databricks/cli/bundle/config"
	"github.com/databricks/cli/bundle/direct/dresources"
	"github.com/databricks/cli/bundle/terraform_dabs_map"
	"github.com/databricks/cli/libs/dyn"
	"github.com/databricks/cli/libs/dyn/dynvar"
	"github.com/databricks/cli/libs/structs/structaccess"
	"github.com/databricks/cli/libs/structs/structpath"
)

// UnknownReferenceField is a reference to a field that the target resource type does not have.
type UnknownReferenceField struct {
	// Reference is the reference without "${" and "}", e.g. "resources.volumes.bar.non_existent".
	Reference string

	// Value is the config value that holds the reference.
	Value dyn.Value

	// Path is the path of Value in the config.
	Path dyn.Path

	Err error
}

// FindUnknownReferenceFields returns the references in the resources of root that the plan
// rejects because the field they name exists neither in the state nor in the remote type of the
// target resource. It checks the references the plan resolves (see extractReferences) the way
// LookupReferencePreDeploy does, so it finds only references the plan rejects too. It skips what
// the plan reports with another error: unsupported resource types, resource keys it cannot read,
// unparsable references and Terraform-only fields.
func FindUnknownReferenceFields(root dyn.Value) ([]UnknownReferenceField, error) {
	adapters, err := dresources.InitAll(nil)
	if err != nil {
		return nil, err
	}

	var nodes []string
	for _, pattern := range planNodePatterns {
		_, _ = dyn.MapByPattern(root, pattern, func(p dyn.Path, v dyn.Value) (dyn.Value, error) {
			nodes = append(nodes, p.String())
			return v, nil
		})
	}
	// MapByPattern visits map keys in random order.
	slices.Sort(nodes)

	var result []UnknownReferenceField
	for _, node := range nodes {
		adapter, ok := adapters[config.GetResourceTypeFromKey(node)]
		if !ok {
			continue
		}
		// An error here (for example a resource key with a dot) is the plan's to report.
		_ = walkReferences(root, node, adapter.StateType(), func(path dyn.Path, _ *structpath.PathNode, v dyn.Value, ref dynvar.Ref) {
			seen := make(map[string]bool)
			for _, target := range ref.References() {
				if seen[target] || !strings.HasPrefix(target, "resources.") {
					continue
				}
				seen[target] = true
				if err := checkReferenceField(adapters, target); err != nil {
					result = append(result, UnknownReferenceField{
						Reference: target,
						Value:     v,
						Path:      path,
						Err:       err,
					})
				}
			}
		})
	}
	return result, nil
}

// checkReferenceField returns the error LookupReferencePreDeploy returns for a reference to a field
// that the target resource type does not have, or nil.
func checkReferenceField(adapters map[string]*dresources.Adapter, target string) error {
	path, err := structpath.ParsePath(target)
	if err != nil {
		return nil
	}
	targetResourceKey, fieldPath := splitResourcePath(path)
	targetGroup := config.GetResourceTypeFromKey(targetResourceKey)
	adapter, ok := adapters[targetGroup]
	if !ok {
		return nil
	}
	fieldPath, err = terraform_dabs_map.TerraformPathToDABs(targetGroup, fieldPath)
	if err != nil || fieldPath == nil || fieldPath.String() == "id" {
		return nil
	}
	if structaccess.ValidatePath(adapter.StateType(), fieldPath) == nil || structaccess.ValidatePath(adapter.RemoteType(), fieldPath) == nil {
		return nil
	}
	return unknownReferenceFieldError(targetGroup, fieldPath)
}

func unknownReferenceFieldError(group string, fieldPath *structpath.PathNode) error {
	return fmt.Errorf("resource type %q has no field %q", group, fieldPath.String())
}
