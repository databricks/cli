package resourcemutator

import (
	"github.com/databricks/cli/bundle/config"
	"github.com/databricks/cli/bundle/config/resources"
	"github.com/databricks/databricks-sdk-go/service/sql"
)

func init() {
	registerTargetModeExamples("sql_warehouses", func(r *config.Resources) {
		r.SqlWarehouses = map[string]*resources.SqlWarehouse{
			"sql_warehouse1": {
				CreateWarehouseRequest: sql.CreateWarehouseRequest{
					Name: "sql_warehouse1",
				},
			},
		}
	})
}
