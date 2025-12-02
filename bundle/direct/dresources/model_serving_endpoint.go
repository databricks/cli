package dresources

import (
	"context"
	"fmt"
	"time"

	"github.com/databricks/cli/bundle/config/resources"
	"github.com/databricks/cli/bundle/deployplan"
	"github.com/databricks/cli/libs/utils"
	"github.com/databricks/databricks-sdk-go"
	"github.com/databricks/databricks-sdk-go/service/serving"
)

type ResourceModelServingEndpoint struct {
	client *databricks.WorkspaceClient
}

func (*ResourceModelServingEndpoint) New(client *databricks.WorkspaceClient) *ResourceModelServingEndpoint {
	return &ResourceModelServingEndpoint{
		client: client,
	}
}

func (*ResourceModelServingEndpoint) PrepareState(input *resources.ModelServingEndpoint) *serving.CreateServingEndpoint {
	return &input.CreateServingEndpoint
}

func autoCaptureConfigOutputToInput(output *serving.AutoCaptureConfigOutput) *serving.AutoCaptureConfigInput {
	if output == nil {
		return nil
	}
	return &serving.AutoCaptureConfigInput{
		CatalogName:     output.CatalogName,
		SchemaName:      output.SchemaName,
		TableNamePrefix: output.TableNamePrefix,
		Enabled:         output.Enabled,
		ForceSendFields: utils.FilterFields[serving.AutoCaptureConfigInput](output.ForceSendFields),
	}
}

func servedEntitiesOutputToInput(output []serving.ServedEntityOutput) []serving.ServedEntityInput {
	entities := make([]serving.ServedEntityInput, len(output))
	for i, entity := range output {
		entities[i] = serving.ServedEntityInput{
			EntityName:                entity.EntityName,
			EntityVersion:             entity.EntityVersion,
			EnvironmentVars:           entity.EnvironmentVars,
			ExternalModel:             entity.ExternalModel,
			InstanceProfileArn:        entity.InstanceProfileArn,
			MaxProvisionedConcurrency: entity.MaxProvisionedConcurrency,
			MaxProvisionedThroughput:  entity.MaxProvisionedThroughput,
			MinProvisionedConcurrency: entity.MinProvisionedConcurrency,
			MinProvisionedThroughput:  entity.MinProvisionedThroughput,
			Name:                      entity.Name,
			ProvisionedModelUnits:     entity.ProvisionedModelUnits,
			ScaleToZeroEnabled:        entity.ScaleToZeroEnabled,
			WorkloadSize:              entity.WorkloadSize,
			WorkloadType:              entity.WorkloadType,
			ForceSendFields:           utils.FilterFields[serving.ServedEntityInput](entity.ForceSendFields),
		}
	}

	return entities
}

func configOutputToInput(output *serving.EndpointCoreConfigOutput) *serving.EndpointCoreConfigInput {
	if output == nil {
		return nil
	}
	return &serving.EndpointCoreConfigInput{
		AutoCaptureConfig: autoCaptureConfigOutputToInput(output.AutoCaptureConfig),
		ServedEntities:    servedEntitiesOutputToInput(output.ServedEntities),
		ServedModels:      nil,
		TrafficConfig:     output.TrafficConfig,
		Name:              "",
	}
}

func (*ResourceModelServingEndpoint) RemapState(state *RefreshOutput) *serving.CreateServingEndpoint {
	details := state.EndpointDetails
	// Map the remote state (ServingEndpointDetailed) to the local state (CreateServingEndpoint)
	// for proper comparison during diff calculation
	return &serving.CreateServingEndpoint{
		AiGateway:          details.AiGateway,
		BudgetPolicyId:     details.BudgetPolicyId,
		Config:             configOutputToInput(details.Config),
		Description:        details.Description,
		EmailNotifications: details.EmailNotifications,
		Name:               details.Name,
		RouteOptimized:     details.RouteOptimized,
		Tags:               details.Tags,
		ForceSendFields:    utils.FilterFields[serving.CreateServingEndpoint](details.ForceSendFields),

		// Rate limits are a deprecated field that are not returned by the API on GET calls. Thus we map them to nil.
		// TODO(shreyas): Add a warning when users try setting top level rate limits.
		RateLimits: nil,
	}
}

