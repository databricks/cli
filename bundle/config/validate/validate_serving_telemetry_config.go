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

		// The telemetry configuration API only supports endpoints with custom served
		// models: it rejects an endpoint that serves nothing ("Telemetry configuration is
		// not supported for endpoint type 'NO_CONFIG'") and one that only proxies external
		// models (the same error with 'EXTERNAL_MODELS'). Create drops telemetry_config
		// instead of failing, so without this the bundle deploys once and then fails on
		// every later deploy, when the plan finally applies the field through that API.
		if !servesRegisteredModel(endpoint.Config) {
			addDiag(
				"telemetry_config is only supported on an endpoint that serves a registered model",
				"The serving endpoints telemetry configuration API only supports endpoints with custom served models, so it rejects an endpoint that serves nothing or only external models. Add a served entity with entity_name, or remove telemetry_config.",
			)
		}

		// The API identifies a telemetry configuration by an existing profile or by the
		// tables to create one from. Given neither, both create and the telemetry
		// configuration API return success and apply nothing, so an inference_table_config
		// on its own is silently discarded. Left unchecked the CLI would record it in state
		// and report a perpetual update against an endpoint that never received it.
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
// indistinguishable from a custom model here (both are named by entity_name), so it is
// accepted; create rejects it outright with "Telemetry configuration is supported only
// for custom CPU or GPU models", which is already a clear failure at the first deploy.
//
// ModelServingEndpointFixups has converted served_models to served_entities by the time
// validation runs, so only served_entities is inspected.
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
