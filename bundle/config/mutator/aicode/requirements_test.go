package aicode

import (
	"testing"

	"github.com/databricks/databricks-sdk-go/service/compute"
	"github.com/databricks/databricks-sdk-go/service/jobs"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestRenderRequirementsFromEnvironmentVersion(t *testing.T) {
	out, err := renderRequirements(&compute.Environment{
		EnvironmentVersion: "5",
		Dependencies:       []string{"numpy", "torch>=2.0.0"},
	})
	require.NoError(t, err)
	assert.Equal(t, "version: \"5\"\ndependencies:\n  - numpy\n  - torch>=2.0.0\n", string(out))
}

func TestRenderRequirementsFallsBackToClient(t *testing.T) {
	out, err := renderRequirements(&compute.Environment{Client: "5"})
	require.NoError(t, err)
	assert.Equal(t, "version: \"5\"\n", string(out))
}

func TestEnvironmentsByKey(t *testing.T) {
	envs := []jobs.JobEnvironment{
		{EnvironmentKey: "default", Spec: &compute.Environment{EnvironmentVersion: "5"}},
		{EnvironmentKey: "nospec"},
	}
	got := environmentsByKey(envs)
	require.Contains(t, got, "default")
	assert.Equal(t, "5", got["default"].EnvironmentVersion)
	assert.NotContains(t, got, "nospec", "environments without a spec are skipped")
}
