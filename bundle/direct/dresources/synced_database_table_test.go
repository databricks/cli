package dresources

import "github.com/databricks/cli/bundle/config/resources"

func init() {
	registerTestConfig("synced_database_tables", &resources.SyncedDatabaseTable{
		Name: "main.myschema.my_synced_table",
	})

	registerCreateRecoveryOptOut("synced_database_tables", "the read response has no create time; the Unity Catalog table it registers would have to be read for one")
}
