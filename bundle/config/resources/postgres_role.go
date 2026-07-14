package resources

import (
	"context"
	"net/url"

	"github.com/databricks/cli/libs/log"
	"github.com/databricks/databricks-sdk-go"
	"github.com/databricks/databricks-sdk-go/apierr"
	"github.com/databricks/databricks-sdk-go/marshal"
	"github.com/databricks/databricks-sdk-go/service/postgres"
)

type PostgresRoleConfig struct {
	postgres.RoleRoleSpec

	// RoleId is the user-specified ID for the role (becomes part of the hierarchical name).
	// This is specified during creation and becomes part of Name: "projects/{project_id}/branches/{branch_id}/roles/{role_id}"
	RoleId string `json:"role_id"`

	// Parent is the branch containing this role. Format: "projects/{project_id}/branches/{branch_id}"
	Parent string `json:"parent"`

	// ReplaceExisting, when true, takes over an existing role with the same ID
	// instead of returning ALREADY_EXISTS. Used to manage a role that already
	// exists on the branch (e.g. inherited from the parent branch). Input-only:
	// not returned by the GET API.
	ReplaceExisting bool `json:"replace_existing,omitempty"`
}

func (c *PostgresRoleConfig) UnmarshalJSON(b []byte) error {
	return marshal.Unmarshal(b, c)
}

func (c *PostgresRoleConfig) MarshalJSON() ([]byte, error) {
	return marshal.Marshal(c)
}

type PostgresRole struct {
	BaseResource
	PostgresRoleConfig
}

func (r *PostgresRole) Exists(ctx context.Context, w *databricks.WorkspaceClient, name string) (bool, error) {
	_, err := w.Postgres.GetRole(ctx, postgres.GetRoleRequest{Name: name})
	if apierr.IsMissing(err) {
		log.Debugf(ctx, "postgres role %s does not exist", name)
		return false, nil
	}
	if err != nil {
		return false, err
	}
	return true, nil
}

func (r *PostgresRole) ResourceDescription() ResourceDescription {
	return ResourceDescription{
		SingularName:  "postgres_role",
		PluralName:    "postgres_roles",
		SingularTitle: "Postgres role",
		PluralTitle:   "Postgres roles",
	}
}

func (r *PostgresRole) GetName() string {
	// Roles don't have a user-visible name field.
	return ""
}

func (r *PostgresRole) GetURL() string {
	// The IDs in the API do not (yet) map to IDs in the web UI.
	return ""
}

func (r *PostgresRole) InitializeURL(_ url.URL) {
	// The IDs in the API do not (yet) map to IDs in the web UI.
}
