package dresources

import (
	"context"
	"errors"
	"strings"
	"time"

	"github.com/databricks/cli/bundle/config/resources"
	"github.com/databricks/databricks-sdk-go"
	"github.com/databricks/databricks-sdk-go/apierr"
	"github.com/databricks/databricks-sdk-go/common/types/fieldmask"
	"github.com/databricks/databricks-sdk-go/service/postgres"
)

// endpointReconciliationTimeout is the maximum time to wait for endpoint reconciliation.
// This value is a heuristic and is being discussed with the backend team.
const endpointReconciliationTimeout = 2 * time.Minute

type ResourcePostgresEndpoint struct {
	client *databricks.WorkspaceClient
}

type PostgresEndpointState = resources.PostgresEndpointConfig

func (*ResourcePostgresEndpoint) New(client *databricks.WorkspaceClient) *ResourcePostgresEndpoint {
	return &ResourcePostgresEndpoint{client: client}
}

func (*ResourcePostgresEndpoint) PrepareState(input *resources.PostgresEndpoint) *PostgresEndpointState {
	return &PostgresEndpointState{
		EndpointId:   input.EndpointId,
		Parent:       input.Parent,
		EndpointSpec: input.EndpointSpec,
	}
}

func (*ResourcePostgresEndpoint) RemapState(remote *postgres.Endpoint) *PostgresEndpointState {
	// Extract endpoint_id from hierarchical name: "projects/{project_id}/branches/{branch_id}/endpoints/{endpoint_id}"
	// TODO: log error when we have access to the context
	components, _ := ParsePostgresName(remote.Name)

	return &PostgresEndpointState{
		EndpointId: components.EndpointID,
		Parent:     remote.Parent,

		// The read API does not return the spec, only the status.
		// This means we cannot detect remote drift for spec fields.
		// Use an empty struct (not nil) so field-level diffing works correctly.
		EndpointSpec: postgres.EndpointSpec{
			AutoscalingLimitMaxCu:  0,
			AutoscalingLimitMinCu:  0,
			Disabled:               false,
			EndpointType:           "",
			Group:                  nil,
			NoSuspension:           false,
			Settings:               nil,
			SuspendTimeoutDuration: nil,
			ForceSendFields:        nil,
		},
	}
}

func (r *ResourcePostgresEndpoint) DoRead(ctx context.Context, id string) (*postgres.Endpoint, error) {
	return r.client.Postgres.GetEndpoint(ctx, postgres.GetEndpointRequest{Name: id})
}

// waitForReconciliation polls the endpoint until PendingState is empty.
// This is needed because the operation can complete while internal reconciliation
// is still in progress, which would cause subsequent operations to fail.
func (r *ResourcePostgresEndpoint) waitForReconciliation(ctx context.Context, name string) (*postgres.Endpoint, error) {
	deadline := time.Now().Add(endpointReconciliationTimeout)
	for {
		endpoint, err := r.client.Postgres.GetEndpoint(ctx, postgres.GetEndpointRequest{Name: name})
		if err != nil {
			return nil, err
		}

		// If there's no pending state, reconciliation is complete
		if endpoint.Status == nil || endpoint.Status.PendingState == "" {
			return endpoint, nil
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

func (r *ResourcePostgresEndpoint) DoCreate(ctx context.Context, config *PostgresEndpointState) (string, *postgres.Endpoint, error) {
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
	result, err = r.waitForReconciliation(ctx, result.Name)
	if err != nil {
		return "", nil, err
	}

	return result.Name, result, nil
}

func (r *ResourcePostgresEndpoint) DoUpdate(ctx context.Context, id string, config *PostgresEndpointState, changes Changes) (*postgres.Endpoint, error) {
	// Build update mask from fields that have action="update" in the changes map.
	// This excludes immutable fields and fields that haven't changed.
	// Prefix with "spec." because the API expects paths relative to the Endpoint object,
	// not relative to our flattened state type.
	fieldPaths := collectUpdatePathsWithPrefix(changes, "spec.")

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
			var apiErr *apierr.APIError
			if errors.As(err, &apiErr) && apiErr.StatusCode == 409 &&
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
