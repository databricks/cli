package generate

import (
	"github.com/databricks/cli/libs/dyn"
	"github.com/databricks/cli/libs/dyn/convert"
	"github.com/databricks/databricks-sdk-go/service/jobs"
)

var taskFieldOrder = []string{"TaskKey", "DependsOn", "ExistingClusterId", "NewCluster", "JobClusterKey"}
var jobFieldOrder = []string{"Name", "Format", "NewCluster", "JobClusters", "ExistingClusterId", "Compute", "Tasks"}

func ConvertJobToValue(job *jobs.Job) (dyn.Value, error) {
	jobOrder := dyn.NewOrder(jobFieldOrder)
	taskOrder := dyn.NewOrder(taskFieldOrder)
	value := make(map[string]dyn.Value)

	if job.Settings.Tasks != nil {
		k, _ := dyn.ConfigKey(job.Settings, "Tasks")
		tasks := make([]dyn.Value, 0)
		for _, task := range job.Settings.Tasks {
			v, err := convertTaskToValue(task, taskOrder)
			if err != nil {
				return dyn.NilValue, err
			}
			tasks = append(tasks, v)
		}
		value[k] = dyn.NewValue(tasks, dyn.Location{Line: jobOrder.Get("Tasks")})
	}

	return convert.ConvertToMapValue(job.Settings, jobOrder, value)
}

func convertTaskToValue(task jobs.Task, order *dyn.Order) (dyn.Value, error) {
	dst := make(map[string]dyn.Value)
	return convert.ConvertToMapValue(task, order, dst)
}
