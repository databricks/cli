package testserver

import (
	"encoding/json"
	"fmt"
	"net/http"

	"github.com/databricks/databricks-sdk-go/service/ml"
)

func (s *FakeWorkspace) ExperimentCreate(req Request) Response {
	defer s.LockUnlock()()

	var experiment ml.CreateExperiment
	if err := json.Unmarshal(req.Body, &experiment); err != nil {
		return Response{
			Body:       fmt.Sprintf("request parsing error: %s", err),
			StatusCode: http.StatusBadRequest,
		}
	}

	// TODO(verify):Server appends these tags automatically to experiments.
	// replicate that behaviour in the test server as well.
	appendTags := []ml.ExperimentTag{
		{
			Key:   "mlflow.ownerId",
			Value: TestUser.Id,
		},
		{
			Key:   "mlflow.experiment.sourceName",
			Value: experiment.Name,
		},
		{
			Key:   "mlflow.ownerId",
			Value: TestUser.Id,
		},
		{
			Key:   "mlflow.ownerEmail",
			Value: TestUser.UserName,
		},
		{
			Key:   "mlflow.experimentType",
			Value: "MLFLOW_EXPERIMENT",
		},
	}

	// Generate a new experiment ID
	experimentId := fmt.Sprintf("%d", len(s.Experiments)+1)

	// Create the experiment
	exp := ml.Experiment{
		ExperimentId:     experimentId,
		Name:             experiment.Name,
		ArtifactLocation: experiment.ArtifactLocation,
		Tags:             append(experiment.Tags, appendTags...),
		LifecycleStage:   "active",
	}

	s.Experiments[experimentId] = exp

	return Response{
		Body: ml.CreateExperimentResponse{
			ExperimentId: experimentId,
		},
	}
}

func (s *FakeWorkspace) ExperimentGet(req Request) Response {
	defer s.LockUnlock()()

	experimentId := req.URL.Query().Get("experiment_id")
	if experimentId == "" {
		return Response{
			StatusCode: http.StatusBadRequest,
			Body:       map[string]string{"message": "experiment_id is required"},
		}
	}

	experiment, exists := s.Experiments[experimentId]
	if !exists {
		return Response{
			StatusCode: http.StatusNotFound,
			Body:       map[string]string{"message": fmt.Sprintf("Experiment %s not found", experimentId)},
		}
	}

	return Response{
		Body: ml.GetExperimentResponse{
			Experiment: &experiment,
		},
	}
}

func (s *FakeWorkspace) ExperimentUpdate(req Request) Response {
	defer s.LockUnlock()()

	var updateReq ml.UpdateExperiment
	if err := json.Unmarshal(req.Body, &updateReq); err != nil {
		return Response{
			Body:       fmt.Sprintf("request parsing error: %s", err),
			StatusCode: http.StatusBadRequest,
		}
	}

	experiment, exists := s.Experiments[updateReq.ExperimentId]
	if !exists {
		return Response{
			StatusCode: http.StatusNotFound,
			Body:       map[string]string{"message": fmt.Sprintf("Experiment %s not found", updateReq.ExperimentId)},
		}
	}

	// Update the experiment
	if updateReq.NewName != "" {
		experiment.Name = updateReq.NewName
	}

	s.Experiments[updateReq.ExperimentId] = experiment

	return Response{}
}

func (s *FakeWorkspace) ExperimentDelete(req Request) Response {
	defer s.LockUnlock()()

	var deleteReq ml.DeleteExperiment
	if err := json.Unmarshal(req.Body, &deleteReq); err != nil {
		return Response{
			Body:       fmt.Sprintf("request parsing error: %s", err),
			StatusCode: http.StatusBadRequest,
		}
	}

	_, exists := s.Experiments[deleteReq.ExperimentId]
	if !exists {
		return Response{
			StatusCode: http.StatusNotFound,
			Body:       map[string]string{"message": fmt.Sprintf("Experiment %s not found", deleteReq.ExperimentId)},
		}
	}

	delete(s.Experiments, deleteReq.ExperimentId)

	return Response{}
}
