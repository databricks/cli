package resourcemutator

import (
	"github.com/databricks/cli/bundle/config"
	"github.com/databricks/cli/bundle/config/resources"
	"github.com/databricks/databricks-sdk-go/service/compute"
)

func init() {
	registerTargetModeExamples("clusters", func(r *config.Resources) {
		r.Clusters = map[string]*resources.Cluster{
			"cluster1": {ClusterSpec: compute.ClusterSpec{ClusterName: "cluster1", SparkVersion: "13.2.x", NumWorkers: 1}},
		}
	})
}
