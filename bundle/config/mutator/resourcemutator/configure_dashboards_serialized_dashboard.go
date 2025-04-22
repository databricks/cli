package resourcemutator

import (
	"context"
	"fmt"

	"github.com/databricks/cli/libs/dyn"

	"github.com/databricks/cli/bundle"
	"github.com/databricks/cli/libs/diag"
)

type configureDashboardSerializedDashboard struct{}

func ConfigureDashboardSerializedDashboard() bundle.Mutator {
	return &configureDashboardSerializedDashboard{}
}

func (c configureDashboardSerializedDashboard) Name() string {
	return "ConfigureDashboardSerializedDashboard"
}

func (c configureDashboardSerializedDashboard) Apply(_ context.Context, b *bundle.Bundle) diag.Diagnostics {
	for _, dashboard := range b.Config.Resources.Dashboards {
		path := dashboard.FilePath
		if path == "" {
			continue
		}
		contents, err := b.SyncRoot.ReadFile(path)
		if err != nil {
			return diag.FromErr(fmt.Errorf("failed to read serialized dashboard from file_path %s: %w", path, err))
		}
		dashboard.SerializedDashboard = string(contents)
	}

	// Drop the "file_path" field. It is mutually exclusive with "serialized_dashboard".
	pattern := dyn.NewPattern(
		dyn.Key("resources"),
		dyn.Key("dashboards"),
		dyn.AnyKey(),
	)
	err := b.Config.Mutate(func(v dyn.Value) (dyn.Value, error) {
		return dyn.MapByPattern(v, pattern, func(p dyn.Path, v dyn.Value) (dyn.Value, error) {
			return dyn.Walk(v, func(p dyn.Path, v dyn.Value) (dyn.Value, error) {
				switch len(p) {
				case 0:
					return v, nil
				case 1:
					if p[0] == dyn.Key("file_path") {
						return v, dyn.ErrDrop
					}
				}

				// Skip everything else.
				return v, dyn.ErrSkip
			})
		})
	})

	var diags diag.Diagnostics
	diags = diags.Extend(diag.FromErr(err))
	return diags
}
