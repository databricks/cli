package direct

import (
	"maps"
	"slices"
	"strings"

	"github.com/databricks/cli/bundle/config"
	"github.com/databricks/cli/bundle/direct/dresources"
	"github.com/databricks/cli/bundle/direct/dstate"
	"github.com/databricks/cli/libs/dyn"
	"github.com/databricks/cli/libs/dyn/dynvar"
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

// FindUnknownReferenceFields returns the references in the resources of configRoot that the plan
// rejects because the field they name exists neither in the state nor in the remote type of the
// target resource. It checks the references the plan resolves (see extractReferences) with the
// plan's own check (checkReferenceFieldExists), so it finds only references the plan rejects too.
// It skips what the plan leaves out or reports with another error: empty resources without state
// in db (nil when the state is not read), targets that are not planned, unsupported resource types,
// resource keys it cannot read, unparsable references and Terraform-only fields.
func FindUnknownReferenceFields(configRoot *config.Root, db *dstate.Database) ([]UnknownReferenceField, error) {
	adapters, err := dresources.InitAll(nil)
	if err != nil {
		return nil, err
	}

	planned := plannedNodes(configRoot, db, adapters)

	var result []UnknownReferenceField
	for _, node := range slices.Sorted(maps.Keys(planned)) {
		adapter := adapters[config.GetResourceTypeFromKey(node)]
		// An error here is the plan's to report.
		_ = walkReferences(configRoot.Value(), node, adapter.StateType(), func(path dyn.Path, _ *structpath.PathNode, v dyn.Value, ref dynvar.Ref) {
			seen := make(map[string]bool)
			for _, target := range ref.References() {
				if seen[target] || !strings.HasPrefix(target, "resources.") {
					continue
				}
				seen[target] = true
				if err := checkReference(adapters, planned, target); err != nil {
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

// plannedNodes returns the plan nodes of configRoot (resources and their permissions and grants)
// that the plan includes: the ones it can read whose state is not empty or that have state in db.
// Without db it treats every node as new, a subset of what the plan includes.
func plannedNodes(configRoot *config.Root, db *dstate.Database, adapters map[string]*dresources.Adapter) map[string]bool {
	planned := make(map[string]bool)
	for _, pattern := range planNodePatterns {
		_, _ = dyn.MapByPattern(configRoot.Value(), pattern, func(p dyn.Path, v dyn.Value) (dyn.Value, error) {
			node := p.String()
			// The plan reads each node back by its key's string form, which a key with a dot does
			// not survive.
			if parsed, err := dyn.NewPathFromString(node); err != nil || !parsed.Equal(p) {
				return v, nil
			}
			adapter, ok := adapters[config.GetResourceTypeFromKey(node)]
			if !ok {
				return v, nil
			}
			inputConfig, err := configRoot.GetResourceConfig(node)
			if err != nil {
				return v, nil
			}
			_, newState, err := prepareNodeState(adapter, inputConfig, node)
			if err != nil {
				return v, nil
			}
			hasState := false
			if db != nil {
				_, hasState = db.State[node]
			}
			if !hasState {
				if empty, err := adapter.IsEmptyState(newState); err != nil || empty {
					return v, nil
				}
			}
			planned[node] = true
			return v, nil
		})
	}
	return planned
}

// checkReference returns the plan's error for a reference to a field that the target resource type
// does not have, or nil, also when the plan reports the reference with another error.
func checkReference(adapters map[string]*dresources.Adapter, planned map[string]bool, target string) error {
	path, err := structpath.ParsePath(target)
	if err != nil {
		return nil
	}
	targetResourceKey, targetGroup, fieldPath, err := referenceTarget(path)
	if err != nil || fieldPath == nil || !planned[targetResourceKey] {
		return nil
	}
	return checkReferenceFieldExists(adapters[targetGroup], targetGroup, fieldPath, path)
}
