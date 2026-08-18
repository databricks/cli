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
	return c.Versions.CreateVersion(ctx, deploymentID, strconv.FormatInt(version, 10), body)
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

// UpdateOperation fills in one operation the version staged; see OperationUpdater.
func (c *Client) UpdateOperation(ctx context.Context, deploymentID string, version int64, key ResourceKey, sequenceID string, update OperationUpdate) (string, error) {
	return c.Operations.UpdateOperation(ctx, deploymentID, version, key, sequenceID, update)
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
