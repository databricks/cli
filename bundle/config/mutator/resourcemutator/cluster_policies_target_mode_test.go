package resourcemutator

import (
	"github.com/databricks/cli/bundle/config"
	"github.com/databricks/cli/bundle/config/resources"
	"github.com/databricks/databricks-sdk-go/service/compute"
)

func init() {
	registerTargetModeExamples("cluster_policies", func(r *config.Resources) {
		r.ClusterPolicies = map[string]*resources.ClusterPolicy{
			"cluster_policy1": {CreatePolicy: compute.CreatePolicy{Name: "cluster_policy1"}},
		}
	})
}
