package client

import (
	"testing"
	"time"

	"github.com/databricks/databricks-sdk-go/service/compute"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
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
		got := buildSSHServerSubmitRun("v1", "scope", notebookPath, "", opts)

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
		got := buildSSHServerSubmitRun("v1", "scope", notebookPath, "", opts)

		assert.Equal(t, compute.HardwareAcceleratorType("GPU_1xA10"), got.Tasks[0].Compute.HardwareAccelerator)
	})

	t.Run("serverless with base environment", func(t *testing.T) {
		opts := ClientOptions{
			ConnectionName:     "conn",
			ServerTimeout:      time.Hour,
			EnvironmentVersion: 4,
			BaseEnvironment:    "my-env",
		}
		got := buildSSHServerSubmitRun("v1", "scope", notebookPath, "workspace-base-environments/dbe_123", opts)

		// A resolved base environment carries its own version, so environment_version is not set.
		require.Len(t, got.Environments, 1)
		assert.Equal(t, "workspace-base-environments/dbe_123", got.Environments[0].Spec.BaseEnvironment)
		assert.Empty(t, got.Environments[0].Spec.EnvironmentVersion)
	})

	t.Run("dedicated cluster", func(t *testing.T) {
		opts := ClientOptions{
			ClusterID:     "abc-123",
			ServerTimeout: time.Hour,
		}
		got := buildSSHServerSubmitRun("v1", "scope", notebookPath, "", opts)

		// Usage policy is serverless-only; a dedicated run carries none and targets the cluster.
		assert.Empty(t, got.BudgetPolicyId)
		assert.Empty(t, got.Tasks[0].NotebookTask.BaseParameters["usagePolicyId"])
		assert.Equal(t, "abc-123", got.Tasks[0].ExistingClusterId)
		assert.Empty(t, got.Tasks[0].EnvironmentKey)
		assert.Empty(t, got.Environments)
	})

	t.Run("server lifecycle", func(t *testing.T) {
		opts := ClientOptions{
			ClusterID:     "abc-123",
			MaxClients:    25,
			ShutdownDelay: 15 * time.Minute,
			ServerTimeout: 48 * time.Hour,
		}
		got := buildSSHServerSubmitRun("v1", "scope", notebookPath, "", opts)

		// This is the only place these two take effect: the server reads maxClients from the
		// widget at startup, and the run's timeout caps the tunnel's lifetime.
		assert.Equal(t, "25", got.Tasks[0].NotebookTask.BaseParameters["maxClients"])
		assert.Equal(t, "15m0s", got.Tasks[0].NotebookTask.BaseParameters["shutdownDelay"])
		assert.Equal(t, 48*60*60, got.TimeoutSeconds)
		assert.Equal(t, 48*60*60, got.Tasks[0].TimeoutSeconds)
	})
}

// Validate's lower bound on --server-timeout exists only to keep timeout_seconds off 0, which the
// Jobs API reads as "no timeout". Tie the two together: no value Validate accepts may submit an
// unbounded run, whichever side is changed later.
func TestServerTimeoutNeverSubmitsUnboundedRun(t *testing.T) {
	const notebookPath = "/Workspace/Users/me/.databricks/ssh-tunnel/v1/conn/ssh-server-bootstrap"

	for _, d := range []time.Duration{
		-time.Minute,
		0,
		time.Nanosecond,
		500 * time.Millisecond,
		999 * time.Millisecond,
		time.Second,
		10 * time.Minute,
		24 * time.Hour,
	} {
		opts := ClientOptions{ClusterID: "abc-123", MaxClients: 10, ServerTimeout: d}
		if err := opts.Validate(); err != nil {
			continue
		}
		got := buildSSHServerSubmitRun("v1", "scope", notebookPath, "", opts)
		assert.NotZero(t, got.TimeoutSeconds, "--server-timeout=%s passed Validate but submits timeout_seconds: 0 (no timeout)", d)
		assert.NotZero(t, got.Tasks[0].TimeoutSeconds, "--server-timeout=%s passed Validate but submits a task with timeout_seconds: 0", d)
	}
}
