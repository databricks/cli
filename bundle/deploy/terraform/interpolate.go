package terraform

import (
	"context"
	"fmt"

	"github.com/databricks/cli/bundle"
	"github.com/databricks/cli/libs/diag"
	"github.com/databricks/cli/libs/dyn"
	"github.com/databricks/cli/libs/dyn/dynvar"
)

type interpolateMutator struct {
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
			switch path[1] {
			case dyn.Key("pipelines"):
				path = dyn.NewPath(dyn.Key("databricks_pipeline")).Append(path[2:]...)
			case dyn.Key("jobs"):
				path = dyn.NewPath(dyn.Key("databricks_job")).Append(path[2:]...)
			case dyn.Key("models"):
				path = dyn.NewPath(dyn.Key("databricks_mlflow_model")).Append(path[2:]...)
			case dyn.Key("experiments"):
				path = dyn.NewPath(dyn.Key("databricks_mlflow_experiment")).Append(path[2:]...)
			case dyn.Key("model_serving_endpoints"):
				path = dyn.NewPath(dyn.Key("databricks_model_serving")).Append(path[2:]...)
			case dyn.Key("registered_models"):
				path = dyn.NewPath(dyn.Key("databricks_registered_model")).Append(path[2:]...)
			default:
				// Trigger "key not found" for unknown resource types.
				return dyn.GetByPath(root, path)
			}

			return dyn.V(fmt.Sprintf("${%s}", path.String())), nil
		})
	})
	return diag.FromErr(err)

}
