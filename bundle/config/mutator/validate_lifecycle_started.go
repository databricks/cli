package mutator

import (
	"context"
	"fmt"

	"github.com/databricks/cli/bundle"
	"github.com/databricks/cli/bundle/config/engine"
	"github.com/databricks/cli/libs/diag"
	"github.com/databricks/cli/libs/dyn"
)

// supportedForLifecycleStarted lists resource types that support lifecycle.started.
var supportedForLifecycleStarted = map[string]bool{
	"apps":           true,
	"clusters":       true,
	"sql_warehouses": true,
}

type validateLifecycleStarted struct {
	engine engine.EngineType
}

// ValidateLifecycleStarted returns a mutator that validates lifecycle.started
// is only used on supported resource types (apps, clusters, sql_warehouses).
// lifecycle.started is only supported in direct deployment mode.
func ValidateLifecycleStarted(e engine.EngineType) bundle.Mutator {
	return &validateLifecycleStarted{engine: e}
}

func (m *validateLifecycleStarted) Name() string {
	return "ValidateLifecycleStarted"
}

func (m *validateLifecycleStarted) Apply(_ context.Context, b *bundle.Bundle) diag.Diagnostics {
	var diags diag.Diagnostics

	_, err := dyn.MapByPattern(
		b.Config.Value(),
		dyn.NewPattern(dyn.Key("resources"), dyn.AnyKey(), dyn.AnyKey()),
		func(path dyn.Path, v dyn.Value) (dyn.Value, error) {
			resourceType := path[1].Key()

			startedV, err := dyn.GetByPath(v, dyn.NewPath(dyn.Key("lifecycle"), dyn.Key("started")))
			if err != nil {
				return v, nil
			}

			started, ok := startedV.AsBool()
			if !ok || !started {
				return v, nil
			}

			// lifecycle.started is a direct-mode-only feature;
			if !m.engine.IsDirect() {
				diags = diags.Append(diag.Diagnostic{
					Severity:  diag.Error,
					Summary:   "lifecycle.started is only supported in direct deployment mode",
					Locations: []dyn.Location{startedV.Location()},
				})
				return v, nil
			}

			if supportedForLifecycleStarted[resourceType] {
				return v, nil
			}

			resourceKey := path.String()
			diags = diags.Append(diag.Diagnostic{
				Severity:  diag.Error,
				Summary:   fmt.Sprintf("lifecycle.started is not supported for %s; it is only supported for apps, clusters, and sql_warehouses", resourceKey),
				Locations: []dyn.Location{startedV.Location()},
			})

			return v, nil
		},
	)
	if err != nil {
		diags = diags.Extend(diag.FromErr(err))
	}

	return diags
}
