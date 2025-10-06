package testserver

import (
	"encoding/json"
	"fmt"

	"github.com/databricks/databricks-sdk-go/service/ml"
)

func (s *FakeWorkspace) ModelRegistryCreateModel(req Request) any {
	defer s.LockUnlock()()

	var request ml.CreateModelRequest
	if err := json.Unmarshal(req.Body, &request); err != nil {
		return Response{
			StatusCode: 400,
			Body:       map[string]string{"message": fmt.Sprintf("Failed to parse request: %s", err)},
		}
	}

	// Create the model
	model := ml.Model{
		Name:        request.Name,
		Description: request.Description,
		Tags:        request.Tags,
	}

	s.ModelRegistryModels[request.Name] = model

	return Response{
		Body: ml.CreateModelResponse{
			RegisteredModel: &model,
		},
	}
}

func (s *FakeWorkspace) ModelRegistryUpdateModel(req Request) any {
	defer s.LockUnlock()()

	var request ml.UpdateModelRequest
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
		Body: ml.UpdateModelResponse{
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
		Body: ml.GetModelResponse{
			RegisteredModelDatabricks: &ml.ModelDatabricks{
				Name:            model.Name,
				Description:     model.Description,
				Tags:            model.Tags,
				ForceSendFields: model.ForceSendFields,
			},
		},
	}
}
