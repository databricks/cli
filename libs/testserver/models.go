package testserver

import (
	"encoding/json"
	"fmt"
	"strconv"

	"github.com/databricks/databricks-sdk-go/service/ml"
)

func init() {
	registerRoutes("models", addModelsRoutes)
	registerStore("models", newModelsStore)
	registerPermissionsObjectType("registered-models", "registered-model")
}

type modelsStore struct {
	ModelRegistryModels   map[string]ml.Model
	ModelRegistryModelIDs map[string]string // model name -> numeric ID
}

func newModelsStore(*FakeWorkspace) *modelsStore {
	return &modelsStore{
		ModelRegistryModels:   map[string]ml.Model{},
		ModelRegistryModelIDs: map[string]string{},
	}
}

func (s *FakeWorkspace) models() *modelsStore {
	return stateOf[modelsStore](s)
}

func addModelsRoutes(server *Server) {
	// Model registry models.
	server.Handle("POST", "/api/2.0/mlflow/registered-models/create", func(req Request) any {
		return req.Workspace.ModelRegistryCreateModel(req)
	})

	server.Handle("GET", "/api/2.0/mlflow/databricks/registered-models/get", func(req Request) any {
		return req.Workspace.ModelRegistryGetModel(req)
	})

	server.Handle("PATCH", "/api/2.0/mlflow/registered-models/update", func(req Request) any {
		return req.Workspace.ModelRegistryUpdateModel(req)
	})

	server.Handle("DELETE", "/api/2.0/mlflow/registered-models/delete", func(req Request) any {
		return MapDelete(req.Workspace, req.Workspace.models().ModelRegistryModels, req.URL.Query().Get("name"))
	})
}

func (s *FakeWorkspace) ModelRegistryCreateModel(req Request) any {
	defer s.LockUnlock()()

	var request ml.CreateModelRequest
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
	model := ml.Model{
		Name:        request.Name,
		Description: request.Description,
		Tags:        request.Tags,
	}

	s.models().ModelRegistryModels[request.Name] = model
	s.models().ModelRegistryModelIDs[request.Name] = modelId

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

	existingModel, ok := s.models().ModelRegistryModels[request.Name]
	if !ok {
		return Response{
			StatusCode: 404,
			Body:       map[string]string{"message": fmt.Sprintf("Model not found: %v", request.Name)},
		}
	}

	// Update the model
	existingModel.Description = request.Description
	s.models().ModelRegistryModels[request.Name] = existingModel

	return Response{
		Body: ml.UpdateModelResponse{
			RegisteredModel: &existingModel,
		},
	}
}

func (s *FakeWorkspace) ModelRegistryGetModel(req Request) any {
	defer s.LockUnlock()()

	name := req.URL.Query().Get("name")

	model, ok := s.models().ModelRegistryModels[name]
	if !ok {
		return Response{
			StatusCode: 404,
			Body:       map[string]string{"message": fmt.Sprintf("Model not found: %v", name)},
		}
	}

	return Response{
		Body: ml.GetModelResponse{
			RegisteredModelDatabricks: &ml.ModelDatabricks{
				Id:              s.models().ModelRegistryModelIDs[name],
				Name:            model.Name,
				Description:     model.Description,
				Tags:            model.Tags,
				ForceSendFields: model.ForceSendFields,
			},
		},
	}
}
