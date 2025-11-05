package testserver

import (
	"encoding/json"
	"fmt"

	"github.com/databricks/databricks-sdk-go/service/serving"
)

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
			config.ServedEntities = make([]serving.ServedEntityOutput, len(createReq.Config.ServedEntities))
			for i, entity := range createReq.Config.ServedEntities {
				config.ServedEntities[i] = serving.ServedEntityOutput{
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
		}

		// Convert ServedModelInput to ServedModelOutput
		if len(createReq.Config.ServedModels) > 0 {
			config.ServedModels = make([]serving.ServedModelOutput, len(createReq.Config.ServedModels))
			for i, model := range createReq.Config.ServedModels {
				config.ServedModels[i] = serving.ServedModelOutput{
					EnvironmentVars:           model.EnvironmentVars,
					InstanceProfileArn:        model.InstanceProfileArn,
					MaxProvisionedConcurrency: model.MaxProvisionedConcurrency,
					MinProvisionedConcurrency: model.MinProvisionedConcurrency,
					ModelName:                 model.ModelName,
					ModelVersion:              model.ModelVersion,
					Name:                      model.Name,
					ProvisionedModelUnits:     model.ProvisionedModelUnits,
					ScaleToZeroEnabled:        model.ScaleToZeroEnabled,
					WorkloadSize:              model.WorkloadSize,
					WorkloadType:              serving.ServingModelWorkloadType(model.WorkloadType),
					ForceSendFields:           model.ForceSendFields,
				}
			}
		}

		// Convert AutoCaptureConfig if present
		if createReq.Config.AutoCaptureConfig != nil {
			config.AutoCaptureConfig = &serving.AutoCaptureConfigOutput{
				CatalogName:     createReq.Config.AutoCaptureConfig.CatalogName,
				SchemaName:      createReq.Config.AutoCaptureConfig.SchemaName,
				TableNamePrefix: createReq.Config.AutoCaptureConfig.TableNamePrefix,
				Enabled:         createReq.Config.AutoCaptureConfig.Enabled,
				ForceSendFields: createReq.Config.AutoCaptureConfig.ForceSendFields,
			}
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
			config.ServedEntities = make([]serving.ServedEntityOutput, len(updateReq.ServedEntities))
			for i, entity := range updateReq.ServedEntities {
				config.ServedEntities[i] = serving.ServedEntityOutput{
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
		}

		// Convert ServedModelInput to ServedModelOutput
		if len(updateReq.ServedModels) > 0 {
			config.ServedModels = make([]serving.ServedModelOutput, len(updateReq.ServedModels))
			for i, model := range updateReq.ServedModels {
				config.ServedModels[i] = serving.ServedModelOutput{
					EnvironmentVars:           model.EnvironmentVars,
					InstanceProfileArn:        model.InstanceProfileArn,
					MaxProvisionedConcurrency: model.MaxProvisionedConcurrency,
					MinProvisionedConcurrency: model.MinProvisionedConcurrency,
					ModelName:                 model.ModelName,
					ModelVersion:              model.ModelVersion,
					Name:                      model.Name,
					ProvisionedModelUnits:     model.ProvisionedModelUnits,
					ScaleToZeroEnabled:        model.ScaleToZeroEnabled,
					WorkloadSize:              model.WorkloadSize,
					WorkloadType:              serving.ServingModelWorkloadType(model.WorkloadType),
					ForceSendFields:           model.ForceSendFields,
				}
			}
		}

		// Convert AutoCaptureConfig if present
		if updateReq.AutoCaptureConfig != nil {
			config.AutoCaptureConfig = &serving.AutoCaptureConfigOutput{
				CatalogName:     updateReq.AutoCaptureConfig.CatalogName,
				SchemaName:      updateReq.AutoCaptureConfig.SchemaName,
				TableNamePrefix: updateReq.AutoCaptureConfig.TableNamePrefix,
				Enabled:         updateReq.AutoCaptureConfig.Enabled,
				ForceSendFields: updateReq.AutoCaptureConfig.ForceSendFields,
			}
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
