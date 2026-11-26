package resourcemutator

import (
	"github.com/databricks/cli/bundle/config"
	"github.com/databricks/cli/bundle/config/resources"
	"github.com/databricks/databricks-sdk-go/service/database"
)

func init() {
	registerTargetModeExamples("synced_database_tables", func(r *config.Resources) {
		r.SyncedDatabaseTables = map[string]*resources.SyncedDatabaseTable{
			"synced_database_table1": {
				SyncedDatabaseTable: database.SyncedDatabaseTable{
					Name: "synced_database_table1",
				},
			},
		}
	})
}
