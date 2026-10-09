package statemgmt

import (
	"encoding/json"
	"testing"

	"github.com/databricks/cli/bundle"
	"github.com/databricks/cli/bundle/config"
	"github.com/databricks/cli/libs/dyn"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestReverseInterpolatePreservesBConfigValue(t *testing.T) {
	// This test verifies that our approach of getting b.Config.Value(),
	// reverse interpolating it, and wrapping in a new config.Root
	// does NOT mutate b.Config

	b := &bundle.Bundle{
		Config: config.Root{
			Bundle: config.Bundle{
				Name: "test",
			},
		},
	}

	err := b.Config.Mutate(func(_ dyn.Value) (dyn.Value, error) {
		return dyn.V(map[string]dyn.Value{
			"bundle": dyn.V(map[string]dyn.Value{
				"name": dyn.V("test"),
			}),
			"resources": dyn.V(map[string]dyn.Value{
				"jobs": dyn.V(map[string]dyn.Value{
					"my_job": dyn.V(map[string]dyn.Value{
						"name":       dyn.V("My Job"),
						"depends_on": dyn.V("${databricks_pipeline.my_pipeline.id}"),
					}),
				}),
			}),
		}), nil
	})
	require.NoError(t, err)

	originalValue := b.Config.Value()
	originalJSON, err := json.Marshal(originalValue.AsAny())
	require.NoError(t, err)

	interpolatedRoot := b.Config.Value()

	uninterpolatedRoot, err := reverseInterpolate(interpolatedRoot)
	require.NoError(t, err)

	dependsOn, err := dyn.GetByPath(uninterpolatedRoot, dyn.MustPathFromString("resources.jobs.my_job.depends_on"))
	require.NoError(t, err)
	dependsOnStr, ok := dependsOn.AsString()
	require.True(t, ok)
	assert.Equal(t, "${resources.pipelines.my_pipeline.id}", dependsOnStr, "should be bundle-style after reverse interpolation")

	var uninterpolatedConfig config.Root
	err = uninterpolatedConfig.Mutate(func(_ dyn.Value) (dyn.Value, error) {
		return uninterpolatedRoot, nil
	})
	require.NoError(t, err)

	afterValue := b.Config.Value()
	afterJSON, err := json.Marshal(afterValue.AsAny())
	require.NoError(t, err)

	assert.Equal(t, string(originalJSON), string(afterJSON), "b.Config.Value() should not change")

	originalDependsOn, err := dyn.GetByPath(afterValue, dyn.MustPathFromString("resources.jobs.my_job.depends_on"))
	require.NoError(t, err)
	originalDependsOnStr, ok := originalDependsOn.AsString()
	require.True(t, ok)
	assert.Equal(t, "${databricks_pipeline.my_pipeline.id}", originalDependsOnStr, "terraform-style reference should be preserved in b.Config")
}

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
