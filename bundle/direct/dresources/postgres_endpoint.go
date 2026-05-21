package dresources

import (
	"context"
	"errors"
	"net/http"
	"strings"
	"time"

	"github.com/databricks/cli/bundle/config/resources"
	"github.com/databricks/databricks-sdk-go"
	"github.com/databricks/databricks-sdk-go/apierr"
	"github.com/databricks/databricks-sdk-go/common/types/fieldmask"
	sdktime "github.com/databricks/databricks-sdk-go/common/types/time"
	"github.com/databricks/databricks-sdk-go/marshal"
	"github.com/databricks/databricks-sdk-go/service/postgres"
)

// endpointReconciliationTimeout is the maximum time to wait for endpoint reconciliation.
// This value is a heuristic and is being discussed with the backend team.
const endpointReconciliationTimeout = 2 * time.Minute

// PostgresEndpointRemote is the return type for DoRead. It embeds EndpointSpec so
// that all paths in StateType are valid paths in RemoteType, enabling drift
// detection for spec fields once the backend echoes spec on GET.
type PostgresEndpointRemote struct {
	postgres.EndpointSpec

	EndpointId string `json:"endpoint_id,omitempty"`
	Parent     string `json:"parent,omitempty"`

	Name       string                   `json:"name,omitempty"`
	Status     *postgres.EndpointStatus `json:"status,omitempty"`
	Uid        string                   `json:"uid,omitempty"`
	CreateTime *sdktime.Time            `json:"create_time,omitempty"`
	UpdateTime *sdktime.Time            `json:"update_time,omitempty"`
}

// Custom marshaler needed because embedded EndpointSpec has its own MarshalJSON
// which would otherwise take over and ignore the additional fields.
func (s *PostgresEndpointRemote) UnmarshalJSON(b []byte) error {
	return marshal.Unmarshal(b, s)
}

func (s PostgresEndpointRemote) MarshalJSON() ([]byte, error) {
	return marshal.Marshal(s)
}

type ResourcePostgresEndpoint struct {
	client *databricks.WorkspaceClient
}

type PostgresEndpointState = resources.PostgresEndpointConfig

func (*ResourcePostgresEndpoint) New(client *databricks.WorkspaceClient) *ResourcePostgresEndpoint {
	return &ResourcePostgresEndpoint{client: client}
}

func (*ResourcePostgresEndpoint) PrepareState(input *resources.PostgresEndpoint) *PostgresEndpointState {
	return &PostgresEndpointState{
		EndpointId:      input.EndpointId,
		Parent:          input.Parent,
		ReplaceExisting: input.ReplaceExisting,
		EndpointSpec:    input.EndpointSpec,
	}
}

func (*ResourcePostgresEndpoint) RemapState(remote *PostgresEndpointRemote) *PostgresEndpointState {
	return &PostgresEndpointState{
		EndpointId: remote.EndpointId,
		Parent:     remote.Parent,

		// replace_existing is a create-time-only flag; the GET API never returns
		// it, so RemapState leaves it false.
		ReplaceExisting: false,

		EndpointSpec: remote.EndpointSpec,
	}
}

// makePostgresEndpointRemote converts the SDK Endpoint into the embedded remote shape.
// GET does not echo spec today (only status is returned); the embedded spec fields
// stay at their zero values, and resources.yml suppresses phantom drift via
// ignore_remote_changes with reason spec:input_only.
func makePostgresEndpointRemote(endpoint *postgres.Endpoint) *PostgresEndpointRemote {
	var spec postgres.EndpointSpec
	if endpoint.Spec != nil {
		spec = *endpoint.Spec
	}
	var endpointID string
	if endpoint.Status != nil {
		endpointID = endpoint.Status.EndpointId
	}
	return &PostgresEndpointRemote{
		EndpointSpec: spec,
		EndpointId:   endpointID,
		Parent:       endpoint.Parent,
		Name:         endpoint.Name,
		Status:       endpoint.Status,
		Uid:          endpoint.Uid,
		CreateTime:   endpoint.CreateTime,
		UpdateTime:   endpoint.UpdateTime,
	}
}

func (r *ResourcePostgresEndpoint) DoRead(ctx context.Context, id string) (*PostgresEndpointRemote, error) {
	endpoint, err := r.client.Postgres.GetEndpoint(ctx, postgres.GetEndpointRequest{Name: id})
	if err != nil {
		return nil, err
	}
	return makePostgresEndpointRemote(endpoint), nil
}

