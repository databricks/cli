package terraform

import (
	"context"
	"fmt"

	"github.com/databricks/cli/bundle"
	"github.com/databricks/cli/libs/diag"
	"github.com/databricks/cli/libs/dyn"
	"github.com/databricks/cli/libs/dyn/dynvar"
)

type interpolateMutator struct{}

// isPostgresResource returns true if the resource type is a postgres resource.
// Postgres resources use "uid" instead of "id" as their identifier attribute.
func isPostgresResource(resourceType string) bool {
	switch resourceType {
	case "postgres_projects", "postgres_branches", "postgres_endpoints":
		return true
	default:
		return false
	}
}

func Interpolate() bundle.Mutator {
	return &interpolateMutator{}
}

func (m *interpolateMutator) Name() string {
	return "terraform.Interpolate"
}

func (m *interpolateMutator) Apply(ctx context.Context, b *bundle.Bundle) diag.Diagnostics {
	err := b.Config.Mutate(func(root dyn.Value) (dyn.Value, error) {
		prefix := dyn.MustPathFromString("resources")

		// Resolve variable references in all values.
		return dynvar.Resolve(root, func(path dyn.Path) (dyn.Value, error) {
			// Expect paths of the form:
			//   - resources.<resource_type>.<resource_name>.<field>...
			if !path.HasPrefix(prefix) || len(path) < 4 {
				return dyn.InvalidValue, dynvar.ErrSkipResolution
			}

			// Rewrite the bundle configuration path:
			//
			//   ${resources.pipelines.my_pipeline.id}
			//
			// into the Terraform-compatible resource identifier:
			//
			//   ${databricks_pipeline.my_pipeline.id}
			//
			resourceType := path[1].Key()
			tfResourceName, ok := GroupToTerraformName[resourceType]
			if !ok {
				// Trigger "key not found" for unknown resource types.
				return dyn.GetByPath(root, path)
			}

			// Replace the resource type with the Terraform resource name.
			path = dyn.NewPath(dyn.Key(tfResourceName)).Append(path[2:]...)

			// Postgres resources use "name" (resource path) instead of "id" as the identifier.
			// Map .id references to .name for these resources so cross-resource references
			// (like branch.parent -> project) resolve to the resource path format.
			if isPostgresResource(resourceType) && len(path) >= 3 {
				if lastKey := path[len(path)-1].Key(); lastKey == "id" {
					path = path[:len(path)-1].Append(dyn.Key("name"))
				}
			}

			return dyn.V(fmt.Sprintf("${%s}", path.String())), nil
		})
	})

	return diag.FromErr(err)
}