type RefreshOutput struct {
	EndpointDetails *serving.ServingEndpointDetailed `json:"endpoint_details"`
	EndpointId      string                           `json:"endpoint_id"`
}

func (r *ResourceModelServingEndpoint) DoRead(ctx context.Context, id string) (*RefreshOutput, error) {
	endpoint, err := r.client.ServingEndpoints.GetByName(ctx, id)
	if err != nil {
		return nil, err
	}
	return &RefreshOutput{
		EndpointDetails: endpoint,
		EndpointId:      endpoint.Id,
	}, nil
}

func (r *ResourceModelServingEndpoint) DoCreate(ctx context.Context, config *serving.CreateServingEndpoint) (string, *RefreshOutput, error) {
	waiter, err := r.client.ServingEndpoints.Create(ctx, *config)
	if err != nil {
		return "", nil, err
	}

	return waiter.Response.Name, nil, nil
}

// waitForEndpointReady waits for the serving endpoint to be ready (not updating)
func (r *ResourceModelServingEndpoint) waitForEndpointReady(ctx context.Context, name string) (*RefreshOutput, error) {
	details, err := r.client.ServingEndpoints.WaitGetServingEndpointNotUpdating(ctx, name, 35*time.Minute, nil)
	if err != nil {
		return nil, err
	}

	return &RefreshOutput{
		EndpointDetails: details,
		EndpointId:      details.Id,
	}, nil
}

func (r *ResourceModelServingEndpoint) WaitAfterCreate(ctx context.Context, config *serving.CreateServingEndpoint) (*RefreshOutput, error) {
	return r.waitForEndpointReady(ctx, config.Name)
}

func (r *ResourceModelServingEndpoint) WaitAfterUpdate(ctx context.Context, config *serving.CreateServingEndpoint) (*RefreshOutput, error) {
	return r.waitForEndpointReady(ctx, config.Name)
}

func (r *ResourceModelServingEndpoint) updateAiGateway(ctx context.Context, id string, aiGateway *serving.AiGatewayConfig) error {
	if aiGateway == nil {
		req := serving.PutAiGatewayRequest{
			Name:                 id,
			FallbackConfig:       nil,
			Guardrails:           nil,
			InferenceTableConfig: nil,
			RateLimits:           nil,
			UsageTrackingConfig:  nil,
		}
		_, err := r.client.ServingEndpoints.PutAiGateway(ctx, req)
		return err
	}

	req := serving.PutAiGatewayRequest{
		Name:                 id,
		FallbackConfig:       aiGateway.FallbackConfig,
		Guardrails:           aiGateway.Guardrails,
		InferenceTableConfig: aiGateway.InferenceTableConfig,
		RateLimits:           aiGateway.RateLimits,
		UsageTrackingConfig:  aiGateway.UsageTrackingConfig,
	}
	_, err := r.client.ServingEndpoints.PutAiGateway(ctx, req)
	if err != nil {
		return fmt.Errorf("failed to update AI gateway: %w", err)
	}
	return nil
}

// TODO(shreyas): Add acceptance tests for update and for unsetting. Will be done once we add selective updates for these fields.
func (r *ResourceModelServingEndpoint) updateConfig(ctx context.Context, id string, config *serving.EndpointCoreConfigInput) error {
	if config == nil {
		// Unset config in resource.
		req := serving.EndpointCoreConfigInput{
			Name:              id,
			AutoCaptureConfig: nil,
			ServedEntities:    nil,
			TrafficConfig:     nil,
			ServedModels:      nil,
		}
		_, err := r.client.ServingEndpoints.UpdateConfig(ctx, req)
		return err
	}
	req := serving.EndpointCoreConfigInput{
		Name:              id,
		AutoCaptureConfig: config.AutoCaptureConfig,
		ServedEntities:    config.ServedEntities,
		TrafficConfig:     config.TrafficConfig,
		ServedModels:      config.ServedModels,
	}
	_, err := r.client.ServingEndpoints.UpdateConfig(ctx, req)
	if err != nil {
		return fmt.Errorf("failed to update config: %w", err)
	}
	return nil
}

func (r *ResourceModelServingEndpoint) updateNotifications(ctx context.Context, id string, notifications *serving.EmailNotifications) error {
	req := serving.UpdateInferenceEndpointNotifications{
		Name:               id,
		EmailNotifications: notifications,
	}
	_, err := r.client.ServingEndpoints.UpdateNotifications(ctx, req)
	if err != nil {
		return fmt.Errorf("failed to update notifications: %w", err)
	}
	return nil
}

