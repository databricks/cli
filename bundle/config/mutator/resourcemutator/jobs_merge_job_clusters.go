package resourcemutator

import (
	"context"

	"github.com/databricks/cli/bundle"
	"github.com/databricks/cli/bundle/config/stages"
	"github.com/databricks/cli/libs/diag"
	"github.com/databricks/cli/libs/dyn"
	"github.com/databricks/cli/libs/dyn/merge"
)

func init() {
	// Reads (dynamic): resources.jobs.*.job_clusters (reads job clusters to merge)
	// Updates (dynamic): resources.jobs.*.job_clusters (merges job clusters with the same job_cluster_key)
	stages.Register(stages.StageAfterTargetMerge, "jobs", MergeJobClusters)
}

type mergeJobClusters struct{}

func MergeJobClusters() bundle.Mutator {
	return &mergeJobClusters{}
}

func (m *mergeJobClusters) Name() string {
	return "MergeJobClusters"
}

func (m *mergeJobClusters) jobClusterKey(v dyn.Value) string {
	switch v.Kind() {
	case dyn.KindInvalid, dyn.KindNil:
		return ""
	case dyn.KindString:
		return v.MustString()
	default:
		panic("job cluster key must be a string")
	}
}

func (m *mergeJobClusters) Apply(ctx context.Context, b *bundle.Bundle) diag.Diagnostics {
	err := b.Config.Mutate(func(v dyn.Value) (dyn.Value, error) {
		if v.Kind() == dyn.KindNil {
			return v, nil
		}

		return dyn.Map(v, "resources.jobs", dyn.Foreach(func(_ dyn.Path, job dyn.Value) (dyn.Value, error) {
			return dyn.Map(job, "job_clusters", merge.ElementsByKey("job_cluster_key", m.jobClusterKey))
		}))
	})

	return diag.FromErr(err)
}
