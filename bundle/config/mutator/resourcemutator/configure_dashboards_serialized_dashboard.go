package resourcemutator

import (
	"context"
	"encoding/json"
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
			// to be able to read file contents in Databricks Workspace (reading a dashboard file via file system fails there)
			filePath, hasFilePath := v.Get(filePathFieldName).AsString()
			sd := v.Get(serializedDashboardFieldName)

			if hasFilePath {
				// file_path and serialized_dashboard are two ways to provide the
				// same content. Accepting both is ambiguous, so reject it instead
				// of silently picking one.
				if sd.IsValid() && sd.Kind() != dyn.KindNil {
					diags = diags.Append(diag.Diagnostic{
						Severity:  diag.Error,
						Summary:   "both file_path and serialized_dashboard are set; specify only one",
						Locations: sd.Locations(),
					})
					return v, nil
				}

				contents, err := b.SyncRoot.ReadFile(filePath)
				if err != nil {
					return dyn.InvalidValue, fmt.Errorf("failed to read serialized dashboard from file_path %s: %w", filePath, err)
				}
				return dyn.Set(v, serializedDashboardFieldName, dyn.V(string(contents)))
			}

			// Marshal an inline structured serialized_dashboard to a JSON string
			switch sd.Kind() {
			case dyn.KindInvalid, dyn.KindNil, dyn.KindString:
				// KindInvalid means serialized_dashboard is absent (neither it nor
				// file_path is set); leave it for backend validation to reject.
				return v, nil
			case dyn.KindMap:
				jsonBytes, err := json.Marshal(sd.AsAny())
				if err != nil {
					return dyn.InvalidValue, fmt.Errorf("failed to marshal inline serialized_dashboard: %w", err)
				}
				return dyn.Set(v, serializedDashboardFieldName, dyn.V(string(jsonBytes)))
			default:
				diags = diags.Append(diag.Diagnostic{
					Severity:  diag.Error,
					Summary:   fmt.Sprintf("serialized_dashboard must be a string or map, got %s", sd.Kind()),
					Locations: sd.Locations(),
				})
				return v, nil
			}
		})
	})

	diags = diags.Extend(diag.FromErr(err))
	return diags
}
