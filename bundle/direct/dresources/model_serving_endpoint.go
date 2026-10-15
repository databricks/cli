package dresources

import (
	"context"
	"fmt"
	"time"

	"github.com/databricks/cli/bundle/config/resources"
	"github.com/databricks/cli/bundle/direct/internal/retries"
	"github.com/databricks/cli/libs/sdkclients"
	"github.com/databricks/cli/libs/sdktypes/modelserving/v1"
	"github.com/databricks/cli/libs/structs/structpath"
	"github.com/databricks/cli/libs/utils"
	sdkmodelserving "github.com/databricks/sdk-go/modelserving/v1"
)

// Precalculated paths for HasChange checks
var (
	pathTags               = structpath.MustParsePath("tags")
	pathAiGateway          = structpath.MustParsePath("ai_gateway")
	pathConfig             = structpath.MustParsePath("config")
	pathEmailNotifications = structpath.MustParsePath("email_notifications")
	pathTelemetryConfig    = structpath.MustParsePath("telemetry_config")
)

type ResourceModelServingEndpoint struct {
	client *sdkclients.Clients
}

func (*ResourceModelServingEndpoint) New(client *sdkclients.Clients) *ResourceModelServingEndpoint {
	return &ResourceModelServingEndpoint{
		client: client,
	}
}

func (*ResourceModelServingEndpoint) PrepareState(input *resources.ModelServingEndpoint) *modelserving.CreateInferenceEndpointRequest {
	return &input.CreateInferenceEndpointRequest
}

// AutoCaptureConfig is the legacy inference-table API; the recommended replacement
// is AI Gateway inference tables. Bundles still expose it, so the conversion has to
// keep working until users have migrated.
//
//nolint:staticcheck
func autoCaptureConfigOutputToInput(output *modelserving.AutoCaptureConfig) *modelserving.AutoCaptureConfig {
	if output == nil {
		return nil
	}
	// state is output-only.
	return &modelserving.AutoCaptureConfig{
		CatalogName:     output.CatalogName,
		SchemaName:      output.SchemaName,
		TableNamePrefix: output.TableNamePrefix,
		Enabled:         output.Enabled,
		State:           nil, // Not set by DABs
	}
}

// servedEntitiesOutputToInput keeps the fields a user can set on a served entity;
// state, creator, creation_timestamp and foundation_model are output-only, and
// model_name/model_version belong to the legacy served_models field.
func servedEntitiesOutputToInput(output []modelserving.ServedModel) []modelserving.ServedModel {
	if output == nil {
		return nil
	}
	entities := make([]modelserving.ServedModel, len(output))
	for i, entity := range output {
		entities[i] = modelserving.ServedModel{
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
			BurstScalingEnabled:       entity.BurstScalingEnabled,
			CreationTimestamp:         nil, // Not set by DABs
			Creator:                   nil, // Not set by DABs
			FoundationModel:           nil, // Not set by DABs
			ModelName:                 nil, // Not set by DABs
			ModelVersion:              nil, // Not set by DABs
			State:                     nil, // Not set by DABs
		}
	}

	return entities
}

func configOutputToInput(output *modelserving.EndpointCoreConfigOutput) *modelserving.EndpointCoreConfig {
	if output == nil {
		return nil
	}
	return &modelserving.EndpointCoreConfig{
		AutoCaptureConfig: autoCaptureConfigOutputToInput(output.AutoCaptureConfig),
		ServedEntities:    servedEntitiesOutputToInput(output.ServedEntities),
		ServedModels:      nil,
		TrafficConfig:     output.TrafficConfig,
	}
}

func (*ResourceModelServingEndpoint) RemapState(state *ModelServingEndpointRemote) *modelserving.CreateInferenceEndpointRequest {
	return &modelserving.CreateInferenceEndpointRequest{
		AiGateway:          state.AiGateway,
		BudgetPolicyId:     state.BudgetPolicyId,
		Config:             state.Config,
		Description:        state.Description,
		EmailNotifications: state.EmailNotifications,
		Name:               state.Name,
		RouteOptimized:     state.RouteOptimized,
		Tags:               state.Tags,
		TelemetryConfig:    state.TelemetryConfig,

		// Rate limits are a deprecated field that are not returned by the API on GET calls. Thus we map them to nil.
		// TODO(shreyas): Add a warning when users try setting top level rate limits.
		RateLimits: nil,
	}
}

