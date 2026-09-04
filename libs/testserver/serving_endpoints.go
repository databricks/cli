package testserver

import (
	"encoding/json"
	"fmt"
	"maps"
	"regexp"
	"slices"

	"github.com/databricks/databricks-sdk-go/service/serving"
)

func servedEntitiesInputToOutput(input []serving.ServedEntityInput) []serving.ServedEntityOutput {
	entities := make([]serving.ServedEntityOutput, len(input))
	for i, entity := range input {
		// Mirror the backend: burst_scaling_enabled is not echoed on GET (not
		// copied here) and external_model secrets are never returned in plaintext.
		entities[i] = serving.ServedEntityOutput{
			EntityName:                entity.EntityName,
			EntityVersion:             entity.EntityVersion,
			EnvironmentVars:           entity.EnvironmentVars,
			ExternalModel:             clearExternalModelSecrets(entity.ExternalModel),
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
			ForceSendFields:           entity.ForceSendFields,
		}
	}
	return entities
}

// applyTelemetryConfig consumes table_names and discards configs that identify no profile.
func applyTelemetryConfig(previous, config *serving.TelemetryConfig) *serving.TelemetryConfig {
	if config == nil {
		return nil
	}
	if config.TableNames == nil && config.TelemetryProfileId == "" {
		return previous
	}

	applied := serving.TelemetryConfig{TelemetryProfileId: config.TelemetryProfileId}
	if applied.TelemetryProfileId == "" {
		// Do not reuse previous: table_names provisions a new profile.
		applied.TelemetryProfileId = nextUUID()
	}
	if config.InferenceTableConfig != nil {
		inferenceTable := *config.InferenceTableConfig
		// The backend names the payload table after the logs table it was given.
		if config.TableNames != nil {
			inferenceTable.Name = config.TableNames.LogsTable + "_payload"
		}
		applied.InferenceTableConfig = &inferenceTable
	}

	return &applied
}

// telemetrySupported returns the unsupported endpoint type when telemetry cannot be applied.
func telemetrySupported(endpoint serving.ServingEndpointDetailed) (string, bool) {
	if endpoint.Config == nil || len(endpoint.Config.ServedEntities) == 0 {
		return "NO_CONFIG", false
	}
	for _, entity := range endpoint.Config.ServedEntities {
		if entity.ExternalModel == nil && entity.EntityName != "" {
			return "", true
		}
	}
	return "EXTERNAL_MODELS", false
}

// clearExternalModelSecrets mirrors the backend, which persists the *_plaintext
// API keys as secrets and never returns them on GET.
func clearExternalModelSecrets(em *serving.ExternalModel) *serving.ExternalModel {
	if em == nil {
		return nil
	}
	out := *em
	if c := out.Ai21labsConfig; c != nil {
		cc := *c
		cc.Ai21labsApiKeyPlaintext = ""
		out.Ai21labsConfig = &cc
	}
	if c := out.AmazonBedrockConfig; c != nil {
		cc := *c
		cc.AwsAccessKeyIdPlaintext = ""
		cc.AwsSecretAccessKeyPlaintext = ""
		out.AmazonBedrockConfig = &cc
	}
	if c := out.AnthropicConfig; c != nil {
		cc := *c
		cc.AnthropicApiKeyPlaintext = ""
		out.AnthropicConfig = &cc
	}
	if c := out.CohereConfig; c != nil {
		cc := *c
		cc.CohereApiKeyPlaintext = ""
		out.CohereConfig = &cc
	}
	if c := out.CustomProviderConfig; c != nil {
		cc := *c
		if a := cc.ApiKeyAuth; a != nil {
			aa := *a
			aa.ValuePlaintext = ""
			cc.ApiKeyAuth = &aa
		}
		if b := cc.BearerTokenAuth; b != nil {
			bb := *b
			bb.TokenPlaintext = ""
			cc.BearerTokenAuth = &bb
		}
		out.CustomProviderConfig = &cc
	}
	if c := out.DatabricksModelServingConfig; c != nil {
		cc := *c
		cc.DatabricksApiTokenPlaintext = ""
		out.DatabricksModelServingConfig = &cc
	}
	if c := out.GoogleCloudVertexAiConfig; c != nil {
		cc := *c
		cc.PrivateKeyPlaintext = ""
		out.GoogleCloudVertexAiConfig = &cc
	}
	if c := out.OpenaiConfig; c != nil {
		cc := *c
		cc.OpenaiApiKeyPlaintext = ""
		cc.MicrosoftEntraClientSecretPlaintext = ""
		out.OpenaiConfig = &cc
	}
	if c := out.PalmConfig; c != nil {
		cc := *c
		cc.PalmApiKeyPlaintext = ""
		out.PalmConfig = &cc
	}
	return &out
}

