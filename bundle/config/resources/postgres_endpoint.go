package resources

import (
	"context"
	"net/url"
	"strings"

	"github.com/databricks/cli/libs/log"
	"github.com/databricks/databricks-sdk-go"
	"github.com/databricks/databricks-sdk-go/service/postgres"
)

type PostgresEndpoint struct {
	BaseResource
	postgres.Endpoint

	// EndpointId is the user-specified ID for the endpoint (becomes part of the hierarchical name).
	// This is specified during creation and becomes part of Name: "projects/{project_id}/branches/{branch_id}/endpoints/{endpoint_id}"
	EndpointId string `json:"endpoint_id,omitempty"`

	// TODO: Enable when PostgresEndpointPermission is defined in Task 6
	// Permissions []PostgresEndpointPermission `json:"permissions,omitempty"`
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
	return e.URL
}

func (e *PostgresEndpoint) InitializeURL(baseURL url.URL) {
	if e.ModifiedStatus == ModifiedStatusCreated {
		return
	}
	if e.Name == "" {
		return
	}
	// Parse: projects/{project_id}/branches/{branch_id}/endpoints/{endpoint_id}
	parts := strings.Split(e.Name, "/")
	if len(parts) >= 6 {
		projectId := parts[1]
		branchId := parts[3]
		endpointId := parts[5]
		baseURL.Path = "postgres/projects/" + projectId + "/branches/" + branchId + "/endpoints/" + endpointId
		e.URL = baseURL.String()
	}
}
