package validate

import (
	"context"

	"github.com/databricks/cli/bundle"
	"github.com/databricks/cli/libs/diag"
	"github.com/databricks/cli/libs/dyn"
	"github.com/databricks/databricks-sdk-go/service/serving"
)

func ValidateServingTelemetryConfig() bundle.ReadOnlyMutator {
	return &validateServingTelemetryConfig{}
}

type validateServingTelemetryConfig struct{ bundle.RO }

func (v *validateServingTelemetryConfig) Name() string {
	return "validate:validate_serving_telemetry_config"
}

func (v *validateServingTelemetryConfig) Apply(_ context.Context, b *bundle.Bundle) diag.Diagnostics {
	var diags diag.Diagnostics

	for key, endpoint := range b.Config.Resources.ModelServingEndpoints {
		if endpoint.TelemetryConfig == nil || identifiesTelemetryProfile(endpoint.TelemetryConfig) {
			continue
		}

		// Create and the telemetry API both report success and apply nothing when the config
		// names no profile, so the field would be planned and re-sent on every deploy.
		path := "resources.model_serving_endpoints." + key + ".telemetry_config"
		diags = append(diags, diag.Diagnostic{
			Severity:  diag.Error,
			Summary:   "telemetry_config must name a table in table_names or set telemetry_profile_id",
			Detail:    "The serving endpoints API identifies a telemetry configuration by the Unity Catalog tables to create a profile from (table_names) or by an existing profile (telemetry_profile_id). Given neither, it discards the configuration instead of applying it.",
			Paths:     []dyn.Path{dyn.MustPathFromString(path)},
			Locations: b.Config.GetLocations(path),
		})
	}

	sortDiagnostics(diags)

	return diags
}

// identifiesTelemetryProfile reports whether the configuration names an existing profile
// or the tables to create one from. An empty table_names names neither.
func identifiesTelemetryProfile(config *serving.TelemetryConfig) bool {
	if config.TelemetryProfileId != "" {
		return true
	}
	t := config.TableNames
	return t != nil && (t.LogsTable != "" || t.TracesTable != "" || t.MetricsTable != "" || t.AnnotationsTable != "")
}
