package resourcemutator

import (
	"context"
	"encoding/json"
	"fmt"

	"github.com/databricks/cli/bundle"
	"github.com/databricks/cli/libs/diag"
	"github.com/databricks/cli/libs/structs/structpath"
	"github.com/databricks/cli/libs/structs/structvar"
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

	// Skip converting the configuration if there is nothing to configure.
	if len(b.Config.Resources.Dashboards) == 0 {
		return nil
	}

	pattern := structpath.MustParsePattern("resources.dashboards.*")

	// Configure serialized_dashboard field for all dashboards.
	err := structvar.ForEach(b.Config.View(), pattern, func(p *structpath.PathNode, v structvar.View) error {
		// Include "serialized_dashboard" field if "file_path" is set.
		// Note: the Terraform resource supports "file_path" natively, but we read the contents of the dashboard here
		// to be able to read file contents in Databricks Workspace (reading a dashboard file via file system fails there)
		filePath, hasFilePath := v.Get(filePathFieldName).AsString()
		sd := v.Get(serializedDashboardFieldName)
		sdPath := structpath.NewStringKey(p, serializedDashboardFieldName)

		if hasFilePath {
			// file_path and serialized_dashboard are two ways to provide the
			// same content. Accepting both is ambiguous, so reject it instead
			// of silently picking one.
			if sd.IsValid() && sd.Kind() != structvar.KindNil {
				diags = diags.Append(diag.Diagnostic{
					Severity:  diag.Error,
					Summary:   "both file_path and serialized_dashboard are set; specify only one",
					Locations: sd.Locations(),
				})
				return nil
			}

			contents, err := b.SyncRoot.ReadFile(filePath)
			if err != nil {
				return fmt.Errorf("failed to read serialized dashboard from file_path %s: %w", filePath, err)
			}
			return b.Config.Set(sdPath, string(contents))
		}

		// Marshal an inline structured serialized_dashboard to a JSON string
		switch sd.Kind() {
		case structvar.KindInvalid, structvar.KindNil, structvar.KindString:
			// KindInvalid means serialized_dashboard is absent (neither it nor
			// file_path is set); leave it for backend validation to reject.
			return nil
		case structvar.KindMap:
			jsonBytes, err := json.Marshal(sd.AsAny())
			if err != nil {
				return fmt.Errorf("failed to marshal inline serialized_dashboard: %w", err)
			}
			return b.Config.Set(sdPath, string(jsonBytes))
		default:
			diags = diags.Append(diag.Diagnostic{
				Severity:  diag.Error,
				Summary:   fmt.Sprintf("serialized_dashboard must be a string or map, got %s", sd.Kind()),
				Locations: sd.Locations(),
			})
			return nil
		}
	})

	diags = diags.Extend(diag.FromErr(err))
	return diags
}
