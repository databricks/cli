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

func TestContinuousTaskRetryModeWarnsWhenUnset(t *testing.T) {
	ctx := t.Context()

	b := &bundle.Bundle{
		Config: config.Root{
			Resources: config.Resources{
				Jobs: map[string]*resources.Job{
					"foo": {
						JobSettings: jobs.JobSettings{
							Continuous: &jobs.Continuous{},
						},
					},
				},
			},
		},
	}

	bundletest.SetLocation(b, "resources.jobs.foo.continuous", []dyn.Location{{File: "a.yml", Line: 1, Column: 1}})

	diags := ContinuousTaskRetryMode().Apply(ctx, b)
	assert.Equal(t, diag.Diagnostics{
		{
			Severity:  diag.Warning,
			Summary:   continuousTaskRetryWarningSummary,
			Detail:    continuousTaskRetryWarningDetail,
			Locations: []dyn.Location{{File: "a.yml", Line: 1, Column: 1}},
			Paths:     []dyn.Path{dyn.MustPathFromString("resources.jobs.foo.continuous")},
		},
	}, diags)
}

func TestContinuousTaskRetryModeNoWarningWhenSet(t *testing.T) {
	ctx := t.Context()

	b := &bundle.Bundle{
		Config: config.Root{
			Resources: config.Resources{
				Jobs: map[string]*resources.Job{
					"foo": {
						JobSettings: jobs.JobSettings{
							Continuous: &jobs.Continuous{
								TaskRetryMode: jobs.TaskRetryModeOnFailure,
							},
						},
					},
				},
			},
		},
	}

	bundletest.SetLocation(b, "resources.jobs.foo.continuous", []dyn.Location{{File: "a.yml", Line: 1, Column: 1}})

	diags := ContinuousTaskRetryMode().Apply(ctx, b)
	assert.Empty(t, diags)
}

func TestContinuousTaskRetryModeNoWarningWithoutContinuous(t *testing.T) {
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

	bundletest.SetLocation(b, "resources.jobs.foo", []dyn.Location{{File: "a.yml", Line: 1, Column: 1}})

	diags := ContinuousTaskRetryMode().Apply(ctx, b)
	assert.Empty(t, diags)
}
