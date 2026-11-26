package resourcemutator

import (
	"github.com/databricks/cli/bundle/config"
	"github.com/databricks/cli/bundle/config/resources"
	"github.com/databricks/databricks-sdk-go/service/database"
)

func init() {
	registerTargetModeExamples("database_instances", func(r *config.Resources) {
		r.DatabaseInstances = map[string]*resources.DatabaseInstance{
			"database_instance1": {
				DatabaseInstance: database.DatabaseInstance{
					Name: "database_instance1",
				},
			},
		}
	})
}
