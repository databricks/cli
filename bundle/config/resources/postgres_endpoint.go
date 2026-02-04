package resources

import (
	"context"
	"net/url"

	"github.com/databricks/cli/libs/log"
	"github.com/databricks/databricks-sdk-go"
	"github.com/databricks/databricks-sdk-go/service/postgres"
)

type PostgresEndpoint struct {
	BaseResource
	postgres.EndpointSpec

	// Parent is the branch containing this endpoint. Format: "projects/{project_id}/branches/{branch_id}"
	Parent string `json:"parent"`

	// EndpointId is the user-specified ID for the endpoint (becomes part of the hierarchical name).
	// This is specified during creation and becomes part of Name: "projects/{project_id}/branches/{branch_id}/endpoints/{endpoint_id}"
	EndpointId string `json:"endpoint_id"`

	// Name is the hierarchical resource name (output-only). Format: "projects/{project_id}/branches/{branch_id}/endpoints/{endpoint_id}"
	Name string `json:"name,omitempty" bundle:"readonly"`
}

func (e *PostgresEndpoint) Exists(ctx context.Context, w *databricks.WorkspaceClient, name string) (bool, error) {
	_, err := w.Postgres.GetEndpoint(ctx, postgres.GetEndpointRequest{Name: name})
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
	// Endpoints don't have a user-visible name field
	return ""
}

func (e *PostgresEndpoint) GetURL() string {
	// The IDs in the API do not (yet) map to IDs in the web UI.
	return ""
}

func (e *PostgresEndpoint) InitializeURL(_ url.URL) {
	// The IDs in the API do not (yet) map to IDs in the web UI.
}
