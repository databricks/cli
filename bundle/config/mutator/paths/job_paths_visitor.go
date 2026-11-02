package paths

import (
	"github.com/databricks/cli/bundle/config"
	"github.com/databricks/cli/libs/structs/structpath"
	"github.com/databricks/databricks-sdk-go/service/jobs"
)

// taskRule visits one kind of path value of a job task.
type taskRule struct {
	mode  TranslateMode
	visit func(task *jobs.Task, c *taskCursor) error
}

// taskCursor is the task being visited. It is reused for all tasks so visiting does
// not allocate; the path is only built for values that are visited.
type taskCursor struct {
	name   string
	index  int
	nested bool
	rule   *taskRule
	fn     VisitFunc
}

// yield visits the value s, the field key of owner, at the parts below the task.
func (c *taskCursor) yield(owner any, key string, s *string, parts ...any) error {
	// Most candidates (e.g. the whl, jar and requirements of every library) are empty.
	if !present(owner, key, s) {
		return nil
	}
	p := structpath.NewPath(resourcePath("jobs", c.name), "tasks", c.index)
	if c.nested {
		p = structpath.NewPath(p, "for_each_task", "task")
	}
	return visit(c.fn, visitArgs{
		owner: owner,
		field: key,
		value: s,
		mode:  c.rule.mode,
		path:  structpath.NewPath(p, parts...),
	})
}

var jobTaskRules = []taskRule{
	{TranslateModeNotebook, func(t *jobs.Task, c *taskCursor) error {
		if t.NotebookTask == nil {
			return nil
		}
		return c.yield(t.NotebookTask, "notebook_path", &t.NotebookTask.NotebookPath, "notebook_task", "notebook_path")
	}},
	{TranslateModeFile, func(t *jobs.Task, c *taskCursor) error {
		if t.SparkPythonTask == nil {
			return nil
		}
		return c.yield(t.SparkPythonTask, "python_file", &t.SparkPythonTask.PythonFile, "spark_python_task", "python_file")
	}},
	{TranslateModeDirectory, func(t *jobs.Task, c *taskCursor) error {
		if t.DbtTask == nil {
			return nil
		}
		return c.yield(t.DbtTask, "project_directory", &t.DbtTask.ProjectDirectory, "dbt_task", "project_directory")
	}},
	{TranslateModeFile, func(t *jobs.Task, c *taskCursor) error {
		if t.SqlTask == nil || t.SqlTask.File == nil {
			return nil
		}
		return c.yield(t.SqlTask.File, "path", &t.SqlTask.File.Path, "sql_task", "file", "path")
	}},
	{TranslateModeFile, func(t *jobs.Task, c *taskCursor) error {
		if t.AlertTask == nil {
			return nil
		}
		return c.yield(t.AlertTask, "workspace_path", &t.AlertTask.WorkspacePath, "alert_task", "workspace_path")
	}},
	{TranslateModeFile, func(t *jobs.Task, c *taskCursor) error {
		for i := range t.Libraries {
			if err := c.yield(&t.Libraries[i], "requirements", &t.Libraries[i].Requirements, "libraries", i, "requirements"); err != nil {
				return err
			}
		}
		return nil
	}},
	// The AI Runtime task runs this bash script on each node; the backend reads it as
	// a workspace file, so translate the local path to its remote (or
	// immutable-snapshot) location like any other file.
	{TranslateModeFile, func(t *jobs.Task, c *taskCursor) error {
		if t.AiRuntimeTask == nil {
			return nil
		}
		for i := range t.AiRuntimeTask.Deployments {
			if err := c.yield(&t.AiRuntimeTask.Deployments[i], "command_path", &t.AiRuntimeTask.Deployments[i].CommandPath, "ai_runtime_task", "deployments", i, "command_path"); err != nil {
				return err
			}
		}
		return nil
	}},
}

// visitTasks applies the rules to the tasks of all jobs, then to the tasks nested in
// their for_each_task, one rule at a time.
func visitTasks(root *config.Root, rules []taskRule, fn VisitFunc) error {
	if err := visitTasksAt(root, false, rules, fn); err != nil {
		return err
	}
	return visitTasksAt(root, true, rules, fn)
}

// visitTasksAt applies the rules, one at a time, to the tasks of all jobs, or (nested)
// to the tasks in their for_each_task.
func visitTasksAt(root *config.Root, nested bool, rules []taskRule, fn VisitFunc) error {
	c := &taskCursor{nested: nested, fn: fn}
	names := sortedKeys(root.Resources.Jobs)
	for r := range rules {
		c.rule = &rules[r]
		for _, name := range names {
			job := root.Resources.Jobs[name]
			if job == nil {
				continue
			}
			c.name = name
			for i := range job.Tasks {
				task := &job.Tasks[i]
				if nested {
					if task.ForEachTask == nil {
						continue
					}
					task = &task.ForEachTask.Task
				}
				c.index = i
				if err := c.rule.visit(task, c); err != nil {
					return err
				}
			}
		}
	}
	return nil
}

// VisitJobPaths visits all paths in job resources and applies a function to each path.
func VisitJobPaths(root *config.Root, fn VisitFunc) error {
	if err := visitTasks(root, jobTaskRules, fn); err != nil {
		return err
	}
	for _, name := range sortedKeys(root.Resources.Jobs) {
		job := root.Resources.Jobs[name]
		if job == nil {
			continue
		}
		for i, ev := range job.EnvironmentVariables {
			if ev.Spec == nil {
				continue
			}
			for j := range ev.Spec.Files {
				if err := visit(fn, visitArgs{
					value: &ev.Spec.Files[j],
					mode:  TranslateModeFile,
					path:  structpath.NewPath(resourcePath("jobs", name), "environment_variables", i, "spec", "files", j),
				}); err != nil {
					return err
				}
			}
		}
	}
	for _, name := range sortedKeys(root.Resources.Jobs) {
		job := root.Resources.Jobs[name]
		if job == nil {
			continue
		}
		for i, env := range job.Environments {
			if env.Spec == nil {
				continue
			}
			if err := visit(fn, visitArgs{
				owner: env.Spec,
				field: "project_environment",
				value: &env.Spec.ProjectEnvironment,
				mode:  TranslateModeFile,
				path:  structpath.NewPath(resourcePath("jobs", name), "environments", i, "spec", "project_environment"),
			}); err != nil {
				return err
			}
		}
	}
	return nil
}
