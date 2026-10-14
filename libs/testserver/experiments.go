package testserver

import (
	experiments "github.com/databricks/cli/libs/sdktypes/experiments/v1"
	"encoding/json"
	"fmt"
	"net/http"
	"strconv"
	"strings"
)

func (s *FakeWorkspace) ExperimentCreate(req Request) Response {
	defer s.LockUnlock()()

	var experiment experiments.CreateExperimentRequest
	if err := json.Unmarshal(req.Body, &experiment); err != nil {
		return Response{
			Body:       fmt.Sprintf("request parsing error: %s", err),
			StatusCode: http.StatusBadRequest,
		}
	}

	testUser := s.CurrentUser()

	// Server appends these tags automatically to experiments.
	// We replicate that behaviour in the test server as well.
	appendTags := []experiments.ExperimentTag{
		{
			Key:   "mlflow.ownerId",
			Value: testUser.Id,
		},
		{
			Key:   "mlflow.experiment.sourceName",
			Value: experiment.Name,
		},
		{
			Key:   "mlflow.ownerId",
			Value: testUser.Id,
		},
		{
			Key:   "mlflow.ownerEmail",
			Value: testUser.UserName,
		},
		{
			Key:   "mlflow.experimentType",
			Value: "MLFLOW_EXPERIMENT",
		},
	}

	experimentId := strconv.FormatInt(nextID(), 10)

	if experiment.ArtifactLocation == "" {
		experiment.ArtifactLocation = "dbfs:/databricks/mlflow-tracking/" + experimentId
	}

	// Strip /Workspace prefix from experiment name to match cloud behavior
	// Input: //Workspace/Users/foo -> Output: /Users/foo
	experimentName := experiment.Name
	// Remove double slash used for Windows compatibility
	if strings.HasPrefix(experimentName, "//") {
		experimentName = strings.TrimPrefix(experimentName, "/")
	}
	// Remove /Workspace prefix
	experimentName = strings.TrimPrefix(experimentName, "/Workspace")

	// Create the experiment
	exp := experiments.Experiment{
		ExperimentId:     experimentId,
		Name:             experimentName,
		ArtifactLocation: experiment.ArtifactLocation,
		Tags:             append(experiment.Tags, appendTags...),
		LifecycleStage:   "active",
		// Echo back like the real GetExperiment; omitting this immutable field triggers a spurious recreate.
		TraceLocation: experiment.TraceLocation,
	}

	s.Experiments[experimentId] = experiments.GetExperimentResponse{
		Experiment: &exp,
	}

	return Response{
		Body: experiments.CreateExperimentResponse{
			ExperimentId: experimentId,
		},
	}
}

func (s *FakeWorkspace) ExperimentUpdate(req Request) Response {
	defer s.LockUnlock()()

	var updateReq experiments.UpdateExperimentRequest
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
		experiment.Experiment.Name = updateReq.NewName

		// The server modifies the value of the tag as well. Mimic that behaviour
		// in the test server as well.
		for i := range experiment.Experiment.Tags {
			if experiment.Experiment.Tags[i].Key == "mlflow.experiment.sourceName" {
				experiment.Experiment.Tags[i].Value = updateReq.NewName
			}
		}
	}

	s.Experiments[updateReq.ExperimentId] = experiment

	return Response{}
}

func (s *FakeWorkspace) ExperimentDelete(req Request) Response {
	var deleteReq experiments.DeleteExperimentRequest
	if err := json.Unmarshal(req.Body, &deleteReq); err != nil {
		return Response{
			Body:       fmt.Sprintf("request parsing error: %s", err),
			StatusCode: http.StatusBadRequest,
		}
	}

	return MapDelete(s, s.Experiments, deleteReq.ExperimentId)
}
