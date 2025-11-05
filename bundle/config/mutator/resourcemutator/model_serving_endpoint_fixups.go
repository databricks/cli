package resourcemutator

import (
	"context"

	"github.com/databricks/cli/bundle"
	"github.com/databricks/cli/bundle/direct/dresources"
	"github.com/databricks/cli/libs/diag"
	"github.com/databricks/cli/libs/dyn"
	"github.com/databricks/databricks-sdk-go/service/serving"
)

type modelServingEndpointFixups struct{}

func ModelServingEndpointFixups() bundle.Mutator {
	return &modelServingEndpointFixups{}
}

func (m *modelServingEndpointFixups) Name() string {
	return "ModelServingEndpointFixups"
}

func servedModelToServedEntity(model serving.ServedModelInput) serving.ServedEntityInput {
	return serving.ServedEntityInput{
		// served_models does not support ExternalModel, so we set ExternalModel to nil
		ExternalModel: nil,

		EntityName:                model.ModelName,
		EntityVersion:             model.ModelVersion,
		EnvironmentVars:           model.EnvironmentVars,
		InstanceProfileArn:        model.InstanceProfileArn,
		MaxProvisionedThroughput:  model.MaxProvisionedThroughput,
		MinProvisionedThroughput:  model.MinProvisionedThroughput,
		Name:                      model.Name,
		ProvisionedModelUnits:     model.ProvisionedModelUnits,
		ScaleToZeroEnabled:        model.ScaleToZeroEnabled,
		WorkloadSize:              model.WorkloadSize,
		WorkloadType:              serving.ServingModelWorkloadType(model.WorkloadType),
		MaxProvisionedConcurrency: model.MaxProvisionedConcurrency,
		MinProvisionedConcurrency: model.MinProvisionedConcurrency,
		ForceSendFields:           dresources.FilterFields[serving.ServedEntityInput](model.ForceSendFields),
	}
}

func (m *modelServingEndpointFixups) Apply(ctx context.Context, b *bundle.Bundle) diag.Diagnostics {
	var diags diag.Diagnostics

	for key, endpoint := range b.Config.Resources.ModelServingEndpoints {
		if endpoint == nil || endpoint.Config == nil {
			continue
		}

		// Validate that both ServedModels and ServedEntities are not used at the same time.
		if len(endpoint.Config.ServedModels) > 0 && len(endpoint.Config.ServedEntities) > 0 {
			diags = diags.Append(diag.Diagnostic{
				Severity: diag.Error,
				Summary:  "Cannot use both served_models and served_entities",
				Detail:   "Model serving endpoint cannot specify both served_models and served_entities at the same time.",
				Locations: []dyn.Location{
					b.Config.GetLocation("resources.model_serving_endpoints." + key),
				},
			})
			continue
		}

		// Convert ServedModels to ServedEntities if specified. ServedModels is a deprecated field, and the service recommends using ServedEntities instead.
		// We perform this translation here so that the deployment plan only has to detect served_entities and can ignore served_models.
		if len(endpoint.Config.ServedModels) > 0 {
			// Add warning recommending served_entities
			diags = diags.Append(diag.Diagnostic{
				Severity: diag.Warning,
				Summary:  "Using served_models is deprecated",
				Detail:   "The served_models field is deprecated. Please use served_entities instead.",
				Locations: []dyn.Location{
					b.Config.GetLocation("resources.model_serving_endpoints." + key + ".config.served_models"),
				},
			})

			endpoint.Config.ServedEntities = make([]serving.ServedEntityInput, len(endpoint.Config.ServedModels))
			for i, model := range endpoint.Config.ServedModels {
				endpoint.Config.ServedEntities[i] = servedModelToServedEntity(model)
			}
			// Clear ServedModels after conversion
			endpoint.Config.ServedModels = nil
		}

		// Apply default workload_size of "Small" if not specified. workload_size has a server side default of
		// "Small" if not specified. Setting it also as a client side default makes diff comparisons straightforward
		// during deployment.
		for i := range endpoint.Config.ServedEntities {
			if endpoint.Config.ServedEntities[i].WorkloadSize == "" {
				endpoint.Config.ServedEntities[i].WorkloadSize = "Small"
			}
		}
	}

	return diags
}
