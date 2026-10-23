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

type PostgresDataApiConfig struct {
	postgres.DataApiDataApiSpec

	// Parent is the database whose Data API this resource manages. Format:
	// "projects/{project_id}/branches/{branch_id}/databases/{database_id}". The
	// Data API has no ID of its own: at most one exists per database, and its
	// resource name (and this resource's ID) is "{parent}/data-api".
	Parent string `json:"parent"`
}

func (c *PostgresDataApiConfig) UnmarshalJSON(b []byte) error {
	return marshal.Unmarshal(b, c)
}

func (c PostgresDataApiConfig) MarshalJSON() ([]byte, error) {
	return marshal.Marshal(c)
}

type PostgresDataApi struct {
	BaseResource
	ID string `json:"id,omitempty" bundle:"readonly"`
	PostgresDataApiConfig
}

func (d *PostgresDataApi) UnmarshalJSON(b []byte) error {
	return marshal.Unmarshal(b, d)
}

func (d PostgresDataApi) MarshalJSON() ([]byte, error) {
	return marshal.Marshal(d)
}

func (d *PostgresDataApi) Exists(ctx context.Context, w *databricks.WorkspaceClient, name string) (bool, error) {
	_, err := w.Postgres.GetDataApi(ctx, postgres.GetDataApiRequest{Name: name})
	if apierr.IsMissing(err) {
		log.Debugf(ctx, "postgres data api %s does not exist", name)
		return false, nil
	}
	if err != nil {
		return false, err
	}
	return true, nil
}

func (d *PostgresDataApi) ResourceDescription() ResourceDescription {
	return ResourceDescription{
		SingularName:  "postgres_data_api",
		PluralName:    "postgres_data_apis",
		SingularTitle: "Postgres Data API",
		PluralTitle:   "Postgres Data APIs",
	}
}

func (d *PostgresDataApi) GetName() string {
	// The Data API has no name field of its own; its resource name is its ID
	// ("{parent}/data-api").
	return d.ID
}

func (d *PostgresDataApi) GetURL() (string, bool) {
	// The IDs in the API do not (yet) map to IDs in the web UI.
	return "", false
}

func (d *PostgresDataApi) InitializeURL(_ url.URL) {
	// The IDs in the API do not (yet) map to IDs in the web UI.
}
