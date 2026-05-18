package resources

import (
	"context"
	"net/url"

	"github.com/databricks/cli/libs/log"
	"github.com/databricks/databricks-sdk-go"
	"github.com/databricks/databricks-sdk-go/marshal"
	"github.com/databricks/databricks-sdk-go/service/postgres"
)

type PostgresDatabaseConfig struct {
	postgres.DatabaseDatabaseSpec

	// DatabaseId is the user-specified ID for the database (becomes part of the hierarchical name).
	// This is specified during creation and becomes part of Name: "projects/{project_id}/branches/{branch_id}/databases/{database_id}"
	DatabaseId string `json:"database_id"`

	// Parent is the branch containing this database. Format: "projects/{project_id}/branches/{branch_id}"
	Parent string `json:"parent"`
}

func (c *PostgresDatabaseConfig) UnmarshalJSON(b []byte) error {
	return marshal.Unmarshal(b, c)
}

func (c *PostgresDatabaseConfig) MarshalJSON() ([]byte, error) {
	return marshal.Marshal(c)
}

type PostgresDatabase struct {
	BaseResource
	PostgresDatabaseConfig
}

func (d *PostgresDatabase) Exists(ctx context.Context, w *databricks.WorkspaceClient, name string) (bool, error) {
	_, err := w.Postgres.GetDatabase(ctx, postgres.GetDatabaseRequest{Name: name})
	if err != nil {
		log.Debugf(ctx, "postgres database %s does not exist", name)
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
	return d.DatabaseId
}

func (d *PostgresDatabase) GetURL() string {
	// The IDs in the API do not (yet) map to IDs in the web UI.
	return ""
}

func (d *PostgresDatabase) InitializeURL(_ url.URL) {
	// The IDs in the API do not (yet) map to IDs in the web UI.
}