func servedModelsInputToOutput(input []serving.ServedModelInput) []serving.ServedModelOutput {
	models := make([]serving.ServedModelOutput, len(input))
	for i, model := range input {
		models[i] = serving.ServedModelOutput{
			ModelName:                 model.ModelName,
			ModelVersion:              model.ModelVersion,
			EnvironmentVars:           model.EnvironmentVars,
			InstanceProfileArn:        model.InstanceProfileArn,
			MaxProvisionedConcurrency: model.MaxProvisionedConcurrency,
			MinProvisionedConcurrency: model.MinProvisionedConcurrency,
			ProvisionedModelUnits:     model.ProvisionedModelUnits,
			ScaleToZeroEnabled:        model.ScaleToZeroEnabled,
			WorkloadSize:              model.WorkloadSize,
			WorkloadType:              serving.ServingModelWorkloadType(model.WorkloadType),
			ForceSendFields:           model.ForceSendFields,
		}
	}
	return models
}

// defaultTrafficConfig mirrors the backend: when the user does not specify a
// traffic_config, the endpoint defaults to routing 100% of traffic to its single
// served entity, and this default is echoed back on GET. This is what makes
// traffic_config a backend-managed field the bundle must not treat as drift; the
// fake has to reproduce it or the persistent-drift regression is invisible locally.
func defaultTrafficConfig(config *serving.EndpointCoreConfigOutput) {
	if config == nil || config.TrafficConfig != nil {
		return
	}
	var names []string
	for _, e := range config.ServedEntities {
		names = append(names, e.Name)
	}
	for _, m := range config.ServedModels {
		names = append(names, m.Name)
	}
	// The backend requires an explicit traffic_config when there is more than one
	// served entity, so only the single-entity default is well-defined here.
	if len(names) != 1 {
		return
	}
	config.TrafficConfig = &serving.TrafficConfig{
		Routes: []serving.Route{{
			ServedEntityName:  names[0],
			ServedModelName:   names[0],
			TrafficPercentage: 100,
		}},
	}
}

// AutoCaptureConfig is the legacy inference-table API; testserver mirrors
// the production conversion until callers migrate to AI Gateway inference tables.
//
//nolint:staticcheck // SA1019: deprecated AutoCaptureConfig{Input,Output} kept for bundle config compatibility
func autoCaptureConfigInputToOutput(input *serving.AutoCaptureConfigInput) *serving.AutoCaptureConfigOutput {
	return &serving.AutoCaptureConfigOutput{
		CatalogName:     input.CatalogName,
		SchemaName:      input.SchemaName,
		TableNamePrefix: input.TableNamePrefix,
		Enabled:         input.Enabled,
		ForceSendFields: input.ForceSendFields,
	}
}

// servingEndpointNamePattern and servingEndpointNameMessage are the backend's own constraint and
// wording for an endpoint name. An empty name fails the pattern, which is how clearing one is refused.
var servingEndpointNamePattern = regexp.MustCompile(`^[a-zA-Z0-9]([a-zA-Z0-9_-]{0,61}[a-zA-Z0-9])?$`)

const servingEndpointNameMessage = "Endpoint name must be maximum 63 characters, and alphanumeric with hyphens and underscores allowed in between."

