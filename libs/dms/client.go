package dms

import (
	"context"
	"fmt"
	"net/http"
	"strconv"
	"strings"

	"github.com/databricks/cli/libs/auth"
	"github.com/databricks/databricks-sdk-go"
	"github.com/databricks/databricks-sdk-go/client"
	"github.com/databricks/databricks-sdk-go/service/bundledeployments"
)

// Client is every call the CLI makes to DMS, as methods below. Each one goes out through one
// of two halves: the generated client for the calls it can express, and a request written by
// hand for the two it cannot.
type Client struct {
	// Service is the generated client.
	Service bundledeployments.BundleDeploymentsInterface

	// api sends the two requests the generated client cannot express; see CreateVersion and
	// UpdateOperation. Both are TODO(DMS): drop them once the spec catches up.
	api *client.DatabricksClient
}

// NewClient returns a Client for the workspace w.
func NewClient(w *databricks.WorkspaceClient) (*Client, error) {
	api, err := client.New(w.Config)
	if err != nil {
		return nil, err
	}
	return &Client{Service: w.BundleDeployments, api: api}, nil
}

// deploymentName and versionName are the two resource-name formats the service uses. Every
// call builds its name here, so a caller only ever passes ids.
func deploymentName(deploymentID string) string {
	return "deployments/" + deploymentID
}

func versionName(deploymentID string, version int64) string {
	return fmt.Sprintf("deployments/%s/versions/%d", deploymentID, version)
}

// CreateDeployment registers a deployment under parentPath and returns the id the server
// assigned it, which is the id of the workspace node it creates there.
func (c *Client) CreateDeployment(ctx context.Context, parentPath, targetName string) (string, error) {
	dep, err := c.Service.CreateDeployment(ctx, bundledeployments.CreateDeploymentRequest{
		Deployment: bundledeployments.Deployment{
			InitialParentPath: parentPath,
			TargetName:        targetName,
		},
	})
	if err != nil {
		return "", err
	}
	return deploymentIDFromName(dep.Name)
}

// GetDeployment reads the deployment record, which carries the last version recorded under it.
func (c *Client) GetDeployment(ctx context.Context, deploymentID string) (*bundledeployments.Deployment, error) {
	return c.Service.GetDeployment(ctx, bundledeployments.GetDeploymentRequest{
		Name: deploymentName(deploymentID),
	})
}

// DeleteDeployment removes the deployment record, which a completed destroy does.
func (c *Client) DeleteDeployment(ctx context.Context, deploymentID string) error {
	return c.Service.DeleteDeployment(ctx, bundledeployments.DeleteDeploymentRequest{
		Name: deploymentName(deploymentID),
	})
}

// CreateVersion claims the version and stages the operations body carries. Written by hand
// because the generated struct has no previous_version_id, which the service needs as its
// concurrency check - without it every deploy after the first is rejected.
func (c *Client) CreateVersion(ctx context.Context, deploymentID string, version int64, body CreateVersionRequest) (*bundledeployments.Version, error) {
	var created bundledeployments.Version
	path := "/api/2.0/bundle/" + deploymentName(deploymentID) + "/versions"
	err := c.api.Do(ctx, http.MethodPost, path,
		auth.WorkspaceIDHeaders(c.api.Config),
		map[string]any{"version_id": strconv.FormatInt(version, 10)},
		body, &created)
	if err != nil {
		return nil, err
	}
	return &created, nil
}

// CompleteVersion closes the version out, which is what stops the service expiring its lease.
func (c *Client) CompleteVersion(ctx context.Context, deploymentID string, version int64, reason bundledeployments.VersionComplete) error {
	_, err := c.Service.CompleteVersion(ctx, bundledeployments.CompleteVersionRequest{
		Name:             versionName(deploymentID, version),
		CompletionReason: reason,
	})
	return err
}

// Heartbeat renews the version's lease.
func (c *Client) Heartbeat(ctx context.Context, deploymentID string, version int64) error {
	_, err := c.Service.Heartbeat(ctx, bundledeployments.HeartbeatRequest{
		Name: versionName(deploymentID, version),
	})
	return err
}

// UpdateOperation fills in one operation the version staged, and returns the sequence id the
// next update for that resource must send. sequenceID is the token the previous update
// returned, or 0 for the first, which is what staging leaves.
//
// Written by hand because the SDK types sequence_id as an int64 while the service sends a
// JSON string, so the generated client cannot read the response.
func (c *Client) UpdateOperation(ctx context.Context, deploymentID string, version int64, key ResourceKey, sequenceID string, update OperationUpdate) (string, error) {
	var result operationResponse
	path := "/api/2.0/bundle/" + versionName(deploymentID, version) + "/operations/" + string(key)
	err := c.api.Do(ctx, http.MethodPatch, path,
		auth.WorkspaceIDHeaders(c.api.Config),
		map[string]any{"update_mask": update.Fields.Mask()},
		newUpdateRequest(update, sequenceID), &result)
	if err != nil {
		return "", err
	}
	return result.SequenceId, nil
}

// deploymentIDFromName extracts the deployment ID from a DMS resource name of
// the form "deployments/{deployment_id}".
func deploymentIDFromName(name string) (string, error) {
	id, ok := strings.CutPrefix(name, deploymentName(""))
	if !ok || id == "" {
		return "", fmt.Errorf("unexpected deployment name %q from deployment metadata service", name)
	}
	return id, nil
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

// newUpdateRequest builds the request body for update. Each field is sent because the mask
// names it: the service ignores the rest, and state is the largest field by far, so a
// failure that keeps the recorded state sends none of it.
func newUpdateRequest(update OperationUpdate, sequenceID string) updateOperationRequest {
	body := updateOperationRequest{SequenceId: sequenceID}
	if update.Fields.Has(FieldState) {
		body.State = string(update.State)
	}
	if update.Fields.Has(FieldResourceID) {
		body.ResourceId = update.ResourceID
	}
	if update.Fields.Has(FieldErrorMessage) {
		body.ErrorMessage = update.ErrorMessage
	}
	if update.Fields.Has(FieldStatus) {
		body.Status = update.Status
	}
	return body
}