type ModelServingEndpointRemote struct {
	EndpointDetails *modelserving.InferenceEndpointDetailed `json:"endpoint_details"`
	EndpointId      string                                  `json:"endpoint_id"`

	// Fields mapped from EndpointDetails in DoRead so that RemapState is a direct copy
	// and these fields participate in normal drift detection.
	AiGateway          *modelserving.AiGatewayConfig    `json:"ai_gateway,omitempty"`
	BudgetPolicyId     *string                          `json:"budget_policy_id,omitempty"`
	Config             *modelserving.EndpointCoreConfig `json:"config,omitempty"`
	Description        *string                          `json:"description,omitempty"`
	EmailNotifications *modelserving.EmailNotifications `json:"email_notifications,omitempty"`
	Name               *string                          `json:"name,omitempty"`
	RouteOptimized     *bool                            `json:"route_optimized,omitempty"`
	Tags               []modelserving.EndpointTag       `json:"tags,omitempty"`
	TelemetryConfig    *modelserving.TelemetryConfig    `json:"telemetry_config,omitempty"`
}

func newModelServingEndpointRemote(resp *sdkmodelserving.InferenceEndpointDetailed) (*ModelServingEndpointRemote, error) {
	details, err := modelserving.InferenceEndpointDetailedFromSDK(resp)
	if err != nil {
		return nil, err
	}
	return &ModelServingEndpointRemote{
		EndpointDetails:    details,
		EndpointId:         utils.Deref(details.Id),
		AiGateway:          details.AiGateway,
		BudgetPolicyId:     details.BudgetPolicyId,
		Config:             configOutputToInput(details.Config),
		Description:        details.Description,
		EmailNotifications: details.EmailNotifications,
		Name:               details.Name,
		RouteOptimized:     details.RouteOptimized,
		Tags:               details.Tags,
		TelemetryConfig:    details.TelemetryConfig,
	}, nil
}

func (r *ResourceModelServingEndpoint) getEndpoint(ctx context.Context, name string) (*sdkmodelserving.InferenceEndpointDetailed, error) {
	return r.client.ServingEndpoints.GetInferenceEndpoint(ctx, sdkmodelserving.GetInferenceEndpointRequest{Name: new(name)})
}

func (r *ResourceModelServingEndpoint) DoRead(ctx context.Context, id string) (*ModelServingEndpointRemote, error) {
	endpoint, err := r.getEndpoint(ctx, id)
	if err != nil {
		return nil, err
	}
	return newModelServingEndpointRemote(endpoint)
}

func (r *ResourceModelServingEndpoint) DoCreate(ctx context.Context, config *modelserving.CreateInferenceEndpointRequest) (string, *ModelServingEndpointRemote, error) {
	request, err := modelserving.CreateInferenceEndpointRequestToSDK(config)
	if err != nil {
		return "", nil, err
	}
	waiter, err := r.client.ServingEndpoints.CreateInferenceEndpoint(ctx, *request)
	if err != nil {
		return "", nil, err
	}

	return waiter.GetName(), nil, nil
}

