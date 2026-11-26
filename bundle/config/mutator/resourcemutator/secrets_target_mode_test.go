package resourcemutator

import (
	"github.com/databricks/cli/bundle/config"
	"github.com/databricks/cli/bundle/config/resources"
	"github.com/databricks/databricks-sdk-go/service/catalog"
)

func init() {
	registerTargetModeExamples("secrets", func(r *config.Resources) {
		r.Secrets = map[string]*resources.Secret{
			"secret1": {
				Secret: catalog.Secret{
					CatalogName: "main",
					SchemaName:  "default",
					Name:        "secret1",
				},
			},
		}
	})
}
