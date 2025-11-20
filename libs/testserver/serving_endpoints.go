package testserver

import (
	"encoding/json"
	"fmt"
	"sort"

	"github.com/databricks/databricks-sdk-go/service/serving"
)

func servedEntitiesInputToOutput(input []serving.ServedEntityInput) []serving.ServedEntityOutput {
	entities := make([]serving.ServedEntityOutput, len(input))
	for i, entity := range input {
		entities[i] = serving.ServedEntityOutput{
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
			ForceSendFields:           entity.ForceSendFields,
		}
	}
	return entities
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

func autoCaptureConfigInputToOutput(input *serving.AutoCaptureConfigInput) *serving.AutoCaptureConfigOutput {
	return &serving.AutoCaptureConfigOutput{
		CatalogName:     input.CatalogName,
		SchemaName:      input.SchemaName,
		TableNamePrefix: input.TableNamePrefix,
		Enabled:         input.Enabled,
		ForceSendFields: input.ForceSendFields,
	}
}

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

	// Check if endpoint with this name already exists
	if _, exists := s.ServingEndpoints[createReq.Name]; exists {
		return Response{
			StatusCode: 409,
			Body:       map[string]string{"error_code": "RESOURCE_ALREADY_EXISTS", "message": fmt.Sprintf("Serving endpoint with name %s already exists", createReq.Name)},
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
	}

	endpoint := serving.ServingEndpointDetailed{
		AiGateway:          createReq.AiGateway,
		BudgetPolicyId:     createReq.BudgetPolicyId,
		Config:             config,
		Creator:            s.CurrentUser().UserName,
		Description:        createReq.Description,
		EmailNotifications: createReq.EmailNotifications,
		Id:                 nextUUID(),
		Name:               createReq.Name,
		RouteOptimized:     createReq.RouteOptimized,
		Tags:               createReq.Tags,
		State: &serving.EndpointState{
			ConfigUpdate: serving.EndpointStateConfigUpdateNotUpdating,
		},
		ForceSendFields: createReq.ForceSendFields,
	}

	s.ServingEndpoints[createReq.Name] = endpoint

	return Response{
		Body: endpoint,
	}
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
	}

	endpoint.Config = config
	endpoint.State = &serving.EndpointState{
		ConfigUpdate: serving.EndpointStateConfigUpdateNotUpdating,
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
	keys := make([]string, 0, len(tagMap))
	for key := range tagMap {
		keys = append(keys, key)
	}
	sort.Strings(keys)
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