func (s *FakeWorkspace) ServingEndpointCreate(req Request) Response {
	defer s.LockUnlock()()

	var createReq serving.CreateServingEndpoint
	err := json.Unmarshal(req.Body, &createReq)
	if err != nil {
		return Response{
			Body:       fmt.Sprintf("cannot unmarshal request body: %s", err),
			StatusCode: 400,
		}
	}

	// The name is validated before the collision check, so an empty one is refused on its own terms
	// rather than colliding with a previous empty-named endpoint.
	if !servingEndpointNamePattern.MatchString(createReq.Name) {
		return Response{
			StatusCode: 400,
			Body: map[string]string{
				"error_code": "INVALID_PARAMETER_VALUE",
				"message":    servingEndpointNameMessage,
			},
		}
	}

	// Check if endpoint with this name already exists
	if _, exists := s.ServingEndpoints[createReq.Name]; exists {
		return Response{
			StatusCode: 409,
			Body:       map[string]string{"error_code": "RESOURCE_ALREADY_EXISTS", "message": fmt.Sprintf("Serving endpoint with name %s already exists", createReq.Name)},
		}
	}

	// An endpoint created without a config block has nothing for these to apply to, so the API
	// rejects them outright rather than storing them.
	if createReq.Config == nil {
		for _, rejected := range []struct {
			field string
			set   bool
		}{
			{"ai_gateway", createReq.AiGateway != nil},
			{"rate_limits", len(createReq.RateLimits) > 0},
		} {
			if rejected.set {
				return Response{
					StatusCode: 400,
					Body: map[string]string{
						"error_code": "INVALID_PARAMETER_VALUE",
						"message":    "Cannot specify " + rejected.field + " when creating endpoints without a config.",
					},
				}
			}
		}
	}

	// Convert config to output format
	var config *serving.EndpointCoreConfigOutput
	if createReq.Config != nil {
		config = &serving.EndpointCoreConfigOutput{
			TrafficConfig: createReq.Config.TrafficConfig,
		}

		// Convert ServedEntityInput to ServedEntityOutput
		if len(createReq.Config.ServedEntities) > 0 {
			config.ServedEntities = servedEntitiesInputToOutput(createReq.Config.ServedEntities)
		}

		// Convert ServedModelInput to ServedModelOutput
		if len(createReq.Config.ServedModels) > 0 {
			config.ServedModels = servedModelsInputToOutput(createReq.Config.ServedModels)
		}

		// Convert AutoCaptureConfig if present
		if createReq.Config.AutoCaptureConfig != nil {
			config.AutoCaptureConfig = autoCaptureConfigInputToOutput(createReq.Config.AutoCaptureConfig)
		}

		defaultTrafficConfig(config)
	}

	now := nowMilli()
	endpoint := serving.ServingEndpointDetailed{
		AiGateway:         createReq.AiGateway,
		BudgetPolicyId:    createReq.BudgetPolicyId,
		Config:            config,
		CreationTimestamp: now,
		Creator:           s.CurrentUser().UserName,
		Description:       createReq.Description,
		// Not carried over from the create: a real workspace does not echo the notifications a
		// create asked for (aws, 2026-09 -- the field is absent from a GET straight afterwards),
		// so the endpoint drifts from the moment it exists. An update does apply them, which is
		// why they are set in the update handler below and not here.
		EmailNotifications:   nil,
		Id:                   nextUUID(),
		LastUpdatedTimestamp: now,
		Name:                 createReq.Name,
		PermissionLevel:      serving.ServingEndpointDetailedPermissionLevelCanManage,
		RouteOptimized:       createReq.RouteOptimized,
		Tags:                 createReq.Tags,
		TelemetryConfig:      applyTelemetryConfig(nil, createReq.TelemetryConfig),
		State: &serving.EndpointState{
			ConfigUpdate: serving.EndpointStateConfigUpdateNotUpdating,
			Ready:        serving.EndpointStateReadyNotReady,
		},
		// Force-send Description so an empty value serializes as "", matching the
		// real backend which always echoes the field back on GET.
		ForceSendFields: append(createReq.ForceSendFields, "PermissionLevel", "RouteOptimized", "Description"),
	}

	// Create drops unsupported telemetry, while the telemetry API rejects it.
	if _, ok := telemetrySupported(endpoint); !ok {
		endpoint.TelemetryConfig = nil
	}

	s.ServingEndpoints[createReq.Name] = endpoint

	return Response{
		Body: endpoint,
	}
}

// ServingEndpointGet reports an in-progress update once before settling it.
func (s *FakeWorkspace) ServingEndpointGet(name string) Response {
	defer s.LockUnlock()()

	endpoint, exists := s.ServingEndpoints[name]
	if !exists {
		return Response{
			StatusCode: 404,
			Body:       map[string]string{"message": fmt.Sprintf("Resource %T not found: %v", endpoint, name)},
		}
	}

	if endpointUpdating(endpoint) {
		settled := endpoint
		settled.State = &serving.EndpointState{
			ConfigUpdate: serving.EndpointStateConfigUpdateNotUpdating,
			Ready:        endpoint.State.Ready,
		}
		s.ServingEndpoints[name] = settled

		// By default this response stays IN_PROGRESS and only the stored copy settles, so
		// the caller has to poll at least once. SettleAsyncImmediately reports the settled
		// state right away instead.
		if s.SettleAsyncImmediately {
			endpoint = settled
		}
	}

	return Response{Body: endpoint}
}

