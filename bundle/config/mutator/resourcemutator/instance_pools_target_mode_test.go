package resourcemutator

import (
	"github.com/databricks/cli/bundle/config"
	"github.com/databricks/cli/bundle/config/resources"
	"github.com/databricks/databricks-sdk-go/service/compute"
)

func init() {
	registerTargetModeExamples("instance_pools", func(r *config.Resources) {
		r.InstancePools = map[string]*resources.InstancePool{
			"instance_pool1": {CreateInstancePool: compute.CreateInstancePool{InstancePoolName: "instance_pool1", NodeTypeId: "i3.xlarge"}},
		}
	})
}
