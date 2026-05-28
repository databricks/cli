package terraform

import (
	"context"
	"fmt"
	"strings"

	"github.com/databricks/cli/bundle"
	"github.com/databricks/cli/bundle/terraform_dabs_map"
	"github.com/databricks/cli/libs/diag"
	"github.com/databricks/cli/libs/dyn"
	"github.com/databricks/cli/libs/dyn/dynvar"
	"github.com/databricks/cli/libs/structs/structpath"
)

type interpolateMutator struct{}

// isPostgresResource returns true if the resource type is a postgres resource.
// Postgres resources use "name" instead of "id" as their identifier attribute.
func isPostgresResource(resourceType string) bool {
	switch resourceType {
	case "postgres_projects", "postgres_branches", "postgres_endpoints", "postgres_catalogs", "postgres_synced_tables":
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

			// Try to resolve as a pre-deploy local config value.
			// This handles TF-style field names (e.g. "git_source.branch" → "git_source.git_branch")
			// and TF list-block [0] indices on single nested blocks (e.g. "task[0].library[0].whl").
			// We translate TF key names to DABs key names via TerraformPathToDABs, reconstruct
			// the full dyn.Path with translated keys and original indices, then look up in root.
			// If the field is not in local config (e.g. "id" which is readonly/post-deploy),
			// we fall through to emit a TF expression instead.
			if len(path) >= 4 {
				resourceName := path[2].Key()

				// Build key-only field string for TF→DABs translation.
				parts := make([]string, 0, len(path)-3)
				for _, p := range path[3:] {
					if k := p.Key(); k != "" {
						parts = append(parts, k)
					}
				}
				fieldStr := strings.Join(parts, ".")

				if fieldNode, err := structpath.ParsePath(fieldStr); err == nil {
					if dabsNode, err := terraform_dabs_map.TerraformPathToDABs(resourceType, fieldNode); err == nil {
						// Reconstruct the dyn.Path with translated DABs keys and original indices.
						// dabsNode contains only key components (fieldStr was key-only); original
						// index components from path[3:] are re-inserted at their original positions.
						dabsKeys := dabsNode.AsSlice()
						dabsKeyIdx := 0
						dabsPath := dyn.NewPath(dyn.Key("resources"), dyn.Key(resourceType), dyn.Key(resourceName))
						for _, p := range path[3:] {
							if p.Key() != "" {
								if dabsKeyIdx < len(dabsKeys) {
									if translatedKey, ok := dabsKeys[dabsKeyIdx].StringKey(); ok {
										dabsPath = dabsPath.Append(dyn.Key(translatedKey))
									}
									dabsKeyIdx++
								}
							} else {
								dabsPath = dabsPath.Append(p)
							}
						}

						if val, err := dyn.GetByPath(root, dabsPath); err == nil && val.Kind() != dyn.KindNil {
							return val, nil
						}
					}
				}
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
