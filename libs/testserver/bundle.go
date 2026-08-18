package testserver

import (
	"bytes"
	"encoding/json"
	"fmt"
	"path"
	"slices"
	"strconv"
	"strings"

	"github.com/databricks/databricks-sdk-go/service/bundledeployments"
	"github.com/databricks/databricks-sdk-go/service/workspace"
)

// Handlers for the Deployment Metadata Service (DMS) API under /api/2.0/bundle.

// maxOperationsPerVersion mirrors the service's compiled-in default. A bundle past it cannot
// be recorded, since the operation set is fixed when the version is created.
const maxOperationsPerVersion = 800

// operationStatusPending is what CreateVersion leaves on a staged operation. Declared here
// because the SDK enum is generated from the OpenAPI spec, which trails the service proto.
const operationStatusPending bundledeployments.OperationStatus = "OPERATION_STATUS_PENDING"

// State is kept in FakeWorkspace.dmsDeployments, keyed by deployment ID.

// dmsDeploymentNodeName is the workspace node name the service uses for deployments.
// It must match DEPLOYMENT_NODE_NAME on the service side (DeploymentWhsClient).
const dmsDeploymentNodeName = "resources.deployment.json"

// dmsUpdatableOperationFields are the update_mask paths UpdateOperation accepts. Any
// other path is rejected, and action_type in particular is fixed when the operation is
// created.
var dmsUpdatableOperationFields = []string{"state", "error_message", "resource_id", "status"}

// dmsDeployment holds a deployment record together with the versions and
// resources recorded under it, so the read APIs (ListVersions/ListResources)
// can serve back what deploys wrote.
type dmsDeployment struct {
	deployment bundledeployments.Deployment
	versions   map[string]*bundledeployments.Version
	// resources is the latest resource state per resource key, updated as
	// operations are recorded.
	resources map[string]bundledeployments.Resource
	// operations holds the recorded operations by resource name. The service keeps
	// one per resource per version, so a resource written twice in a version updates
	// its operation rather than adding another.
	operations map[string]*bundledeployments.Operation
	// lastSuccessfulVersionID is the highest version completed successfully.
	// The read path treats a non-empty value as "DMS owns the state";
	// the SDK Deployment struct does not yet carry this field.
	lastSuccessfulVersionID string
}

func (s *FakeWorkspace) CreateDeployment(req Request) Response {
	var dep bundledeployments.Deployment
	if err := json.Unmarshal(req.Body, &dep); err != nil {
		return Response{StatusCode: 400, Body: map[string]string{"message": err.Error()}}
	}
	if dep.InitialParentPath == "" {
		return Response{
			StatusCode: 400,
			Body:       map[string]string{"error_code": "INVALID_PARAMETER_VALUE", "message": "initial_parent_path is required"},
		}
	}

	defer s.LockUnlock()()

	// The service registers the deployment as a workspace node under
	// initial_parent_path and uses that node's ID as the deployment ID, so a
	// get-status on the node path is how clients look the deployment back up.
	nodePath := path.Join(dep.InitialParentPath, dmsDeploymentNodeName)
	if resp, ok := s.requireParentDirectory(nodePath); !ok {
		return resp
	}
	objectID := nextID()
	s.files[nodePath] = FileEntry{
		Info: workspace.ObjectInfo{
			ObjectType: "FILE",
			Path:       nodePath,
			ObjectId:   objectID,
		},
	}

	// The record carries no version yet; last_version_id stays empty until
	// the first CreateVersion. A failed registration leaves a record with no versions.
	deploymentID := strconv.FormatInt(objectID, 10)
	s.dmsDeploymentNodes[deploymentID] = nodePath

	dep.Name = "deployments/" + deploymentID
	dep.Status = bundledeployments.DeploymentStatusDeploymentStatusActive
	s.dmsDeployments[deploymentID] = &dmsDeployment{
		deployment: dep,
		versions:   map[string]*bundledeployments.Version{},
		resources:  map[string]bundledeployments.Resource{},
		operations: map[string]*bundledeployments.Operation{},
	}
	return Response{Body: dep}
}

