package resources

import (
	"context"
	"net/url"

	"github.com/databricks/cli/libs/log"
	"github.com/databricks/cli/libs/workspaceurls"

	"github.com/databricks/databricks-sdk-go"
	"github.com/databricks/databricks-sdk-go/marshal"
	"github.com/databricks/databricks-sdk-go/service/database"
)

type SyncedDatabaseTable struct {
	BaseResource
	ID string `json:"id,omitempty" bundle:"readonly"`
	database.SyncedDatabaseTable
}

func (s *SyncedDatabaseTable) UnmarshalJSON(b []byte) error {
	return marshal.Unmarshal(b, s)
}

func (s SyncedDatabaseTable) MarshalJSON() ([]byte, error) {
	return marshal.Marshal(s)
}

func (s *SyncedDatabaseTable) Exists(ctx context.Context, w *databricks.WorkspaceClient, name string) (bool, error) {
	_, err := w.Database.GetSyncedDatabaseTable(ctx, database.GetSyncedDatabaseTableRequest{Name: name})
	if err != nil {
		log.Debugf(ctx, "synced database table %s does not exist", name)
		return false, err
	}
	return true, nil
}

func (*SyncedDatabaseTable) RunAsSupport() RunAsSupport {
	return RunAsNotApplicable
}

func (*SyncedDatabaseTable) Approval() Approval {
	return Approval{
		Order: 60,
		Deploy: `
This action will result in the deletion or recreation of the following synced database tables.
The synced data in the destination database will be lost (the source table is preserved):`,
		Destroy: `This action will result in the deletion of the following synced database tables.
The synced data in the destination database will be lost (the source table is preserved):`,
	}
}

func (s *SyncedDatabaseTable) ResourceDescription() ResourceDescription {
	return ResourceDescription{
		SingularName:  "synced_database_table",
		PluralName:    "synced_database_tables",
		SingularTitle: "Synced database table",
		PluralTitle:   "Synced database tables",
	}
}

func (*SyncedDatabaseTable) NamePrefixExemptReason() string {
	return unityCatalogNamePrefixExemptReason
}

func (s *SyncedDatabaseTable) GetName() string {
	return s.Name
}

func (*SyncedDatabaseTable) WorkspaceURL() (workspaceurls.Pattern, bool) {
	return workspaceurls.Pattern{Path: "explore/data/%s"}, true
}

func (s *SyncedDatabaseTable) InitializeURL(baseURL url.URL) {
	if s.Name == "" {
		return
	}
	s.URL = workspaceURL(s, baseURL, s.Name)
}
