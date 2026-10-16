package resourcemutator

import (
	"context"

	"github.com/databricks/cli/bundle"
	"github.com/databricks/cli/bundle/config/resources"
	"github.com/databricks/cli/libs/diag"
	"github.com/databricks/databricks-sdk-go/service/jobs"
)

type mergeJobClusters struct{}

func MergeJobClusters() bundle.Mutator {
	return &mergeJobClusters{}
}

func (m *mergeJobClusters) Name() string {
	return "MergeJobClusters"
}

func (m *mergeJobClusters) Apply(ctx context.Context, b *bundle.Bundle) diag.Diagnostics {
	err := mergeByKey(b, "jobs", b.Config.Resources.Jobs, "job_clusters", "job_cluster_key",
		func(r *resources.Job) []jobs.JobCluster { return r.JobClusters },
		func(c jobs.JobCluster) string { return c.JobClusterKey },
		stringKey("job cluster key"), false)
	return diag.FromErr(err)
}
