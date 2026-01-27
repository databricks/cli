package dresources

import (
	"context"
	"fmt"

	"github.com/databricks/cli/bundle/config/resources"
	"github.com/databricks/databricks-sdk-go"
	"github.com/databricks/databricks-sdk-go/common/types/fieldmask"
	"github.com/databricks/databricks-sdk-go/service/postgres"
)

type ResourcePostgresEndpoint struct {
	client *databricks.WorkspaceClient
}

// PostgresEndpointState contains only the fields needed for creation/update.
// It does NOT include output-only fields like Name, which are only available after API response.
type PostgresEndpointState struct {
	Parent     string                 `json:"parent,omitempty"`
	Spec       *postgres.EndpointSpec `json:"spec,omitempty"`
	EndpointId string                 `json:"endpoint_id,omitempty"`
}

func (*ResourcePostgresEndpoint) New(client *databricks.WorkspaceClient) *ResourcePostgresEndpoint {
	return &ResourcePostgresEndpoint{client: client}
}

func (*ResourcePostgresEndpoint) PrepareState(input *resources.PostgresEndpoint) *PostgresEndpointState {
	return &PostgresEndpointState{
		Parent:     input.Parent,
		Spec:       &input.EndpointSpec,
		EndpointId: input.EndpointId,
	}
}

func (*ResourcePostgresEndpoint) RemapState(remote *postgres.Endpoint) *PostgresEndpointState {
	return &PostgresEndpointState{
		Parent: remote.Parent,
		Spec:   remote.Spec,
		// EndpointId is not available in remote state, it's already part of the Name
	}
}

func (r *ResourcePostgresEndpoint) DoRead(ctx context.Context, id string) (*postgres.Endpoint, error) {
	return r.client.Postgres.GetEndpoint(ctx, postgres.GetEndpointRequest{Name: id})
}

func (r *ResourcePostgresEndpoint) DoCreate(ctx context.Context, config *PostgresEndpointState) (string, *postgres.Endpoint, error) {
	endpointId := config.EndpointId
	if endpointId == "" {
		return "", nil, fmt.Errorf("endpoint_id must be specified")
	}

	parent := config.Parent
	if parent == "" {
		return "", nil, fmt.Errorf("parent (branch name) must be specified")
	}

	waiter, err := r.client.Postgres.CreateEndpoint(ctx, postgres.CreateEndpointRequest{
		EndpointId: endpointId,
		Parent:     parent,
		Endpoint: postgres.Endpoint{
			Spec: config.Spec,
		},
	})
	if err != nil {
		return "", nil, err
	}

	// Wait for the endpoint to be ready (long-running operation)
	result, err := waiter.Wait(ctx)
	if err != nil {
		return "", nil, err
	}

	return result.Name, result, nil
}

func (r *ResourcePostgresEndpoint) DoUpdate(ctx context.Context, id string, config *PostgresEndpointState, _ Changes) (*postgres.Endpoint, error) {
	waiter, err := r.client.Postgres.UpdateEndpoint(ctx, postgres.UpdateEndpointRequest{
		Endpoint: postgres.Endpoint{
			Name: id,
			Spec: config.Spec,
		},
		Name: id,
		UpdateMask: fieldmask.FieldMask{
			Paths: []string{"*"},
		},
	})
	if err != nil {
		return nil, err
	}

	// Wait for the update to complete
	result, err := waiter.Wait(ctx)
	return result, err
}

func (r *ResourcePostgresEndpoint) DoDelete(ctx context.Context, id string) error {
	waiter, err := r.client.Postgres.DeleteEndpoint(ctx, postgres.DeleteEndpointRequest{
		Name: id,
	})
	if err != nil {
		return err
	}
	return waiter.Wait(ctx)
}
