package validate

import (
	"testing"

	"github.com/databricks/cli/bundle"
	"github.com/databricks/cli/bundle/config"
	"github.com/databricks/cli/bundle/config/engine"
	"github.com/databricks/cli/bundle/config/resources"
	"github.com/databricks/cli/bundle/internal/bundletest"
	"github.com/databricks/cli/libs/diag"
	"github.com/databricks/cli/libs/dyn"
	"github.com/databricks/cli/libs/env"
	"github.com/databricks/databricks-sdk-go/service/jobs"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func makeBundle(t *testing.T) *bundle.Bundle {
	t.Helper()
	b := &bundle.Bundle{
		Config: config.Root{
			Resources: config.Resources{
				Jobs: map[string]*resources.Job{
					"src": {JobSettings: jobs.JobSettings{Name: "source"}},
					"dst": {JobSettings: jobs.JobSettings{Name: "placeholder"}},
				},
			},
		},
	}
	return b
}

func TestTFOnlyReferences_DirectError(t *testing.T) {
	b := makeBundle(t)
	bundletest.Mutate(t, b, func(v dyn.Value) (dyn.Value, error) {
		return dyn.Set(v, "resources.jobs.dst.name", dyn.V("${resources.jobs.src.always_running}"))
	})

	ctx := env.Set(t.Context(), engine.EnvVar, "direct")
	diags := TFOnlyReferences().Apply(ctx, b)
	require.Len(t, diags, 1)
	assert.Equal(t, diag.Error, diags[0].Severity)
	assert.Contains(t, diags[0].Summary, "resources.jobs.src.always_running")
	assert.Contains(t, diags[0].Summary, "Terraform-only field")
}

func TestTFOnlyReferences_TerraformWarning(t *testing.T) {
	b := makeBundle(t)
	bundletest.Mutate(t, b, func(v dyn.Value) (dyn.Value, error) {
		return dyn.Set(v, "resources.jobs.dst.name", dyn.V("${resources.jobs.src.always_running}"))
	})

	ctx := env.Set(t.Context(), engine.EnvVar, "terraform")
	diags := TFOnlyReferences().Apply(ctx, b)
	require.Len(t, diags, 1)
	assert.Equal(t, diag.Warning, diags[0].Severity)
	assert.Contains(t, diags[0].Summary, "resources.jobs.src.always_running")
	assert.Contains(t, diags[0].Summary, "Terraform-only field")
}

func TestTFOnlyReferences_NormalReference(t *testing.T) {
	b := makeBundle(t)
	bundletest.Mutate(t, b, func(v dyn.Value) (dyn.Value, error) {
		// "name" is not a TF-only field; no diagnostic expected.
		return dyn.Set(v, "resources.jobs.dst.name", dyn.V("${resources.jobs.src.name}"))
	})

	ctx := env.Set(t.Context(), engine.EnvVar, "direct")
	diags := TFOnlyReferences().Apply(ctx, b)
	assert.Empty(t, diags)
}

func TestTFOnlyReferences_RenamedField(t *testing.T) {
	b := makeBundle(t)
	bundletest.Mutate(t, b, func(v dyn.Value) (dyn.Value, error) {
		// "git_source[0].branch" is a TF rename (not TF-only), should not error.
		return dyn.Set(v, "resources.jobs.dst.name", dyn.V("${resources.jobs.src.git_source[0].branch}"))
	})

	ctx := env.Set(t.Context(), engine.EnvVar, "direct")
	diags := TFOnlyReferences().Apply(ctx, b)
	assert.Empty(t, diags)
}
