package testserver

import (
	"encoding/json"
	"fmt"
	"net/http"
	"strconv"
	"strings"
	"time"

	"github.com/databricks/cli/libs/tmpdms"
)

// deploymentMetadata holds in-memory state for the deployment metadata service.
// Stored per-workspace inside FakeWorkspace.
type deploymentMetadata struct {
	// deployments keyed by deployment_id
	deployments map[string]tmpdms.Deployment

	// versions keyed by "deploymentId/versionId"
	versions map[string]tmpdms.Version

	// operations keyed by "deploymentId/versionId/resourceKey"
	operations map[string]tmpdms.Operation

	// resources keyed by "deploymentId/resourceKey"
	resources map[string]tmpdms.Resource

	// lock state per deployment: which version holds the lock and when it expires
	lockHolder map[string]string    // deploymentId -> "deployments/{id}/versions/{vid}"
	lockExpiry map[string]time.Time // deploymentId -> expiry time
}

func newDeploymentMetadata() *deploymentMetadata {
	return &deploymentMetadata{
		deployments: map[string]tmpdms.Deployment{},
		versions:    map[string]tmpdms.Version{},
		operations:  map[string]tmpdms.Operation{},
		resources:   map[string]tmpdms.Resource{},
		lockHolder:  map[string]string{},
		lockExpiry:  map[string]time.Time{},
	}
}

const lockDuration = 2 * time.Minute

func (s *FakeWorkspace) DeploymentMetadataCreateDeployment(req Request) Response {
	defer s.LockUnlock()()

	// deployment_id is a query parameter, not in the body.
	deploymentID := req.URL.Query().Get("deployment_id")
	if deploymentID == "" {
		return Response{
			StatusCode: http.StatusBadRequest,
			Body:       map[string]string{"error_code": "INVALID_PARAMETER_VALUE", "message": "deployment_id is required"},
		}
	}

	// The body maps to the Deployment sub-message.
	var bodyDeployment tmpdms.Deployment
	if len(req.Body) > 0 {
		if err := json.Unmarshal(req.Body, &bodyDeployment); err != nil {
			return Response{
				StatusCode: http.StatusBadRequest,
				Body:       map[string]string{"error_code": "INVALID_PARAMETER_VALUE", "message": fmt.Sprintf("invalid request: %s", err)},
			}
		}
	}

	state := s.deploymentMetadata
	if _, exists := state.deployments[deploymentID]; exists {
		return Response{
			StatusCode: http.StatusConflict,
			Body:       map[string]string{"error_code": "ALREADY_EXISTS", "message": fmt.Sprintf("deployment %s already exists", deploymentID)},
		}
	}

	now := time.Now().UTC()
	deployment := tmpdms.Deployment{
		Name:        "deployments/" + deploymentID,
		DisplayName: deploymentID,
		TargetName:  bodyDeployment.TargetName,
		Status:      tmpdms.DeploymentStatusActive,
		CreatedBy:   s.CurrentUser().UserName,
		CreateTime:  &now,
		UpdateTime:  &now,
	}

	state.deployments[deploymentID] = deployment
	return Response{Body: deployment}
}

func (s *FakeWorkspace) DeploymentMetadataGetDeployment(deploymentID string) Response {
	defer s.LockUnlock()()

	state := s.deploymentMetadata
	deployment, ok := state.deployments[deploymentID]
	if !ok {
		return Response{
			StatusCode: http.StatusNotFound,
			Body:       map[string]string{"error_code": "NOT_FOUND", "message": fmt.Sprintf("deployment %s not found", deploymentID)},
		}
	}
	return Response{Body: deployment}
}

func (s *FakeWorkspace) DeploymentMetadataDeleteDeployment(deploymentID string) Response {
	defer s.LockUnlock()()

	state := s.deploymentMetadata
	deployment, ok := state.deployments[deploymentID]
	if !ok {
		return Response{
			StatusCode: http.StatusNotFound,
			Body:       map[string]string{"error_code": "NOT_FOUND", "message": fmt.Sprintf("deployment %s not found", deploymentID)},
		}
	}

	now := time.Now().UTC()
	deployment.Status = tmpdms.DeploymentStatusDeleted
	deployment.DestroyTime = &now
	deployment.DestroyedBy = s.CurrentUser().UserName
	deployment.UpdateTime = &now
	state.deployments[deploymentID] = deployment

	return Response{Body: deployment}
}

