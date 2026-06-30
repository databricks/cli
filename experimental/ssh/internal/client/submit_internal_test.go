package client

import (
	"testing"
	"time"

	"github.com/databricks/databricks-sdk-go/service/compute"
	"github.com/stretchr/testify/assert"
)

func TestBuildSSHServerSubmitRun(t *testing.T) {
	const notebookPath = "/Workspace/Users/me/.databricks/ssh-tunnel/v1/conn/ssh-server-bootstrap"

	t.Run("serverless with usage policy", func(t *testing.T) {
		opts := ClientOptions{
			ConnectionName:     "conn",
			UsagePolicyID:      "pol-1",
			ServerTimeout:      time.Hour,
			EnvironmentVersion: 4,
		}
		got := buildSSHServerSubmitRun("v1", "scope", notebookPath, opts)

		// Usage policy flows onto the run and into the base params the server reads.
		assert.Equal(t, "pol-1", got.BudgetPolicyId)
		assert.Equal(t, "pol-1", got.Tasks[0].NotebookTask.BaseParameters["usagePolicyId"])
		assert.Equal(t, "true", got.Tasks[0].NotebookTask.BaseParameters["serverless"])

		// Serverless runs on an environment, not an existing cluster.
		assert.Equal(t, serverlessEnvironmentKey, got.Tasks[0].EnvironmentKey)
		assert.Empty(t, got.Tasks[0].ExistingClusterId)
		assert.Len(t, got.Environments, 1)
		assert.Nil(t, got.Tasks[0].Compute)
	})

	t.Run("serverless with accelerator", func(t *testing.T) {
		opts := ClientOptions{
			ConnectionName: "conn",
			Accelerator:    "GPU_1xA10",
			ServerTimeout:  time.Hour,
		}
		got := buildSSHServerSubmitRun("v1", "scope", notebookPath, opts)

		assert.Equal(t, compute.HardwareAcceleratorType("GPU_1xA10"), got.Tasks[0].Compute.HardwareAccelerator)
	})

	t.Run("dedicated cluster", func(t *testing.T) {
		opts := ClientOptions{
			ClusterID:     "abc-123",
			ServerTimeout: time.Hour,
		}
		got := buildSSHServerSubmitRun("v1", "scope", notebookPath, opts)

		// Usage policy is serverless-only; a dedicated run carries none and targets the cluster.
		assert.Empty(t, got.BudgetPolicyId)
		assert.Empty(t, got.Tasks[0].NotebookTask.BaseParameters["usagePolicyId"])
		assert.Equal(t, "abc-123", got.Tasks[0].ExistingClusterId)
		assert.Empty(t, got.Tasks[0].EnvironmentKey)
		assert.Empty(t, got.Environments)
	})
}
