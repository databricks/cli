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
		if endpoint.TelemetryConfig == nil {
			continue
		}

		path := "resources.model_serving_endpoints." + key + ".telemetry_config"
		addDiag := func(summary, detail string) {
			diags = append(diags, diag.Diagnostic{
				Severity:  diag.Error,
				Summary:   summary,
				Detail:    detail,
				Paths:     []dyn.Path{dyn.MustPathFromString(path)},
				Locations: b.Config.GetLocations(path),
			})
		}

		// The telemetry API rejects endpoints typed NO_CONFIG or EXTERNAL_MODELS, but create
		// drops the field instead of failing, so without this the bundle deploys once and
		// then fails on every later deploy.
		if !servesRegisteredModel(endpoint.Config) {
			addDiag(
				"telemetry_config is only supported on an endpoint that serves a registered model",
				"The serving endpoints telemetry configuration API only supports endpoints with custom served models, so it rejects an endpoint that serves nothing or only external models. Add a served entity with entity_name, or remove telemetry_config.",
			)
		}

		// Create and the telemetry API both report success and apply nothing when the
		// configuration names no profile, so an inference_table_config on its own would land
		// in state and show as a perpetual update against an endpoint that never got it.
		if !identifiesTelemetryProfile(endpoint.TelemetryConfig) {
			addDiag(
				"telemetry_config must set table_names or telemetry_profile_id",
				"The serving endpoints API identifies a telemetry configuration by the Unity Catalog tables to create a profile from (table_names) or by an existing profile (telemetry_profile_id). Given neither, it discards the configuration instead of applying it.",
			)
		}
	}

	sortDiagnostics(diags)

	return diags
}

// identifiesTelemetryProfile reports whether the configuration names the telemetry
// profile to use or the tables to create one from.
func identifiesTelemetryProfile(config *serving.TelemetryConfig) bool {
	return config.TableNames != nil || config.TelemetryProfileId != ""
}

// servesRegisteredModel reports whether the endpoint serves at least one registered
// model, as opposed to nothing at all or only external models. A foundation model is
// named the same way and is accepted here; create rejects it outright with a clear error.
//
// ModelServingEndpointFixups has already folded served_models into served_entities.
func servesRegisteredModel(config *serving.EndpointCoreConfigInput) bool {
	if config == nil {
		return false
	}

	for _, entity := range config.ServedEntities {
		if entity.ExternalModel == nil && entity.EntityName != "" {
			return true
		}
	}

	return false
}
