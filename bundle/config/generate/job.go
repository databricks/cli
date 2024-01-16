package generate

import (
	"github.com/databricks/cli/libs/dyn"
	"github.com/databricks/cli/libs/dyn/yamlsaver"
	"github.com/databricks/databricks-sdk-go/service/jobs"
)

var jobOrder = yamlsaver.NewOrder([]string{"Name", "Format", "NewCluster", "JobClusters", "ExistingClusterId", "Compute", "Tasks"})
var taskOrder = yamlsaver.NewOrder([]string{"TaskKey", "DependsOn", "ExistingClusterId", "NewCluster", "JobClusterKey"})

func ConvertJobToValue(job *jobs.Job) (dyn.Value, error) {
	value := make(map[string]dyn.Value)

	if job.Settings.Tasks != nil {
		k, _ := yamlsaver.ConfigKey(job.Settings, "Tasks")
		tasks := make([]dyn.Value, 0)
		for _, task := range job.Settings.Tasks {
			v, err := convertTaskToValue(task, taskOrder)
			if err != nil {
				return dyn.NilValue, err
			}
			tasks = append(tasks, v)
		}
		// We're using location lines to define the order of keys in exported YAML.
		value[k] = dyn.NewValue(tasks, dyn.Location{Line: jobOrder.Get("Tasks")})
	}

	return yamlsaver.ConvertToMapValue(job.Settings, jobOrder, value)
}

func convertTaskToValue(task jobs.Task, order *yamlsaver.Order) (dyn.Value, error) {
	dst := make(map[string]dyn.Value)
	return yamlsaver.ConvertToMapValue(task, order, dst)
}
