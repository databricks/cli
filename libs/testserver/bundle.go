package testserver

import (
	"encoding/json"
	"strconv"

	sdkbundle "github.com/databricks/databricks-sdk-go/service/bundle"
)

// Handlers for the Deployment Metadata Service (DMS) API under /api/2.0/bundle.
// State is kept in FakeWorkspace.Deployments, keyed by deployment ID.

func (s *FakeWorkspace) BundleCreateDeployment(req Request) Response {
	deploymentID := req.URL.Query().Get("deployment_id")

	var dep sdkbundle.Deployment
	if err := json.Unmarshal(req.Body, &dep); err != nil {
		return Response{StatusCode: 400, Body: map[string]string{"message": err.Error()}}
	}

	defer s.LockUnlock()()

	dep.Name = "deployments/" + deploymentID
	dep.Status = sdkbundle.DeploymentStatusDeploymentStatusActive
	s.Deployments[deploymentID] = &dep
	return Response{Body: dep}
}

func (s *FakeWorkspace) BundleGetDeployment(deploymentID string) Response {
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

func (s *FakeWorkspace) BundleDeleteDeployment(deploymentID string) Response {
	defer s.LockUnlock()()

	delete(s.Deployments, deploymentID)
	return Response{Body: map[string]any{}}
}

func (s *FakeWorkspace) BundleCreateVersion(req Request, deploymentID string) Response {
	versionID := req.URL.Query().Get("version_id")

	var version sdkbundle.Version
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

	// Mirror the server-side optimistic concurrency check: the new version must
	// be exactly last_version_id + 1.
	last, _ := strconv.ParseInt(dep.LastVersionId, 10, 64)
	want := strconv.FormatInt(last+1, 10)
	if dep.LastVersionId == "" {
		want = "1"
	}
	if versionID != want {
		return Response{
			StatusCode: 409,
			Headers:    map[string][]string{"Content-Type": {"application/json"}},
			Body:       map[string]string{"error_code": "ABORTED", "message": "expected version " + want + ", got " + versionID},
		}
	}

	dep.LastVersionId = versionID
	version.Name = "deployments/" + deploymentID + "/versions/" + versionID
	version.VersionId = versionID
	version.Status = sdkbundle.VersionStatusVersionStatusInProgress
	return Response{Body: version}
}

func (s *FakeWorkspace) BundleCompleteVersion(req Request, deploymentID, versionID string) Response {
	var completeReq sdkbundle.CompleteVersionRequest
	_ = json.Unmarshal(req.Body, &completeReq)

	return Response{Body: sdkbundle.Version{
		Name:             "deployments/" + deploymentID + "/versions/" + versionID,
		VersionId:        versionID,
		Status:           sdkbundle.VersionStatusVersionStatusCompleted,
		CompletionReason: completeReq.CompletionReason,
	}}
}

func (s *FakeWorkspace) BundleHeartbeat() Response {
	return Response{Body: sdkbundle.HeartbeatResponse{}}
}
