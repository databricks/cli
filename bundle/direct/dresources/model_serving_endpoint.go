package dresources

import (
	"context"
	"fmt"
	"time"

	"github.com/databricks/cli/bundle/config/resources"
	"github.com/databricks/cli/libs/structs/structpath"
	"github.com/databricks/cli/libs/utils"
	"github.com/databricks/databricks-sdk-go"
	"github.com/databricks/databricks-sdk-go/service/serving"
)

// Precalculated paths for HasChange checks
var (
	pathTags               = structpath.MustParsePath("tags")
	pathAiGateway          = structpath.MustParsePath("ai_gateway")
	pathConfig             = structpath.MustParsePath("config")
	pathEmailNotifications = structpath.MustParsePath("email_notifications")
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

// AutoCaptureConfig is the legacy inference-table API; the recommended replacement
// is AI Gateway inference tables. Bundles still expose it, so the conversion has to
// keep working until users have migrated.
//
//nolint:staticcheck // SA1019: deprecated AutoCaptureConfig{Input,Output} kept for bundle config compatibility
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
			BurstScalingEnabled:       entity.BurstScalingEnabled,
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

func (*ResourceModelServingEndpoint) RemapState(state *ModelServingEndpointRemote) *serving.CreateServingEndpoint {
	return &serving.CreateServingEndpoint{
		AiGateway:          state.AiGateway,
		BudgetPolicyId:     state.BudgetPolicyId,
		Config:             state.Config,
		Description:        state.Description,
		EmailNotifications: state.EmailNotifications,
		Name:               state.Name,
		RouteOptimized:     state.RouteOptimized,
		Tags:               state.Tags,
		TelemetryConfig:    state.TelemetryConfig,
		ForceSendFields:    utils.FilterFields[serving.CreateServingEndpoint](state.EndpointDetails.ForceSendFields),

		// Rate limits are a deprecated field that are not returned by the API on GET calls. Thus we map them to nil.
		// TODO(shreyas): Add a warning when users try setting top level rate limits.
		RateLimits: nil,
	}
}

type ModelServingEndpointRemote struct {
	EndpointDetails *serving.ServingEndpointDetailed `json:"endpoint_details"`
	EndpointId      string                           `json:"endpoint_id"`

	// Fields mapped from EndpointDetails in DoRead so that RemapState is a direct copy
	// and these fields participate in normal drift detection.
	AiGateway          *serving.AiGatewayConfig         `json:"ai_gateway,omitempty"`
	BudgetPolicyId     string                           `json:"budget_policy_id,omitempty"`
	Config             *serving.EndpointCoreConfigInput `json:"config,omitempty"`
	Description        string                           `json:"description,omitempty"`
	EmailNotifications *serving.EmailNotifications      `json:"email_notifications,omitempty"`
	Name               string                           `json:"name,omitempty"`
	RouteOptimized     bool                             `json:"route_optimized,omitempty"`
	Tags               []serving.EndpointTag            `json:"tags,omitempty"`
	TelemetryConfig    *serving.TelemetryConfig         `json:"telemetry_config,omitempty"`
}

func newModelServingEndpointRemote(details *serving.ServingEndpointDetailed) *ModelServingEndpointRemote {
	return &ModelServingEndpointRemote{
		EndpointDetails:    details,
		EndpointId:         details.Id,
		AiGateway:          details.AiGateway,
		BudgetPolicyId:     details.BudgetPolicyId,
		Config:             configOutputToInput(details.Config),
		Description:        details.Description,
		EmailNotifications: details.EmailNotifications,
		Name:               details.Name,
		RouteOptimized:     details.RouteOptimized,
		Tags:               details.Tags,
		TelemetryConfig:    details.TelemetryConfig,
	}
}

func (r *ResourceModelServingEndpoint) DoRead(ctx context.Context, id string) (*ModelServingEndpointRemote, error) {
	endpoint, err := r.client.ServingEndpoints.GetByName(ctx, id)
	if err != nil {
		return nil, err
	}
	return newModelServingEndpointRemote(endpoint), nil
}

func (r *ResourceModelServingEndpoint) DoCreate(ctx context.Context, config *serving.CreateServingEndpoint) (string, *ModelServingEndpointRemote, error) {
	waiter, err := r.client.ServingEndpoints.Create(ctx, *config)
	if err != nil {
		return "", nil, err
	}

	return waiter.Response.Name, nil, nil
}

// waitForEndpointReady waits for the serving endpoint to be ready (not updating)
func (r *ResourceModelServingEndpoint) waitForEndpointReady(ctx context.Context, name string) (*ModelServingEndpointRemote, error) {
	details, err := r.client.ServingEndpoints.WaitGetServingEndpointNotUpdating(ctx, name, 35*time.Minute, nil)
	if err != nil {
		return nil, err
	}
	return newModelServingEndpointRemote(details), nil
}

func (r *ResourceModelServingEndpoint) WaitAfterCreate(ctx context.Context, id string, config *serving.CreateServingEndpoint) (*ModelServingEndpointRemote, error) {
	return r.waitForEndpointReady(ctx, config.Name)
}

func (r *ResourceModelServingEndpoint) WaitAfterUpdate(ctx context.Context, id string, config *serving.CreateServingEndpoint) (*ModelServingEndpointRemote, error) {
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

func (r *ResourceModelServingEndpoint) DoUpdate(ctx context.Context, id string, config *serving.CreateServingEndpoint, entry *PlanEntry) (*ModelServingEndpointRemote, error) {
	var err error

	// Terraform makes these API calls sequentially. We do the same here.
	// It's an unknown as of 1st Dec 2025 if these APIs are safe to make in parallel. (we did not check)
	// https://github.com/databricks/terraform-provider-databricks/blob/c61a32300445f84efb2bb6827dee35e6e523f4ff/serving/resource_model_serving.go#L373
	if entry.Changes.HasChange(pathTags) {
		err = r.updateTags(ctx, id, config.Tags)
		if err != nil {
			return nil, err
		}
	}

	if entry.Changes.HasChange(pathAiGateway) {
		err = r.updateAiGateway(ctx, id, config.AiGateway)
		if err != nil {
			return nil, err
		}
	}

	if entry.Changes.HasChange(pathConfig) {
		err = r.updateConfig(ctx, id, config.Config)
		if err != nil {
			return nil, err
		}
	}

	if entry.Changes.HasChange(pathEmailNotifications) {
		err = r.updateNotifications(ctx, id, config.EmailNotifications)
		if err != nil {
			return nil, err
		}
	}

	return nil, nil
}

func (r *ResourceModelServingEndpoint) DoDelete(ctx context.Context, id string, _ *serving.CreateServingEndpoint) error {
	return r.client.ServingEndpoints.DeleteByName(ctx, id)
}
