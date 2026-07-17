package validate

import (
	"context"
	"fmt"

	"github.com/databricks/cli/bundle"
	"github.com/databricks/cli/bundle/config/engine"
	"github.com/databricks/cli/libs/diag"
	"github.com/databricks/cli/libs/dyn"
	"github.com/databricks/databricks-sdk-go/service/jobs"
)

// EmptyTaskParameters warns when a task parameter is an empty string and the
// bundle deploys with the Terraform engine. The Terraform provider reads an
// empty list element back as nil and job creation panics with
// "parameters[<nil>] is not a string". The Jobs API accepts empty string
// parameters and the direct engine deploys them, so this is a warning and it
// is only emitted for the Terraform engine.
//
// Empty parameters usually come from a variable that is intentionally blank
// in some targets, e.g. a per-developer prefix passed as its own list element.
func EmptyTaskParameters() bundle.ReadOnlyMutator {
	return &emptyTaskParameters{}
}

type emptyTaskParameters struct{ bundle.RO }

func (v *emptyTaskParameters) Name() string {
	return "validate:empty_task_parameters"
}

// Apply walks every job task (including tasks nested in a for_each_task) and
// collects a warning for each empty string element in a parameters list. It
// returns nothing when the effective engine is direct, because the direct
// engine deploys empty parameters correctly; the panic is specific to the
// Terraform provider. It must run after variable references in resources are
// resolved, so that a parameter written as ${var.x} is seen as its final
// string value.
func (v *emptyTaskParameters) Apply(ctx context.Context, b *bundle.Bundle) diag.Diagnostics {
	// Resolve effective engine: config takes precedence over env var.
	effectiveEngine := b.Config.Bundle.Engine
	if effectiveEngine == engine.EngineNotSet {
		if envEngine, err := engine.FromEnv(ctx); err == nil {
			effectiveEngine = envEngine
		}
	}
	if effectiveEngine.IsDirect() {
		return nil
	}

	var diags diag.Diagnostics
	for key, job := range b.Config.Resources.Jobs {
		for i, task := range job.Tasks {
			diags = diags.Extend(checkEmptyTaskParameters(b, &task, fmt.Sprintf("resources.jobs.%s.tasks[%d]", key, i)))
			if task.ForEachTask != nil {
				diags = diags.Extend(checkEmptyTaskParameters(b, &task.ForEachTask.Task, fmt.Sprintf("resources.jobs.%s.tasks[%d].for_each_task.task", key, i)))
			}
		}
	}
	return diags
}

// checkEmptyTaskParameters returns one warning per empty string element in the
// given task's parameters list, covering the four task types whose parameters
// are a plain list of strings (spark_python_task, python_wheel_task,
// spark_jar_task, spark_submit_task). basePath is the config path of the task,
// e.g. "resources.jobs.my_job.tasks[0]"; it anchors each warning to the exact
// element (".<task_type>.parameters[<i>]") so the rendered diagnostic points
// at the offending line in the user's YAML.
func checkEmptyTaskParameters(b *bundle.Bundle, task *jobs.Task, basePath string) diag.Diagnostics {
	var diags diag.Diagnostics

	check := func(params []string, taskType string) {
		for i, p := range params {
			if p != "" {
				continue
			}
			path := fmt.Sprintf("%s.%s.parameters[%d]", basePath, taskType, i)
			diags = append(diags, diag.Diagnostic{
				Severity: diag.Warning,
				Summary:  fmt.Sprintf("empty parameter in %s fails deploys that use the Terraform engine", taskType),
				Detail: "The Terraform provider reads an empty string element in a parameters list back as nil, " +
					"and creating the job panics with \"parameters[<nil>] is not a string\". " +
					"Remove the empty element or make its value non-empty. " +
					"The direct engine (bundle.engine: direct) deploys empty parameters correctly.",
				Locations: b.Config.GetLocations(path),
				Paths:     []dyn.Path{dyn.MustPathFromString(path)},
			})
		}
	}

	if task.SparkPythonTask != nil {
		check(task.SparkPythonTask.Parameters, "spark_python_task")
	}
	if task.PythonWheelTask != nil {
		check(task.PythonWheelTask.Parameters, "python_wheel_task")
	}
	if task.SparkJarTask != nil {
		check(task.SparkJarTask.Parameters, "spark_jar_task")
	}
	if task.SparkSubmitTask != nil {
		check(task.SparkSubmitTask.Parameters, "spark_submit_task")
	}

	return diags
}
