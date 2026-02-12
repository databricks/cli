package resources

import (
	"context"
	"net/url"

	"github.com/databricks/cli/libs/log"
	"github.com/databricks/databricks-sdk-go"
	"github.com/databricks/databricks-sdk-go/marshal"
	"github.com/databricks/databricks-sdk-go/service/postgres"
)

type PostgresEndpointConfig struct {
	postgres.EndpointSpec

	// EndpointId is the user-specified ID for the endpoint (becomes part of the hierarchical name).
	// This is specified during creation and becomes part of Name: "projects/{project_id}/branches/{branch_id}/endpoints/{endpoint_id}"
	EndpointId string `json:"endpoint_id"`

	// Parent is the branch containing this endpoint. Format: "projects/{project_id}/branches/{branch_id}"
	Parent string `json:"parent"`
}

func (c *PostgresEndpointConfig) UnmarshalJSON(b []byte) error {
	return marshal.Unmarshal(b, c)
}

func (c *PostgresEndpointConfig) MarshalJSON() ([]byte, error) {
	return marshal.Marshal(c)
}

type PostgresEndpoint struct {
	BaseResource
	PostgresEndpointConfig
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
