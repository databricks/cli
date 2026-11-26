package generate

import (
	"context"
	"strings"

	"github.com/databricks/cli/libs/diag"
	"github.com/databricks/cli/libs/dyn"
	"github.com/databricks/cli/libs/dyn/yamlsaver"
	"github.com/databricks/databricks-sdk-go/service/jobs"
)

var (
	jobOrder  = yamlsaver.NewOrder([]string{"name", "job_clusters", "compute", "tasks"})
	taskOrder = yamlsaver.NewOrder([]string{"task_key", "depends_on", "existing_cluster_id", "new_cluster", "job_cluster_key"})
)

func ConvertJobToValue(job *jobs.Job) (dyn.Value, error) {
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
				"name":    dyn.NewValue(parameter.Name, []diag.Location{{Line: 0}}), // We use Line: 0 to ensure that the name goes first.
				"default": dyn.NewValue(parameter.Default, []diag.Location{{Line: 1}}),
			}
			params = append(params, dyn.V(p))
		}

		value["parameters"] = dyn.NewValue(params, []diag.Location{{Line: jobOrder.Get("parameters")}})
	}

	return yamlsaver.ConvertToMapValue(job.Settings, jobOrder, []string{"format", "new_cluster", "existing_cluster_id", "deployment"}, value)
}

func convertTaskToValue(task jobs.Task, order *yamlsaver.Order) (dyn.Value, error) {
	dst := make(map[string]dyn.Value)
	return yamlsaver.ConvertToMapValue(task, order, []string{"format"}, dst)
}

// WithSparkPythonFiles enables downloading workspace files referenced by a
// spark_python_task.
func WithSparkPythonFiles() DownloaderOption {
	return func(n *Downloader) {
		n.downloadSparkPythonFiles = true
	}
}

func (n *Downloader) MarkTaskForDownload(ctx context.Context, task *jobs.Task) error {
	if task.NotebookTask != nil {
		return n.markNotebookForDownload(ctx, &task.NotebookTask.NotebookPath)
	}

	if n.downloadSparkPythonFiles && task.SparkPythonTask != nil && isWorkspaceFileTask(task.SparkPythonTask.Source, task.SparkPythonTask.PythonFile) {
		return n.markFileForDownload(ctx, &task.SparkPythonTask.PythonFile)
	}

	return nil
}

// isWorkspaceFileTask reports whether a task file lives in the Databricks
// workspace and should be downloaded. Files sourced from Git (source: GIT) are
// left to the Git repository, and python_file also accepts cloud URIs
// (dbfs:/, s3:/, adls:/, gcs:/) which are not workspace paths; per the Jobs API,
// workspace files are absolute and begin with "/".
func isWorkspaceFileTask(source jobs.Source, filePath string) bool {
	return source != jobs.SourceGit && strings.HasPrefix(filePath, "/")
}

func (n *Downloader) MarkTasksForDownload(ctx context.Context, tasks []jobs.Task) error {
	var paths []string
	for _, task := range tasks {
		if task.NotebookTask != nil {
			paths = append(paths, task.NotebookTask.NotebookPath)
		}
		if n.downloadSparkPythonFiles && task.SparkPythonTask != nil && isWorkspaceFileTask(task.SparkPythonTask.Source, task.SparkPythonTask.PythonFile) {
			paths = append(paths, task.SparkPythonTask.PythonFile)
		}
	}
	if len(paths) > 0 {
		n.basePath = commonDirPrefix(paths)
	}
	for i := range tasks {
		if err := n.MarkTaskForDownload(ctx, &tasks[i]); err != nil {
			return err
		}
	}
	return nil
}
