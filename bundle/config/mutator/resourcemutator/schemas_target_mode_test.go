package resourcemutator

import (
	"github.com/databricks/cli/bundle/config"
	"github.com/databricks/cli/bundle/config/resources"
	"github.com/databricks/databricks-sdk-go/service/catalog"
)

func init() {
	registerTargetModeExamples("schemas", func(r *config.Resources) {
		r.Schemas = map[string]*resources.Schema{
			"schema1": {CreateSchema: catalog.CreateSchema{Name: "schema1"}},
		}
	})
}
