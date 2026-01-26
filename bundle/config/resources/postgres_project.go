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
	postgres.Project

	// ProjectId is the user-specified ID for the project (becomes part of the hierarchical name).
	// This is specified during creation and becomes part of Name: "projects/{project_id}"
	ProjectId string `json:"project_id,omitempty"`

	// TODO: Enable when PostgresProjectPermission is defined in Task 6
	// Permissions []PostgresProjectPermission `json:"permissions,omitempty"`
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
	// Use display_name from spec if set
	if p.Spec != nil && p.Spec.DisplayName != "" {
		return p.Spec.DisplayName
	}
	// Use display_name from status if set (after deployment)
	if p.Status != nil && p.Status.DisplayName != "" {
		return p.Status.DisplayName
	}
	return ""
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
