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

// statePrefix is what a bundle state key carries and a DMS resource key does not: state calls a
// job "resources.jobs.foo", DMS calls it "jobs.foo". Every exported name here takes the state
// form; the prefix comes off where a request is built, and back on where a resource is read.
const statePrefix = "resources."

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
func (c *Client) CreateDeployment(ctx context.Context, parentPath string, metadata Metadata) (string, error) {
	dep := metadata.deployment()
	dep.InitialParentPath = parentPath

	created, err := c.Service.CreateDeployment(ctx, bundledeployments.CreateDeploymentRequest{Deployment: dep})
	if err != nil {
		return "", err
	}
	return deploymentIDFromName(created.Name)
}

// UpdateDeployment writes the fields mask names onto the deployment. The service ignores every
// other field, so the mask is what decides the write.
func (c *Client) UpdateDeployment(ctx context.Context, deploymentID string, metadata Metadata, mask string) error {
	return c.raw.UpdateDeployment(ctx, deploymentID, metadata.deployment(), mask)
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
func (c *Client) UpdateOperation(ctx context.Context, deploymentID string, version int64, stateKey, sequenceID string, update OperationUpdate) (string, error) {
	return c.raw.UpdateOperation(ctx, deploymentID, version, stateKey, sequenceID, update)
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

	// UpdateDeployment is hand-written because the generated client has no such call yet.
	UpdateDeployment(ctx context.Context, deploymentID string, deployment bundledeployments.Deployment, mask string) error

	// UpdateOperation is hand-written because the SDK types sequence_id as an int64 while
	// the service sends a JSON string, so it cannot read the response. sequenceID is the
	// token the previous update for this resource returned, or 0 for the first, which is
	// what staging leaves.
	UpdateOperation(ctx context.Context, deploymentID string, version int64, stateKey, sequenceID string, update OperationUpdate) (next string, err error)
}

// CreateVersionRequest is the CreateVersion request body.
type CreateVersionRequest struct {
	CliVersion  string      `json:"cli_version"`
	VersionType VersionType `json:"version_type"`
	// PreviousVersionId is the deployment's most recent version, unset for a
	// deployment's first version.
	PreviousVersionId string `json:"previous_version_id,omitempty"`
	// GitInfo records where this version's source came from. The rest of the provenance -
	// display name, target, mode, workspace paths - belongs to the deployment.
	GitInfo *bundledeployments.GitInfo `json:"git_info,omitempty"`
	// Operations is every resource this version will touch; see StagedOperation. It sits in this
	// body with the version's own fields because the request binds body: "version", and is input
	// only - the response never carries it back.
	Operations []StagedOperation `json:"operations,omitempty"`
}

// StagedOperation is one resource the version will record an operation for. The service
// creates it in OPERATION_STATUS_PENDING at sequence id 0, and the CLI fills in the outcome
// with UpdateOperation as the resource is applied.
type StagedOperation struct {
	// ResourceKey is the bundle state key; the request carries the form the service uses.
	ResourceKey string                                `json:"resource_key"`
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
	staged := make([]StagedOperation, len(body.Operations))
	for i, op := range body.Operations {
		op.ResourceKey = strings.TrimPrefix(op.ResourceKey, statePrefix)
		staged[i] = op
	}
	body.Operations = staged

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

// newDeploymentUpdate builds the request body for update, holding exactly the masked fields for
// the same reason newUpdateRequest does. A map, not the SDK struct, whose omitempty tags would
// drop a masked field that is empty - which is how deployment_mode is cleared when a target stops
// setting mode.
func newDeploymentUpdate(deployment bundledeployments.Deployment, mask string) map[string]any {
	values := map[string]any{
		"display_name":    deployment.DisplayName,
		"target_name":     deployment.TargetName,
		"deployment_mode": deployment.DeploymentMode,
		"workspace_info":  deployment.WorkspaceInfo,
	}

	body := map[string]any{}
	for field := range strings.SplitSeq(mask, ",") {
		body[field] = values[field]
	}
	return body
}

func (r *rawClient) UpdateDeployment(ctx context.Context, deploymentID string, deployment bundledeployments.Deployment, mask string) error {
	path := "/api/2.0/bundle/" + deploymentName(deploymentID)
	return r.client.Do(ctx, http.MethodPatch, path,
		auth.WorkspaceIDHeaders(r.client.Config),
		map[string]any{"update_mask": mask},
		newDeploymentUpdate(deployment, mask), nil)
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

func (r *rawClient) UpdateOperation(ctx context.Context, deploymentID string, version int64, stateKey, sequenceID string, update OperationUpdate) (string, error) {
	body := newUpdateRequest(update, sequenceID)

	var result operationResponse
	path := "/api/2.0/bundle/" + versionName(deploymentID, version) + "/operations/" + strings.TrimPrefix(stateKey, statePrefix)
	err := r.client.Do(ctx, http.MethodPatch, path,
		auth.WorkspaceIDHeaders(r.client.Config),
		map[string]any{"update_mask": update.Fields.Mask()},
		body, &result)
	if err != nil {
		return "", err
	}
	return result.SequenceId, nil
}