func (s *FakeWorkspace) GetDeployment(deploymentID string) Response {
	defer s.LockUnlock()()

	d, ok := s.dmsDeployments[deploymentID]
	if !ok {
		return dmsNotFound("deployment " + deploymentID)
	}

	body, err := deploymentBody(d)
	if err != nil {
		return Response{StatusCode: 500, Body: map[string]string{"message": err.Error()}}
	}
	return Response{Body: body}
}

// deploymentBody renders a deployment with last_successful_version_id, which
// the SDK struct doesn't yet carry. Embedding in a wrapper won't work because
// Deployment.MarshalJSON silently drops sibling fields.
func deploymentBody(d *dmsDeployment) (map[string]any, error) {
	raw, err := json.Marshal(d.deployment)
	if err != nil {
		return nil, err
	}

	var body map[string]any
	dec := json.NewDecoder(bytes.NewReader(raw))
	dec.UseNumber()
	if err := dec.Decode(&body); err != nil {
		return nil, err
	}

	if d.lastSuccessfulVersionID != "" {
		body["last_successful_version_id"] = d.lastSuccessfulVersionID
	}
	return body, nil
}

func (s *FakeWorkspace) DeleteDeployment(deploymentID string) Response {
	defer s.LockUnlock()()

	// The service trashes the deployment's workspace node, so a later get-status
	// on the node path reports the deployment as absent.
	if nodePath, ok := s.dmsDeploymentNodes[deploymentID]; ok {
		delete(s.files, nodePath)
	}
	delete(s.dmsDeploymentNodes, deploymentID)
	delete(s.dmsDeployments, deploymentID)
	return Response{Body: map[string]any{}}
}

