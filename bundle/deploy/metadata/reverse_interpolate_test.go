package metadata

import (
	"testing"

	"github.com/databricks/cli/libs/dyn"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestReverseInterpolate(t *testing.T) {
	tests := []struct {
		name     string
		input    dyn.Value
		expected dyn.Value
	}{
		{
			name: "converts terraform-style job reference to bundle-style",
			input: dyn.V(map[string]dyn.Value{
				"job_id": dyn.V("${databricks_job.my_job.id}"),
			}),
			expected: dyn.V(map[string]dyn.Value{
				"job_id": dyn.V("${resources.jobs.my_job.id}"),
			}),
		},
		{
			name: "leaves bundle-style references unchanged",
			input: dyn.V(map[string]dyn.Value{
				"pipeline_id": dyn.V("${resources.pipelines.my_pipeline.id}"),
			}),
			expected: dyn.V(map[string]dyn.Value{
				"pipeline_id": dyn.V("${resources.pipelines.my_pipeline.id}"),
			}),
		},
		{
			name: "handles nested paths",
			input: dyn.V(map[string]dyn.Value{
				"config": dyn.V(map[string]dyn.Value{
					"source": dyn.V("${databricks_pipeline.my_pipeline.url}"),
				}),
			}),
			expected: dyn.V(map[string]dyn.Value{
				"config": dyn.V(map[string]dyn.Value{
					"source": dyn.V("${resources.pipelines.my_pipeline.url}"),
				}),
			}),
		},
		{
			name: "skips unknown terraform resource types",
			input: dyn.V(map[string]dyn.Value{
				"unknown": dyn.V("${unknown_resource.my_resource.id}"),
			}),
			expected: dyn.V(map[string]dyn.Value{
				"unknown": dyn.V("${unknown_resource.my_resource.id}"),
			}),
		},
		{
			name: "handles multiple references in one value",
			input: dyn.V(map[string]dyn.Value{
				"combined": dyn.V("${databricks_job.job1.id}/${databricks_pipeline.pipeline1.id}"),
			}),
			expected: dyn.V(map[string]dyn.Value{
				"combined": dyn.V("${resources.jobs.job1.id}/${resources.pipelines.pipeline1.id}"),
			}),
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result, err := reverseInterpolate(tt.input)
			require.NoError(t, err)
			assert.Equal(t, tt.expected.AsAny(), result.AsAny())
		})
	}
}
