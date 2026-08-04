package direct

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"strings"

	"github.com/databricks/cli/bundle/deployplan"
	"github.com/databricks/cli/bundle/direct/dstate"
	"github.com/databricks/cli/libs/log"
	"github.com/databricks/databricks-sdk-go/apierr"
	"github.com/databricks/databricks-sdk-go/service/bundledeployments"
)

// maxOperationStateSize is the largest serialized state DMS accepts per
// operation. Uploading more is rejected server-side, so fail early with a
// message that names the resource.
const maxOperationStateSize = 64 * 1024

// recordedOperation is an applied resource operation, serialized and waiting to be
// uploaded to the deployment metadata service (DMS).
//
// The payload is built on the apply worker rather than in the uploader so the
// queue does not hold on to the live resource struct, and so a malformed state
// fails the resource that produced it instead of the drain at the end of apply.
type recordedOperation struct {
	action     bundledeployments.OperationActionType
	resourceID string

	// state is the serialized local config after the operation. It is nil for a
	// delete, where the resource no longer exists.
	state json.RawMessage
}

// newRecordedOperation serializes an applied operation for upload. state is the
// local config after the operation and must be nil for delete operations. It
// errors when the serialized state exceeds maxOperationStateSize.
func newRecordedOperation(action deployplan.ActionType, resourceID string, state any, dependsOn []deployplan.DependsOnEntry) (recordedOperation, error) {
	actionType, err := deployActionToSDK(action)
	if err != nil {
		return recordedOperation{}, err
	}

	op := recordedOperation{action: actionType, resourceID: resourceID}

	// Operation.State carries the serialized state, which DMS serves back as
	// resource state. Unset for delete: the resource is gone.
	if state != nil {
		config, err := json.Marshal(state)
		if err != nil {
			return recordedOperation{}, fmt.Errorf("serializing state: %w", err)
		}
		raw, err := json.Marshal(dstate.RecordedState{State: config, DependsOn: dependsOn})
		if err != nil {
			return recordedOperation{}, fmt.Errorf("serializing state: %w", err)
		}
		if len(raw) > maxOperationStateSize {
			return recordedOperation{}, fmt.Errorf("serialized state is %d bytes, which exceeds the %d byte limit for recording deployment history", len(raw), maxOperationStateSize)
		}
		op.state = raw
	}

	return op, nil
}

// operationUploader records an applied resource operation with DMS. Uploads run
// on the operationQueue workers, off the apply path.
type operationUploader interface {
	upload(ctx context.Context, resourceKey string, op recordedOperation) error
}

// operationRecorder uploads operations via the DMS CreateOperation API.
type operationRecorder struct {
	client bundledeployments.BundleDeploymentsInterface
	// parent is the version the operations are recorded under, formatted as
	// "deployments/{deployment_id}/versions/{version_id}".
	parent string
}

// NewOperationRecorder returns an operationUploader backed by the DMS
// CreateOperation API. deploymentID and version identify the deployment version
// assigned by DMS that the operations are recorded under.
func NewOperationRecorder(client bundledeployments.BundleDeploymentsInterface, deploymentID string, version int64) operationUploader {
	return &operationRecorder{
		client: client,
		parent: fmt.Sprintf("deployments/%s/versions/%d", deploymentID, version),
	}
}

func (r *operationRecorder) upload(ctx context.Context, resourceKey string, op recordedOperation) error {
	// DMS resource keys are unprefixed (e.g. "jobs.foo"), while the CLI's state
	// keys carry a leading "resources." (e.g. "resources.jobs.foo"). Strip it on
	// the way out; the read path re-adds it (see dstate.fetchDeploymentResources).
	dmsKey := strings.TrimPrefix(resourceKey, "resources.")

	operation := bundledeployments.Operation{
		ActionType:  op.action,
		ResourceId:  op.resourceID,
		ResourceKey: dmsKey,
		Status:      bundledeployments.OperationStatusOperationStatusSucceeded,
	}
	if op.state != nil {
		// Operation.State is an opaque UTF-8 JSON string the service stores
		// unchanged (see the SDK docs), so send the serialized state as-is.
		operation.State = string(op.state)
	}

	_, err := r.client.CreateOperation(ctx, bundledeployments.CreateOperationRequest{
		Parent:      r.parent,
		ResourceKey: dmsKey,
		Operation:   operation,
	})
	// The CLI discards the response, so a failure to deserialize it does not mean
	// the operation was not recorded: DMS serves sequence_id as a JSON string
	// ("1") per proto3 int64 encoding, but bundledeployments.Operation.SequenceId
	// is an int64 the SDK cannot parse from a string. This surfaces intermittently
	// (only responses that carry sequence_id trip it) as a spurious deploy
	// failure on a call the server accepted.
	//
	// The SDK emits this only on the 2xx body-parse path - a status >= 400 is
	// mapped to *apierr.APIError before the body is read - so a "failed to
	// unmarshal response body" error means the operation was recorded. Tolerate
	// exactly that, and nothing broader: a transport error means the request may
	// not have reached DMS and must still fail the deploy.
	if err != nil && !errors.As(err, new(*apierr.APIError)) && strings.Contains(err.Error(), "failed to unmarshal response body") {
		log.Debugf(ctx, "ignoring response deserialization error from CreateOperation for %s (operation was recorded): %v", dmsKey, err)
		return nil
	}
	return err
}

// deployActionToSDK maps a deployplan action to its DMS operation action type.
// Only actions that mutate a resource are recordable; Skip and Undefined never
// reach a recorder and are rejected rather than silently coerced.
func deployActionToSDK(a deployplan.ActionType) (bundledeployments.OperationActionType, error) {
	switch a {
	case deployplan.Create:
		return bundledeployments.OperationActionTypeOperationActionTypeCreate, nil
	case deployplan.Update:
		return bundledeployments.OperationActionTypeOperationActionTypeUpdate, nil
	case deployplan.UpdateWithID:
		return bundledeployments.OperationActionTypeOperationActionTypeUpdateWithId, nil
	case deployplan.Recreate:
		return bundledeployments.OperationActionTypeOperationActionTypeRecreate, nil
	case deployplan.Resize:
		return bundledeployments.OperationActionTypeOperationActionTypeResize, nil
	case deployplan.Delete:
		return bundledeployments.OperationActionTypeOperationActionTypeDelete, nil
	default:
		return "", fmt.Errorf("cannot record operation: unsupported action %q", a)
	}
}
