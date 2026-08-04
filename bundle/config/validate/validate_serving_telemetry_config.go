package validate

import (
	"cmp"
	"context"
	"slices"

	"github.com/databricks/cli/bundle"
	"github.com/databricks/cli/libs/diag"
	"github.com/databricks/cli/libs/dyn"
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

	// The serving endpoints API types an endpoint that serves nothing as NO_CONFIG and
	// rejects telemetry configuration on it with "Telemetry configuration is not supported
	// for endpoint type 'NO_CONFIG'". Create drops the field instead of failing, so without
	// this the bundle deploys once and then fails on every later deploy, when the plan
	// finally applies telemetry_config through the telemetry-config API.
	for key, endpoint := range b.Config.Resources.ModelServingEndpoints {
		if endpoint.TelemetryConfig == nil {
			continue
		}
		if endpoint.Config != nil && (len(endpoint.Config.ServedEntities) > 0 || len(endpoint.Config.ServedModels) > 0) {
			continue
		}

		path := "resources.model_serving_endpoints." + key + ".telemetry_config"
		diags = append(diags, diag.Diagnostic{
			Severity:  diag.Error,
			Summary:   "telemetry_config is not supported on an endpoint that does not serve a model",
			Detail:    "The serving endpoints API rejects telemetry configuration on an endpoint with no served entities. Add an entry to config.served_entities, or remove telemetry_config.",
			Paths:     []dyn.Path{dyn.MustPathFromString(path)},
			Locations: b.Config.GetLocations(path),
		})
	}

	// Map iteration order is randomized; sort by path for stable output.
	slices.SortFunc(diags, func(x, y diag.Diagnostic) int {
		return cmp.Compare(x.Paths[0].String(), y.Paths[0].String())
	})

	return diags
}
