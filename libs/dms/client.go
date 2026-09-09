package dms

import (
	"context"
	"fmt"
	"strconv"
	"strings"

	"github.com/databricks/databricks-sdk-go"
	"github.com/databricks/databricks-sdk-go/common/types/fieldmask"
	"github.com/databricks/databricks-sdk-go/service/bundledeployments"
)

// DeploymentNodeName is the workspace node DMS creates per deployment. Must
// match DeploymentWhsClient.DEPLOYMENT_NODE_NAME on the service side.
const DeploymentNodeName = "resources.deployment.json"

// statePrefix is what a bundle state key carries and a DMS resource key does not: state calls a
// job "resources.jobs.foo", DMS calls it "jobs.foo". Every exported name here takes the state
// form; the prefix comes off where a request is built, and back on where a resource is read.
const statePrefix = "resources."

// Client carries the calls the CLI makes to DMS, as methods below. Each one goes out through the
// generated SDK client.
type Client struct {
	// Service is the generated client.
	Service bundledeployments.BundleDeploymentsInterface
}

// NewClient returns a Client for the workspace w.
func NewClient(w *databricks.WorkspaceClient) (*Client, error) {
	return &Client{Service: w.BundleDeployments}, nil
}

// DeploymentName and versionName are the two resource-name formats the service uses. Every
// call builds its name here, so a caller only ever passes ids.
func DeploymentName(deploymentID string) string {
	return "deployments/" + deploymentID
}

func versionName(deploymentID string, version int) string {
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
	_, err := c.Service.UpdateDeployment(ctx, bundledeployments.UpdateDeploymentRequest{
		Name:       DeploymentName(deploymentID),
		Deployment: newDeploymentUpdate(metadata, mask),
		UpdateMask: fieldmask.FieldMask{Paths: strings.Split(mask, ",")},
	})
	return err
}

// DeleteDeployment removes the deployment record, which a completed destroy does.
func (c *Client) DeleteDeployment(ctx context.Context, deploymentID string) error {
	return c.Service.DeleteDeployment(ctx, bundledeployments.DeleteDeploymentRequest{
		Name: DeploymentName(deploymentID),
	})
}

// CreateVersion claims the version and stages the operations body carries.
func (c *Client) CreateVersion(ctx context.Context, deploymentID string, version int, body CreateVersionRequest) (*bundledeployments.Version, error) {
	operations := make([]bundledeployments.StagedOperation, len(body.Operations))
	for i, op := range body.Operations {
		operations[i] = bundledeployments.StagedOperation{
			ActionType:  op.ActionType,
			ResourceKey: strings.TrimPrefix(op.ResourceKey, statePrefix),
		}
	}

	return c.Service.CreateVersion(ctx, bundledeployments.CreateVersionRequest{
		Parent:    DeploymentName(deploymentID),
		VersionId: strconv.Itoa(version),
		Version: bundledeployments.Version{
			CliVersion:        body.CliVersion,
			VersionType:       body.VersionType,
			PreviousVersionId: body.PreviousVersionId,
			GitInfo:           body.GitInfo,
			Operations:        operations,
		},
	})
}

// CompleteVersion closes the version out, which is what stops the service expiring its lease.
func (c *Client) CompleteVersion(ctx context.Context, deploymentID string, version int, reason bundledeployments.VersionComplete) error {
	_, err := c.Service.CompleteVersion(ctx, bundledeployments.CompleteVersionRequest{
		Name:             versionName(deploymentID, version),
		CompletionReason: reason,
	})
	return err
}

// UpdateOperation fills in one operation the version staged, and returns the sequence id the
// next update for that resource must send.
func (c *Client) UpdateOperation(ctx context.Context, deploymentID string, version int, stateKey, sequenceID string, update OperationUpdate) (string, error) {
	operation, err := newOperationUpdate(update, sequenceID)
	if err != nil {
		return "", err
	}

	result, err := c.Service.UpdateOperation(ctx, bundledeployments.UpdateOperationRequest{
		Name:       versionName(deploymentID, version) + "/operations/" + strings.TrimPrefix(stateKey, statePrefix),
		Operation:  operation,
		UpdateMask: fieldmask.FieldMask{Paths: strings.Split(update.Fields.Mask(), ",")},
	})
	if err != nil {
		return "", err
	}
	return strconv.FormatInt(result.SequenceId, 10), nil
}