func (s *FakeWorkspace) DeploymentMetadataCreateVersion(req Request, deploymentID string) Response {
	defer s.LockUnlock()()

	state := s.deploymentMetadata
	deployment, ok := state.deployments[deploymentID]
	if !ok {
		return Response{
			StatusCode: http.StatusNotFound,
			Body:       map[string]string{"error_code": "NOT_FOUND", "message": fmt.Sprintf("deployment %s not found", deploymentID)},
		}
	}

	// version_id is a query parameter, not in the body.
	versionID := req.URL.Query().Get("version_id")
	if versionID == "" {
		return Response{
			StatusCode: http.StatusBadRequest,
			Body:       map[string]string{"error_code": "INVALID_PARAMETER_VALUE", "message": "version_id is required"},
		}
	}

	// The body maps to the Version sub-message.
	var bodyVersion tmpdms.Version
	if len(req.Body) > 0 {
		if err := json.Unmarshal(req.Body, &bodyVersion); err != nil {
			return Response{
				StatusCode: http.StatusBadRequest,
				Body:       map[string]string{"error_code": "INVALID_PARAMETER_VALUE", "message": fmt.Sprintf("invalid request: %s", err)},
			}
		}
	}

	// Validate version_id == last_version_id + 1 (matching server behavior).
	var expectedVersionID string
	if deployment.LastVersionID == "" {
		expectedVersionID = "1"
	} else {
		lastVersion, err := strconv.ParseInt(deployment.LastVersionID, 10, 64)
		if err != nil {
			return Response{
				StatusCode: http.StatusInternalServerError,
				Body:       map[string]string{"error_code": "INTERNAL_ERROR", "message": "stored last_version_id is not a valid number: " + deployment.LastVersionID},
			}
		}
		expectedVersionID = strconv.FormatInt(lastVersion+1, 10)
	}
	if versionID != expectedVersionID {
		return Response{
			StatusCode: http.StatusConflict,
			Body: map[string]string{
				"error_code": "ABORTED",
				"message":    fmt.Sprintf("version_id must be %s (last_version_id + 1), got: %s", expectedVersionID, versionID),
			},
		}
	}

	// Check lock: if a lock is held and not expired, reject with 409.
	now := time.Now().UTC()
	if holder, hasLock := state.lockHolder[deploymentID]; hasLock {
		if expiry, ok := state.lockExpiry[deploymentID]; ok && expiry.After(now) {
			return Response{
				StatusCode: http.StatusConflict,
				Body: map[string]string{
					"error_code": "ABORTED",
					"message":    fmt.Sprintf("deployment is locked by %s until %s", holder, expiry.Format(time.RFC3339)),
				},
			}
		}
	}

	versionKey := deploymentID + "/" + versionID
	version := tmpdms.Version{
		Name:       fmt.Sprintf("deployments/%s/versions/%s", deploymentID, versionID),
		VersionID:  versionID,
		CreatedBy:  s.CurrentUser().UserName,
		CreateTime: &now,
		Status:     tmpdms.VersionStatusInProgress,
	}
	version.CliVersion = bodyVersion.CliVersion
	version.VersionType = bodyVersion.VersionType
	version.TargetName = bodyVersion.TargetName

	state.versions[versionKey] = version

	// Acquire the lock.
	lockExpiry := now.Add(lockDuration)
	state.lockHolder[deploymentID] = version.Name
	state.lockExpiry[deploymentID] = lockExpiry

	// Update the deployment's last_version_id and status.
	deployment.LastVersionID = versionID
	deployment.Status = tmpdms.DeploymentStatusInProgress
	deployment.UpdateTime = &now
	state.deployments[deploymentID] = deployment

	return Response{Body: version}
}

func (s *FakeWorkspace) DeploymentMetadataGetVersion(deploymentID, versionID string) Response {
	defer s.LockUnlock()()

	state := s.deploymentMetadata
	versionKey := deploymentID + "/" + versionID
	version, ok := state.versions[versionKey]
	if !ok {
		return Response{
			StatusCode: http.StatusNotFound,
			Body:       map[string]string{"error_code": "NOT_FOUND", "message": fmt.Sprintf("version %s not found", versionKey)},
		}
	}
	return Response{Body: version}
}

func (s *FakeWorkspace) DeploymentMetadataHeartbeat(req Request, deploymentID, versionID string) Response {
	defer s.LockUnlock()()

	state := s.deploymentMetadata
	versionKey := deploymentID + "/" + versionID
	version, ok := state.versions[versionKey]
	if !ok {
		return Response{
			StatusCode: http.StatusNotFound,
			Body:       map[string]string{"error_code": "NOT_FOUND", "message": fmt.Sprintf("version %s not found", versionKey)},
		}
	}

	if version.Status != tmpdms.VersionStatusInProgress {
		return Response{
			StatusCode: http.StatusConflict,
			Body:       map[string]string{"error_code": "ABORTED", "message": "version is no longer in progress"},
		}
	}

	// Verify this version holds the lock.
	expectedHolder := fmt.Sprintf("deployments/%s/versions/%s", deploymentID, versionID)
	if state.lockHolder[deploymentID] != expectedHolder {
		return Response{
			StatusCode: http.StatusConflict,
			Body:       map[string]string{"error_code": "ABORTED", "message": "lock is not held by this version"},
		}
	}

	// Renew the lock.
	now := time.Now().UTC()
	newExpiry := now.Add(lockDuration)
	state.lockExpiry[deploymentID] = newExpiry

	return Response{Body: tmpdms.HeartbeatResponse{ExpireTime: &newExpiry}}
}

