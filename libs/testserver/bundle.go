package testserver

import (
	"encoding/json"
	"slices"
	"strconv"

	"github.com/databricks/databricks-sdk-go/service/bundledeployments"
)

// Handlers for the Deployment Metadata Service (DMS) API under /api/2.0/bundle.
// State is kept in FakeWorkspace.dmsDeployments, keyed by deployment ID.

// dmsDeployment holds a deployment record together with the versions and
// resources recorded under it, so the read APIs (ListVersions/ListResources)
// can serve back what deploys wrote.
type dmsDeployment struct {
	deployment bundledeployments.Deployment
	versions   map[string]*bundledeployments.Version
	// resources is the latest resource state per resource key, updated as
	// operations are recorded.
	resources map[string]bundledeployments.Resource
	// lastSuccessfulVersionID is the highest version that completed
	// successfully. The server advances last_successful_version_id only on
	// success (unlike last_version_id), and the read path treats a non-empty
	// value as "DMS owns the state". Tracked separately because the SDK
	// Deployment struct does not yet carry the field (still stage:DEVELOPMENT).
	lastSuccessfulVersionID string
}

func (s *FakeWorkspace) CreateDeployment(req Request) Response {
	// The client either supplies the deployment ID or, in the server-generated
	// flow, leaves it empty for the server to mint one.
	deploymentID := req.URL.Query().Get("deployment_id")
	if deploymentID == "" {
		deploymentID = nextUUID()
	}

	var dep bundledeployments.Deployment
	if err := json.Unmarshal(req.Body, &dep); err != nil {
		return Response{StatusCode: 400, Body: map[string]string{"message": err.Error()}}
	}

	defer s.LockUnlock()()

	dep.Name = "deployments/" + deploymentID
	dep.Status = bundledeployments.DeploymentStatusDeploymentStatusActive
	s.dmsDeployments[deploymentID] = &dmsDeployment{
		deployment: dep,
		versions:   map[string]*bundledeployments.Version{},
		resources:  map[string]bundledeployments.Resource{},
	}
	return Response{Body: dep}
}

func (s *FakeWorkspace) GetDeployment(deploymentID string) Response {
	defer s.LockUnlock()()

	d, ok := s.dmsDeployments[deploymentID]
	if !ok {
		return dmsNotFound("deployment " + deploymentID)
	}

	// The SDK Deployment struct does not yet carry last_successful_version_id
	// (still stage:DEVELOPMENT, so stripped from generation), but the read path
	// reads it off the raw JSON. Serve it as an extra field alongside the typed
	// deployment so the overlay behaves as it will against the real server.
	return Response{Body: struct {
		bundledeployments.Deployment
		LastSuccessfulVersionID string `json:"last_successful_version_id,omitempty"`
	}{
		Deployment:              d.deployment,
		LastSuccessfulVersionID: d.lastSuccessfulVersionID,
	}}
}

func (s *FakeWorkspace) DeleteDeployment(deploymentID string) Response {
	defer s.LockUnlock()()

	delete(s.dmsDeployments, deploymentID)
	return Response{Body: map[string]any{}}
}

func (s *FakeWorkspace) CreateVersion(req Request, deploymentID string) Response {
	versionID := req.URL.Query().Get("version_id")

	var version bundledeployments.Version
	if err := json.Unmarshal(req.Body, &version); err != nil {
		return Response{StatusCode: 400, Body: map[string]string{"message": err.Error()}}
	}

	defer s.LockUnlock()()

	d, ok := s.dmsDeployments[deploymentID]
	if !ok {
		return dmsNotFound("deployment " + deploymentID)
	}

	// Mirror the server-side optimistic concurrency check: the new version must
	// be exactly last_version_id + 1.
	want := "1"
	if d.deployment.LastVersionId != "" {
		last, _ := strconv.ParseInt(d.deployment.LastVersionId, 10, 64)
		want = strconv.FormatInt(last+1, 10)
	}
	if versionID != want {
		return dmsAborted("expected version " + want + ", got " + versionID)
	}

	d.deployment.LastVersionId = versionID
	version.Name = "deployments/" + deploymentID + "/versions/" + versionID
	version.VersionId = versionID
	version.Status = bundledeployments.VersionStatusVersionStatusInProgress
	d.versions[versionID] = &version
	return Response{Body: version}
}

