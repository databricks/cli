package apps

import (
	"context"

	"github.com/databricks/cli/bundle"
	"github.com/databricks/cli/bundle/deploy/terraform"
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
				key, ok := terraform.TerraformToGroupName[path[0].Key()]
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
