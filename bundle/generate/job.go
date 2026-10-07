package generate

import (
	"github.com/databricks/cli/libs/structs/structyaml"
	"github.com/databricks/databricks-sdk-go/service/jobs"
)

var (
	jobOrder  = []string{"name", "job_clusters", "compute", "tasks", "parameters"}
	taskOrder = []string{"task_key", "depends_on", "existing_cluster_id", "new_cluster", "job_cluster_key"}
)

func ConvertJobToValue(job *jobs.Job) (structyaml.Map, error) {
	// Tasks and parameters are processed separately.
	value, err := structyaml.Struct(job.Settings, "format", "new_cluster", "existing_cluster_id", "tasks", "parameters")
	if err != nil {
		return nil, err
	}

	if job.Settings.Tasks != nil {
		var tasks []any
		for _, task := range job.Settings.Tasks {
			v, err := convertTaskToValue(task)
			if err != nil {
				return nil, err
			}
			tasks = append(tasks, v)
		}
		value.Add("tasks", tasks)
	}

	// We're processing job.Settings.Parameters separately to retain empty default values.
	if len(job.Settings.Parameters) > 0 {
		var params []any
		for _, parameter := range job.Settings.Parameters {
			params = append(params, structyaml.M("name", parameter.Name, "default", parameter.Default))
		}

		value.Add("parameters", params)
	}

	return value.Order(jobOrder...), nil
}

func convertTaskToValue(task jobs.Task) (structyaml.Map, error) {
	value, err := structyaml.Struct(task, "format")
	if err != nil {
		return nil, err
	}
	return value.Order(taskOrder...), nil
}
