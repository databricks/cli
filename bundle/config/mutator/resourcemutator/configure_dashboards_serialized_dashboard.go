package resourcemutator

import (
	"context"
	"fmt"
	"maps"
	"slices"

	"github.com/databricks/cli/bundle"
	"github.com/databricks/cli/libs/diag"
	"github.com/databricks/cli/libs/structs/structpath"
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

	dashboards := b.Config.Resources.Dashboards
	for _, name := range slices.Sorted(maps.Keys(dashboards)) {
		d := dashboards[name]
		if d == nil {
			continue
		}
		sdPath := structpath.NewPath(nil, "resources", "dashboards", name, serializedDashboardFieldName)

		// Include "serialized_dashboard" field if "file_path" is set.
		// Note: the Terraform resource supports "file_path" natively, but we read the contents of the dashboard here
		// to be able to read file contents in Databricks Workspace (reading a dashboard file via file system fails there)
		if d.FilePath != "" {
			// file_path and serialized_dashboard are two ways to provide the
			// same content. Accepting both is ambiguous, so reject it instead
			// of silently picking one.
			if d.SerializedDashboard != nil {
				diags = diags.Append(diag.Diagnostic{
					Severity:  diag.Error,
					Summary:   "both file_path and serialized_dashboard are set; specify only one",
					Locations: b.Config.GetLocationsOfExact(sdPath),
				})
				continue
			}

			contents, err := b.SyncRoot.ReadFile(d.FilePath)
			if err != nil {
				return diags.Extend(diag.FromErr(fmt.Errorf("failed to read serialized dashboard from file_path %s: %w", d.FilePath, err)))
			}
			if err := b.Config.Set(sdPath, string(contents)); err != nil {
				return diags.Extend(diag.FromErr(err))
			}
			continue
		}

		// Marshal an inline structured serialized_dashboard to a JSON string.
		// Absent (neither it nor file_path is set) is left for backend validation.
		s, ok, kind, err := inlineJSON(d.SerializedDashboard)
		if err != nil {
			return diags.Extend(diag.FromErr(fmt.Errorf("failed to marshal inline serialized_dashboard: %w", err)))
		}
		if kind != "" {
			diags = diags.Append(diag.Diagnostic{
				Severity:  diag.Error,
				Summary:   fmt.Sprintf("serialized_dashboard must be a string or map, got %s", kind),
				Locations: b.Config.GetLocationsOfExact(sdPath),
			})
			continue
		}
		if ok {
			if err := b.Config.Set(sdPath, s); err != nil {
				return diags.Extend(diag.FromErr(err))
			}
		}
	}

	return diags
}