func (s *FakeWorkspace) DeploymentMetadataCompleteVersion(req Request, deploymentID, versionID string) Response {
	defer s.LockUnlock()()

	state := s.deploymentMetadata

	// Allow tests to simulate a complete version failure. If the deployment's
	// target_name is "fail-complete", return a 500 error.
	if deployment, ok := state.deployments[deploymentID]; ok && deployment.TargetName == "fail-complete" {
		return Response{
			StatusCode: http.StatusInternalServerError,
			Body:       map[string]string{"error_code": "INTERNAL_ERROR", "message": "simulated complete version failure"},
		}
	}

	versionKey := deploymentID + "/" + versionID
	version, ok := state.versions[versionKey]
	if !ok {
		return Response{
			StatusCode: http.StatusNotFound,
			Body:       map[string]string{"error_code": "NOT_FOUND", "message": fmt.Sprintf("version %s not found", versionKey)},
		}
	}

	if version.Status != tmpdms.VersionStatusInProgress {
		return Response{
			StatusCode: http.StatusConflict,
			Body:       map[string]string{"error_code": "ABORTED", "message": "version is already completed"},
		}
	}

	var completeReq tmpdms.CompleteVersionRequest
	if err := json.Unmarshal(req.Body, &completeReq); err != nil {
		return Response{
			StatusCode: http.StatusBadRequest,
			Body:       map[string]string{"error_code": "INVALID_PARAMETER_VALUE", "message": fmt.Sprintf("invalid request: %s", err)},
		}
	}

	now := time.Now().UTC()
	version.Status = tmpdms.VersionStatusCompleted
	version.CompleteTime = &now
	version.CompletionReason = completeReq.CompletionReason
	version.CompletedBy = s.CurrentUser().UserName
	state.versions[versionKey] = version

	// Release the lock.
	delete(state.lockHolder, deploymentID)
	delete(state.lockExpiry, deploymentID)

	// Update deployment status based on completion reason.
	if deployment, ok := state.deployments[deploymentID]; ok {
		switch completeReq.CompletionReason {
		case tmpdms.VersionCompleteSuccess:
			deployment.Status = tmpdms.DeploymentStatusActive
		case tmpdms.VersionCompleteFailure, tmpdms.VersionCompleteForceAbort, tmpdms.VersionCompleteLeaseExpired:
			deployment.Status = tmpdms.DeploymentStatusFailed
		case tmpdms.VersionCompleteUnspecified:
			// No status change for unspecified completion reason.
		}
		deployment.UpdateTime = &now
		state.deployments[deploymentID] = deployment
	}

	return Response{Body: version}
}

func (s *FakeWorkspace) DeploymentMetadataCreateOperation(req Request, deploymentID, versionID string) Response {
	defer s.LockUnlock()()

	state := s.deploymentMetadata

	// resource_key is a query parameter, not in the body.
	resourceKey := req.URL.Query().Get("resource_key")
	if resourceKey == "" {
		return Response{
			StatusCode: http.StatusBadRequest,
			Body:       map[string]string{"error_code": "INVALID_PARAMETER_VALUE", "message": "resource_key is required"},
		}
	}

	// The body maps to the Operation sub-message.
	var bodyOperation tmpdms.Operation
	if len(req.Body) > 0 {
		if err := json.Unmarshal(req.Body, &bodyOperation); err != nil {
			return Response{
				StatusCode: http.StatusBadRequest,
				Body:       map[string]string{"error_code": "INVALID_PARAMETER_VALUE", "message": fmt.Sprintf("invalid request: %s", err)},
			}
		}
	}

	now := time.Now().UTC()
	opKey := deploymentID + "/" + versionID + "/" + resourceKey
	operation := tmpdms.Operation{
		Name:         fmt.Sprintf("deployments/%s/versions/%s/operations/%s", deploymentID, versionID, resourceKey),
		ResourceKey:  resourceKey,
		CreateTime:   &now,
		ActionType:   bodyOperation.ActionType,
		State:        bodyOperation.State,
		ResourceID:   bodyOperation.ResourceID,
		Status:       bodyOperation.Status,
		ErrorMessage: bodyOperation.ErrorMessage,
	}

	state.operations[opKey] = operation

	// Upsert the deployment-level resource.
	resKey := deploymentID + "/" + resourceKey
	resource := tmpdms.Resource{
		Name:           fmt.Sprintf("deployments/%s/resources/%s", deploymentID, resourceKey),
		ResourceKey:    resourceKey,
		State:          bodyOperation.State,
		ResourceID:     bodyOperation.ResourceID,
		LastActionType: bodyOperation.ActionType,
		LastVersionID:  versionID,
	}
	state.resources[resKey] = resource

	return Response{Body: operation}
}

func (s *FakeWorkspace) DeploymentMetadataListResources(deploymentID string) Response {
	defer s.LockUnlock()()

	state := s.deploymentMetadata
	prefix := deploymentID + "/"
	var resources []tmpdms.Resource
	for key, resource := range state.resources {
		if strings.HasPrefix(key, prefix) {
			resources = append(resources, resource)
		}
	}
	if resources == nil {
		resources = []tmpdms.Resource{}
	}
	return Response{Body: tmpdms.ListResourcesResponse{Resources: resources}}
}
