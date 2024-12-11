package apps

import (
	"context"

	"github.com/databricks/cli/bundle"
	"github.com/databricks/cli/libs/diag"
	"github.com/databricks/cli/libs/dyn"
	"github.com/databricks/cli/libs/dyn/dynvar"
)

type interpolateVariables struct{}

func (i *interpolateVariables) Apply(ctx context.Context, b *bundle.Bundle) diag.Diagnostics {
	pattern := dyn.NewPattern(
		dyn.Key("resources"),
		dyn.Key("apps"),
		dyn.AnyKey(),
		dyn.Key("config"),
	)

	err := b.Config.Mutate(func(root dyn.Value) (dyn.Value, error) {
		return dyn.MapByPattern(root, pattern, func(p dyn.Path, v dyn.Value) (dyn.Value, error) {
			return dynvar.Resolve(v, func(path dyn.Path) (dyn.Value, error) {
				switch path[0] {
				case dyn.Key("databricks_pipeline"):
					path = dyn.NewPath(dyn.Key("resources"), dyn.Key("pipelines")).Append(path[1:]...)
				case dyn.Key("databricks_job"):
					path = dyn.NewPath(dyn.Key("resources"), dyn.Key("jobs")).Append(path[1:]...)
				case dyn.Key("databricks_mlflow_model"):
					path = dyn.NewPath(dyn.Key("resources"), dyn.Key("models")).Append(path[1:]...)
				case dyn.Key("databricks_mlflow_experiment"):
					path = dyn.NewPath(dyn.Key("resources"), dyn.Key("experiments")).Append(path[1:]...)
				case dyn.Key("databricks_model_serving"):
					path = dyn.NewPath(dyn.Key("resources"), dyn.Key("model_serving_endpoints")).Append(path[1:]...)
				case dyn.Key("databricks_registered_model"):
					path = dyn.NewPath(dyn.Key("resources"), dyn.Key("registered_models")).Append(path[1:]...)
				case dyn.Key("databricks_quality_monitor"):
					path = dyn.NewPath(dyn.Key("resources"), dyn.Key("quality_monitors")).Append(path[1:]...)
				case dyn.Key("databricks_schema"):
					path = dyn.NewPath(dyn.Key("resources"), dyn.Key("schemas")).Append(path[1:]...)
				case dyn.Key("databricks_volume"):
					path = dyn.NewPath(dyn.Key("resources"), dyn.Key("volumes")).Append(path[1:]...)
				case dyn.Key("databricks_cluster"):
					path = dyn.NewPath(dyn.Key("resources"), dyn.Key("clusters")).Append(path[1:]...)
				case dyn.Key("databricks_dashboard"):
					path = dyn.NewPath(dyn.Key("resources"), dyn.Key("dashboards")).Append(path[1:]...)
				case dyn.Key("databricks_app"):
					path = dyn.NewPath(dyn.Key("resources"), dyn.Key("apps")).Append(path[1:]...)
				default:
					// Trigger "key not found" for unknown resource types.
					return dyn.GetByPath(root, path)
				}

				return dyn.GetByPath(root, path)
			})
		})
	})

	return diag.FromErr(err)
}

func (i *interpolateVariables) Name() string {
	return "apps.InterpolateVariables"
}

func InterpolateVariables() bundle.Mutator {
	return &interpolateVariables{}
}
