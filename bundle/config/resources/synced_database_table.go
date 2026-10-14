package resources

import (
	"context"
	"github.com/databricks/cli/libs/sdkclients"
	"github.com/databricks/cli/libs/sdktypes/database/v1"
	"github.com/databricks/cli/libs/utils"
	sdkdatabase "github.com/databricks/sdk-go/database/v1"
	"net/url"

	"github.com/databricks/cli/libs/log"
	"github.com/databricks/cli/libs/workspaceurls"
)

type SyncedDatabaseTable struct {
	BaseResource
	ID string `json:"id,omitempty" bundle:"readonly"`
	database.SyncedDatabaseTable
}

func (s *SyncedDatabaseTable) Exists(ctx context.Context, w *sdkclients.Clients, name string) (bool, error) {
	_, err := w.Database.GetSyncedDatabaseTable(ctx, sdkdatabase.GetSyncedDatabaseTableRequest{Name: new(name)})
	if err != nil {
		log.Debugf(ctx, "synced database table %s does not exist", name)
		return false, err
	}
	return true, nil
}

func (s *SyncedDatabaseTable) ResourceDescription() ResourceDescription {
	return ResourceDescription{
		SingularName:  "synced_database_table",
		PluralName:    "synced_database_tables",
		SingularTitle: "Synced database table",
		PluralTitle:   "Synced database tables",
	}
}

func (s *SyncedDatabaseTable) GetName() string {
	return utils.Deref(s.Name)
}

func (s *SyncedDatabaseTable) InitializeURL(baseURL url.URL) {
	if s.Name == nil {
		return
	}
	s.URL = workspaceurls.ResourceURL(baseURL, "synced_database_tables", *s.Name)
}
