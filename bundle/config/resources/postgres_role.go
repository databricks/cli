package resources

import (
	"context"
	"github.com/databricks/cli/libs/apierrs"
	"github.com/databricks/cli/libs/sdkclients"
	"github.com/databricks/cli/libs/sdktypes/postgres/v1"
	sdkpostgres "github.com/databricks/sdk-go/postgres/v1"
	"net/url"

	"github.com/databricks/cli/libs/log"
)

type PostgresRoleConfig struct {
	postgres.Role_RoleSpec

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

type PostgresRole struct {
	BaseResource
	ID string `json:"id,omitempty" bundle:"readonly"`
	PostgresRoleConfig
}

func (r *PostgresRole) Exists(ctx context.Context, w *sdkclients.Clients, name string) (bool, error) {
	_, err := w.Postgres.GetRole(ctx, sdkpostgres.GetRoleRequest{Name: new(name)})
	if apierrs.IsMissing(err) {
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
	// Roles have no name field of their own; the resource name is the ID
	// ("projects/{project_id}/branches/{branch_id}/roles/{role_id}").
	return r.ID
}

func (r *PostgresRole) GetURL() (string, bool) {
	// The IDs in the API do not (yet) map to IDs in the web UI.
	return "", false
}

func (r *PostgresRole) InitializeURL(_ url.URL) {
	// The IDs in the API do not (yet) map to IDs in the web UI.
}
