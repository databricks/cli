package testserver

import (
	"encoding/json"

	"github.com/databricks/databricks-sdk-go/service/bundledeployments"
)

// Handlers for the Deployment Metadata Service (DMS) API under /api/2.0/bundle.
// State is kept in FakeWorkspace.Deployments, keyed by deployment ID.

func (s *FakeWorkspace) CreateDeployment(req Request) Response {
	deploymentID := req.URL.Query().Get("deployment_id")

	var dep bundledeployments.Deployment
	if err := json.Unmarshal(req.Body, &dep); err != nil {
		return Response{StatusCode: 400, Body: map[string]string{"message": err.Error()}}
	}

	defer s.LockUnlock()()

	dep.Name = "deployments/" + deploymentID
	dep.Status = bundledeployments.DeploymentStatusDeploymentStatusActive
	s.Deployments[deploymentID] = &dep
	return Response{Body: dep}
}

func (s *FakeWorkspace) GetDeployment(deploymentID string) Response {
	defer s.LockUnlock()()

	dep, ok := s.Deployments[deploymentID]
	if !ok {
		return Response{
			StatusCode: 404,
			Body: map[string]string{
				"error_code": "RESOURCE_DOES_NOT_EXIST",
				"message":    "deployment " + deploymentID + " does not exist",
			},
		}
	}
	return Response{Body: *dep}
}

func (s *FakeWorkspace) DeleteDeployment(deploymentID string) Response {
	defer s.LockUnlock()()

	delete(s.Deployments, deploymentID)
	return Response{Body: map[string]any{}}
}

func (s *FakeWorkspace) CreateVersion(req Request, deploymentID string) Response {
	versionID := req.URL.Query().Get("version_id")

	var version bundledeployments.Version
	if err := json.Unmarshal(req.Body, &version); err != nil {
		return Response{StatusCode: 400, Body: map[string]string{"message": err.Error()}}
	}

	defer s.LockUnlock()()

	dep, ok := s.Deployments[deploymentID]
	if !ok {
		return Response{
			StatusCode: 404,
			Body:       map[string]string{"error_code": "RESOURCE_DOES_NOT_EXIST", "message": "deployment does not exist"},
		}
	}

	// The server accepts client-chosen version IDs (the CLI sends the state
	// serial), so we record whatever the client requested.
	dep.LastVersionId = versionID
	version.Name = "deployments/" + deploymentID + "/versions/" + versionID
	version.VersionId = versionID
	version.Status = bundledeployments.VersionStatusVersionStatusInProgress
	return Response{Body: version}
}

func (s *FakeWorkspace) CompleteVersion(req Request, deploymentID, versionID string) Response {
	var completeReq bundledeployments.CompleteVersionRequest
	_ = json.Unmarshal(req.Body, &completeReq)

	return Response{Body: bundledeployments.Version{
		Name:             "deployments/" + deploymentID + "/versions/" + versionID,
		VersionId:        versionID,
		Status:           bundledeployments.VersionStatusVersionStatusCompleted,
		CompletionReason: completeReq.CompletionReason,
	}}
}

func (s *FakeWorkspace) Heartbeat() Response {
	return Response{Body: bundledeployments.HeartbeatResponse{}}
}
