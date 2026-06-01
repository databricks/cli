package testserver

import (
	"encoding/json"
	"slices"

	sdkbundle "github.com/databricks/databricks-sdk-go/service/bundle"
)

// Minimal Deployment Metadata Service (DMS) handlers under /api/2.0/bundle,
// sufficient to seed deployment state and read it back: CreateVersion /
// CompleteVersion / ListVersions track a deployment's versions, CreateOperation
// upserts (or deletes) a deployment-level resource, and ListResources returns
// the recorded resources for a deployment.

// BundleCreateVersion records a new in-progress version for a deployment.
func (s *FakeWorkspace) BundleCreateVersion(req Request, deploymentID string) Response {
	var version sdkbundle.Version
	if err := json.Unmarshal(req.Body, &version); err != nil {
		return Response{StatusCode: 400, Body: map[string]string{"message": err.Error()}}
	}

	defer s.LockUnlock()()

	version.VersionId = req.URL.Query().Get("version_id")
	version.Name = "deployments/" + deploymentID + "/versions/" + version.VersionId
	version.Status = sdkbundle.VersionStatusVersionStatusInProgress
	s.BundleVersions[deploymentID] = append(s.BundleVersions[deploymentID], version)

	return Response{Body: version}
}

// BundleCompleteVersion marks a deployment version completed with the given reason.
func (s *FakeWorkspace) BundleCompleteVersion(req Request, deploymentID, versionID string) Response {
	var completeReq sdkbundle.CompleteVersionRequest
	if err := json.Unmarshal(req.Body, &completeReq); err != nil {
		return Response{StatusCode: 400, Body: map[string]string{"message": err.Error()}}
	}

	defer s.LockUnlock()()

	versions := s.BundleVersions[deploymentID]
	for i := range versions {
		if versions[i].VersionId == versionID {
			versions[i].Status = sdkbundle.VersionStatusVersionStatusCompleted
			versions[i].CompletionReason = completeReq.CompletionReason
			return Response{Body: versions[i]}
		}
	}
	return Response{StatusCode: 404, Body: map[string]string{"message": "version not found"}}
}

// BundleListVersions returns the versions recorded for a deployment.
func (s *FakeWorkspace) BundleListVersions(deploymentID string) Response {
	defer s.LockUnlock()()

	return Response{Body: sdkbundle.ListVersionsResponse{Versions: s.BundleVersions[deploymentID]}}
}

// BundleCreateOperation records a resource operation against a deployment. A
// create/update operation upserts the deployment-level resource; a delete
// operation removes it.
func (s *FakeWorkspace) BundleCreateOperation(req Request, deploymentID string) Response {
	var op sdkbundle.Operation
	if err := json.Unmarshal(req.Body, &op); err != nil {
		return Response{StatusCode: 400, Body: map[string]string{"message": err.Error()}}
	}

	defer s.LockUnlock()()

	if s.BundleResources[deploymentID] == nil {
		s.BundleResources[deploymentID] = map[string]sdkbundle.Resource{}
	}

	if op.ActionType == sdkbundle.OperationActionTypeOperationActionTypeDelete {
		delete(s.BundleResources[deploymentID], op.ResourceKey)
	} else {
		s.BundleResources[deploymentID][op.ResourceKey] = sdkbundle.Resource{
			Name:           "deployments/" + deploymentID + "/resources/" + op.ResourceKey,
			ResourceKey:    op.ResourceKey,
			ResourceId:     op.ResourceId,
			LastActionType: op.ActionType,
			State:          op.State,
		}
	}

	return Response{Body: op}
}

// BundleListResources returns the resources recorded for a deployment, sorted by
// resource key for deterministic output.
func (s *FakeWorkspace) BundleListResources(deploymentID string) Response {
	defer s.LockUnlock()()

	resources := s.BundleResources[deploymentID]
	keys := make([]string, 0, len(resources))
	for k := range resources {
		keys = append(keys, k)
	}
	slices.Sort(keys)

	out := make([]sdkbundle.Resource, 0, len(keys))
	for _, k := range keys {
		out = append(out, resources[k])
	}

	return Response{Body: sdkbundle.ListResourcesResponse{Resources: out}}
}