// waitForReconciliation polls the endpoint until PendingState is empty.
// This is needed because the operation can complete while internal reconciliation
// is still in progress, which would cause subsequent operations to fail.
func (r *ResourcePostgresEndpoint) waitForReconciliation(ctx context.Context, name string) (*PostgresEndpointRemote, error) {
	deadline := time.Now().Add(endpointReconciliationTimeout)
	for {
		endpoint, err := r.client.Postgres.GetEndpoint(ctx, postgres.GetEndpointRequest{Name: name})
		if err != nil {
			return nil, err
		}

		// If there's no pending state, reconciliation is complete
		if endpoint.Status == nil || endpoint.Status.PendingState == "" {
			return makePostgresEndpointRemote(endpoint), nil
		}

		// Check if we've exceeded the timeout
		if time.Now().After(deadline) {
			return nil, errors.New("timeout waiting for endpoint reconciliation to complete")
		}

		// Wait before polling again
		select {
		case <-ctx.Done():
			return nil, ctx.Err()
		case <-time.After(2 * time.Second):
		}
	}
}

func (r *ResourcePostgresEndpoint) DoCreate(ctx context.Context, config *PostgresEndpointState) (string, *PostgresEndpointRemote, error) {
	waiter, err := r.client.Postgres.CreateEndpoint(ctx, postgres.CreateEndpointRequest{
		EndpointId: config.EndpointId,
		Parent:     config.Parent,
		Endpoint: postgres.Endpoint{
			Spec: &config.EndpointSpec,

			// Output-only fields.
			CreateTime:      nil,
			Name:            "",
			Parent:          "",
			Status:          nil,
			Uid:             "",
			UpdateTime:      nil,
			ForceSendFields: nil,
		},
		ReplaceExisting: config.ReplaceExisting,
		ForceSendFields: nil,
	})
	if err != nil {
		return "", nil, err
	}

	// Wait for the operation to complete
	result, err := waiter.Wait(ctx)
	if err != nil {
		return "", nil, err
	}

	// Wait for reconciliation to complete
	remote, err := r.waitForReconciliation(ctx, result.Name)
	if err != nil {
		return "", nil, err
	}

	return remote.Name, remote, nil
}

func (r *ResourcePostgresEndpoint) DoUpdate(ctx context.Context, id string, config *PostgresEndpointState, entry *PlanEntry) (*PostgresEndpointRemote, error) {
	// Build update mask from fields that have action="update" in the changes map.
	// This excludes immutable fields and fields that haven't changed.
	// Prefix with "spec." because the API expects paths relative to the Endpoint object,
	// not relative to our flattened state type.
	fieldPaths := collectUpdatePathsWithPrefix(entry.Changes, "spec.")

	waiter, err := r.client.Postgres.UpdateEndpoint(ctx, postgres.UpdateEndpointRequest{
		Endpoint: postgres.Endpoint{
			Spec: &config.EndpointSpec,

			// Output-only fields.
			CreateTime:      nil,
			Name:            "",
			Parent:          "",
			Status:          nil,
			Uid:             "",
			UpdateTime:      nil,
			ForceSendFields: nil,
		},
		Name: id,
		UpdateMask: fieldmask.FieldMask{
			Paths: fieldPaths,
		},
	})
	if err != nil {
		return nil, err
	}

	// Wait for the update to complete
	_, err = waiter.Wait(ctx)
	if err != nil {
		return nil, err
	}

	// Wait for reconciliation to complete
	return r.waitForReconciliation(ctx, id)
}

func (r *ResourcePostgresEndpoint) DoDelete(ctx context.Context, id string) error {
	// Retry loop to handle "Endpoint reconciliation still in progress" errors
	deadline := time.Now().Add(endpointReconciliationTimeout)
	for {
		waiter, err := r.client.Postgres.DeleteEndpoint(ctx, postgres.DeleteEndpointRequest{
			Name: id,
		})
		if err != nil {
			// Check if this is a reconciliation in progress error
			if apiErr, ok := errors.AsType[*apierr.APIError](err); ok && apiErr.StatusCode == http.StatusConflict &&
				strings.Contains(apiErr.Message, "reconciliation") {
				// Check if we've exceeded the timeout
				if time.Now().After(deadline) {
					return errors.New("timeout waiting for endpoint reconciliation to complete before delete")
				}
				// Wait and retry
				select {
				case <-ctx.Done():
					return ctx.Err()
				case <-time.After(2 * time.Second):
					continue
				}
			}
			return err
		}
		return waiter.Wait(ctx)
	}
}
