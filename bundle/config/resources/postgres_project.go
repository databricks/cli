package resources

import (
	"context"

	"github.com/databricks/cli/libs/log"
	"github.com/databricks/cli/libs/workspaceurls"
	"github.com/databricks/databricks-sdk-go"
	"github.com/databricks/databricks-sdk-go/marshal"
	"github.com/databricks/databricks-sdk-go/service/postgres"
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

func (c *PostgresProjectConfig) UnmarshalJSON(b []byte) error {
	return marshal.Unmarshal(b, c)
}

func (c PostgresProjectConfig) MarshalJSON() ([]byte, error) {
	return marshal.Marshal(c)
}

type PostgresProject struct {
	BaseResource
	ID string `json:"id,omitempty" bundle:"readonly"`
	PostgresProjectConfig

	Permissions []Permission `json:"permissions,omitempty"`
}

func (p *PostgresProject) UnmarshalJSON(b []byte) error {
	return marshal.Unmarshal(b, p)
}

func (p PostgresProject) MarshalJSON() ([]byte, error) {
	return marshal.Marshal(p)
}

func (p *PostgresProject) Exists(ctx context.Context, w *databricks.WorkspaceClient, name string) (bool, error) {
	_, err := w.Postgres.GetProject(ctx, postgres.GetProjectRequest{Name: name})
	if err != nil {
		log.Debugf(ctx, "postgres project %s does not exist", name)
		return false, err
	}
	return true, nil
}

func (*PostgresProject) RunAsSupport() RunAsSupport {
	return RunAsNotApplicable
}

func (p *PostgresProject) ResourceDescription() ResourceDescription {
	return ResourceDescription{
		SingularName:  "postgres_project",
		PluralName:    "postgres_projects",
		SingularTitle: "Postgres project",
		PluralTitle:   "Postgres projects",
	}
}

func (*PostgresProject) PermissionsSupport() PermissionsSupport {
	return PermissionsSupport{
		BundleLevels: BundlePermissionLevels{
			CanManage: "CAN_MANAGE",
			CanView:   "CAN_USE",
		},
		RequestObjectType:  "database-projects",
		ResponseObjectType: "database-projects",
		ObjectIDField:      "project_id", // bare project_id, not the hierarchical "projects/{id}" state ID
	}
}

func (*PostgresProject) NamePrefixExemptReason() string {
	return "the project is addressed by its ID (projects/{project_id}); its display_name is not prefixed today and, " +
		"unlike for dashboards, no decision has been recorded on whether it should be"
}

func (p *PostgresProject) GetName() string {
	return p.DisplayName
}

func (*PostgresProject) WorkspaceURL() (workspaceurls.Pattern, bool) {
	// The IDs in the API do not (yet) map to IDs in the web UI.
	return workspaceurls.Pattern{}, false
}
