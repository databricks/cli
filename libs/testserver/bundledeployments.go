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

// State is kept in FakeWorkspace.DmsDeployments, keyed by deployment ID.

// dmsDeploymentNodeName is the workspace node name the service uses for deployments.
// It must match DEPLOYMENT_NODE_NAME on the service side (DeploymentWhsClient).
const dmsDeploymentNodeName = "resources.deployment.json"

// dmsUpdatableOperationFields are the update_mask paths UpdateOperation accepts. Any
// other path is rejected, and action_type in particular is fixed when the operation is
// created.
var dmsUpdatableOperationFields = []string{"state", "error_message", "resource_id", "status"}

// DmsDeployment holds a deployment record together with the versions and
// resources recorded under it, so the read APIs (ListVersions/ListResources)
// can serve back what deploys wrote.
type DmsDeployment struct {
	Deployment bundledeployments.Deployment
	Versions   map[string]*bundledeployments.Version
	// resources is the latest resource state per resource key, updated as
	// operations are recorded.
	Resources map[string]bundledeployments.Resource
	// operations holds the recorded operations by resource name. The service keeps
	// one per resource per version, so a resource written twice in a version updates
	// its operation rather than adding another.
	Operations map[string]*bundledeployments.Operation
	// lastSuccessfulVersionID is the highest version completed successfully.
	// The read path treats a non-empty value as "DMS owns the state";
	// the SDK Deployment struct does not yet carry this field.
	LastSuccessfulVersionID string
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
			// The service registers its own node type, which get-status reports back;
			// a plain FILE is what the fake used to claim and the real API never returns.
			ObjectType: "BUNDLE_DEPLOYMENT",
			Path:       nodePath,
			ObjectId:   objectID,
		},
	}

	// The record carries no version yet; last_version_id stays empty until
	// the first CreateVersion. A failed registration leaves a record with no versions.
	deploymentID := strconv.FormatInt(objectID, 10)
	s.DmsDeploymentNodes[deploymentID] = nodePath

	if resp, ok := checkWorkspaceInfo(dep.WorkspaceInfo); !ok {
		return resp
	}

	dep.Name = "deployments/" + deploymentID
	dep.Status = bundledeployments.DeploymentStatusDeploymentStatusActive
	s.DmsDeployments[deploymentID] = &DmsDeployment{
		Deployment: dep,
		Versions:   map[string]*bundledeployments.Version{},
		Resources:  map[string]bundledeployments.Resource{},
		Operations: map[string]*bundledeployments.Operation{},
	}
	return Response{Body: dep}
}

