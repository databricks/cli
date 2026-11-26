package resourcemutator

import (
	"github.com/databricks/cli/bundle/config"
	"github.com/databricks/cli/bundle/config/resources"
)

func init() {
	registerTargetModeExamples("secret_scopes", func(r *config.Resources) {
		r.SecretScopes = map[string]*resources.SecretScope{
			"secretScope1": {
				Name: "secretScope1",
			},
		}
	})
}
