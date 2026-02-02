package resources

import (
	"context"
	"net/url"
	"strings"

	"github.com/databricks/cli/libs/log"
	"github.com/databricks/databricks-sdk-go"
	"github.com/databricks/databricks-sdk-go/service/postgres"
)

type PostgresProject struct {
	BaseResource
	postgres.ProjectSpec

	// ProjectId is the user-specified ID for the project (becomes part of the hierarchical name).
	// This is specified during creation and becomes part of Name: "projects/{project_id}"
	ProjectId string `json:"project_id,omitempty"`

	// Name is the hierarchical resource name (output-only). Format: "projects/{project_id}"
	Name string `json:"name,omitempty" bundle:"readonly"`
}

func (p *PostgresProject) Exists(ctx context.Context, w *databricks.WorkspaceClient, name string) (bool, error) {
	_, err := w.Postgres.GetProject(ctx, postgres.GetProjectRequest{Name: name})
	if err != nil {
		log.Debugf(ctx, "postgres project %s does not exist", name)
		return false, err
	}
	return true, nil
}

func (p *PostgresProject) ResourceDescription() ResourceDescription {
	return ResourceDescription{
		SingularName:  "postgres_project",
		PluralName:    "postgres_projects",
		SingularTitle: "Postgres project",
		PluralTitle:   "Postgres projects",
	}
}

func (p *PostgresProject) GetName() string {
	// Return display_name directly (now embedded from ProjectSpec)
	return p.DisplayName
}

func (p *PostgresProject) GetURL() string {
	return p.URL
}

func (p *PostgresProject) InitializeURL(baseURL url.URL) {
	if p.ModifiedStatus == ModifiedStatusCreated {
		return
	}
	if p.Name == "" {
		return
	}
	// Extract project_id from hierarchical name: projects/{project_id}
	parts := strings.Split(p.Name, "/")
	if len(parts) >= 2 {
		projectId := parts[1]
		baseURL.Path = "postgres/projects/" + projectId
		p.URL = baseURL.String()
	}
}