func (s *FakeWorkspace) GetDeployment(deploymentID string) Response {
	defer s.LockUnlock()()

	d, ok := s.DmsDeployments[deploymentID]
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
func deploymentBody(d *DmsDeployment) (map[string]any, error) {
	raw, err := json.Marshal(d.Deployment)
	if err != nil {
		return nil, err
	}

	var body map[string]any
	dec := json.NewDecoder(bytes.NewReader(raw))
	dec.UseNumber()
	if err := dec.Decode(&body); err != nil {
		return nil, err
	}

	if d.LastSuccessfulVersionID != "" {
		body["last_successful_version_id"] = d.LastSuccessfulVersionID
	}
	return body, nil
}

// dmsUpdatableDeploymentFields are the update_mask paths UpdateDeployment accepts.
var dmsUpdatableDeploymentFields = []string{"display_name", "target_name", "deployment_mode", "workspace_info"}

// checkWorkspaceInfo rejects a bundle_root_path without the git_folder_path it is relative to,
// which is what the service does.
func checkWorkspaceInfo(ws *bundledeployments.WorkspaceInfo) (Response, bool) {
	if ws != nil && (ws.GitFolderPath == "") != (ws.BundleRootPath == "") {
		return dmsInvalidArgument("workspace_info.git_folder_path and workspace_info.bundle_root_path must be set together"), false
	}
	return Response{}, true
}

func (s *FakeWorkspace) UpdateDeployment(req Request, deploymentID string) Response {
	var raw map[string]json.RawMessage
	if err := json.Unmarshal(req.Body, &raw); err != nil {
		return Response{StatusCode: 400, Body: map[string]string{"message": err.Error()}}
	}
	var dep bundledeployments.Deployment
	if err := json.Unmarshal(req.Body, &dep); err != nil {
		return Response{StatusCode: 400, Body: map[string]string{"message": err.Error()}}
	}

	updateMask := req.URL.Query().Get("update_mask")
	if updateMask == "" {
		return dmsInvalidArgument("update_mask is required")
	}
	update := map[string]bool{}
	for path := range strings.SplitSeq(updateMask, ",") {
		path = strings.TrimSpace(path)
		if !slices.Contains(dmsUpdatableDeploymentFields, path) {
			return dmsInvalidArgument("update_mask path " + path + " is not updatable")
		}
		if _, sent := raw[path]; !sent {
			// An empty value is how a field is cleared, so the field has to be there to say so.
			return dmsInvalidArgument(path + " is required when '" + path + "' is in update_mask (an empty value clears it)")
		}
		update[path] = true
	}

	if resp, ok := checkWorkspaceInfo(dep.WorkspaceInfo); !ok {
		return resp
	}

	defer s.LockUnlock()()

	d, ok := s.DmsDeployments[deploymentID]
	if !ok {
		return dmsNotFound("deployment " + deploymentID)
	}

	// Only the masked paths are written; every other field of the body is ignored.
	if update["display_name"] {
		d.Deployment.DisplayName = dep.DisplayName
	}
	if update["target_name"] {
		d.Deployment.TargetName = dep.TargetName
	}
	if update["deployment_mode"] {
		d.Deployment.DeploymentMode = dep.DeploymentMode
	}
	if update["workspace_info"] {
		d.Deployment.WorkspaceInfo = dep.WorkspaceInfo
	}

	body, err := deploymentBody(d)
	if err != nil {
		return Response{StatusCode: 500, Body: map[string]string{"message": err.Error()}}
	}
	return Response{Body: body}
}

func (s *FakeWorkspace) DeleteDeployment(deploymentID string) Response {
	defer s.LockUnlock()()

	// The service trashes the deployment's workspace node, so a later get-status
	// on the node path reports the deployment as absent.
	if nodePath, ok := s.DmsDeploymentNodes[deploymentID]; ok {
		delete(s.files, nodePath)
	}
	delete(s.DmsDeploymentNodes, deploymentID)
	delete(s.DmsDeployments, deploymentID)
	return Response{Body: map[string]any{}}
}

func (s *FakeWorkspace) CreateVersion(req Request, deploymentID string) Response {
	versionID := req.URL.Query().Get("version_id")

	var version bundledeployments.Version
	if err := json.Unmarshal(req.Body, &version); err != nil {
		return Response{StatusCode: 400, Body: map[string]string{"message": err.Error()}}
	}

	// operations rides in the version body like every other field, but is absent from the
	// generated struct while the field is at DEVELOPMENT stage, so it is read separately.
	// Embedding Version instead would promote its own UnmarshalJSON and silently drop this
	// field. It is input only, so it is read here and never returned.
	var staged struct {
		Operations []struct {
			ResourceKey string                                `json:"resource_key"`
			ActionType  bundledeployments.OperationActionType `json:"action_type"`
		} `json:"operations"`
	}
	if err := json.Unmarshal(req.Body, &staged); err != nil {
		return Response{StatusCode: 400, Body: map[string]string{"message": err.Error()}}
	}

	defer s.LockUnlock()()

	d, ok := s.DmsDeployments[deploymentID]
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
	if d.Deployment.LastVersionId != "" {
		last, _ = strconv.ParseInt(d.Deployment.LastVersionId, 10, 64)
	}
	if next <= last {
		return dmsInvalidArgument("version_id " + versionID + " must be greater than the most recent version " + d.Deployment.LastVersionId)
	}
	if version.PreviousVersionId != d.Deployment.LastVersionId {
		return dmsAborted("previous_version_id is outdated; the deployment's most recent version is " + d.Deployment.LastVersionId)
	}

	// Note: deployment lock not modelled. Tests kill the CLI mid-apply, leaving
	// the version in-progress forever, whereas the real service lets the lease expire.

	// A version records its whole operation set up front; there is no API to add one later.
	if len(staged.Operations) > maxOperationsPerVersion {
		return Response{
			StatusCode: 429,
			Body: map[string]string{
				"error_code": "RESOURCE_EXHAUSTED",
				"message":    fmt.Sprintf("a version may stage at most %d operations, got %d", maxOperationsPerVersion, len(staged.Operations)),
			},
		}
	}
	seen := make(map[string]bool, len(staged.Operations))
	for _, staged := range staged.Operations {
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

	d.Deployment.LastVersionId = versionID
	version.Name = "deployments/" + deploymentID + "/versions/" + versionID
	version.VersionId = versionID
	version.Status = bundledeployments.VersionStatusVersionStatusInProgress
	d.Versions[versionID] = &version

	// The deployment's git provenance is derived from the version that carried it; the rest of
	// its metadata is written through CreateDeployment and UpdateDeployment.
	d.Deployment.GitInfo = version.GitInfo

	// Each staged operation starts pending at sequence 0, and the CLI fills in its outcome
	// with UpdateOperation as the resource is applied.
	for _, staged := range staged.Operations {
		opName := "deployments/" + deploymentID + "/versions/" + versionID + "/operations/" + staged.ResourceKey
		d.Operations[opName] = &bundledeployments.Operation{
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

	d, ok := s.DmsDeployments[deploymentID]
	if !ok {
		return dmsNotFound("deployment " + deploymentID)
	}
	v, ok := d.Versions[versionID]
	if !ok {
		return dmsNotFound("version " + versionID)
	}

	v.Status = bundledeployments.VersionStatusVersionStatusCompleted
	v.CompletionReason = completeReq.CompletionReason
	if completeReq.CompletionReason == bundledeployments.VersionCompleteVersionCompleteSuccess {
		d.LastSuccessfulVersionID = versionID
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
		// state is the exception: masked with no value is how the service is told the resource
		// is gone, which is what stops it being listed.
		if _, sent := raw[path]; !sent && path != "state" {
			// An empty value is how a field is cleared, so the field has to be there to say so.
			return dmsInvalidArgument(path + " is required when '" + path + "' is in update_mask (an empty value clears it)")
		}
		update[path] = true
	}

	// The service parses the status as a proto enum, so a value outside the enum arrives as
	// UNSPECIFIED and is refused. Checked here too: a status the API does not have otherwise
	// passes every local test and fails only against a real workspace.
	if update["status"] {
		switch op.Status {
		case bundledeployments.OperationStatusOperationStatusSucceeded,
			bundledeployments.OperationStatusOperationStatusFailed,
			operationStatusPending:
		default:
			return dmsInvalidArgument("status is required and must not be UNSPECIFIED when 'status' is in update_mask")
		}
	}

	// The service counts only a state it was given as recording one, so these ask whether a
	// value arrived rather than whether the mask names the field.
	_, stateSent := raw["state"]
	recordsState := update["state"] && stateSent

	// A resource_id can be written but not cleared, and only alongside the state it belongs to.
	if update["resource_id"] {
		if op.ResourceId == "" {
			return dmsInvalidArgument("resource_id is required when 'resource_id' is in update_mask")
		}
		if !recordsState {
			return dmsInvalidArgument("state must be in update_mask when 'resource_id' is")
		}
	}

	defer s.LockUnlock()()

	d, ok := s.DmsDeployments[deploymentID]
	if !ok {
		return dmsNotFound("deployment " + deploymentID)
	}

	opName := "deployments/" + deploymentID + "/versions/" + versionID + "/operations/" + resourceKey
	existing, ok := d.Operations[opName]
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

	// A failure may leave no state only where nothing is left to describe: a create that never
	// landed, or a recreate whose delete already went through.
	if failed && after.State == "" && leavesLiveResource(after.ActionType) {
		return dmsInvalidArgument("state is required for a " + string(after.ActionType) +
			" operation, because it acts on a resource that already exists and cannot destroy it, even when it fails")
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

	// Only an update that names state moves the resource: naming it with no value clears
	// it and removes the resource, and an update that leaves it out - a failure reporting
	// its outcome - must not disturb what the deployment already holds. Every version
	// stages its operations without state, so re-deriving this from the operation
	// regardless of the mask would drop a resource whose deploy failed before writing.
	if update["state"] {
		if existing.State == "" {
			delete(d.Resources, resourceKey)
		} else {
			d.Resources[resourceKey] = bundledeployments.Resource{
				Name:           "deployments/" + deploymentID + "/resources/" + resourceKey,
				ResourceKey:    resourceKey,
				ResourceId:     existing.ResourceId,
				ResourceType:   existing.ResourceType,
				LastActionType: existing.ActionType,
				LastVersionId:  versionID,
				State:          existing.State,
			}
		}
	} else if resource, projected := d.Resources[resourceKey]; projected && update["resource_id"] {
		// resource_id is mirrored too, for a resource the deployment still holds.
		resource.ResourceId = existing.ResourceId
		d.Resources[resourceKey] = resource
	}

	return Response{Body: body}
}

func (s *FakeWorkspace) ListResources(deploymentID string) Response {
	defer s.LockUnlock()()

	d, ok := s.DmsDeployments[deploymentID]
	if !ok {
		return dmsNotFound("deployment " + deploymentID)
	}

	// Sort by resource key so the response order is deterministic.
	keys := make([]string, 0, len(d.Resources))
	for key := range d.Resources {
		keys = append(keys, key)
	}
	slices.Sort(keys)

	resources := make([]bundledeployments.Resource, 0, len(keys))
	for _, key := range keys {
		resources = append(resources, d.Resources[key])
	}
	return Response{Body: bundledeployments.ListResourcesResponse{Resources: resources}}
}

// leavesLiveResource reports whether a failed operation of this type leaves a resource behind
// that it still has to describe. A create leaves nothing, and a recreate already deleted.
func leavesLiveResource(action bundledeployments.OperationActionType) bool {
	switch action {
	case bundledeployments.OperationActionTypeOperationActionTypeCreate,
		bundledeployments.OperationActionTypeOperationActionTypeRecreate:
		return false
	default:
		return true
	}
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
