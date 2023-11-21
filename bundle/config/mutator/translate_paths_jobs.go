package mutator

import (
	"fmt"

	"github.com/databricks/cli/bundle"
	"github.com/databricks/databricks-sdk-go/service/compute"
	"github.com/databricks/databricks-sdk-go/service/jobs"
)

func transformNotebookTask(resource any, dir string) *transformer {
	task, ok := resource.(*jobs.Task)
	if !ok || task.NotebookTask == nil {
		return nil
	}

	return &transformer{
		dir,
		&task.NotebookTask.NotebookPath,
		"tasks.notebook_task.notebook_path",
		translateNotebookPath,
	}
}

func transformSparkTask(resource any, dir string) *transformer {
	task, ok := resource.(*jobs.Task)
	if !ok || task.SparkPythonTask == nil {
		return nil
	}

	return &transformer{
		dir,
		&task.SparkPythonTask.PythonFile,
		"tasks.spark_python_task.python_file",
		translateFilePath,
	}
}

func transformWhlLibrary(resource any, dir string) *transformer {
	library, ok := resource.(*compute.Library)
	if !ok || library.Whl == "" {
		return nil
	}

	return &transformer{
		dir,
		&library.Whl,
		"libraries.whl",
		translateNoOp, // Does not convert to remote path but makes sure that nested paths resolved correctly
	}
}

func transformDbtTask(resource any, dir string) *transformer {
	task, ok := resource.(*jobs.Task)
	if !ok || task.DbtTask == nil {
		return nil
	}

	return &transformer{
		dir,
		&task.DbtTask.ProjectDirectory,
		"tasks.dbt_task.project_directory",
		translateDirectoryPath,
	}
}

func transformSqlFileTask(resource any, dir string) *transformer {
	task, ok := resource.(*jobs.Task)
	if !ok || task.SqlTask == nil || task.SqlTask.File == nil {
		return nil
	}

	return &transformer{
		dir,
		&task.SqlTask.File.Path,
		"tasks.sql_task.file.path",
		translateFilePath,
	}
}

func transformJarLibrary(resource any, dir string) *transformer {
	library, ok := resource.(*compute.Library)
	if !ok || library.Jar == "" {
		return nil
	}

	return &transformer{
		dir,
		&library.Jar,
		"libraries.jar",
		translateNoOp, // Does not convert to remote path but makes sure that nested paths resolved correctly
	}
}

func applyJobTransformers(m *translatePaths, b *bundle.Bundle) error {
	jobTransformers := []transformFunc{
		transformNotebookTask,
		transformSparkTask,
		transformWhlLibrary,
		transformJarLibrary,
		transformDbtTask,
		transformSqlFileTask,
	}

	for key, job := range b.Config.Resources.Jobs {
		dir, err := job.ConfigFileDirectory()
		if err != nil {
			return fmt.Errorf("unable to determine directory for job %s: %w", key, err)
		}

		// Do not translate job task paths if using git source
		if job.GitSource != nil {
			continue
		}

		for i := 0; i < len(job.Tasks); i++ {
			task := &job.Tasks[i]
			err := m.applyTransformers(jobTransformers, b, task, dir)
			if err != nil {
				return err
			}
			for j := 0; j < len(task.Libraries); j++ {
				library := &task.Libraries[j]
				err := m.applyTransformers(jobTransformers, b, library, dir)
				if err != nil {
					return err
				}
			}
		}
	}

	return nil
}