// endpointUpdating reports whether a config update is in progress.
func endpointUpdating(endpoint serving.ServingEndpointDetailed) bool {
	return endpoint.State != nil && endpoint.State.ConfigUpdate == serving.EndpointStateConfigUpdateInProgress
}

func (s *FakeWorkspace) ServingEndpointUpdate(req Request, name string) Response {
	defer s.LockUnlock()()

	var updateReq serving.EndpointCoreConfigInput
	err := json.Unmarshal(req.Body, &updateReq)
	if err != nil {
		return Response{
			Body:       fmt.Sprintf("cannot unmarshal request body: %s", err),
			StatusCode: 400,
		}
	}

	endpoint, exists := s.ServingEndpoints[name]
	if !exists {
		return Response{
			StatusCode: 404,
			Body:       map[string]string{"error_code": "RESOURCE_DOES_NOT_EXIST", "message": fmt.Sprintf("Serving endpoint with name %s not found", name)},
		}
	}

	// Convert config to output format
	var config *serving.EndpointCoreConfigOutput
	if updateReq.ServedEntities != nil || updateReq.ServedModels != nil || updateReq.TrafficConfig != nil {
		config = &serving.EndpointCoreConfigOutput{
			TrafficConfig: updateReq.TrafficConfig,
		}

		// Convert ServedEntityInput to ServedEntityOutput
		if len(updateReq.ServedEntities) > 0 {
			config.ServedEntities = servedEntitiesInputToOutput(updateReq.ServedEntities)
		}

		// Convert ServedModelInput to ServedModelOutput
		if len(updateReq.ServedModels) > 0 {
			config.ServedModels = servedModelsInputToOutput(updateReq.ServedModels)
		}

		// Convert AutoCaptureConfig if present
		if updateReq.AutoCaptureConfig != nil {
			config.AutoCaptureConfig = autoCaptureConfigInputToOutput(updateReq.AutoCaptureConfig)
		}

		defaultTrafficConfig(config)
	}

	endpoint.Config = config
	endpoint.LastUpdatedTimestamp = nowMilli()
	// Keep the update in progress until GET observes it.
	endpoint.State = &serving.EndpointState{
		ConfigUpdate: serving.EndpointStateConfigUpdateInProgress,
		Ready:        serving.EndpointStateReadyNotReady,
	}

	s.ServingEndpoints[name] = endpoint

	return Response{
		Body: endpoint,
	}
}

func (s *FakeWorkspace) ServingEndpointPutAiGateway(req Request, name string) Response {
	defer s.LockUnlock()()

	var putReq serving.PutAiGatewayRequest
	err := json.Unmarshal(req.Body, &putReq)
	if err != nil {
		return Response{
			Body:       fmt.Sprintf("cannot unmarshal request body: %s", err),
			StatusCode: 400,
		}
	}

	endpoint, exists := s.ServingEndpoints[name]
	if !exists {
		return Response{
			StatusCode: 404,
			Body:       map[string]string{"error_code": "RESOURCE_DOES_NOT_EXIST", "message": fmt.Sprintf("Serving endpoint with name %s not found", name)},
		}
	}

	// Update AI gateway config
	if putReq.FallbackConfig != nil || putReq.Guardrails != nil || putReq.InferenceTableConfig != nil || putReq.RateLimits != nil || putReq.UsageTrackingConfig != nil {
		endpoint.AiGateway = &serving.AiGatewayConfig{
			FallbackConfig:       putReq.FallbackConfig,
			Guardrails:           putReq.Guardrails,
			InferenceTableConfig: putReq.InferenceTableConfig,
			RateLimits:           putReq.RateLimits,
			UsageTrackingConfig:  putReq.UsageTrackingConfig,
		}
	} else {
		// Unset AI gateway if no fields provided
		endpoint.AiGateway = nil
	}

	s.ServingEndpoints[name] = endpoint

	return Response{
		Body: endpoint.AiGateway,
	}
}

