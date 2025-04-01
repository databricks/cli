package mutator

import (
	"context"
	"testing"

	"github.com/databricks/cli/bundle"
	"github.com/databricks/cli/bundle/config"
	"github.com/databricks/cli/bundle/config/resources"
	"github.com/stretchr/testify/assert"
)

func TestProcessYamlResources(t *testing.T) {
	initializeMutator := recordingMutator{}
	normalizeMutator := recordingMutator{}
	processor := NewResourceProcessor(
		[]bundle.Mutator{&initializeMutator},
		[]bundle.Mutator{&normalizeMutator},
	)
	b := &bundle.Bundle{
		Config: config.Root{
			Resources: config.Resources{
				Jobs: map[string]*resources.Job{
					"job_1": {},
					"job_2": {},
				},
			},
		},
	}

	m := ProcessYamlResources(processor)
	diags := bundle.Apply(context.Background(), b, m)

	assert.NoError(t, diags.Error())
	assert.ElementsMatch(t, initializeMutator.jobNames, []string{"job_1", "job_2"})
	assert.ElementsMatch(t, normalizeMutator.jobNames, []string{"job_1", "job_2"})
}
