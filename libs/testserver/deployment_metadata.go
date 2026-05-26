package testserver

import (
	"encoding/json"
	"fmt"
	"net/http"
	"strconv"
	"strings"
	"time"

	sdktime "github.com/databricks/databricks-sdk-go/common/types/time"
	sdkbundle "github.com/databricks/databricks-sdk-go/service/bundle"
)

// deploymentMetadata holds in-memory state for the deployment metadata
// service. One instance lives inside each FakeWorkspace so tests can drive
// CRUD against the DMS routes the same way they drive jobs/apps/etc.
type deploymentMetadata struct {
	// deployments keyed by deployment_id.
	deployments map[string]sdkbundle.Deployment

	// versions keyed by "deploymentId/versionId".
	versions map[string]sdkbundle.Version

	// operations keyed by "deploymentId/versionId/resourceKey".
	operations map[string]sdkbundle.Operation

	// resources keyed by "deploymentId/resourceKey".
	resources map[string]sdkbundle.Resource

	// lockHolder maps deploymentId -> the full version name that holds the
	// lock (e.g. "deployments/{id}/versions/{vid}"). Absent when no lock is
	// held.
	lockHolder map[string]string

	// lockExpiry maps deploymentId -> when the lock expires; checked against
	// time.Now() on every lock-acquiring or lock-respecting call.
	lockExpiry map[string]time.Time
}

func newDeploymentMetadata() *deploymentMetadata {
	return &deploymentMetadata{
		deployments: map[string]sdkbundle.Deployment{},
		versions:    map[string]sdkbundle.Version{},
		operations:  map[string]sdkbundle.Operation{},
		resources:   map[string]sdkbundle.Resource{},
		lockHolder:  map[string]string{},
		lockExpiry:  map[string]time.Time{},
	}
}

// lockDuration matches the real service's default lease so heartbeat-renewal
// tests have a comfortable margin.
const lockDuration = 2 * time.Minute

func nowPtr() *sdktime.Time {
	return sdktime.New(time.Now().UTC())
}

func toSDKTime(t time.Time) *sdktime.Time {
	return sdktime.New(t.UTC())
}

func badRequest(msg string) Response {
	return Response{
		StatusCode: http.StatusBadRequest,
		Body:       map[string]string{"error_code": "INVALID_PARAMETER_VALUE", "message": msg},
	}
}

func notFound(msg string) Response {
	return Response{
		StatusCode: http.StatusNotFound,
		Body:       map[string]string{"error_code": "NOT_FOUND", "message": msg},
	}
}

func aborted(msg string) Response {
	return Response{
		StatusCode: http.StatusConflict,
		Body:       map[string]string{"error_code": "ABORTED", "message": msg},
	}
}

// DeploymentMetadataCreateDeployment is mounted at
// POST /api/2.0/bundle/deployments. The SDK sends the inner Deployment as the
// body and passes deployment_id as a query parameter.
func (s *FakeWorkspace) DeploymentMetadataCreateDeployment(req Request) Response {
	defer s.LockUnlock()()

	deploymentID := req.URL.Query().Get("deployment_id")
	if deploymentID == "" {
		return badRequest("deployment_id is required")
	}

	var body sdkbundle.Deployment
	if len(req.Body) > 0 {
		if err := json.Unmarshal(req.Body, &body); err != nil {
			return badRequest(fmt.Sprintf("invalid request: %s", err))
		}
	}

	state := s.deploymentMetadata
	if _, exists := state.deployments[deploymentID]; exists {
		return Response{
			StatusCode: http.StatusConflict,
			Body: map[string]string{
				"error_code": "ALREADY_EXISTS",
				"message":    fmt.Sprintf("deployment %s already exists", deploymentID),
			},
		}
	}

	now := nowPtr()
	dep := sdkbundle.Deployment{
		Name:        "deployments/" + deploymentID,
		DisplayName: deploymentID,
		TargetName:  body.TargetName,
		Status:      sdkbundle.DeploymentStatusDeploymentStatusActive,
		CreatedBy:   s.CurrentUser().UserName,
		CreateTime:  now,
		UpdateTime:  now,
	}
	state.deployments[deploymentID] = dep
	return Response{Body: dep}
}

// DeploymentMetadataGetDeployment is mounted at
// GET /api/2.0/bundle/deployments/{deployment_id}.
func (s *FakeWorkspace) DeploymentMetadataGetDeployment(deploymentID string) Response {
	defer s.LockUnlock()()

	dep, ok := s.deploymentMetadata.deployments[deploymentID]
	if !ok {
		return notFound(fmt.Sprintf("deployment %s not found", deploymentID))
	}
	return Response{Body: dep}
}

