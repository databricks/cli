package mutator

import (
	"context"
	"sort"

	"github.com/databricks/cli/bundle"
	"github.com/databricks/cli/libs/diag"
)

type sortJobTasks struct{}

// SortJobTasks sorts the tasks of each job in the bundle by task key. Sorting
// the task keys ensures that the diff computed by terraform is correct. For
// more details see the NOTE at https://registry.terraform.io/providers/databricks/databricks/latest/docs/resources/job#example-usage
// and https://github.com/databricks/terraform-provider-databricks/issues/4011.
func SortJobTasks() bundle.Mutator {
	return &sortJobTasks{}
}

func (m *sortJobTasks) Name() string {
	return "SortJobTasks"
}

func (m *sortJobTasks) Apply(ctx context.Context, b *bundle.Bundle) diag.Diagnostics {
	for _, job := range b.Config.Resources.Jobs {
		sort.Slice(job.Tasks, func(i, j int) bool {
			return job.Tasks[i].TaskKey < job.Tasks[j].TaskKey
		})
	}

	return nil
}
