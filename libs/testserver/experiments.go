package testserver

import (
	"encoding/json"
	"fmt"
	"net/http"
	"strconv"
	"strings"

	"github.com/databricks/databricks-sdk-go/service/ml"
)

// experimentNameUCTracing scopes the out-of-band UC trace-location injection to the
// recreate repro test, keeping unrelated experiment tests free of it. Mirrors
// catalogNameManagedDefaults. It mimics an enable-UC-tracing job that binds a trace
// location to the experiment after creation, one the bundle config never declares.
const experimentNameUCTracing = "uc-tracing"

func (s *FakeWorkspace) ExperimentCreate(req Request) Response {
	defer s.LockUnlock()()

	var experiment ml.CreateExperiment
	if err := json.Unmarshal(req.Body, &experiment); err != nil {
		return Response{
			Body:       fmt.Sprintf("request parsing error: %s", err),
			StatusCode: http.StatusBadRequest,
		}
	}

	testUser := s.CurrentUser()

	// Server appends these tags automatically to experiments.
	// We replicate that behaviour in the test server as well.
	appendTags := []ml.ExperimentTag{
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
	exp := ml.Experiment{
		ExperimentId:     experimentId,
		Name:             experimentName,
		ArtifactLocation: experiment.ArtifactLocation,
		Tags:             append(experiment.Tags, appendTags...),
		LifecycleStage:   "active",
		// Echo back like the real GetExperiment; omitting this immutable field triggers a spurious recreate.
		TraceLocation: experiment.TraceLocation,
	}

	// Mimic an out-of-band enable-UC-tracing job: the bundle config never declares
	// trace_location, but GetExperiment then echoes one. See SUP-37168 and
	// acceptance/bundle/resources/experiments/uc-tracing-recreate.
	if exp.TraceLocation == nil && strings.Contains(experimentName, experimentNameUCTracing) {
		exp.TraceLocation = &ml.ExperimentTraceLocation{
			UcTraceLocation: &ml.UcTraceLocation{
				Catalog:              "main",
				Schema:               "traces",
				EffectiveTablePrefix: "exp_" + experimentId + "_traces",
			},
		}
	}

	s.Experiments[experimentId] = ml.GetExperimentResponse{
		Experiment: &exp,
	}

	return Response{
		Body: ml.CreateExperimentResponse{
			ExperimentId: experimentId,
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
	var deleteReq ml.DeleteExperiment
	if err := json.Unmarshal(req.Body, &deleteReq); err != nil {
		return Response{
			Body:       fmt.Sprintf("request parsing error: %s", err),
			StatusCode: http.StatusBadRequest,
		}
	}

	return MapDelete(s, s.Experiments, deleteReq.ExperimentId)
}
