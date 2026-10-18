package resourcemutator

import (
	"testing"

	"github.com/databricks/cli/bundle/config"
	"github.com/databricks/cli/bundle/config/resources"
	"github.com/databricks/databricks-sdk-go/service/jobs"
	"github.com/stretchr/testify/assert"
)

func TestResourceKeySet_AddResources(t *testing.T) {
	var r config.Resources
	r.Jobs = map[string]*resources.Job{
		"job_1": {JobSettings: jobs.JobSettings{Name: "job_1"}},
		"job_2": nil,
	}
	r.Pipelines = map[string]*resources.Pipeline{
		"pipeline_1": {},
	}

	all := NewResourceKeySet()
	all.AddResources(&r)
	assert.ElementsMatch(t, []ResourceKey{
		{Type: "jobs", Name: "job_1"},
		{Type: "jobs", Name: "job_2"},
		{Type: "pipelines", Name: "pipeline_1"},
	}, all.ToArray())

	jobKeys := NewResourceKeySet()
	jobKeys.AddResourcesOfType(&r, "jobs")
	assert.ElementsMatch(t, []ResourceKey{
		{Type: "jobs", Name: "job_1"},
		{Type: "jobs", Name: "job_2"},
	}, jobKeys.ToArray())
}

func TestResourceKeySet_AddResourceKey(t *testing.T) {
	set := NewResourceKeySet()

	set.AddResourceKey(ResourceKey{Type: "jobs", Name: "job_1"})
	set.AddResourceKey(ResourceKey{Type: "pipelines", Name: "pipeline_1"})

	assert.ElementsMatch(t, []string{"jobs", "pipelines"}, set.Types())
	assert.ElementsMatch(t, []string{"job_1"}, set.Names("jobs"))
	assert.ElementsMatch(t, []string{"pipeline_1"}, set.Names("pipelines"))
	assert.ElementsMatch(t,
		[]ResourceKey{
			{Type: "jobs", Name: "job_1"},
			{Type: "pipelines", Name: "pipeline_1"},
		},
		set.ToArray(),
	)
	assert.Equal(t, 2, set.Size())
}
