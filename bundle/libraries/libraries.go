package libraries

import (
	"github.com/databricks/cli/bundle"
	"github.com/databricks/databricks-sdk-go/service/jobs"
)

func findAllTasks(b *bundle.Bundle) map[string]([]jobs.Task) {
	r := b.Config.Resources
	result := make(map[string]([]jobs.Task), 0)
	for k := range b.Config.Resources.Jobs {
		result[k] = append(result[k], r.Jobs[k].JobSettings.Tasks...)
	}

	return result
}

func FindAllEnvironments(b *bundle.Bundle) map[string]([]jobs.JobEnvironment) {
	jobEnvs := make(map[string]([]jobs.JobEnvironment), 0)
	for jobKey, job := range b.Config.Resources.Jobs {
		if len(job.Environments) == 0 {
			continue
		}

		jobEnvs[jobKey] = job.Environments
	}

	return jobEnvs
}

func isEnvsWithLocalLibraries(envs []jobs.JobEnvironment) bool {
	for _, e := range envs {
		if e.Spec == nil {
			continue
		}

		for _, l := range e.Spec.Dependencies {
			if IsLibraryLocal(l) {
				return true
			}
		}
	}

	return false
}

func FindTasksWithLocalLibraries(b *bundle.Bundle) []jobs.Task {
	tasks := findAllTasks(b)
	envs := FindAllEnvironments(b)

	var allTasks []jobs.Task
	for k, jobTasks := range tasks {
		for i := range jobTasks {
			task := jobTasks[i]
			if isTaskWithLocalLibraries(task) {
				allTasks = append(allTasks, task)
			}
		}

		if envs[k] != nil && isEnvsWithLocalLibraries(envs[k]) {
			allTasks = append(allTasks, jobTasks...)
		}
	}

	return allTasks
}

func isTaskWithLocalLibraries(task jobs.Task) bool {
	for _, l := range task.Libraries {
		p, err := libraryPath(&l)
		// If there's an error, skip the library because it's not of supported type
		if err != nil {
			continue
		}
		if IsLibraryLocal(p) {
			return true
		}
	}

	return false
}

func IsTaskWithWorkspaceLibraries(task jobs.Task) bool {
	for _, l := range task.Libraries {
		if IsWorkspaceLibrary(&l) {
			return true
		}
	}

	return false
}
