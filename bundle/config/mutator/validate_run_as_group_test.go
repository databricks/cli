package mutator_test

import (
	"testing"

	"github.com/databricks/cli/bundle"
	"github.com/databricks/cli/bundle/config"
	"github.com/databricks/cli/bundle/config/engine"
	"github.com/databricks/cli/bundle/config/mutator"
	"github.com/databricks/cli/bundle/config/resources"
	"github.com/databricks/cli/libs/dyn"
	"github.com/databricks/databricks-sdk-go/service/jobs"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestValidateRunAsGroup(t *testing.T) {
	for _, tc := range []struct {
		engine engine.EngineType
		paths  []string
	}{
		{engine: engine.EngineDirect},
		{engine: engine.EngineTerraform, paths: []string{"resources.jobs.a.run_as.group_name", "resources.jobs.z.run_as.group_name"}},
	} {
		t.Run(string(tc.engine), func(t *testing.T) {
			b := &bundle.Bundle{Config: config.Root{Resources: config.Resources{Jobs: map[string]*resources.Job{
				"z":    {JobSettings: jobs.JobSettings{RunAs: &jobs.JobRunAs{GroupName: "group"}}},
				"a":    {JobSettings: jobs.JobSettings{RunAs: &jobs.JobRunAs{GroupName: "group"}}},
				"user": {JobSettings: jobs.JobSettings{RunAs: &jobs.JobRunAs{UserName: "user@example.test"}}},
				"sp":   {JobSettings: jobs.JobSettings{RunAs: &jobs.JobRunAs{ServicePrincipalName: "sp"}}},
				"none": {},
			}}}}
			diags := bundle.Apply(t.Context(), b, mutator.ValidateRunAsGroup(tc.engine))
			require.Len(t, diags, len(tc.paths))
			for i, path := range tc.paths {
				assert.Equal(t, []dyn.Path{dyn.MustPathFromString(path)}, diags[i].Paths)
			}
		})
	}
}