// deploymentIDFromName extracts the deployment ID from a DMS resource name of
// the form "deployments/{deployment_id}".
func deploymentIDFromName(name string) (string, error) {
	id, ok := strings.CutPrefix(name, DeploymentName(""))
	if !ok || id == "" {
		return "", fmt.Errorf("unexpected deployment name %q from the deployment history service", name)
	}
	return id, nil
}

// CreateVersionRequest is the input to Client.CreateVersion.
type CreateVersionRequest struct {
	CliVersion  string
	VersionType VersionType
	// PreviousVersionId is the deployment's most recent version, unset for a
	// deployment's first version.
	PreviousVersionId string
	// GitInfo records where this version's source came from. The rest of the provenance -
	// display name, target, mode, workspace paths - belongs to the deployment.
	GitInfo *bundledeployments.GitInfo
	// Operations is every resource this version will touch; see StagedOperation.
	Operations []StagedOperation
}

// StagedOperation is one resource the version will record an operation for. The service
// creates it in OPERATION_STATUS_PENDING at sequence id 0, and the CLI fills in the outcome
// with UpdateOperation as the resource is applied.
type StagedOperation struct {
	// ResourceKey is the bundle state key; CreateVersion strips the prefix to the form the service uses.
	ResourceKey string
	ActionType  bundledeployments.OperationActionType
}

// newDeploymentUpdate builds the deployment carrying exactly the masked fields, empty ones
// included: the service requires every masked field to be present in the body and reads an empty
// value as a clear (a target that stops setting mode clears deployment_mode). ForceSendFields
// keeps those empty values on the wire, which omitempty would drop.
func newDeploymentUpdate(metadata Metadata, mask string) bundledeployments.Deployment {
	full := metadata.deployment()
	var dep bundledeployments.Deployment
	for path := range strings.SplitSeq(mask, ",") {
		switch path {
		case "display_name":
			dep.DisplayName = full.DisplayName
			dep.ForceSendFields = append(dep.ForceSendFields, "DisplayName")
		case "target_name":
			dep.TargetName = full.TargetName
			dep.ForceSendFields = append(dep.ForceSendFields, "TargetName")
		case "deployment_mode":
			dep.DeploymentMode = full.DeploymentMode
			dep.ForceSendFields = append(dep.ForceSendFields, "DeploymentMode")
		case "workspace_info":
			dep.WorkspaceInfo = full.WorkspaceInfo
			dep.ForceSendFields = append(dep.ForceSendFields, "WorkspaceInfo")
		}
	}
	return dep
}

// newOperationUpdate builds the operation carrying exactly the fields update.Fields masks, plus
// the sequence_id precondition. The service requires a masked field to be present and reads an
// empty value as a write (error_message="" clears it), so masked fields that can be empty are
// forced onto the wire; sequence_id is always sent and a freshly staged operation sits at 0,
// which omitempty would drop. State is the exception: an absent value is how the service is told
// the resource is gone, so a nil state is left off (never forced) while still named in the mask.
func newOperationUpdate(update OperationUpdate, sequenceID string) (bundledeployments.Operation, error) {
	sequence, err := strconv.ParseInt(sequenceID, 10, 64)
	if err != nil {
		return bundledeployments.Operation{}, fmt.Errorf("invalid sequence id %q: %w", sequenceID, err)
	}

	operation := bundledeployments.Operation{
		SequenceId:      sequence,
		ForceSendFields: []string{"SequenceId"},
	}
	if update.Fields.Has(FieldState) && update.State != nil {
		operation.State = string(update.State)
	}
	if update.Fields.Has(FieldErrorMessage) {
		operation.ErrorMessage = update.ErrorMessage
		operation.ForceSendFields = append(operation.ForceSendFields, "ErrorMessage")
	}
	if update.Fields.Has(FieldResourceID) {
		operation.ResourceId = update.ResourceID
		operation.ForceSendFields = append(operation.ForceSendFields, "ResourceId")
	}
	if update.Fields.Has(FieldStatus) {
		operation.Status = update.Status
		operation.ForceSendFields = append(operation.ForceSendFields, "Status")
	}
	return operation, nil
}
