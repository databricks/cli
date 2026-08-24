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
// of two halves: the generated client for the calls it can express, and hand-written requests
// for the two it cannot.
type Client struct {
	// Service is the generated client.
	Service bundledeployments.BundleDeploymentsInterface

	// raw sends what the generated client cannot; see requester.
	raw requester
}

// NewClient returns a Client for the workspace w.
func NewClient(w *databricks.WorkspaceClient) (*Client, error) {
	api, err := client.New(w.Config)
	if err != nil {
		return nil, err
	}
	return &Client{Service: w.BundleDeployments, raw: &rawClient{client: api}}, nil
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

// CreateVersion claims the version and stages the operations body carries.
func (c *Client) CreateVersion(ctx context.Context, deploymentID string, version int64, body CreateVersionRequest) (*bundledeployments.Version, error) {
	return c.raw.CreateVersion(ctx, deploymentID, strconv.FormatInt(version, 10), body)
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
// next update for that resource must send.
func (c *Client) UpdateOperation(ctx context.Context, deploymentID string, version int64, key ResourceKey, sequenceID string, update OperationUpdate) (string, error) {
	return c.raw.UpdateOperation(ctx, deploymentID, version, key, sequenceID, update)
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

// requester sends the two requests the generated client cannot express, so a test can capture
// what the CLI puts on the wire. Both are TODO(DMS): drop them once the spec catches up.
type requester interface {
	// CreateVersion is hand-written because the generated struct has no operations: the field
	// is at DEVELOPMENT stage, which keeps it out of the SDK until it is promoted.
	CreateVersion(ctx context.Context, deploymentID, versionID string, body CreateVersionRequest) (*bundledeployments.Version, error)

	// UpdateOperation is hand-written because the SDK types sequence_id as an int64 while
	// the service sends a JSON string, so it cannot read the response. sequenceID is the
	// token the previous update for this resource returned, or 0 for the first, which is
	// what staging leaves.
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
	// Operations is every resource this version will touch; see StagedOperation. It sits in this
	// body with the version's own fields because the request binds body: "version", and is input
	// only - the response never carries it back.
	Operations []StagedOperation `json:"operations,omitempty"`
}

// StagedOperation is one resource the version will record an operation for. The service
// creates it in OPERATION_STATUS_PENDING at sequence id 0, and the CLI fills in the outcome
// with UpdateOperation as the resource is applied.
type StagedOperation struct {
	ResourceKey ResourceKey                           `json:"resource_key"`
	ActionType  bundledeployments.OperationActionType `json:"action_type"`
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
	path := "/api/2.0/bundle/" + deploymentName(deploymentID) + "/versions"
	err := r.client.Do(ctx, http.MethodPost, path,
		auth.WorkspaceIDHeaders(r.client.Config),
		map[string]any{"version_id": versionID},
		body, &version)
	if err != nil {
		return nil, err
	}
	return &version, nil
}

// newUpdateRequest builds the request body for update. A field is in the body when the mask
// names it and absent otherwise, which is what the service requires: it rejects an update
// whose mask names a field the body leaves out, and an empty value is how a field is cleared -
// no state means the resource is gone, no error_message means an earlier failure is resolved.
// A map, not a struct, so presence cannot drift from the mask through an omitempty tag.
func newUpdateRequest(update OperationUpdate, sequenceID string) map[string]any {
	body := map[string]any{"sequence_id": sequenceID}
	if update.Fields.Has(FieldState) {
		body["state"] = string(update.State)
	}
	if update.Fields.Has(FieldResourceID) {
		body["resource_id"] = update.ResourceID
	}
	if update.Fields.Has(FieldErrorMessage) {
		body["error_message"] = update.ErrorMessage
	}
	if update.Fields.Has(FieldStatus) {
		body["status"] = update.Status
	}
	return body
}

func (r *rawClient) UpdateOperation(ctx context.Context, deploymentID string, version int64, key ResourceKey, sequenceID string, update OperationUpdate) (string, error) {
	body := newUpdateRequest(update, sequenceID)

	var result operationResponse
	path := "/api/2.0/bundle/" + versionName(deploymentID, version) + "/operations/" + string(key)
	err := r.client.Do(ctx, http.MethodPatch, path,
		auth.WorkspaceIDHeaders(r.client.Config),
		map[string]any{"update_mask": update.Fields.Mask()},
		body, &result)
	if err != nil {
		return "", err
	}
	return result.SequenceId, nil
}
