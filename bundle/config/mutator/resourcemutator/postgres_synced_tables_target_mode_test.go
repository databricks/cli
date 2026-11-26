package resourcemutator

import (
	"github.com/databricks/cli/bundle/config"
	"github.com/databricks/cli/bundle/config/resources"
)

func init() {
	registerTargetModeExamples("postgres_synced_tables", func(r *config.Resources) {
		r.PostgresSyncedTables = map[string]*resources.PostgresSyncedTable{
			"postgres_synced_table1": {
				PostgresSyncedTableConfig: resources.PostgresSyncedTableConfig{
					SyncedTableId: "catalog.schema.table1",
				},
			},
		}
	})
}
