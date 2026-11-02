package validate

import (
	"testing"

	"github.com/databricks/cli/bundle"
	"github.com/databricks/cli/bundle/config"
	"github.com/databricks/cli/bundle/config/resources"
	"github.com/databricks/cli/libs/diag"
	"github.com/databricks/cli/libs/structs/structpath"
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

func TestTFOnlyReferences_Error(t *testing.T) {
	b := makeBundle(t)
	require.NoError(t, b.Config.Set(structpath.MustParsePath("resources.jobs.dst.name"), "${resources.jobs.src.always_running}"))

	diags := TFOnlyReferences().Apply(t.Context(), b)
	require.Len(t, diags, 1)
	assert.Equal(t, diag.Error, diags[0].Severity)
	assert.Contains(t, diags[0].Summary, "resources.jobs.src.always_running")
	assert.Contains(t, diags[0].Summary, "Terraform-only field")
}

func TestTFOnlyReferences_NormalReference(t *testing.T) {
	b := makeBundle(t)
	// "name" is not a TF-only field; no diagnostic expected.
	require.NoError(t, b.Config.Set(structpath.MustParsePath("resources.jobs.dst.name"), "${resources.jobs.src.name}"))

	diags := TFOnlyReferences().Apply(t.Context(), b)
	assert.Empty(t, diags)
}

func TestTFOnlyReferences_RenamedField(t *testing.T) {
	b := makeBundle(t)
	// "git_source[0].branch" is a TF rename (not TF-only), should not error.
	require.NoError(t, b.Config.Set(structpath.MustParsePath("resources.jobs.dst.name"), "${resources.jobs.src.git_source[0].branch}"))

	diags := TFOnlyReferences().Apply(t.Context(), b)
	assert.Empty(t, diags)
}