func diffTags(currentTags, desiredTags []serving.EndpointTag) (addTags []serving.EndpointTag, deleteTags []string) {
	addTags = make([]serving.EndpointTag, 0)

	// build maps for easy lookup.
	currentTagsMap := make(map[string]string)
	desiredTagsMap := make(map[string]string)
	for _, tag := range currentTags {
		currentTagsMap[tag.Key] = tag.Value
	}
	for _, tag := range desiredTags {
		desiredTagsMap[tag.Key] = tag.Value
	}

	// Compute keys to be added.
	for key, desiredValue := range desiredTagsMap {
		v, ok := currentTagsMap[key]
		if !ok {
			addTags = append(addTags, serving.EndpointTag{
				Key:             key,
				Value:           desiredValue,
				ForceSendFields: nil,
			})
			continue
		}
		if v != desiredValue {
			addTags = append(addTags, serving.EndpointTag{
				Key:             key,
				Value:           desiredValue,
				ForceSendFields: nil,
			})
		}
	}

	// Compute keys to be deleted.
	for key := range currentTagsMap {
		if _, ok := desiredTagsMap[key]; !ok {
			deleteTags = append(deleteTags, key)
		}
	}

	return addTags, deleteTags
}

func (r *ResourceModelServingEndpoint) updateTags(ctx context.Context, id string, tags []serving.EndpointTag) error {
	endpoint, err := r.client.ServingEndpoints.GetByName(ctx, id)
	if err != nil {
		return err
	}

	addTags, deleteTags := diffTags(endpoint.Tags, tags)

	req := serving.PatchServingEndpointTags{
		Name:       id,
		AddTags:    addTags,
		DeleteTags: deleteTags,
	}
	_, err = r.client.ServingEndpoints.Patch(ctx, req)
	if err != nil {
		return fmt.Errorf("failed to update tags: %w", err)
	}
	return nil
}

func (r *ResourceModelServingEndpoint) DoUpdate(ctx context.Context, id string, config *serving.CreateServingEndpoint, changes *Changes) (*RefreshOutput, error) {
	var err error

	// Terraform makes these API calls sequentially. We do the same here.
	// It's an unknown as of 1st Dec 2025 if these APIs are safe to make in parallel. (we did not check)
	// https://github.com/databricks/terraform-provider-databricks/blob/c61a32300445f84efb2bb6827dee35e6e523f4ff/serving/resource_model_serving.go#L373
	if changes.HasChange("tags") {
		err = r.updateTags(ctx, id, config.Tags)
		if err != nil {
			return nil, err
		}
	}

	if changes.HasChange("ai_gateway") {
		err = r.updateAiGateway(ctx, id, config.AiGateway)
		if err != nil {
			return nil, err
		}
	}

	if changes.HasChange("config") {
		err = r.updateConfig(ctx, id, config.Config)
		if err != nil {
			return nil, err
		}
	}

	if changes.HasChange("email_notifications") {
		err = r.updateNotifications(ctx, id, config.EmailNotifications)
		if err != nil {
			return nil, err
		}
	}

	return nil, nil
}

func (r *ResourceModelServingEndpoint) DoDelete(ctx context.Context, id string) error {
	return r.client.ServingEndpoints.DeleteByName(ctx, id)
}

func (*ResourceModelServingEndpoint) FieldTriggers(_ bool) map[string]deployplan.ActionType {
	// TF implementation: https://github.com/databricks/terraform-provider-databricks/blob/6c106e8e7052bb2726148d66309fd460ed444236/mlflow/resource_mlflow_experiment.go#L22
	return map[string]deployplan.ActionType{
		"name":        deployplan.ActionTypeRecreate,
		"description": deployplan.ActionTypeRecreate, // description is immutable, can't be updated via API
		"config.auto_capture_config.catalog_name":      deployplan.ActionTypeRecreate,
		"config.auto_capture_config.schema_name":       deployplan.ActionTypeRecreate,
		"config.auto_capture_config.table_name_prefix": deployplan.ActionTypeRecreate,
		"route_optimized":                              deployplan.ActionTypeRecreate,
	}
}