func (s *FakeWorkspace) ServingEndpointUpdateNotifications(req Request, name string) Response {
	defer s.LockUnlock()()

	var updateReq serving.UpdateInferenceEndpointNotifications
	err := json.Unmarshal(req.Body, &updateReq)
	if err != nil {
		return Response{
			Body:       fmt.Sprintf("cannot unmarshal request body: %s", err),
			StatusCode: 400,
		}
	}

	endpoint, exists := s.ServingEndpoints[name]
	if !exists {
		return Response{
			StatusCode: 404,
			Body:       map[string]string{"error_code": "RESOURCE_DOES_NOT_EXIST", "message": fmt.Sprintf("Serving endpoint with name %s not found", name)},
		}
	}

	endpoint.EmailNotifications = updateReq.EmailNotifications
	s.ServingEndpoints[name] = endpoint

	return Response{
		Body: endpoint,
	}
}

// ServingEndpointPatchTelemetryConfig applies telemetry after validating endpoint state.
func (s *FakeWorkspace) ServingEndpointPatchTelemetryConfig(req Request, name string) Response {
	defer s.LockUnlock()()

	var patchReq serving.PatchTelemetryConfigRequest
	err := json.Unmarshal(req.Body, &patchReq)
	if err != nil {
		return Response{
			Body:       fmt.Sprintf("cannot unmarshal request body: %s", err),
			StatusCode: 400,
		}
	}

	endpoint, exists := s.ServingEndpoints[name]
	if !exists {
		return Response{
			StatusCode: 404,
			Body:       map[string]string{"error_code": "RESOURCE_DOES_NOT_EXIST", "message": fmt.Sprintf("Serving endpoint with name %s not found", name)},
		}
	}

	if endpointType, ok := telemetrySupported(endpoint); !ok {
		return Response{
			StatusCode: 400,
			Body: map[string]string{
				"error_code": "INVALID_PARAMETER_VALUE",
				"message":    fmt.Sprintf("Telemetry configuration is not supported for endpoint type '%s'. This API only supports endpoints with custom served models.", endpointType),
			},
		}
	}

	// The telemetry API returns 409 while another update is in progress.
	if endpointUpdating(endpoint) {
		return Response{
			StatusCode: 409,
			Body: map[string]string{
				"error_code": "RESOURCE_CONFLICT",
				"message":    fmt.Sprintf("Endpoint %s is currently updating. Wait for the update to complete before updating its telemetry configuration.", name),
			},
		}
	}

	// An omitted telemetry_config removes the configuration from the endpoint.
	endpoint.TelemetryConfig = applyTelemetryConfig(endpoint.TelemetryConfig, patchReq.TelemetryConfig)
	endpoint.LastUpdatedTimestamp = nowMilli()
	s.ServingEndpoints[name] = endpoint

	return Response{
		Body: endpoint,
	}
}

func (s *FakeWorkspace) ServingEndpointPatchTags(req Request, name string) Response {
	defer s.LockUnlock()()

	var patchReq serving.PatchServingEndpointTags
	err := json.Unmarshal(req.Body, &patchReq)
	if err != nil {
		return Response{
			Body:       fmt.Sprintf("cannot unmarshal request body: %s", err),
			StatusCode: 400,
		}
	}

	endpoint, exists := s.ServingEndpoints[name]
	if !exists {
		return Response{
			StatusCode: 404,
			Body:       map[string]string{"error_code": "RESOURCE_DOES_NOT_EXIST", "message": fmt.Sprintf("Serving endpoint with name %s not found", name)},
		}
	}

	// Build map of current tags
	tagMap := make(map[string]string)
	for _, tag := range endpoint.Tags {
		tagMap[tag.Key] = tag.Value
	}

	// Add or update tags
	for _, tag := range patchReq.AddTags {
		tagMap[tag.Key] = tag.Value
	}

	// Delete tags
	for _, key := range patchReq.DeleteTags {
		delete(tagMap, key)
	}

	// Convert back to slice sorted by key for stable output
	tags := make([]serving.EndpointTag, 0, len(tagMap))
	keys := slices.Sorted(maps.Keys(tagMap))
	for _, key := range keys {
		tags = append(tags, serving.EndpointTag{Key: key, Value: tagMap[key]})
	}

	endpoint.Tags = tags
	s.ServingEndpoints[name] = endpoint

	// Return the tags as EndpointTags struct, not as array
	return Response{
		Body: serving.EndpointTags{
			Tags: tags,
		},
	}
}
