package resources

import (
	"context"
	"net/url"

	"github.com/databricks/cli/libs/sdkclients"
	"github.com/databricks/cli/libs/sdktypes/postgres/v1"
	"github.com/databricks/cli/libs/utils"
	sdkpostgres "github.com/databricks/sdk-go/postgres/v1"

	"github.com/databricks/cli/libs/log"
)

type PostgresProjectConfig struct {
	postgres.ProjectSpec

	// ProjectId is the user-specified ID for the project (becomes part of the hierarchical name).
	// This is specified during creation and becomes part of Name: "projects/{project_id}"
	ProjectId string `json:"project_id"`

	// PurgeOnDelete, when true, hard-deletes the project on destroy (Purge=true on
	// DeleteProject). When false or unset, the backend performs a soft delete that
	// can be undone within the project's retention window. Input-only: not
	// returned by the GET API.
	PurgeOnDelete *bool `json:"purge_on_delete,omitempty"`
}

type PostgresProject struct {
	BaseResource
	ID string `json:"id,omitempty" bundle:"readonly"`
	PostgresProjectConfig

	Permissions []Permission `json:"permissions,omitempty"`
}

func (p *PostgresProject) Exists(ctx context.Context, w *sdkclients.Clients, name string) (bool, error) {
	_, err := w.Postgres.GetProject(ctx, sdkpostgres.GetProjectRequest{Name: new(name)})
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
	return utils.Deref(p.DisplayName)
}

func (p *PostgresProject) GetURL() (string, bool) {
	// The IDs in the API do not (yet) map to IDs in the web UI.
	return "", false
}

func (p *PostgresProject) InitializeURL(_ url.URL) {
	// The IDs in the API do not (yet) map to IDs in the web UI.
}
