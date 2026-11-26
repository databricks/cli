package resourcemutator

import (
	"github.com/databricks/cli/bundle/config"
	"github.com/databricks/cli/bundle/config/resources"
)

func init() {
	registerTargetModeExamples("postgres_branches", func(r *config.Resources) {
		r.PostgresBranches = map[string]*resources.PostgresBranch{
			"postgres_branch1": {
				PostgresBranchConfig: resources.PostgresBranchConfig{
					BranchId: "postgres-branch-1",
					Parent:   "projects/postgres-project-1",
				},
			},
		}
	})
}
