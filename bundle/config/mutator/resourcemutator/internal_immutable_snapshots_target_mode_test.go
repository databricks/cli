package resourcemutator

import (
	"github.com/databricks/cli/bundle/config"
	"github.com/databricks/cli/bundle/config/resources"
)

func init() {
	registerTargetModeExamples("internal_immutable_snapshots", func(r *config.Resources) {
		r.Snapshots = map[string]*resources.Snapshot{
			"snapshot1": {
				BundleID: "bundle1",
				ACL:      nil,
			},
		}
	})
}
