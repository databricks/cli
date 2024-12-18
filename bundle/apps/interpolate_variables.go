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

	tfToConfigMap := map[string]string{
		"databricks_pipeline":          "pipelines",
		"databricks_job":               "jobs",
		"databricks_mlflow_model":      "models",
		"databricks_mlflow_experiment": "experiments",
		"databricks_model_serving":     "model_serving_endpoints",
		"databricks_registered_model":  "registered_models",
		"databricks_quality_monitor":   "quality_monitors",
		"databricks_schema":            "schemas",
		"databricks_volume":            "volumes",
		"databricks_cluster":           "clusters",
		"databricks_dashboard":         "dashboards",
		"databricks_app":               "apps",
	}

	err := b.Config.Mutate(func(root dyn.Value) (dyn.Value, error) {
		return dyn.MapByPattern(root, pattern, func(p dyn.Path, v dyn.Value) (dyn.Value, error) {
			return dynvar.Resolve(v, func(path dyn.Path) (dyn.Value, error) {
				key, ok := tfToConfigMap[path[0].Key()]
				if ok {
					path = dyn.NewPath(dyn.Key("resources"), dyn.Key(key)).Append(path[1:]...)
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