// waitForEndpointReady waits for the serving endpoint to be ready (not updating)
func (r *ResourceModelServingEndpoint) waitForEndpointReady(ctx context.Context, name string) (*ModelServingEndpointRemote, error) {
	details, err := retries.Poll(ctx, 35*time.Minute, func() (*sdkmodelserving.InferenceEndpointDetailed, *retries.Err) {
		details, err := r.getEndpoint(ctx, name)
		if err != nil {
			return nil, retries.Halt(err)
		}
		var status sdkmodelserving.InferenceEndpointState_ConfigUpdateState
		if details.State != nil {
			status = details.State.ConfigUpdate
		}
		statusMessage := fmt.Sprintf("current status: %s", status)
		switch status {
		case sdkmodelserving.InferenceEndpointState_ConfigUpdateState_NotUpdating:
			return details, nil
		case sdkmodelserving.InferenceEndpointState_ConfigUpdateState_UpdateFailed, sdkmodelserving.InferenceEndpointState_ConfigUpdateState_UpdateCanceled:
			return nil, retries.Halt(fmt.Errorf("failed to reach %s, got %s: %s",
				sdkmodelserving.InferenceEndpointState_ConfigUpdateState_NotUpdating, status, statusMessage))
		default:
			return nil, retries.Continues(statusMessage)
		}
	})
	if err != nil {
		return nil, err
	}
	return newModelServingEndpointRemote(details)
}

func (r *ResourceModelServingEndpoint) WaitAfterCreate(ctx context.Context, id string, config *modelserving.CreateInferenceEndpointRequest) (*ModelServingEndpointRemote, error) {
	return r.waitForEndpointReady(ctx, utils.Deref(config.Name))
}

func (r *ResourceModelServingEndpoint) WaitAfterUpdate(ctx context.Context, id string, config *modelserving.CreateInferenceEndpointRequest) (*ModelServingEndpointRemote, error) {
	return r.waitForEndpointReady(ctx, utils.Deref(config.Name))
}

func (r *ResourceModelServingEndpoint) updateAiGateway(ctx context.Context, id string, aiGateway *modelserving.AiGatewayConfig) error {
	req := modelserving.PutInferenceEndpointAiGatewayRequest{
		Name:                 new(id),
		FallbackConfig:       nil, // Not set by DABs
		Guardrails:           nil, // Not set by DABs
		InferenceTableConfig: nil, // Not set by DABs
		RateLimits:           nil, // Not set by DABs
		UsageTrackingConfig:  nil, // Not set by DABs
	}
	if aiGateway != nil {
		req.FallbackConfig = aiGateway.FallbackConfig
		req.Guardrails = aiGateway.Guardrails
		req.InferenceTableConfig = aiGateway.InferenceTableConfig
		req.RateLimits = aiGateway.RateLimits
		req.UsageTrackingConfig = aiGateway.UsageTrackingConfig
	}
	request, err := modelserving.PutInferenceEndpointAiGatewayRequestToSDK(&req)
	if err != nil {
		return err
	}
	_, err = r.client.ServingEndpoints.PutInferenceEndpointAiGateway(ctx, *request)
	if err != nil && aiGateway != nil {
		return fmt.Errorf("failed to update AI gateway: %w", err)
	}
	return err
}

// TODO(shreyas): Add acceptance tests for update and for unsetting. Will be done once we add selective updates for these fields.
func (r *ResourceModelServingEndpoint) updateConfig(ctx context.Context, id string, config *modelserving.EndpointCoreConfig) error {
	// A nil config unsets the config in the resource.
	req := modelserving.PutInferenceEndpointConfigRequest{
		Name:              new(id),
		AutoCaptureConfig: nil, // Not set by DABs
		ServedEntities:    nil, // Not set by DABs
		ServedModels:      nil, // Not set by DABs
		TrafficConfig:     nil, // Not set by DABs
	}
	if config != nil {
		req.AutoCaptureConfig = config.AutoCaptureConfig
		req.ServedEntities = config.ServedEntities
		req.TrafficConfig = config.TrafficConfig
		req.ServedModels = config.ServedModels
	}
	request, err := modelserving.PutInferenceEndpointConfigRequestToSDK(&req)
	if err != nil {
		return err
	}
	_, err = r.client.ServingEndpoints.PutInferenceEndpointConfig(ctx, *request)
	if err != nil && config != nil {
		return fmt.Errorf("failed to update config: %w", err)
	}
	return err
}

