package resourcemutator

import (
	"testing"

	"github.com/databricks/cli/bundle"
	"github.com/databricks/cli/bundle/config"
	"github.com/databricks/cli/bundle/config/resources"
	"github.com/stretchr/testify/assert"
)

func TestGetAllResources(t *testing.T) {
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

	set, err := getAllResources(b)

	assert.NoError(t, err)
	assert.ElementsMatch(
		t,
		set.ToArray(),
		[]ResourceKey{
			{Type: "jobs", Name: "job_1"},
			{Type: "jobs", Name: "job_2"},
		},
	)
}
