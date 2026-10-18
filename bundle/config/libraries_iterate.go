package config

import (
	"github.com/databricks/cli/libs/structs/structpath"
	"github.com/databricks/databricks-sdk-go/service/compute"
	"github.com/databricks/databricks-sdk-go/service/jobs"
)

// The iterators below visit the places that hold libraries, in the order of the job
// names (and task indices). They visit lists that are set (not nil); the path is built
// for each visited list.

// IterateJobTasks calls fn for the tasks of all jobs, or (nested) for the tasks of
// their for_each_task, with the path of the task, until fn returns false.
func (r *Resources) IterateJobTasks(nested bool, fn func(p *structpath.PathNode, task *jobs.Task) bool) {
	for _, name := range sortedKeys(r.Jobs) {
		job := r.Jobs[name]
		if job == nil {
			continue
		}
		for i := range job.Tasks {
			task := &job.Tasks[i]
			if nested {
				if task.ForEachTask == nil {
					continue
				}
				task = &task.ForEachTask.Task
			}
			p := structpath.NewPath(nil, "resources", "jobs", name, "tasks", i)
			if nested {
				p = structpath.NewPath(p, "for_each_task", "task")
			}
			if !fn(p, task) {
				return
			}
		}
	}
}

// IterateTaskLibraries calls fn for the libraries of the tasks of all jobs, or (nested)
// of the tasks of their for_each_task, until fn returns false.
func (r *Resources) IterateTaskLibraries(nested bool, fn func(p *structpath.PathNode, libs *[]compute.Library) bool) {
	r.IterateJobTasks(nested, func(p *structpath.PathNode, task *jobs.Task) bool {
		if task.Libraries == nil {
			return true
		}
		return fn(structpath.NewPath(p, "libraries"), &task.Libraries)
	})
}

// IterateAiRuntimeCodeSources calls fn for the code_source_path of the AI Runtime tasks
// of all jobs, or (nested) of the tasks of their for_each_task, until fn returns false.
func (r *Resources) IterateAiRuntimeCodeSources(nested bool, fn func(p *structpath.PathNode, s *string) bool) {
	r.IterateJobTasks(nested, func(p *structpath.PathNode, task *jobs.Task) bool {
		if task.AiRuntimeTask == nil || task.AiRuntimeTask.CodeSourcePath == "" {
			return true
		}
		return fn(structpath.NewPath(p, "ai_runtime_task", "code_source_path"), &task.AiRuntimeTask.CodeSourcePath)
	})
}

// IterateJobEnvironmentDependencies calls fn for the dependencies of the environments of
// all jobs, until fn returns false.
func (r *Resources) IterateJobEnvironmentDependencies(fn func(p *structpath.PathNode, deps *[]string) bool) {
	for _, name := range sortedKeys(r.Jobs) {
		job := r.Jobs[name]
		if job == nil {
			continue
		}
		for i := range job.Environments {
			spec := job.Environments[i].Spec
			if spec == nil || spec.Dependencies == nil {
				continue
			}
			if !fn(structpath.NewPath(nil, "resources", "jobs", name, "environments", i, "spec", "dependencies"), &spec.Dependencies) {
				return
			}
		}
	}
}

// IteratePipelineEnvironmentDependencies calls fn for the environment dependencies of
// all pipelines, until fn returns false.
func (r *Resources) IteratePipelineEnvironmentDependencies(fn func(p *structpath.PathNode, deps *[]string) bool) {
	for _, name := range sortedKeys(r.Pipelines) {
		pl := r.Pipelines[name]
		if pl == nil || pl.Environment == nil || pl.Environment.Dependencies == nil {
			continue
		}
		if !fn(structpath.NewPath(nil, "resources", "pipelines", name, "environment", "dependencies"), &pl.Environment.Dependencies) {
			return
		}
	}
}

// IterateClusterLibraries calls fn for the libraries of all clusters, until fn returns
// false.
func (r *Resources) IterateClusterLibraries(fn func(p *structpath.PathNode, libs *[]compute.Library) bool) {
	for _, name := range sortedKeys(r.Clusters) {
		c := r.Clusters[name]
		if c == nil || c.Libraries == nil {
			continue
		}
		if !fn(structpath.NewPath(nil, "resources", "clusters", name, "libraries"), &c.Libraries) {
			return
		}
	}
}
