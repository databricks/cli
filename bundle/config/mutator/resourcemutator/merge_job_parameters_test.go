package resourcemutator_test

import (
	"context"
	"testing"

	"github.com/databricks/cli/bundle/config/mutator/resourcemutator"

	"github.com/databricks/cli/bundle"
	"github.com/databricks/cli/bundle/config"
	"github.com/databricks/cli/bundle/config/resources"
	"github.com/databricks/databricks-sdk-go/service/jobs"
	"github.com/stretchr/testify/assert"
)

func TestMergeJobParameters(t *testing.T) {
	b := &bundle.Bundle{
		Config: config.Root{
			Resources: config.Resources{
				Jobs: map[string]*resources.Job{
					"foo": {
						JobSettings: jobs.JobSettings{
							Parameters: []jobs.JobParameterDefinition{
								{
									Name:    "foo",
									Default: "v1",
								},
								{
									Name:    "bar",
									Default: "v1",
								},
								{
									Name:    "foo",
									Default: "v2",
								},
							},
						},
					},
				},
			},
		},
	}

	diags := bundle.Apply(context.Background(), b, resourcemutator.MergeJobParameters())
	assert.NoError(t, diags.Error())

	j := b.Config.Resources.Jobs["foo"]

	assert.Len(t, j.Parameters, 2)
	assert.Equal(t, "foo", j.Parameters[0].Name)
	assert.Equal(t, "v2", j.Parameters[0].Default)
	assert.Equal(t, "bar", j.Parameters[1].Name)
	assert.Equal(t, "v1", j.Parameters[1].Default)
}

func TestMergeJobParametersWithNilKey(t *testing.T) {
	b := &bundle.Bundle{
		Config: config.Root{
			Resources: config.Resources{
				Jobs: map[string]*resources.Job{
					"foo": {
						JobSettings: jobs.JobSettings{
							Parameters: []jobs.JobParameterDefinition{
								{
									Default: "v1",
								},
								{
									Default: "v2",
								},
							},
						},
					},
				},
			},
		},
	}

	diags := bundle.Apply(context.Background(), b, resourcemutator.MergeJobParameters())
	assert.NoError(t, diags.Error())
	assert.Len(t, b.Config.Resources.Jobs["foo"].Parameters, 1)
}
