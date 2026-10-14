package resources

import (
	"context"
	"github.com/databricks/cli/libs/sdkclients"
	"github.com/databricks/cli/libs/sdktypes/postgres/v1"
	sdkpostgres "github.com/databricks/sdk-go/postgres/v1"
	"net/url"

	"github.com/databricks/cli/libs/log"
)

type PostgresEndpointConfig struct {
	postgres.EndpointSpec

	// EndpointId is the user-specified ID for the endpoint (becomes part of the hierarchical name).
	// This is specified during creation and becomes part of Name: "projects/{project_id}/branches/{branch_id}/endpoints/{endpoint_id}"
	EndpointId string `json:"endpoint_id"`

	// Parent is the branch containing this endpoint. Format: "projects/{project_id}/branches/{branch_id}"
	Parent string `json:"parent"`

	// ReplaceExisting, when true, takes over an existing endpoint with the same ID
	// instead of returning ALREADY_EXISTS. Used to manage the implicitly-created
	// primary read-write endpoint of a new branch. Input-only: not returned by the GET API.
	ReplaceExisting bool `json:"replace_existing,omitempty"`
}

type PostgresEndpoint struct {
	BaseResource
	ID string `json:"id,omitempty" bundle:"readonly"`
	PostgresEndpointConfig
}

func (e *PostgresEndpoint) Exists(ctx context.Context, w *sdkclients.Clients, name string) (bool, error) {
	_, err := w.Postgres.GetEndpoint(ctx, sdkpostgres.GetEndpointRequest{Name: new(name)})
	if err != nil {
		log.Debugf(ctx, "postgres endpoint %s does not exist", name)
		return false, err
	}
	return true, nil
}

func (e *PostgresEndpoint) ResourceDescription() ResourceDescription {
	return ResourceDescription{
		SingularName:  "postgres_endpoint",
		PluralName:    "postgres_endpoints",
		SingularTitle: "Postgres endpoint",
		PluralTitle:   "Postgres endpoints",
	}
}

func (e *PostgresEndpoint) GetName() string {
	// Endpoints have no name field of their own; the resource name is the ID
	// ("projects/{project_id}/branches/{branch_id}/endpoints/{endpoint_id}").
	return e.ID
}

func (e *PostgresEndpoint) GetURL() (string, bool) {
	// The IDs in the API do not (yet) map to IDs in the web UI.
	return "", false
}

func (e *PostgresEndpoint) InitializeURL(_ url.URL) {
	// The IDs in the API do not (yet) map to IDs in the web UI.
}
