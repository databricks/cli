package validate

import (
	"fmt"
	"testing"

	"github.com/databricks/cli/bundle"
	"github.com/databricks/cli/bundle/config"
	"github.com/databricks/cli/libs/diag"
	"github.com/databricks/cli/libs/dyn"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func loadBundle(t *testing.T, yaml string) *bundle.Bundle {
	root, diags := config.LoadFromBytes("databricks.yml", []byte(yaml))
	require.Empty(t, diags)
	return &bundle.Bundle{Config: *root}
}

// summarize renders diagnostics as "summary at path (line)".
func summarize(diags diag.Diagnostics) []string {
	var out []string
	for _, d := range diags {
		line := 0
		if len(d.Locations) > 0 {
			line = d.Locations[0].Line
		}
		out = append(out, fmt.Sprintf("%s at %s (%d)", d.Summary, d.Paths[0], line))
	}
	return out
}

func TestRequiredWalkNestedTasks(t *testing.T) {
	b := loadBundle(t, `
bundle:
  name: test-bundle
resources:
  jobs:
    job:
      tasks:
        - task_key: outer
          for_each_task:
            inputs: "[1, 2]"
            task:
              notebook_task: {}
        - depends_on:
            - outcome: "true"
          task_key: ""
`)
	diags := bundle.Apply(t.Context(), b, Required())
	assert.Equal(t, []string{
		`required field "notebook_path" is not set at resources.jobs.job.tasks[0].for_each_task.task.notebook_task (12)`,
		`required field "task_key" is not set at resources.jobs.job.tasks[0].for_each_task.task (12)`,
		`required field "task_key" is not set at resources.jobs.job.tasks[1].depends_on[0] (14)`,
	}, summarize(diags))
}

func TestRequiredWalkSetValues(t *testing.T) {
	// A pure reference and an explicit zero value set a field; null does not.
	b := loadBundle(t, `
bundle:
  name: test-bundle
variables:
  threshold:
    default: 10
resources:
  jobs:
    job:
      health:
        rules:
          - metric: RUN_DURATION_SECONDS
            op: GREATER_THAN
            value: ${var.threshold}
          - metric: RUN_DURATION_SECONDS
            op: GREATER_THAN
            value: 0
      tasks:
        - task_key: run
          run_job_task:
        - task_key: dbt
          dbt_task:
            commands: null
`)
	diags := bundle.Apply(t.Context(), b, Required())
	assert.Equal(t, []string{
		`required field "commands" is not set at resources.jobs.job.tasks[1].dbt_task (23)`,
		`required field "job_id" is not set at resources.jobs.job.tasks[0].run_job_task (20)`,
	}, summarize(diags))
}

func TestEnumWalk(t *testing.T) {
	b := loadBundle(t, `
bundle:
  name: test-bundle
resources:
  jobs:
    job:
      tasks:
        - task_key: a
          run_if: SOMETIMES
          notebook_task:
            notebook_path: ./nb.py
            source: ${var.source}
        - task_key: b
          run_if: ALL_DONE
`)
	diags := bundle.Apply(t.Context(), b, Enum())
	assert.Equal(t, []string{
		`invalid value "SOMETIMES" for enum field. Valid values are [ALL_DONE ALL_FAILED ALL_SUCCESS AT_LEAST_ONE_FAILED AT_LEAST_ONE_SUCCESS NONE_FAILED] at resources.jobs.job.tasks[0].run_if (9)`,
	}, summarize(diags))
}

func TestEnumWalkKeepsWalkOrderForTies(t *testing.T) {
	// Values without a location tie in the sort; they keep the order of the
	// configuration, not that of the walk over the typed maps.
	b := loadBundle(t, `
bundle:
  name: test-bundle
resources:
  jobs:
    zeta:
      tasks:
        - task_key: a
          run_if: SOMETIMES
    alpha:
      tasks:
        - task_key: a
          run_if: SOMETIMES
    mid:
      tasks:
        - task_key: a
          run_if: SOMETIMES
`)
	err := b.Config.Mutate(func(v dyn.Value) (dyn.Value, error) {
		return dyn.MapByPattern(v, dyn.NewPattern(dyn.Key("resources"), dyn.Key("jobs"), dyn.AnyKey(), dyn.Key("tasks"), dyn.AnyIndex(), dyn.Key("run_if")), func(_ dyn.Path, v dyn.Value) (dyn.Value, error) {
			return dyn.V(v.MustString()), nil
		})
	})
	require.NoError(t, err)
	diags := bundle.Apply(t.Context(), b, Enum())
	var paths []string
	for _, d := range diags {
		paths = append(paths, d.Paths[0].String())
	}
	assert.Equal(t, []string{
		"resources.jobs.zeta.tasks[0].run_if",
		"resources.jobs.alpha.tasks[0].run_if",
		"resources.jobs.mid.tasks[0].run_if",
	}, paths)
}