func (s *FakeWorkspace) CreateVersion(req Request, deploymentID string) Response {
	versionID := req.URL.Query().Get("version_id")

	var version bundledeployments.Version
	if err := json.Unmarshal(req.Body, &version); err != nil {
		return Response{StatusCode: 400, Body: map[string]string{"message": err.Error()}}
	}

	// previous_version_id and operations are absent from the generated struct, so read them
	// separately.
	var extra struct {
		PreviousVersionId string `json:"previous_version_id"`
		Operations        []struct {
			ResourceKey string                                `json:"resource_key"`
			ActionType  bundledeployments.OperationActionType `json:"action_type"`
		} `json:"operations"`
	}
	if err := json.Unmarshal(req.Body, &extra); err != nil {
		return Response{StatusCode: 400, Body: map[string]string{"message": err.Error()}}
	}
	concurrency := extra

	defer s.LockUnlock()()

	d, ok := s.dmsDeployments[deploymentID]
	if !ok {
		return dmsNotFound("deployment " + deploymentID)
	}

	// version_id must be numerically greater than the most recent version,
	// and previous_version_id must name that version to detect concurrent deploys.
	next, err := strconv.ParseInt(versionID, 10, 64)
	if err != nil || next < 1 {
		return dmsInvalidArgument("version_id must be a positive integer, got " + versionID)
	}
	var last int64
	if d.deployment.LastVersionId != "" {
		last, _ = strconv.ParseInt(d.deployment.LastVersionId, 10, 64)
	}
	if next <= last {
		return dmsInvalidArgument("version_id " + versionID + " must be greater than the most recent version " + d.deployment.LastVersionId)
	}
	if concurrency.PreviousVersionId != d.deployment.LastVersionId {
		return dmsAborted("previous_version_id is outdated; the deployment's most recent version is " + d.deployment.LastVersionId)
	}

	// Note: deployment lock not modelled. Tests kill the CLI mid-apply, leaving
	// the version in-progress forever, whereas the real service lets the lease expire.

	// bundle_root_path is relative to git_folder_path, so the service rejects
	// workspace_info that carries one without the other.
	if ws := version.WorkspaceInfo; ws != nil && (ws.GitFolderPath == "") != (ws.BundleRootPath == "") {
		return dmsInvalidArgument("workspace_info.git_folder_path and workspace_info.bundle_root_path must be set together")
	}

	// A version records its whole operation set up front; there is no API to add one later.
	if len(extra.Operations) > maxOperationsPerVersion {
		return Response{
			StatusCode: 429,
			Body: map[string]string{
				"error_code": "RESOURCE_EXHAUSTED",
				"message":    fmt.Sprintf("a version may stage at most %d operations, got %d", maxOperationsPerVersion, len(extra.Operations)),
			},
		}
	}
	seen := make(map[string]bool, len(extra.Operations))
	for _, staged := range extra.Operations {
		switch {
		case staged.ResourceKey == "":
			return dmsInvalidArgument("operations.resource_key is required")
		case !strings.Contains(staged.ResourceKey, "."):
			return dmsInvalidArgument("operations.resource_key must have a known resource type prefix (e.g. 'jobs.', 'pipelines.'): " + staged.ResourceKey)
		case staged.ActionType == "":
			return dmsInvalidArgument("operations.action_type is required and must not be UNSPECIFIED for resource " + staged.ResourceKey)
		case seen[staged.ResourceKey]:
			return dmsInvalidArgument("operations must have distinct resource_keys; duplicate: " + staged.ResourceKey)
		}
		seen[staged.ResourceKey] = true
	}

	d.deployment.LastVersionId = versionID
	version.Name = "deployments/" + deploymentID + "/versions/" + versionID
	version.VersionId = versionID
	version.Status = bundledeployments.VersionStatusVersionStatusInProgress
	d.versions[versionID] = &version

	// The service denormalizes the version's provenance onto the deployment, which
	// is where the read APIs serve it from. display_name is excluded: the service
	// keeps that on the deployment's workspace node instead.
	d.deployment.TargetName = version.TargetName
	d.deployment.DeploymentMode = version.DeploymentMode
	d.deployment.GitInfo = version.GitInfo
	d.deployment.WorkspaceInfo = version.WorkspaceInfo

	// Each staged operation starts pending at sequence 0, and the CLI fills in its outcome
	// with UpdateOperation as the resource is applied.
	for _, staged := range extra.Operations {
		opName := "deployments/" + deploymentID + "/versions/" + versionID + "/operations/" + staged.ResourceKey
		d.operations[opName] = &bundledeployments.Operation{
			Name:        opName,
			ResourceKey: staged.ResourceKey,
			ActionType:  staged.ActionType,
			Status:      operationStatusPending,
			SequenceId:  0,
		}
	}

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

// operationBody renders an operation the way the service does: sequence_id as a
// JSON string, which the SDK struct cannot express (it types the field int64).
func operationBody(op *bundledeployments.Operation) (map[string]any, error) {
	raw, err := json.Marshal(op)
	if err != nil {
		return nil, err
	}

	var body map[string]any
	dec := json.NewDecoder(bytes.NewReader(raw))
	dec.UseNumber()
	if err := dec.Decode(&body); err != nil {
		return nil, err
	}

	body["sequence_id"] = strconv.FormatInt(op.SequenceId, 10)
	return body, nil
}

// UpdateOperation applies a later write for a resource already recorded in this
// version. sequence_id is the concurrency precondition and increments on success.
func (s *FakeWorkspace) UpdateOperation(req Request, deploymentID, versionID, resourceKey string) Response {
	// sequence_id arrives as a string, which the SDK struct cannot hold (it types the
	// field int64), so read the body twice: once for the typed fields with that key
	// removed, and once for the precondition alone.
	var raw map[string]json.RawMessage
	if err := json.Unmarshal(req.Body, &raw); err != nil {
		return Response{StatusCode: 400, Body: map[string]string{"message": err.Error()}}
	}
	var precondition struct {
		SequenceId string `json:"sequence_id"`
	}
	if err := json.Unmarshal(req.Body, &precondition); err != nil {
		return Response{StatusCode: 400, Body: map[string]string{"message": err.Error()}}
	}
	delete(raw, "sequence_id")
	typedBody, err := json.Marshal(raw)
	if err != nil {
		return Response{StatusCode: 500, Body: map[string]string{"message": err.Error()}}
	}
	var op bundledeployments.Operation
	if err := json.Unmarshal(typedBody, &op); err != nil {
		return Response{StatusCode: 400, Body: map[string]string{"message": err.Error()}}
	}

	updateMask := req.URL.Query().Get("update_mask")
	if updateMask == "" {
		return dmsInvalidArgument("update_mask is required")
	}
	// Only masked paths are written; other fields keep their values.
	// Omitting state keeps the already-recorded state.
	update := map[string]bool{}
	for path := range strings.SplitSeq(updateMask, ",") {
		path = strings.TrimSpace(path)
		if !slices.Contains(dmsUpdatableOperationFields, path) {
			return dmsInvalidArgument("update_mask path " + path + " is not updatable")
		}
		update[path] = true
	}

	defer s.LockUnlock()()

	d, ok := s.dmsDeployments[deploymentID]
	if !ok {
		return dmsNotFound("deployment " + deploymentID)
	}

	opName := "deployments/" + deploymentID + "/versions/" + versionID + "/operations/" + resourceKey
	existing, ok := d.operations[opName]
	if !ok {
		return dmsNotFound("operation " + opName)
	}
	if precondition.SequenceId != strconv.FormatInt(existing.SequenceId, 10) {
		return dmsAborted("sequence_id is outdated; the operation is at " + strconv.FormatInt(existing.SequenceId, 10))
	}

	// Invariants check the operation after the update, not the request.
	// The mask leaves unspecified fields unchanged.
	after := *existing
	if update["state"] {
		after.State = op.State
	}
	if update["error_message"] {
		after.ErrorMessage = op.ErrorMessage
	}
	if update["resource_id"] {
		after.ResourceId = op.ResourceId
	}
	if update["status"] {
		after.Status = op.Status
	}

	failed := after.Status == bundledeployments.OperationStatusOperationStatusFailed
	if !failed && after.ErrorMessage != "" {
		return dmsInvalidArgument("error_message is only allowed when status is OPERATION_STATUS_FAILED")
	}

	// Delete operations require resource_id. Create-flavored actions may lack an ID initially.
	if after.ActionType == bundledeployments.OperationActionTypeOperationActionTypeDelete && after.ResourceId == "" {
		return dmsInvalidArgument("resource_id is required for OPERATION_ACTION_TYPE_DELETE operations")
	}

	// An operation with state must identify its resource via resource_id,
	// even for failed operations reporting prior state.
	if after.State != "" && after.ResourceId == "" {
		return dmsInvalidArgument("resource_id is required for an operation that records state")
	}

	// Only the masked fields change; action_type and resource_key stay as created.
	if update["state"] {
		existing.State = op.State
	}
	if update["error_message"] {
		existing.ErrorMessage = op.ErrorMessage
	}
	if update["resource_id"] {
		existing.ResourceId = op.ResourceId
	}
	if update["status"] {
		existing.Status = op.Status
	}
	existing.SequenceId++

	body, err := operationBody(existing)
	if err != nil {
		return Response{StatusCode: 500, Body: map[string]string{"message": err.Error()}}
	}

	// Mirror onto the resource set the same way CreateOperation does, so the read
	// path reflects the newest write.
	if existing.State == "" {
		delete(d.resources, resourceKey)
	} else {
		d.resources[resourceKey] = bundledeployments.Resource{
			Name:           "deployments/" + deploymentID + "/resources/" + resourceKey,
			ResourceKey:    resourceKey,
			ResourceId:     existing.ResourceId,
			ResourceType:   existing.ResourceType,
			LastActionType: existing.ActionType,
			LastVersionId:  versionID,
			State:          existing.State,
		}
	}

	return Response{Body: body}
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
		// Content-Type is required for the SDK to parse the body into a typed error,
		// which is what callers match against apierr.ErrResourceDoesNotExist.
		Headers: map[string][]string{"Content-Type": {"application/json"}},
		Body: map[string]string{
			"error_code": "RESOURCE_DOES_NOT_EXIST",
			"message":    what + " does not exist",
		},
	}
}

// dmsAborted returns the 409 ABORTED error the server uses for the version
// optimistic-concurrency check.
func dmsInvalidArgument(message string) Response {
	return Response{
		StatusCode: 400,
		Headers:    map[string][]string{"Content-Type": {"application/json"}},
		Body:       map[string]string{"error_code": "INVALID_PARAMETER_VALUE", "message": message},
	}
}

func dmsAborted(message string) Response {
	return Response{
		StatusCode: 409,
		Headers:    map[string][]string{"Content-Type": {"application/json"}},
		Body:       map[string]string{"error_code": "ABORTED", "message": message},
	}
}