// DeploymentMetadataDeleteDeployment is mounted at
// DELETE /api/2.0/bundle/deployments/{deployment_id}.
func (s *FakeWorkspace) DeploymentMetadataDeleteDeployment(deploymentID string) Response {
	defer s.LockUnlock()()

	state := s.deploymentMetadata
	dep, ok := state.deployments[deploymentID]
	if !ok {
		return notFound(fmt.Sprintf("deployment %s not found", deploymentID))
	}

	now := nowPtr()
	dep.Status = sdkbundle.DeploymentStatusDeploymentStatusDeleted
	dep.DestroyTime = now
	dep.DestroyedBy = s.CurrentUser().UserName
	dep.UpdateTime = now
	state.deployments[deploymentID] = dep
	return Response{Body: dep}
}

// DeploymentMetadataCreateVersion is mounted at
// POST /api/2.0/bundle/deployments/{deployment_id}/versions. Body = Version,
// query = version_id. Validates monotonic version IDs and enforces the
// deployment-level lock.
func (s *FakeWorkspace) DeploymentMetadataCreateVersion(req Request, deploymentID string) Response {
	defer s.LockUnlock()()

	state := s.deploymentMetadata
	dep, ok := state.deployments[deploymentID]
	if !ok {
		return notFound(fmt.Sprintf("deployment %s not found", deploymentID))
	}

	versionID := req.URL.Query().Get("version_id")
	if versionID == "" {
		return badRequest("version_id is required")
	}

	var body sdkbundle.Version
	if len(req.Body) > 0 {
		if err := json.Unmarshal(req.Body, &body); err != nil {
			return badRequest(fmt.Sprintf("invalid request: %s", err))
		}
	}

	// Enforce monotonic versions: version_id must equal last_version_id + 1.
	expected := "1"
	if dep.LastVersionId != "" {
		n, err := strconv.ParseInt(dep.LastVersionId, 10, 64)
		if err != nil {
			return Response{
				StatusCode: http.StatusInternalServerError,
				Body: map[string]string{
					"error_code": "INTERNAL_ERROR",
					"message":    "stored last_version_id is not a valid number: " + dep.LastVersionId,
				},
			}
		}
		expected = strconv.FormatInt(n+1, 10)
	}
	if versionID != expected {
		return aborted(fmt.Sprintf("version_id must be %s (last_version_id + 1), got: %s",
			expected, versionID))
	}

	// Enforce lock: if a lock is held and not expired, reject.
	now := time.Now().UTC()
	if holder, hasLock := state.lockHolder[deploymentID]; hasLock {
		if exp, ok := state.lockExpiry[deploymentID]; ok && exp.After(now) {
			return aborted(fmt.Sprintf("deployment is locked by %s until %s",
				holder, exp.Format(time.RFC3339)))
		}
	}

	versionKey := deploymentID + "/" + versionID
	createTime := toSDKTime(now)
	version := sdkbundle.Version{
		Name:        fmt.Sprintf("deployments/%s/versions/%s", deploymentID, versionID),
		VersionId:   versionID,
		CreatedBy:   s.CurrentUser().UserName,
		CreateTime:  createTime,
		Status:      sdkbundle.VersionStatusVersionStatusInProgress,
		CliVersion:  body.CliVersion,
		VersionType: body.VersionType,
		TargetName:  body.TargetName,
	}
	state.versions[versionKey] = version

	state.lockHolder[deploymentID] = version.Name
	state.lockExpiry[deploymentID] = now.Add(lockDuration)

	dep.LastVersionId = versionID
	dep.Status = sdkbundle.DeploymentStatusDeploymentStatusInProgress
	dep.UpdateTime = createTime
	state.deployments[deploymentID] = dep

	return Response{Body: version}
}

// DeploymentMetadataGetVersion is mounted at
// GET /api/2.0/bundle/deployments/{deployment_id}/versions/{version_id}.
func (s *FakeWorkspace) DeploymentMetadataGetVersion(deploymentID, versionID string) Response {
	defer s.LockUnlock()()

	versionKey := deploymentID + "/" + versionID
	version, ok := s.deploymentMetadata.versions[versionKey]
	if !ok {
		return notFound(fmt.Sprintf("version %s not found", versionKey))
	}
	return Response{Body: version}
}

// DeploymentMetadataHeartbeat is mounted at
// POST /api/2.0/bundle/deployments/{deployment_id}/versions/{version_id}/heartbeat.
// Validates the version is in-progress and holds the lock, then resets the
// lock expiry.
func (s *FakeWorkspace) DeploymentMetadataHeartbeat(_ Request, deploymentID, versionID string) Response {
	defer s.LockUnlock()()

	state := s.deploymentMetadata
	versionKey := deploymentID + "/" + versionID
	version, ok := state.versions[versionKey]
	if !ok {
		return notFound(fmt.Sprintf("version %s not found", versionKey))
	}
	if version.Status != sdkbundle.VersionStatusVersionStatusInProgress {
		return aborted("version is no longer in progress")
	}

	expectedHolder := fmt.Sprintf("deployments/%s/versions/%s", deploymentID, versionID)
	if state.lockHolder[deploymentID] != expectedHolder {
		return aborted("lock is not held by this version")
	}

	now := time.Now().UTC()
	expiry := now.Add(lockDuration)
	state.lockExpiry[deploymentID] = expiry
	return Response{Body: sdkbundle.HeartbeatResponse{ExpireTime: toSDKTime(expiry)}}
}

