package resourcemutator

import (
	"context"
	"fmt"

	"github.com/databricks/cli/bundle"
	"github.com/databricks/cli/libs/diag"
	"github.com/databricks/cli/libs/dyn"
)

const (
	filePathFieldName            = "file_path"
	serializedDashboardFieldName = "serialized_dashboard"
)

type configureDashboardSerializedDashboard struct{}

func ConfigureDashboardSerializedDashboard() bundle.Mutator {
	return &configureDashboardSerializedDashboard{}
}

func (c configureDashboardSerializedDashboard) Name() string {
	return "ConfigureDashboardSerializedDashboard"
}

func (c configureDashboardSerializedDashboard) Apply(_ context.Context, b *bundle.Bundle) diag.Diagnostics {
	var diags diag.Diagnostics

	pattern := dyn.NewPattern(
		dyn.Key("resources"),
		dyn.Key("dashboards"),
		dyn.AnyKey(),
	)

	// Configure serialized_dashboard field for all dashboards.
	err := b.Config.Mutate(func(v dyn.Value) (dyn.Value, error) {
		return dyn.MapByPattern(v, pattern, func(p dyn.Path, v dyn.Value) (dyn.Value, error) {
			// Include "serialized_dashboard" field if "file_path" is set.
			// Note: the Terraform resource supports "file_path" natively, but we read the contents of the dashboard here
			// to cover the use case of deployments from the workspace
			path, ok := v.Get(filePathFieldName).AsString()
			if !ok {
				return v, nil
			}

			contents, err := b.SyncRoot.ReadFile(path)
			if err != nil {
				return dyn.InvalidValue, fmt.Errorf("failed to read serialized dashboard from file_path %s: %w", path, err)
			}

			v, err = dyn.Set(v, serializedDashboardFieldName, dyn.V(string(contents)))
			if err != nil {
				return dyn.InvalidValue, fmt.Errorf("failed to set serialized_dashboard: %w", err)
			}

			// Drop the "file_path" field. It is mutually exclusive with "serialized_dashboard".
			return dyn.Walk(v, func(p dyn.Path, v dyn.Value) (dyn.Value, error) {
				switch len(p) {
				case 0:
					return v, nil
				case 1:
					if p[0] == dyn.Key(filePathFieldName) {
						return v, dyn.ErrDrop
					}
				}

				// Skip everything else.
				return v, dyn.ErrSkip
			})
		})
	})

	diags = diags.Extend(diag.FromErr(err))
	return diags
}
