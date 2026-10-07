package statemgmt

import (
	"encoding/json"
	"testing"

	"github.com/databricks/cli/bundle"
	"github.com/databricks/cli/bundle/config"
	"github.com/databricks/cli/libs/structs/structpath"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestReverseInterpolatePreservesBConfigValue(t *testing.T) {
	// This test verifies that reverse interpolating a config returns a new
	// config.Root and does NOT mutate the original.

	root, diags := config.LoadFromBytes("test.yml", []byte(`
bundle:
  name: test
resources:
  jobs:
    my_job:
      name: My Job
      description: ${databricks_pipeline.my_pipeline.id}
      max_concurrent_runs: ${databricks_job.other.id}
`))
	require.NoError(t, diags.Error())
	b := &bundle.Bundle{Config: *root}

	originalJSON, err := json.Marshal(b.Config.View().AsAny())
	require.NoError(t, err)

	uninterpolatedConfig, err := reverseInterpolateConfig(&b.Config)
	require.NoError(t, err)

	uninterpolated := uninterpolatedConfig.View()
	description, ok := uninterpolated.Lookup(structpath.MustParsePath("resources.jobs.my_job.description")).AsString()
	require.True(t, ok)
	assert.Equal(t, "${resources.pipelines.my_pipeline.id}", description, "should be bundle-style after reverse interpolation")

	// References in fields that cannot hold a string are rewritten too.
	maxRuns, ok := uninterpolated.Lookup(structpath.MustParsePath("resources.jobs.my_job.max_concurrent_runs")).AsString()
	require.True(t, ok)
	assert.Equal(t, "${resources.jobs.other.id}", maxRuns)

	// Locations are kept.
	assert.Equal(t, b.Config.GetLocations("resources.jobs.my_job.description"), uninterpolatedConfig.GetLocations("resources.jobs.my_job.description"))

	afterJSON, err := json.Marshal(b.Config.View().AsAny())
	require.NoError(t, err)
	assert.Equal(t, string(originalJSON), string(afterJSON), "b.Config should not change")

	originalDescription, ok := b.Config.View().Lookup(structpath.MustParsePath("resources.jobs.my_job.description")).AsString()
	require.True(t, ok)
	assert.Equal(t, "${databricks_pipeline.my_pipeline.id}", originalDescription, "terraform-style reference should be preserved in b.Config")
}

func TestReverseInterpolate(t *testing.T) {
	tests := []struct {
		name     string
		input    string
		expected string
	}{
		{
			name:     "converts terraform-style job reference to bundle-style",
			input:    "${databricks_job.my_job.id}",
			expected: "${resources.jobs.my_job.id}",
		},
		{
			name:     "leaves bundle-style references unchanged",
			input:    "${resources.pipelines.my_pipeline.id}",
			expected: "${resources.pipelines.my_pipeline.id}",
		},
		{
			name:     "handles nested paths",
			input:    "${databricks_pipeline.my_pipeline.url}",
			expected: "${resources.pipelines.my_pipeline.url}",
		},
		{
			name:     "skips unknown terraform resource types",
			input:    "${unknown_resource.my_resource.id}",
			expected: "${unknown_resource.my_resource.id}",
		},
		{
			name:     "handles multiple references in one value",
			input:    "${databricks_job.job1.id}/${databricks_pipeline.pipeline1.id}",
			expected: "${resources.jobs.job1.id}/${resources.pipelines.pipeline1.id}",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result, err := reverseInterpolate(tt.input)
			require.NoError(t, err)
			assert.Equal(t, tt.expected, result)
		})
	}
}
