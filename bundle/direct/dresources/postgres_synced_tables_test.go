package dresources

import "github.com/databricks/cli/bundle/config/resources"

func init() {
	registerTestConfig("postgres_synced_tables", &resources.PostgresSyncedTable{
		SyncedTableId: "main.public.trips_synced",
	})
}
