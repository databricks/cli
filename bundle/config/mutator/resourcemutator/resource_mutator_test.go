package resourcemutator

import (
	"testing"

	"github.com/databricks/cli/bundle"
	"github.com/databricks/cli/bundle/config"
	"github.com/databricks/cli/bundle/config/resources"
	"github.com/databricks/cli/libs/structs/structpath"
	"github.com/databricks/databricks-sdk-go/service/jobs"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func newJob(name string) *resources.Job {
	return &resources.Job{JobSettings: jobs.JobSettings{Name: name}}
}

func TestSelectResources(t *testing.T) {
	job1 := newJob("job_1")
	job2 := newJob("job_2")

	b := &bundle.Bundle{Config: config.Root{
		Resources: config.Resources{
			Jobs:      map[string]*resources.Job{"job_1": job1, "job_2": job2},
			Pipelines: map[string]*resources.Pipeline{"pipeline_1": {}},
		},
	}}

	keys := NewResourceKeySet()
	keys.AddResourceKey(ResourceKey{Type: "jobs", Name: "job_1"})

	restore := selectResources(b, keys)
	assert.Equal(t, map[string]*resources.Job{"job_1": job1}, b.Config.Resources.Jobs)
	assert.Nil(t, b.Config.Resources.Pipelines)

	require.NoError(t, restore())
	assert.Equal(t, map[string]*resources.Job{"job_1": job1, "job_2": job2}, b.Config.Resources.Jobs)
	assert.Len(t, b.Config.Resources.Pipelines, 1)
}

func TestSelectResourcesRestoreMergesUpdates(t *testing.T) {
	b := &bundle.Bundle{Config: config.Root{
		Resources: config.Resources{
			Jobs: map[string]*resources.Job{"job_1": newJob("job_1"), "job_2": newJob("job_2")},
		},
	}}
	b.Config.Bundle.Name = "before"

	keys := NewResourceKeySet()
	keys.AddResourceKey(ResourceKey{Type: "jobs", Name: "job_1"})

	restore := selectResources(b, keys)

	// Changes to selected resources are kept, other changes are discarded.
	require.NoError(t, b.Config.Set(structpath.MustParsePath("resources.jobs.job_1.name"), "updated"))
	b.Config.Bundle.Name = "after"

	require.NoError(t, restore())
	assert.Equal(t, "updated", b.Config.Resources.Jobs["job_1"].Name)
	assert.Equal(t, "job_2", b.Config.Resources.Jobs["job_2"].Name)
	assert.Equal(t, "before", b.Config.Bundle.Name)
}

func TestSelectResourcesRestoreKeepsResourceType(t *testing.T) {
	b := &bundle.Bundle{Config: config.Root{
		Resources: config.Resources{
			Jobs: map[string]*resources.Job{"job_1": newJob("job_1")},
		},
	}}

	keys := NewResourceKeySet()
	keys.AddResourceKey(ResourceKey{Type: "jobs", Name: "job_1"})

	restore := selectResources(b, keys)
	require.NoError(t, restore())
	assert.Equal(t, "job_1", b.Config.Resources.Jobs["job_1"].Name)
	assert.Nil(t, b.Config.Resources.Pipelines)
}
