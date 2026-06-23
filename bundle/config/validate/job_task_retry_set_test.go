package validate

import (
	"testing"

	"github.com/databricks/cli/bundle"
	"github.com/databricks/cli/bundle/config"
	"github.com/databricks/cli/bundle/config/resources"
	"github.com/databricks/cli/bundle/internal/bundletest"
	"github.com/databricks/cli/libs/diag"
	"github.com/databricks/cli/libs/dyn"
	"github.com/databricks/databricks-sdk-go/service/jobs"
	"github.com/stretchr/testify/assert"
)

func TestJobTaskRetrySetWarnsWhenUnset(t *testing.T) {
	ctx := t.Context()

	b := &bundle.Bundle{
		Config: config.Root{
			Resources: config.Resources{
				Jobs: map[string]*resources.Job{
					"foo": {
						JobSettings: jobs.JobSettings{
							Tasks: []jobs.Task{
								{
									TaskKey:      "my_task",
									NotebookTask: &jobs.NotebookTask{},
								},
							},
						},
					},
				},
			},
		},
	}

	bundletest.SetLocation(b, "resources.jobs.foo.tasks[0]", []dyn.Location{{File: "a.yml", Line: 1, Column: 1}})

	diags := JobTaskRetrySet().Apply(ctx, b)
	assert.Equal(t, diag.Diagnostics{
		{
			Severity:  diag.Warning,
			Summary:   jobTaskRetryWarningSummary,
			Detail:    jobTaskRetryWarningDetail,
			Locations: []dyn.Location{{File: "a.yml", Line: 1, Column: 1}},
			Paths:     []dyn.Path{dyn.MustPathFromString("resources.jobs.foo.tasks[0]")},
		},
	}, diags)
}

func TestJobTaskRetrySetNoWarningWhenSet(t *testing.T) {
	ctx := t.Context()

	b := &bundle.Bundle{
		Config: config.Root{
			Resources: config.Resources{
				Jobs: map[string]*resources.Job{
					"foo": {
						JobSettings: jobs.JobSettings{
							Tasks: []jobs.Task{
								{
									TaskKey:      "my_task",
									NotebookTask: &jobs.NotebookTask{},
									MaxRetries:   3,
								},
							},
						},
					},
				},
			},
		},
	}

	bundletest.SetLocation(b, "resources.jobs.foo.tasks[0]", []dyn.Location{{File: "a.yml", Line: 1, Column: 1}})

	diags := JobTaskRetrySet().Apply(ctx, b)
	assert.Empty(t, diags)
}

func TestJobTaskRetrySetNoWarningWhenExplicitZero(t *testing.T) {
	ctx := t.Context()

	b := &bundle.Bundle{
		Config: config.Root{
			Resources: config.Resources{
				Jobs: map[string]*resources.Job{
					"foo": {
						JobSettings: jobs.JobSettings{
							Tasks: []jobs.Task{
								{
									TaskKey:      "my_task",
									NotebookTask: &jobs.NotebookTask{},
								},
							},
						},
					},
				},
			},
		},
	}

	bundletest.SetLocation(b, "resources.jobs.foo.tasks[0]", []dyn.Location{{File: "a.yml", Line: 1, Column: 1}})

	// max_retries: 0 is omitted by typed conversion, so set it explicitly on the
	// dyn value to exercise the deliberate "never retry" case.
	bundletest.Mutate(t, b, func(v dyn.Value) (dyn.Value, error) {
		return dyn.Set(v, "resources.jobs.foo.tasks[0].max_retries", dyn.V(0))
	})

	diags := JobTaskRetrySet().Apply(ctx, b)
	assert.Empty(t, diags)
}

func TestJobTaskRetrySetWarnsForEachTask(t *testing.T) {
	ctx := t.Context()

	b := &bundle.Bundle{
		Config: config.Root{
			Resources: config.Resources{
				Jobs: map[string]*resources.Job{
					"foo": {
						JobSettings: jobs.JobSettings{
							Tasks: []jobs.Task{
								{
									TaskKey:    "my_task",
									MaxRetries: 3,
									ForEachTask: &jobs.ForEachTask{
										Task: jobs.Task{
											TaskKey:      "inner",
											NotebookTask: &jobs.NotebookTask{},
										},
									},
								},
							},
						},
					},
				},
			},
		},
	}

	bundletest.SetLocation(b, "resources.jobs.foo.tasks[0].for_each_task.task", []dyn.Location{{File: "a.yml", Line: 1, Column: 1}})

	diags := JobTaskRetrySet().Apply(ctx, b)
	assert.Equal(t, diag.Diagnostics{
		{
			Severity:  diag.Warning,
			Summary:   jobTaskRetryWarningSummary,
			Detail:    jobTaskRetryWarningDetail,
			Locations: []dyn.Location{{File: "a.yml", Line: 1, Column: 1}},
			Paths:     []dyn.Path{dyn.MustPathFromString("resources.jobs.foo.tasks[0].for_each_task.task")},
		},
	}, diags)
}