// DeploymentMetadataCompleteVersion is mounted at
// POST /api/2.0/bundle/deployments/{deployment_id}/versions/{version_id}/complete.
// Tests can inject a simulated failure by setting the deployment's target_name
// to "fail-complete": the endpoint returns a 500 so the caller exercises its
// "lock release failed" path.
func (s *FakeWorkspace) DeploymentMetadataCompleteVersion(req Request, deploymentID, versionID string) Response {
	defer s.LockUnlock()()

	state := s.deploymentMetadata

	if dep, ok := state.deployments[deploymentID]; ok && dep.TargetName == "fail-complete" {
		return Response{
			StatusCode: http.StatusInternalServerError,
			Body: map[string]string{
				"error_code": "INTERNAL_ERROR",
				"message":    "simulated complete version failure",
			},
		}
	}

	versionKey := deploymentID + "/" + versionID
	version, ok := state.versions[versionKey]
	if !ok {
		return notFound(fmt.Sprintf("version %s not found", versionKey))
	}
	if version.Status != sdkbundle.VersionStatusVersionStatusInProgress {
		return aborted("version is already completed")
	}

	var body sdkbundle.CompleteVersionRequest
	if err := json.Unmarshal(req.Body, &body); err != nil {
		return badRequest(fmt.Sprintf("invalid request: %s", err))
	}

	now := nowPtr()
	version.Status = sdkbundle.VersionStatusVersionStatusCompleted
	version.CompleteTime = now
	version.CompletionReason = body.CompletionReason
	version.CompletedBy = s.CurrentUser().UserName
	state.versions[versionKey] = version

	delete(state.lockHolder, deploymentID)
	delete(state.lockExpiry, deploymentID)

	if dep, ok := state.deployments[deploymentID]; ok {
		switch body.CompletionReason {
		case sdkbundle.VersionCompleteVersionCompleteSuccess:
			dep.Status = sdkbundle.DeploymentStatusDeploymentStatusActive
		case sdkbundle.VersionCompleteVersionCompleteFailure,
			sdkbundle.VersionCompleteVersionCompleteForceAbort,
			sdkbundle.VersionCompleteVersionCompleteLeaseExpired:
			dep.Status = sdkbundle.DeploymentStatusDeploymentStatusFailed
		}
		dep.UpdateTime = now
		state.deployments[deploymentID] = dep
	}

	return Response{Body: version}
}

// DeploymentMetadataCreateOperation is mounted at
// POST /api/2.0/bundle/deployments/{deployment_id}/versions/{version_id}/operations.
// Records the operation and upserts the deployment-level Resource so a
// follow-up ListResources sees the merged view.
func (s *FakeWorkspace) DeploymentMetadataCreateOperation(req Request, deploymentID, versionID string) Response {
	defer s.LockUnlock()()

	state := s.deploymentMetadata

	resourceKey := req.URL.Query().Get("resource_key")
	if resourceKey == "" {
		return badRequest("resource_key is required")
	}

	var body sdkbundle.Operation
	if len(req.Body) > 0 {
		if err := json.Unmarshal(req.Body, &body); err != nil {
			return badRequest(fmt.Sprintf("invalid request: %s", err))
		}
	}

	now := nowPtr()
	opKey := deploymentID + "/" + versionID + "/" + resourceKey
	op := sdkbundle.Operation{
		Name:         fmt.Sprintf("deployments/%s/versions/%s/operations/%s", deploymentID, versionID, resourceKey),
		ResourceKey:  resourceKey,
		CreateTime:   now,
		ActionType:   body.ActionType,
		State:        body.State,
		ResourceId:   body.ResourceId,
		Status:       body.Status,
		ErrorMessage: body.ErrorMessage,
	}
	state.operations[opKey] = op

	resKey := deploymentID + "/" + resourceKey
	state.resources[resKey] = sdkbundle.Resource{
		Name:           fmt.Sprintf("deployments/%s/resources/%s", deploymentID, resourceKey),
		ResourceKey:    resourceKey,
		State:          body.State,
		ResourceId:     body.ResourceId,
		LastActionType: body.ActionType,
		LastVersionId:  versionID,
	}

	return Response{Body: op}
}

// DeploymentMetadataListResources is mounted at
// GET /api/2.0/bundle/deployments/{deployment_id}/resources.
func (s *FakeWorkspace) DeploymentMetadataListResources(deploymentID string) Response {
	defer s.LockUnlock()()

	state := s.deploymentMetadata
	prefix := deploymentID + "/"
	var resources []sdkbundle.Resource
	for key, r := range state.resources {
		if strings.HasPrefix(key, prefix) {
			resources = append(resources, r)
		}
	}
	if resources == nil {
		resources = []sdkbundle.Resource{}
	}
	return Response{Body: sdkbundle.ListResourcesResponse{Resources: resources}}
}
