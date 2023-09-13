package jobs_utils

import (
	"github.com/databricks/cli/bundle"
	"github.com/databricks/databricks-sdk-go/service/jobs"
)

type TaskWithJobKey struct {
	Task   *jobs.Task
	JobKey string
}

func GetTasksWithJobKeyBy(b *bundle.Bundle, filter func(*jobs.Task) bool) []TaskWithJobKey {
	tasks := make([]TaskWithJobKey, 0)
	for k := range b.Config.Resources.Jobs {
		for i := range b.Config.Resources.Jobs[k].Tasks {
			t := &b.Config.Resources.Jobs[k].Tasks[i]
			if filter(t) {
				tasks = append(tasks, TaskWithJobKey{
					JobKey: k,
					Task:   t,
				})
			}
		}
	}
	return tasks
}
