package libraries

import (
	"path/filepath"

	"github.com/databricks/cli/bundle"
	"github.com/databricks/databricks-sdk-go/service/jobs"
)

func findAllTasks(b *bundle.Bundle) map[string]([]*jobs.Task) {
	r := b.Config.Resources
	result := make(map[string]([]*jobs.Task), 0)
	for k := range b.Config.Resources.Jobs {
		result[k] = make([]*jobs.Task, 0)
		tasks := r.Jobs[k].JobSettings.Tasks
		for i := range tasks {
			task := &tasks[i]
			result[k] = append(result[k], task)
		}
	}

	return result
}

func findAllEnvironments(b *bundle.Bundle) map[string]([]jobs.JobEnvironment) {
	jobEnvs := make(map[string]([]jobs.JobEnvironment), 0)
	for _, job := range b.Config.Resources.Jobs {
		if len(job.Environments) == 0 {
			continue
		}

		jobEnvs[job.Name] = job.Environments
	}

	return jobEnvs
}

func isEnvsWithLocalLibraries(envs []jobs.JobEnvironment) bool {
	for _, e := range envs {
		for _, l := range e.Spec.Dependencies {
			if IsLocalPath(l) {
				return true
			}
		}
	}

	return false
}

func FindAllWheelTasksWithLocalLibraries(b *bundle.Bundle) []*jobs.Task {
	tasks := findAllTasks(b)
	envs := findAllEnvironments(b)

	wheelTasks := make([]*jobs.Task, 0)
	for k, jobTasks := range tasks {
		for _, task := range jobTasks {
			if task.PythonWheelTask == nil {
				continue
			}

			if isTaskWithLocalLibraries(task) {
				wheelTasks = append(wheelTasks, task)
			}

			if envs[k] != nil && isEnvsWithLocalLibraries(envs[k]) {
				wheelTasks = append(wheelTasks, task)
			}
		}
	}

	return wheelTasks
}

func isTaskWithLocalLibraries(task *jobs.Task) bool {
	for _, l := range task.Libraries {
		if IsLocalLibrary(&l) {
			return true
		}
	}

	return false
}

func IsTaskWithWorkspaceLibraries(task *jobs.Task) bool {
	for _, l := range task.Libraries {
		if IsWorkspaceLibrary(&l) {
			return true
		}
	}

	return false
}

func findMatches(path string, b *bundle.Bundle) ([]string, error) {
	fullPath := filepath.Join(b.RootPath, path)
	return filepath.Glob(fullPath)
}

func AbsPathForResource(b *bundle.Bundle, resource string, path string) string {
	p := filepath.Dir(b.Config.GetLocation(resource).File)
	if !filepath.IsAbs(p) {
		p = filepath.Join(b.RootPath, p)
	}

	return filepath.Join(p, path)
}
