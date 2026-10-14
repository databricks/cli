package dresources

import (
	"context"
	"errors"
	"net/http"
	"strings"
	"time"

	"github.com/databricks/cli/bundle/config/resources"
	"github.com/databricks/cli/libs/apierrs"
	"github.com/databricks/cli/libs/sdkclients"
	postgres "github.com/databricks/cli/libs/sdktypes/postgres/v1"
	"github.com/databricks/cli/libs/utils"
	"github.com/databricks/sdk-go/core/types"
	sdkpostgres "github.com/databricks/sdk-go/postgres/v1"
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
	CreateTime *types.Time              `json:"create_time,omitempty"`
	UpdateTime *types.Time              `json:"update_time,omitempty"`
}

type ResourcePostgresEndpoint struct {
	client *sdkclients.Clients
}

type PostgresEndpointState = resources.PostgresEndpointConfig

func (*ResourcePostgresEndpoint) New(client *sdkclients.Clients) *ResourcePostgresEndpoint {
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
// stay at their zero values, and postgres_endpoints.yml suppresses phantom drift via
// ignore_remote_changes with reason spec:input_only.
func makePostgresEndpointRemote(resp *sdkpostgres.Endpoint) (*PostgresEndpointRemote, error) {
	endpoint, err := postgres.EndpointFromSDK(resp)
	if err != nil {
		return nil, err
	}
	var spec postgres.EndpointSpec
	if endpoint.Spec != nil {
		spec = *endpoint.Spec
	}
	return &PostgresEndpointRemote{
		EndpointSpec: spec,
		EndpointId:   utils.Deref(endpoint.EndpointId),
		Parent:       utils.Deref(endpoint.Parent),
		Name:         utils.Deref(endpoint.Name),
		Status:       endpoint.Status,
		Uid:          utils.Deref(endpoint.Uid),
		CreateTime:   endpoint.CreateTime,
		UpdateTime:   endpoint.UpdateTime,
	}, nil
}

func (r *ResourcePostgresEndpoint) DoRead(ctx context.Context, id string) (*PostgresEndpointRemote, error) {
	endpoint, err := r.client.Postgres.GetEndpoint(ctx, sdkpostgres.GetEndpointRequest{Name: new(id)})
	if err != nil {
		return nil, err
	}
	return makePostgresEndpointRemote(endpoint)
}

// waitForReconciliation polls the endpoint until PendingState is empty.
// This is needed because the operation can complete while internal reconciliation
// is still in progress, which would cause subsequent operations to fail.
func (r *ResourcePostgresEndpoint) waitForReconciliation(ctx context.Context, name string) (*PostgresEndpointRemote, error) {
	deadline := time.Now().Add(endpointReconciliationTimeout)
	for {
		endpoint, err := r.client.Postgres.GetEndpoint(ctx, sdkpostgres.GetEndpointRequest{Name: new(name)})
		if err != nil {
			return nil, err
		}

		// If there's no pending state, reconciliation is complete
		if endpoint.Status == nil || endpoint.Status.PendingState == "" {
			return makePostgresEndpointRemote(endpoint)
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
	// Only spec is set; the other Endpoint fields are output-only.
	request, err := postgres.CreateEndpointRequestToSDK(&postgres.CreateEndpointRequest{
		EndpointId:      new(config.EndpointId),
		Parent:          new(config.Parent),
		Endpoint:        &postgres.Endpoint{Spec: &config.EndpointSpec},
		ReplaceExisting: trueOrNil(config.ReplaceExisting),
	})
	if err != nil {
		return "", nil, err
	}
	waiter, err := r.client.Postgres.CreateEndpoint(ctx, *request)
	if err != nil {
		return "", nil, err
	}

	// Wait for the operation to complete
	result, err := waiter.Wait(ctx)
	if err != nil {
		return "", nil, err
	}

	// Wait for reconciliation to complete
	remote, err := r.waitForReconciliation(ctx, utils.Deref(result.Name))
	if err != nil {
		return "", nil, err
	}

	return remote.Name, remote, nil
}

// no_suspension and suspend_timeout_duration are two sides of one oneof, which the API
// accepts in update_mask only under the group name; see branchOneofGroups.
var endpointOneofGroups = map[string]string{
	"no_suspension":            "suspension",
	"suspend_timeout_duration": "suspension",
}

func (r *ResourcePostgresEndpoint) DoUpdate(ctx context.Context, id string, config *PostgresEndpointState, entry *PlanEntry) (*PostgresEndpointRemote, error) {
	// Build update mask from fields that have action="update" in the changes map.
	// This excludes immutable fields and fields that haven't changed.
	// Prefix with "spec." because the API expects paths relative to the Endpoint object,
	// not relative to our flattened state type.
	fieldPaths := collectUpdatePathsWithPrefix(entry.Changes, "spec.", endpointOneofGroups)

	// Only name (which addresses the endpoint) and spec are set; the other
	// Endpoint fields are output-only.
	request, err := postgres.UpdateEndpointRequestToSDK(&postgres.UpdateEndpointRequest{
		Endpoint: &postgres.Endpoint{
			Name: new(id),
			Spec: &config.EndpointSpec,
		},
		UpdateMask: new(strings.Join(fieldPaths, ",")),
	})
	if err != nil {
		return nil, err
	}
	waiter, err := r.client.Postgres.UpdateEndpoint(ctx, *request)
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

func (r *ResourcePostgresEndpoint) DoDelete(ctx context.Context, id string, _ *PostgresEndpointState) error {
	// Retry loop to handle "Endpoint reconciliation still in progress" errors
	deadline := time.Now().Add(endpointReconciliationTimeout)
	for {
		waiter, err := r.client.Postgres.DeleteEndpoint(ctx, sdkpostgres.DeleteEndpointRequest{
			Name: new(id),
		})
		if err != nil {
			// Check if this is a reconciliation in progress error
			if apierrs.StatusCode(err) == http.StatusConflict && strings.Contains(apierrs.Message(err), "reconciliation") {
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
