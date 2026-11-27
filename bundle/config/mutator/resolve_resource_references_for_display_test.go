package mutator

import (
	"testing"

	"github.com/databricks/cli/bundle"
	"github.com/databricks/cli/bundle/config"
	"github.com/databricks/cli/bundle/config/resources"
	"github.com/databricks/cli/libs/dyn"
	"github.com/databricks/databricks-sdk-go/service/jobs"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestResolveResourceReferencesForDisplay(t *testing.T) {
	b := &bundle.Bundle{
		Config: config.Root{
			Resources: config.Resources{
				Jobs: map[string]*resources.Job{
					"a":             {ID: "123", JobSettings: jobs.JobSettings{Name: "job-a"}},
					"ref_id":        {JobSettings: jobs.JobSettings{Name: "ref ${resources.jobs.a.id}"}},
					"ref_name":      {JobSettings: jobs.JobSettings{Name: "after ${resources.jobs.a.name}"}},
					"undeployed":    {JobSettings: jobs.JobSettings{Name: "ref ${resources.jobs.b.id}"}},
					"b":             {JobSettings: jobs.JobSettings{Name: "job-b"}},
					"ref_secret":    {JobSettings: jobs.JobSettings{Name: "secret ${resources.secrets.s.value}"}},
					"cycle_a":       {JobSettings: jobs.JobSettings{Name: "${resources.jobs.cycle_b.name}"}},
					"cycle_b":       {JobSettings: jobs.JobSettings{Name: "${resources.jobs.cycle_a.name}"}},
					"bad_type":      {JobSettings: jobs.JobSettings{Name: "bad"}},
					"ref_null":      {JobSettings: jobs.JobSettings{Name: "schedule ${resources.jobs.null_schedule.schedule}"}},
					"null_schedule": {JobSettings: jobs.JobSettings{Name: "n"}},
					"run_job": {JobSettings: jobs.JobSettings{Tasks: []jobs.Task{{
						TaskKey:    "t",
						RunJobTask: &jobs.RunJobTask{},
					}}}},
				},
				Secrets: map[string]*resources.Secret{
					"s": {},
				},
			},
		},
	}

	// Set fields whose typed form can't hold a reference.
	require.NoError(t, b.Config.Mutate(func(v dyn.Value) (dyn.Value, error) {
		for path, value := range map[string]string{
			"resources.secrets.s.value":                           "s3cr3t",
			"resources.jobs.bad_type.max_concurrent_runs":         "${resources.jobs.a.name}",
			"resources.jobs.run_job.tasks[0].run_job_task.job_id": "${resources.jobs.a.id}",
		} {
			var err error
			v, err = dyn.SetByPath(v, dyn.MustPathFromString(path), dyn.V(value))
			if err != nil {
				return v, err
			}
		}
		return dyn.SetByPath(v, dyn.MustPathFromString("resources.jobs.null_schedule.schedule"), dyn.NilValue)
	}))

	diags := bundle.Apply(t.Context(), b, ResolveResourceReferencesForDisplay())
	require.NoError(t, diags.Error())

	j := b.Config.Resources.Jobs
	assert.Equal(t, "ref 123", j["ref_id"].Name)
	assert.Equal(t, "after job-a", j["ref_name"].Name)
	assert.Equal(t, int64(123), j["run_job"].Tasks[0].RunJobTask.JobId)
	// b has no id: the reference stays rather than becoming "ref ".
	assert.Equal(t, "ref ${resources.jobs.b.id}", j["undeployed"].Name)
	assert.Equal(t, "secret ${resources.secrets.s.value}", j["ref_secret"].Name)
	assert.Equal(t, "${resources.jobs.cycle_b.name}", j["cycle_a"].Name)
	// An explicit null is not interpolated as "<nil>".
	assert.Equal(t, "schedule ${resources.jobs.null_schedule.schedule}", j["ref_null"].Name)

	// A name does not fit the int field: the resource keeps its references.
	v, err := dyn.GetByPath(b.Config.Value(), dyn.MustPathFromString("resources.jobs.bad_type.max_concurrent_runs"))
	require.NoError(t, err)
	assert.Equal(t, "${resources.jobs.a.name}", v.MustString())
}
