package testserver

import (
	"encoding/json"
	"fmt"
	"net/http"
	"strconv"
	"strings"

	experiments "github.com/databricks/cli/libs/sdktypes/experiments/v1"
	"github.com/databricks/cli/libs/utils"
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
			Key:   utils.NonZero("mlflow.ownerId"),
			Value: testUser.Id,
		},
		{
			Key:   utils.NonZero("mlflow.experiment.sourceName"),
			Value: experiment.Name,
		},
		{
			Key:   utils.NonZero("mlflow.ownerId"),
			Value: testUser.Id,
		},
		{
			Key:   utils.NonZero("mlflow.ownerEmail"),
			Value: testUser.UserName,
		},
		{
			Key:   utils.NonZero("mlflow.experimentType"),
			Value: utils.NonZero("MLFLOW_EXPERIMENT"),
		},
	}

	experimentId := strconv.FormatInt(nextID(), 10)

	if utils.Deref(experiment.ArtifactLocation) == "" {
		experiment.ArtifactLocation = utils.NonZero("dbfs:/databricks/mlflow-tracking/" + experimentId)
	}

	// Strip /Workspace prefix from experiment name to match cloud behavior
	// Input: //Workspace/Users/foo -> Output: /Users/foo
	experimentName := experiment.Name
	// Remove double slash used for Windows compatibility
	if strings.HasPrefix(utils.Deref(experimentName), "//") {
		experimentName = utils.NonZero(strings.TrimPrefix(utils.Deref(experimentName), "/"))
	}
	// Remove /Workspace prefix
	experimentName = utils.NonZero(strings.TrimPrefix(utils.Deref(experimentName), "/Workspace"))

	// Create the experiment
	exp := experiments.Experiment{
		ExperimentId:     utils.NonZero(experimentId),
		Name:             experimentName,
		ArtifactLocation: experiment.ArtifactLocation,
		Tags:             append(experiment.Tags, appendTags...),
		LifecycleStage:   utils.NonZero("active"),
		// Echo back like the real GetExperiment; omitting this immutable field triggers a spurious recreate.
		TraceLocation: experiment.TraceLocation,
	}

	s.Experiments[experimentId] = experiments.GetExperimentResponse{
		Experiment: &exp,
	}

	return Response{
		Body: experiments.CreateExperimentResponse{
			ExperimentId: utils.NonZero(experimentId),
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

	experiment, exists := s.Experiments[utils.Deref(updateReq.ExperimentId)]
	if !exists {
		return Response{
			StatusCode: http.StatusNotFound,
			Body:       map[string]string{"message": fmt.Sprintf("Experiment %s not found", utils.Deref(updateReq.ExperimentId))},
		}
	}

	// Update the experiment
	if utils.Deref(updateReq.NewName) != "" {
		experiment.Experiment.Name = updateReq.NewName

		// The server modifies the value of the tag as well. Mimic that behaviour
		// in the test server as well.
		for i := range experiment.Experiment.Tags {
			if utils.Deref(experiment.Experiment.Tags[i].Key) == "mlflow.experiment.sourceName" {
				experiment.Experiment.Tags[i].Value = updateReq.NewName
			}
		}
	}

	s.Experiments[utils.Deref(updateReq.ExperimentId)] = experiment

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

	return MapDelete(s, s.Experiments, utils.Deref(deleteReq.ExperimentId))
}
