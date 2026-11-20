package dresources

import (
	"context"
	"fmt"
	"time"

	"github.com/databricks/cli/bundle/config/resources"
	"github.com/databricks/cli/libs/structs/structcopy"
	"github.com/databricks/cli/libs/structs/structpath"
	"github.com/databricks/databricks-sdk-go"
	"github.com/databricks/databricks-sdk-go/marshal"
	"github.com/databricks/databricks-sdk-go/service/serving"
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

// The API reports a config with types of its own (the *Output ones); these convert them
// to the types a request takes.
var (
	//nolint:staticcheck // SA1019: deprecated AutoCaptureConfig{Input,Output} kept for bundle config compatibility
	toAutoCaptureConfigInput = newCopier[serving.AutoCaptureConfigOutput, serving.AutoCaptureConfigInput]()
	toServedEntityInput      = newCopier[serving.ServedEntityOutput, serving.ServedEntityInput]()

	// AutoCaptureConfig and ServedEntities are converted on their own. ServedModels, which
	// served_entities deprecates, is left unset.
	toEndpointCoreConfigInput = newCopier[serving.EndpointCoreConfigOutput, serving.EndpointCoreConfigInput](
		structcopy.Skip("AutoCaptureConfig", "ServedEntities", "ServedModels"),
	)
)

// AutoCaptureConfig is the legacy inference-table API; the recommended replacement
// is AI Gateway inference tables. Bundles still expose it, so the conversion has to
// keep working until users have migrated.
//
//nolint:staticcheck // SA1019: deprecated AutoCaptureConfig{Input,Output} kept for bundle config compatibility
func autoCaptureConfigOutputToInput(output *serving.AutoCaptureConfigOutput) *serving.AutoCaptureConfigInput {
	if output == nil {
		return nil
	}
	return toAutoCaptureConfigInput.Copy(output)
}

func servedEntitiesOutputToInput(output []serving.ServedEntityOutput) []serving.ServedEntityInput {
	entities := make([]serving.ServedEntityInput, len(output))
	for i := range output {
		entities[i] = *toServedEntityInput.Copy(&output[i])
	}

	return entities
}

func configOutputToInput(output *serving.EndpointCoreConfigOutput) *serving.EndpointCoreConfigInput {
	if output == nil {
		return nil
	}
	config := toEndpointCoreConfigInput.Copy(output)
	config.AutoCaptureConfig = autoCaptureConfigOutputToInput(output.AutoCaptureConfig)
	config.ServedEntities = servedEntitiesOutputToInput(output.ServedEntities)
	return config
}

// Rate limits are a deprecated field that are not returned by the API on GET calls, so the
// remote state has no value for them.
// TODO(shreyas): Add a warning when users try setting top level rate limits.
var toCreateServingEndpoint = newCopier[ModelServingEndpointRemote, serving.CreateServingEndpoint]()

func (*ResourceModelServingEndpoint) RemapState(state *ModelServingEndpointRemote) *serving.CreateServingEndpoint {
	return toCreateServingEndpoint.Copy(state)
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

	// ForceSendFields surfaces EndpointDetails.ForceSendFields at the root so RemapState is a
	// plain field copy rather than reaching into the nested EndpointDetails struct.
	ForceSendFields []string `json:"-"`
}

// Custom marshalers so the SDK's ForceSendFields convention is honored on the wrapper and its
// extra fields survive a JSON state round-trip (mirrors VectorSearchEndpointRemote).
func (s *ModelServingEndpointRemote) UnmarshalJSON(b []byte) error {
	return marshal.Unmarshal(b, s)
}

func (s ModelServingEndpointRemote) MarshalJSON() ([]byte, error) {
	return marshal.Marshal(s)
}

// Config is converted on its own, see configOutputToInput.
var toModelServingEndpointRemote = newCopier[serving.ServingEndpointDetailed, ModelServingEndpointRemote](
	structcopy.Skip("Config"),
)

func newModelServingEndpointRemote(details *serving.ServingEndpointDetailed) *ModelServingEndpointRemote {
	remote := toModelServingEndpointRemote.Copy(details)
	remote.EndpointDetails = details
	remote.EndpointId = details.Id
	remote.Config = configOutputToInput(details.Config)
	return remote
}

func (r *ResourceModelServingEndpoint) DoRead(ctx context.Context, id string) (*ModelServingEndpointRemote, error) {
	endpoint, err := r.client.ServingEndpoints.GetByName(ctx, id)
	if err != nil {
		return nil, err
	}
	return newModelServingEndpointRemote(endpoint), nil
}

func (r *ResourceModelServingEndpoint) DoCreate(ctx context.Context, config *serving.CreateServingEndpoint) (string, *ModelServingEndpointRemote, error) {
	create := func(ctx context.Context) (*serving.ServingEndpointDetailed, error) {
		waiter, err := r.client.ServingEndpoints.Create(ctx, *config)
		if err != nil {
			return nil, err
		}
		return waiter.Response, nil
	}
	start := time.Now()
	endpoint, err := create(ctx)
	if err != nil {
		// Endpoint names are chosen by the bundle and unique in the workspace. WaitAfterCreate waits
		// for a recovered endpoint as it does for one the first request created.
		endpoint, err = recoverCreate(ctx, createRecovery[serving.ServingEndpointDetailed]{
			name:  "serving endpoint " + config.Name,
			start: start,
			get: func(ctx context.Context) (*serving.ServingEndpointDetailed, error) {
				return r.client.ServingEndpoints.Get(ctx, serving.GetServingEndpointRequest{Name: config.Name})
			},
			createTime: func(found *serving.ServingEndpointDetailed) (time.Time, bool) {
				return time.UnixMilli(found.CreationTimestamp), found.CreationTimestamp != 0
			},
			create: create,
		}, err)
		if err != nil {
			return "", nil, err
		}
	}

	return endpoint.Name, nil, nil
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

var toPutAiGateway = newCopier[serving.AiGatewayConfig, serving.PutAiGatewayRequest]()

func (r *ResourceModelServingEndpoint) updateAiGateway(ctx context.Context, id string, aiGateway *serving.AiGatewayConfig) error {
	if aiGateway == nil {
		var req serving.PutAiGatewayRequest
		req.Name = id
		_, err := r.client.ServingEndpoints.PutAiGateway(ctx, req)
		return err
	}

	req := toPutAiGateway.Copy(aiGateway)
	req.Name = id
	_, err := r.client.ServingEndpoints.PutAiGateway(ctx, *req)
	if err != nil {
		return fmt.Errorf("failed to update AI gateway: %w", err)
	}
	return nil
}

// TODO(shreyas): Add acceptance tests for update and for unsetting. Will be done once we add selective updates for these fields.
func (r *ResourceModelServingEndpoint) updateConfig(ctx context.Context, id string, config *serving.EndpointCoreConfigInput) error {
	if config == nil {
		// Unset config in resource.
		var req serving.EndpointCoreConfigInput
		req.Name = id
		_, err := r.client.ServingEndpoints.UpdateConfig(ctx, req)
		return err
	}
	req := *config
	req.Name = id
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

// updateTelemetryConfig removes telemetry when telemetryConfig is nil.
func (r *ResourceModelServingEndpoint) updateTelemetryConfig(ctx context.Context, id string, telemetryConfig *serving.TelemetryConfig) error {
	req := serving.PatchTelemetryConfigRequest{
		Name:            id,
		TelemetryConfig: telemetryConfig,
	}
	_, err := r.client.ServingEndpoints.PatchTelemetryConfig(ctx, req)
	if err != nil {
		return fmt.Errorf("failed to update telemetry config: %w", err)
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
				Key:   key,
				Value: desiredValue,
			})
			continue
		}
		if v != desiredValue {
			addTags = append(addTags, serving.EndpointTag{
				Key:   key,
				Value: desiredValue,
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

func (r *ResourceModelServingEndpoint) DoDelete(ctx context.Context, id string, _ *serving.CreateServingEndpoint) error {
	return r.client.ServingEndpoints.DeleteByName(ctx, id)
}