func (s *FakeWorkspace) CompleteVersion(req Request, deploymentID, versionID string) Response {
	var completeReq bundledeployments.CompleteVersionRequest
	if err := json.Unmarshal(req.Body, &completeReq); err != nil {
		return Response{StatusCode: 400, Body: map[string]string{"message": err.Error()}}
	}

	defer s.LockUnlock()()

	d, ok := s.dmsDeployments[deploymentID]
	if !ok {
		return dmsNotFound("deployment " + deploymentID)
	}
	v, ok := d.versions[versionID]
	if !ok {
		return dmsNotFound("version " + versionID)
	}

	v.Status = bundledeployments.VersionStatusVersionStatusCompleted
	v.CompletionReason = completeReq.CompletionReason
	if completeReq.CompletionReason == bundledeployments.VersionCompleteVersionCompleteSuccess {
		d.lastSuccessfulVersionID = versionID
	}
	return Response{Body: *v}
}

func (s *FakeWorkspace) Heartbeat() Response {
	return Response{Body: bundledeployments.HeartbeatResponse{}}
}

func (s *FakeWorkspace) CreateOperation(req Request, deploymentID, versionID string) Response {
	resourceKey := req.URL.Query().Get("resource_key")

	var op bundledeployments.Operation
	if err := json.Unmarshal(req.Body, &op); err != nil {
		return Response{StatusCode: 400, Body: map[string]string{"message": err.Error()}}
	}

	defer s.LockUnlock()()

	d, ok := s.dmsDeployments[deploymentID]
	if !ok {
		return dmsNotFound("deployment " + deploymentID)
	}

	op.Name = "deployments/" + deploymentID + "/versions/" + versionID + "/operations/" + resourceKey
	op.ResourceKey = resourceKey

	// Reflect the operation onto the deployment-level resource set the way the
	// backend does: a delete removes the resource, anything else upserts it.
	if op.ActionType == bundledeployments.OperationActionTypeOperationActionTypeDelete {
		delete(d.resources, resourceKey)
	} else {
		d.resources[resourceKey] = bundledeployments.Resource{
			Name:           "deployments/" + deploymentID + "/resources/" + resourceKey,
			ResourceKey:    resourceKey,
			ResourceId:     op.ResourceId,
			ResourceType:   op.ResourceType,
			LastActionType: op.ActionType,
			LastVersionId:  versionID,
			State:          op.State,
		}
	}
	return Response{Body: op}
}

func (s *FakeWorkspace) ListResources(deploymentID string) Response {
	defer s.LockUnlock()()

	d, ok := s.dmsDeployments[deploymentID]
	if !ok {
		return dmsNotFound("deployment " + deploymentID)
	}

	// Sort by resource key so the response order is deterministic.
	keys := make([]string, 0, len(d.resources))
	for key := range d.resources {
		keys = append(keys, key)
	}
	slices.Sort(keys)

	resources := make([]bundledeployments.Resource, 0, len(keys))
	for _, key := range keys {
		resources = append(resources, d.resources[key])
	}
	return Response{Body: bundledeployments.ListResourcesResponse{Resources: resources}}
}

// dmsNotFound returns the RESOURCE_DOES_NOT_EXIST error shape the DMS API uses,
// which the SDK maps to apierr.ErrNotFound.
func dmsNotFound(what string) Response {
	return Response{
		StatusCode: 404,
		Body: map[string]string{
			"error_code": "RESOURCE_DOES_NOT_EXIST",
			"message":    what + " does not exist",
		},
	}
}

// dmsAborted returns the 409 ABORTED error the server uses for the version
// optimistic-concurrency check.
func dmsAborted(message string) Response {
	return Response{
		StatusCode: 409,
		Headers:    map[string][]string{"Content-Type": {"application/json"}},
		Body:       map[string]string{"error_code": "ABORTED", "message": message},
	}
}
