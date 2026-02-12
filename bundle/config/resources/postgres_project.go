package resources

import (
	"context"
	"net/url"

	"github.com/databricks/cli/libs/log"
	"github.com/databricks/databricks-sdk-go"
	"github.com/databricks/databricks-sdk-go/marshal"
	"github.com/databricks/databricks-sdk-go/service/postgres"
)

type PostgresProjectConfig struct {
	postgres.ProjectSpec

	// ProjectId is the user-specified ID for the project (becomes part of the hierarchical name).
	// This is specified during creation and becomes part of Name: "projects/{project_id}"
	ProjectId string `json:"project_id"`
}

func (c *PostgresProjectConfig) UnmarshalJSON(b []byte) error {
	return marshal.Unmarshal(b, c)
}

func (c *PostgresProjectConfig) MarshalJSON() ([]byte, error) {
	return marshal.Marshal(c)
}

type PostgresProject struct {
	BaseResource
	PostgresProjectConfig
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
	return p.DisplayName
}

func (p *PostgresProject) GetURL() string {
	// The IDs in the API do not (yet) map to IDs in the web UI.
	return ""
}

func (p *PostgresProject) InitializeURL(_ url.URL) {
	// The IDs in the API do not (yet) map to IDs in the web UI.
}
