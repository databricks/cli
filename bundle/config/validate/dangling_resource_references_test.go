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
	"github.com/stretchr/testify/require"
)

func TestDanglingResourceReferences_MissingResource(t *testing.T) {
	b := &bundle.Bundle{
		Config: config.Root{
			Resources: config.Resources{
				Jobs: map[string]*resources.Job{
					"my_job": {JobSettings: jobs.JobSettings{Name: "my_job"}},
				},
			},
		},
	}
	bundletest.Mutate(t, b, func(v dyn.Value) (dyn.Value, error) {
		return dyn.Set(v, "resources.jobs.my_job.name", dyn.V("${resources.jobs.does_not_exist.id}"))
	})

	diags := DanglingResourceReferences().Apply(t.Context(), b)
	require.Len(t, diags, 1)
	assert.Equal(t, diag.Error, diags[0].Severity)
	assert.Equal(t, "reference does not exist: ${resources.jobs.does_not_exist.id}", diags[0].Summary)
}

func TestDanglingResourceReferences_ExistingResource(t *testing.T) {
	b := &bundle.Bundle{
		Config: config.Root{
			Resources: config.Resources{
				Jobs: map[string]*resources.Job{
					"src": {JobSettings: jobs.JobSettings{Name: "src"}},
					"dst": {JobSettings: jobs.JobSettings{Name: "dst"}},
				},
			},
		},
	}
	bundletest.Mutate(t, b, func(v dyn.Value) (dyn.Value, error) {
		return dyn.Set(v, "resources.jobs.dst.name", dyn.V("${resources.jobs.src.id}"))
	})

	diags := DanglingResourceReferences().Apply(t.Context(), b)
	assert.Empty(t, diags)
}

func TestDanglingResourceReferences_UnknownType(t *testing.T) {
	b := &bundle.Bundle{
		Config: config.Root{
			Resources: config.Resources{
				Jobs: map[string]*resources.Job{
					"my_job": {JobSettings: jobs.JobSettings{Name: "my_job"}},
				},
			},
		},
	}
	bundletest.Mutate(t, b, func(v dyn.Value) (dyn.Value, error) {
		return dyn.Set(v, "resources.jobs.my_job.name", dyn.V("${resources.unknown.foo.id}"))
	})

	diags := DanglingResourceReferences().Apply(t.Context(), b)
	require.Len(t, diags, 1)
	assert.Contains(t, diags[0].Summary, "${resources.unknown.foo.id}")
}
