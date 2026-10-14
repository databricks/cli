package generate

import (
	"github.com/databricks/cli/libs/diag"
	"github.com/databricks/cli/libs/dyn"
	"github.com/databricks/cli/libs/dyn/yamlsaver"
	"github.com/databricks/cli/libs/sdktypes/jobs/v2"
	"github.com/databricks/cli/libs/utils"
)

var (
	jobOrder  = yamlsaver.NewOrder([]string{"name", "job_clusters", "compute", "tasks"})
	taskOrder = yamlsaver.NewOrder([]string{"task_key", "depends_on", "existing_cluster_id", "new_cluster", "job_cluster_key"})
)

func ConvertJobToValue(job *jobs.GetJobResponse) (dyn.Value, error) {
	value := make(map[string]dyn.Value)
	if job.Settings.Tasks != nil {
		var tasks []dyn.Value
		for _, task := range job.Settings.Tasks {
			v, err := convertTaskToValue(task, taskOrder)
			if err != nil {
				return dyn.InvalidValue, err
			}
			tasks = append(tasks, v)
		}
		// We're using location lines to define the order of keys in exported YAML.
		value["tasks"] = dyn.NewValue(tasks, []diag.Location{{Line: jobOrder.Get("tasks")}})
	}

	// We're processing job.Settings.Parameters separately to retain empty default values.
	if len(job.Settings.Parameters) > 0 {
		var params []dyn.Value
		for _, parameter := range job.Settings.Parameters {
			p := map[string]dyn.Value{
				"name":    dyn.NewValue(utils.Deref(parameter.Name), []diag.Location{{Line: 0}}), // We use Line: 0 to ensure that the name goes first.
				"default": dyn.NewValue(utils.Deref(parameter.Default), []diag.Location{{Line: 1}}),
			}
			params = append(params, dyn.V(p))
		}

		value["parameters"] = dyn.NewValue(params, []diag.Location{{Line: jobOrder.Get("parameters")}})
	}

	return yamlsaver.ConvertToMapValue(job.Settings, jobOrder, []string{"format", "new_cluster", "existing_cluster_id"}, value)
}

func convertTaskToValue(task jobs.TaskSettings, order *yamlsaver.Order) (dyn.Value, error) {
	dst := make(map[string]dyn.Value)
	return yamlsaver.ConvertToMapValue(task, order, []string{"format"}, dst)
}
