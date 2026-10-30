package resourcemutator

import (
	"context"

	"github.com/databricks/cli/bundle"
	"github.com/databricks/cli/bundle/config/resources"
	"github.com/databricks/cli/libs/diag"
	"github.com/databricks/databricks-sdk-go/service/jobs"
)

type mergeJobTasks struct{}

func MergeJobTasks() bundle.Mutator {
	return &mergeJobTasks{}
}

func (m *mergeJobTasks) Name() string {
	return "MergeJobTasks"
}

func (m *mergeJobTasks) Apply(ctx context.Context, b *bundle.Bundle) diag.Diagnostics {
	// Sorting keys here since it'll be sorted by TF anyway
	// https://github.com/databricks/terraform-provider-databricks/blob/0a932c2/jobs/resource_job.go#L343
	// However, if we don't sort we have a difference between direct and TF and between configs in
	// "bundle validate" and configs sent to backend.
	err := mergeByKey(b, "jobs", b.Config.Resources.Jobs, "tasks", "task_key",
		func(r *resources.Job) []jobs.Task { return r.Tasks },
		func(t jobs.Task) string { return t.TaskKey },
		stringKey("task key"), true)
	return diag.FromErr(err)
}
