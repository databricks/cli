package resources

import (
	"context"
	"net/url"

	"github.com/databricks/cli/libs/apierrs"
	"github.com/databricks/cli/libs/sdkclients"
	"github.com/databricks/cli/libs/sdktypes/postgres/v1"
	sdkpostgres "github.com/databricks/sdk-go/postgres/v1"

	"github.com/databricks/cli/libs/log"
)

type PostgresDatabaseConfig struct {
	postgres.Database_DatabaseSpec

	// DatabaseId is the user-specified ID for the database (becomes part of the hierarchical name).
	// This is specified during creation and becomes part of Name: "projects/{project_id}/branches/{branch_id}/databases/{database_id}"
	DatabaseId string `json:"database_id"`

	// Parent is the branch containing this database. Format: "projects/{project_id}/branches/{branch_id}"
	Parent string `json:"parent"`

	// ReplaceExisting, when true, takes over an existing database with the same ID
	// instead of returning ALREADY_EXISTS. Used to manage a database that already
	// exists on the branch. Input-only: not returned by the GET API.
	ReplaceExisting bool `json:"replace_existing,omitempty"`
}

type PostgresDatabase struct {
	BaseResource
	ID string `json:"id,omitempty" bundle:"readonly"`
	PostgresDatabaseConfig
}

func (d *PostgresDatabase) Exists(ctx context.Context, w *sdkclients.Clients, name string) (bool, error) {
	_, err := w.Postgres.GetDatabase(ctx, sdkpostgres.GetDatabaseRequest{Name: new(name)})
	if apierrs.IsMissing(err) {
		log.Debugf(ctx, "postgres database %s does not exist", name)
		return false, nil
	}
	if err != nil {
		return false, err
	}
	return true, nil
}

func (d *PostgresDatabase) ResourceDescription() ResourceDescription {
	return ResourceDescription{
		SingularName:  "postgres_database",
		PluralName:    "postgres_databases",
		SingularTitle: "Postgres database",
		PluralTitle:   "Postgres databases",
	}
}

func (d *PostgresDatabase) GetName() string {
	// Databases have no name field of their own; the resource name is the ID
	// ("projects/{project_id}/branches/{branch_id}/databases/{database_id}").
	return d.ID
}

func (d *PostgresDatabase) GetURL() (string, bool) {
	// The IDs in the API do not (yet) map to IDs in the web UI.
	return "", false
}

func (d *PostgresDatabase) InitializeURL(_ url.URL) {
	// The IDs in the API do not (yet) map to IDs in the web UI.
}
