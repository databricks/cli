package testserver

import (
	modelregistry "github.com/databricks/cli/libs/sdktypes/modelregistry/v1"
	"encoding/json"
	"fmt"
	"strconv"
)

func (s *FakeWorkspace) ModelRegistryCreateModel(req Request) any {
	defer s.LockUnlock()()

	var request modelregistry.CreateRegisteredModelRequest
	if err := json.Unmarshal(req.Body, &request); err != nil {
		return Response{
			StatusCode: 400,
			Body:       map[string]string{"message": fmt.Sprintf("Failed to parse request: %s", err)},
		}
	}

	// MLflow rejects an empty name; the fake would otherwise store a model under a key
	// nothing can look up.
	if request.Name == "" {
		return Response{
			StatusCode: 400,
			Body: map[string]string{
				"error_code": "INVALID_PARAMETER_VALUE",
				"message":    "Got an invalid name ''. Registered Model names cannot be empty strings.",
			},
		}
	}

	// Create the model with a numeric ID (matching real API behavior)
	modelId := strconv.FormatInt(nextID(), 10)
	model := modelregistry.RegisteredModel{
		Name:        request.Name,
		Description: request.Description,
		Tags:        request.Tags,
	}

	s.ModelRegistryModels[request.Name] = model
	s.ModelRegistryModelIDs[request.Name] = modelId

	return Response{
		Body: modelregistry.CreateRegisteredModelResponse{
			RegisteredModel: &model,
		},
	}
}

func (s *FakeWorkspace) ModelRegistryUpdateModel(req Request) any {
	defer s.LockUnlock()()

	var request modelregistry.UpdateRegisteredModelRequest
	if err := json.Unmarshal(req.Body, &request); err != nil {
		return Response{
			StatusCode: 400,
			Body:       map[string]string{"message": fmt.Sprintf("Failed to parse request: %s", err)},
		}
	}

	existingModel, ok := s.ModelRegistryModels[request.Name]
	if !ok {
		return Response{
			StatusCode: 404,
			Body:       map[string]string{"message": fmt.Sprintf("Model not found: %v", request.Name)},
		}
	}

	// Update the model
	existingModel.Description = request.Description
	s.ModelRegistryModels[request.Name] = existingModel

	return Response{
		Body: modelregistry.UpdateRegisteredModelResponse{
			RegisteredModel: &existingModel,
		},
	}
}

func (s *FakeWorkspace) ModelRegistryGetModel(req Request) any {
	defer s.LockUnlock()()

	name := req.URL.Query().Get("name")

	model, ok := s.ModelRegistryModels[name]
	if !ok {
		return Response{
			StatusCode: 404,
			Body:       map[string]string{"message": fmt.Sprintf("Model not found: %v", name)},
		}
	}

	return Response{
		Body: modelregistry.GetRegisteredModelDatabricksResponse{
			RegisteredModelDatabricks: &modelregistry.RegisteredModelDatabricks{
				Id:              s.ModelRegistryModelIDs[name],
				Name:            model.Name,
				Description:     model.Description,
				Tags:            model.Tags,
				ForceSendFields: model.ForceSendFields,
			},
		},
	}
}
