package aircmd

import (
	"testing"

	"github.com/databricks/databricks-sdk-go/service/jobs"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// airRunWithCompute builds a run whose AI runtime task reports the given
// accelerator type and count, for node-count resolution tests.
func airRunWithCompute(accelType string, count int) *jobs.Run {
	return &jobs.Run{
		RunId: 123,
		Tasks: []jobs.RunTask{{
			RunId: 456,
			AiRuntimeTask: &jobs.AiRuntimeTask{
				Deployments: []jobs.DeploymentSpec{{
					Compute: jobs.ComputeSpec{
						AcceleratorType:  jobs.ComputeSpecAcceleratorType(accelType),
						AcceleratorCount: count,
					},
				}},
			},
		}},
	}
}

func TestResolveNodeCount(t *testing.T) {
	tests := []struct {
		accelType string
		count     int
		want      int
	}{
		{"GPU_1xA10", 2, 2},
		{"GPU_1xH100", 4, 4},
		{"GPU_8xH100", 16, 2},
	}
	for _, tt := range tests {
		n, err := resolveNodeCount(airRunWithCompute(tt.accelType, tt.count))
		require.NoError(t, err)
		assert.Equal(t, tt.want, n)
	}

	// A run with no AI runtime compute errors.
	_, err := resolveNodeCount(&jobs.Run{RunId: 1})
	require.Error(t, err)
}
