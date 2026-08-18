package dms

import (
	"context"
	"fmt"
	"net/http"

	"github.com/databricks/cli/libs/auth"
	"github.com/databricks/databricks-sdk-go"
	"github.com/databricks/databricks-sdk-go/client"
	"github.com/databricks/databricks-sdk-go/service/bundledeployments"
)

// Client is every call the CLI makes to DMS. Most go through the generated client; the two
// the SDK cannot express are written by hand below, each with its own interface so a test
// can capture what the CLI sends.
type Client struct {
	// Service is the generated client, used for every call it can express.
	Service bundledeployments.BundleDeploymentsInterface

	// Versions creates versions; see VersionCreator.
	Versions VersionCreator

	// Operations fills in staged operations; see OperationUpdater.
	Operations OperationUpdater
}

// NewClient returns a Client for the workspace w.
func NewClient(w *databricks.WorkspaceClient) (*Client, error) {
	api, err := client.New(w.Config)
	if err != nil {
		return nil, err
	}
	raw := &rawClient{client: api}
	return &Client{Service: w.BundleDeployments, Versions: raw, Operations: raw}, nil
}

// VersionCreator creates a version under a deployment. Hand-written because the generated
// struct has no previous_version_id, which the service needs as its concurrency check -
// without it every deploy after the first is rejected.
type VersionCreator interface {
	CreateVersion(ctx context.Context, deploymentID, versionID string, body CreateVersionRequest) (*bundledeployments.Version, error)
}

// OperationUpdater fills in an operation the version staged, and returns the sequence id the
// next update for that resource must send. Hand-written because the SDK types sequence_id as
// an int64 while the service sends a JSON string. TODO(DMS): drop once the spec agrees.
type OperationUpdater interface {
	// sequenceID is the token the previous update for this resource returned, or 0 for the
	// first, which is what staging leaves.
	UpdateOperation(ctx context.Context, deploymentID string, version int64, key ResourceKey, sequenceID string, update OperationUpdate) (next string, err error)
}

// CreateVersionRequest is the CreateVersion request body.
type CreateVersionRequest struct {
	CliVersion  string      `json:"cli_version"`
	VersionType VersionType `json:"version_type"`
	TargetName  string      `json:"target_name,omitempty"`
	// DisplayName names the deployment in the UI. The service keeps it on the
	// deployment's node, so a version that omits it leaves the deployment unnamed.
	DisplayName string `json:"display_name,omitempty"`
	// PreviousVersionId is the deployment's most recent version, unset for a
	// deployment's first version.
	PreviousVersionId string `json:"previous_version_id,omitempty"`
	// DeploymentMode is the bundle target's mode, unset when the target sets none.
	DeploymentMode bundledeployments.DeploymentMode `json:"deployment_mode,omitempty"`
	// GitInfo and WorkspaceInfo record where the deployed source came from and
	// where it landed. The service denormalizes both onto the deployment.
	GitInfo       *bundledeployments.GitInfo       `json:"git_info,omitempty"`
	WorkspaceInfo *bundledeployments.WorkspaceInfo `json:"workspace_info,omitempty"`
	// Operations is every resource this version will touch; see StagedOperation.
	Operations []StagedOperation `json:"operations,omitempty"`
}

// StagedOperation is one resource the version will record an operation for. The service
// creates it in OPERATION_STATUS_PENDING at sequence id 0, and the CLI fills in the outcome
// with UpdateOperation as the resource is applied.
type StagedOperation struct {
	ResourceKey ResourceKey                           `json:"resource_key"`
	ActionType  bundledeployments.OperationActionType `json:"action_type"`
}

// updateOperationRequest carries the values an update writes. action_type and resource_key
// are left out: the service fixes them when the version stages the operation.
type updateOperationRequest struct {
	State        string                            `json:"state,omitempty"`
	ErrorMessage string                            `json:"error_message,omitempty"`
	ResourceId   string                            `json:"resource_id,omitempty"`
	Status       bundledeployments.OperationStatus `json:"status,omitempty"`
	SequenceId   string                            `json:"sequence_id,omitempty"`
}

// operationResponse is the part of an operation response the CLI reads back.
type operationResponse struct {
	// SequenceId is the concurrency token for the next update, typed as the service sends it.
	SequenceId string `json:"sequence_id,omitempty"`
}

// rawClient sends the requests the generated client cannot express.
type rawClient struct {
	client *client.DatabricksClient
}

func (r *rawClient) CreateVersion(ctx context.Context, deploymentID, versionID string, body CreateVersionRequest) (*bundledeployments.Version, error) {
	var version bundledeployments.Version
	path := fmt.Sprintf("/api/2.0/bundle/deployments/%s/versions", deploymentID)
	err := r.client.Do(ctx, http.MethodPost, path,
		auth.WorkspaceIDHeaders(r.client.Config),
		map[string]any{"version_id": versionID},
		body, &version)
	if err != nil {
		return nil, err
	}
	return &version, nil
}

// newUpdateRequest builds the request body for update. Only what the mask names is sent:
// the service would ignore the rest, and state is the largest field by far, so a failure
// that keeps the recorded state sends none.
func newUpdateRequest(update OperationUpdate, sequenceID string) updateOperationRequest {
	body := updateOperationRequest{
		ErrorMessage: update.ErrorMessage,
		Status:       update.Status,
		SequenceId:   sequenceID,
	}
	if update.Fields.Has(FieldState) {
		body.State = string(update.State)
		body.ResourceId = update.ResourceID
	}
	return body
}

func (r *rawClient) UpdateOperation(ctx context.Context, deploymentID string, version int64, key ResourceKey, sequenceID string, update OperationUpdate) (string, error) {
	body := newUpdateRequest(update, sequenceID)

	var result operationResponse
	path := fmt.Sprintf("/api/2.0/bundle/deployments/%s/versions/%d/operations/%s", deploymentID, version, key)
	err := r.client.Do(ctx, http.MethodPatch, path,
		auth.WorkspaceIDHeaders(r.client.Config),
		map[string]any{"update_mask": update.Fields.Mask()},
		body, &result)
	if err != nil {
		return "", err
	}
	return result.SequenceId, nil
}