func (r *ResourceModelServingEndpoint) updateNotifications(ctx context.Context, id string, notifications *modelserving.EmailNotifications) error {
	request, err := modelserving.UpdateInferenceEndpointNotificationsRequestToSDK(&modelserving.UpdateInferenceEndpointNotificationsRequest{
		Name:               new(id),
		EmailNotifications: notifications,
	})
	if err != nil {
		return err
	}
	_, err = r.client.ServingEndpoints.UpdateInferenceEndpointNotifications(ctx, *request)
	if err != nil {
		return fmt.Errorf("failed to update notifications: %w", err)
	}
	return nil
}

// updateTelemetryConfig removes telemetry when telemetryConfig is nil.
func (r *ResourceModelServingEndpoint) updateTelemetryConfig(ctx context.Context, id string, telemetryConfig *modelserving.TelemetryConfig) error {
	request, err := modelserving.PatchInferenceEndpointTelemetryConfigRequestToSDK(&modelserving.PatchInferenceEndpointTelemetryConfigRequest{
		Name:            new(id),
		TelemetryConfig: telemetryConfig,
	})
	if err != nil {
		return err
	}
	_, err = r.client.ServingEndpoints.PatchInferenceEndpointTelemetryConfig(ctx, *request)
	if err != nil {
		return fmt.Errorf("failed to update telemetry config: %w", err)
	}
	return nil
}

func diffTags(currentTags, desiredTags []modelserving.EndpointTag) (addTags []modelserving.EndpointTag, deleteTags []string) {
	addTags = make([]modelserving.EndpointTag, 0)

	// build maps for easy lookup.
	currentTagsMap := make(map[string]string)
	desiredTagsMap := make(map[string]string)
	for _, tag := range currentTags {
		currentTagsMap[utils.Deref(tag.Key)] = utils.Deref(tag.Value)
	}
	for _, tag := range desiredTags {
		desiredTagsMap[utils.Deref(tag.Key)] = utils.Deref(tag.Value)
	}

	// Compute keys to be added.
	for key, desiredValue := range desiredTagsMap {
		v, ok := currentTagsMap[key]
		if !ok || v != desiredValue {
			addTags = append(addTags, modelserving.EndpointTag{
				Key:   new(key),
				Value: nonEmpty(desiredValue),
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

func (r *ResourceModelServingEndpoint) updateTags(ctx context.Context, id string, tags []modelserving.EndpointTag) error {
	resp, err := r.getEndpoint(ctx, id)
	if err != nil {
		return err
	}
	endpoint, err := modelserving.InferenceEndpointDetailedFromSDK(resp)
	if err != nil {
		return err
	}

	addTags, deleteTags := diffTags(endpoint.Tags, tags)

	request, err := modelserving.PatchInferenceEndpointTagsRequestToSDK(&modelserving.PatchInferenceEndpointTagsRequest{
		Name:       new(id),
		AddTags:    addTags,
		DeleteTags: deleteTags,
	})
	if err != nil {
		return err
	}
	_, err = r.client.ServingEndpoints.PatchInferenceEndpointTags(ctx, *request)
	if err != nil {
		return fmt.Errorf("failed to update tags: %w", err)
	}
	return nil
}

func (r *ResourceModelServingEndpoint) DoUpdate(ctx context.Context, id string, config *modelserving.CreateInferenceEndpointRequest, entry *PlanEntry) (*ModelServingEndpointRemote, error) {
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

	if entry.Changes.HasChange(pathTelemetryConfig) {
		// The telemetry API rejects endpoints with an update in progress.
		_, err = r.waitForEndpointReady(ctx, id)
		if err != nil {
			return nil, err
		}

		err = r.updateTelemetryConfig(ctx, id, config.TelemetryConfig)
		if err != nil {
			return nil, err
		}
	}

	return nil, nil
}

func (r *ResourceModelServingEndpoint) DoDelete(ctx context.Context, id string, _ *modelserving.CreateInferenceEndpointRequest) error {
	_, err := r.client.ServingEndpoints.DeleteInferenceEndpoint(ctx, sdkmodelserving.DeleteInferenceEndpointRequest{Name: new(id)})
	return err
}
