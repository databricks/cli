package mutator

import (
	"fmt"

	"github.com/databricks/cli/bundle"
	"github.com/databricks/databricks-sdk-go/service/compute"
	"github.com/databricks/databricks-sdk-go/service/jobs"
)

func selectNotebookTask(resource interface{}, m *translatePaths) *selector {
	task, ok := resource.(*jobs.Task)
	if !ok || task.NotebookTask == nil {
		return nil
	}

	return &selector{
		&task.NotebookTask.NotebookPath,
		"tasks.notebook_task.notebook_path",
		m.translateNotebookPath,
	}
}

func selectSparkTask(resource interface{}, m *translatePaths) *selector {
	task, ok := resource.(*jobs.Task)
	if !ok || task.SparkPythonTask == nil {
		return nil
	}

	return &selector{
		&task.SparkPythonTask.PythonFile,
		"tasks.spark_python_task.python_file",
		m.translateFilePath,
	}
}

func selectWhlLibrary(resource interface{}, m *translatePaths) *selector {
	library, ok := resource.(*compute.Library)
	if !ok || library.Whl == "" {
		return nil
	}

	return &selector{
		&library.Whl,
		"libraries.whl",
		m.translateToBundleRootRelativePath,
	}
}

func selectJarLibrary(resource interface{}, m *translatePaths) *selector {
	library, ok := resource.(*compute.Library)
	if !ok || library.Jar == "" {
		return nil
	}

	return &selector{
		&library.Jar,
		"libraries.jar",
		m.translateFilePath,
	}
}

func getJobsTransformers(m *translatePaths, b *bundle.Bundle) ([]*transformer, error) {
	var transformers []*transformer = make([]*transformer, 0)

	for key, job := range b.Config.Resources.Jobs {
		dir, err := job.ConfigFileDirectory()
		if err != nil {
			return nil, fmt.Errorf("unable to determine directory for job %s: %w", key, err)
		}

		// Do not translate job task paths if using git source
		if job.GitSource != nil {
			continue
		}

		for i := 0; i < len(job.Tasks); i++ {
			task := &job.Tasks[i]
			transformers = addTransformerForResource(transformers, m, task, dir)
			for j := 0; j < len(task.Libraries); j++ {
				library := &task.Libraries[j]
				transformers = addTransformerForResource(transformers, m, library, dir)
			}
		}
	}

	return transformers, nil
}
