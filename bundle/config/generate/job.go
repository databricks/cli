package generate

import (
	"github.com/databricks/cli/libs/config"
	"github.com/databricks/cli/libs/config/convert"
	"github.com/databricks/databricks-sdk-go/service/jobs"
)

var taskFieldOrder = []string{"TaskKey", "DependsOn", "ExistingClusterId", "NewCluster", "JobClusterKey"}
var jobFieldOrder = []string{"Name", "Format", "NewCluster", "JobClusters", "ExistingClusterId", "Compute", "Tasks"}

func ConvertJobToValue(job *jobs.Job) (config.Value, error) {
	jobOrder := config.NewOrder(jobFieldOrder)
	taskOrder := config.NewOrder(taskFieldOrder)
	value := make(map[string]config.Value)

	if job.Settings.Tasks != nil {
		k, _ := config.Key(job.Settings, "Tasks")
		tasks := make([]config.Value, 0)
		for _, task := range job.Settings.Tasks {
			v, err := convertTaskToValue(task, taskOrder)
			if err != nil {
				return config.NilValue, err
			}
			tasks = append(tasks, v)
		}
		value[k] = config.NewValue(tasks, config.Location{Line: jobOrder.Get("Tasks")})
	}

	return convert.ConvertToMapValue(job.Settings, jobOrder, value)
}

func convertTaskToValue(task jobs.Task, order *config.Order) (config.Value, error) {
	dst := make(map[string]config.Value)
	return convert.ConvertToMapValue(task, order, dst)
}
