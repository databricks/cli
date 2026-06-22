package dbconnect

import (
	"context"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

type stubCompute struct {
	clusterVersion string
	clusterErr     error
}

func (s stubCompute) GetClusterSparkVersion(_ context.Context, _ string) (string, error) {
	return s.clusterVersion, s.clusterErr
}
func (s stubCompute) GetJobSparkVersion(_ context.Context, _ string) (string, bool, string, error) {
	return "", false, "", nil
}

func TestResolveServerlessFlag(t *testing.T) {
	ti, err := ResolveTarget(t.Context(), TargetFlags{Serverless: "v4"}, stubCompute{}, BundleTarget{})
	require.NoError(t, err)
	assert.Equal(t, "serverless", ti.Kind)
	assert.Equal(t, "serverless/serverless-v4", ti.EnvKey)
}

func TestResolveClusterFlag(t *testing.T) {
	c := stubCompute{clusterVersion: "15.4.x-scala2.12"}
	ti, err := ResolveTarget(t.Context(), TargetFlags{Cluster: "abc"}, c, BundleTarget{})
	require.NoError(t, err)
	assert.Equal(t, "cluster", ti.Kind)
	assert.Equal(t, "dbr/15.4.x-scala2.12", ti.EnvKey)
	assert.Equal(t, "abc", ti.ClusterID)
}

func TestResolveBundleNothingSelected(t *testing.T) {
	_, err := ResolveTarget(t.Context(), TargetFlags{}, stubCompute{}, BundleTarget{Selected: false})
	var pe *PipelineError
	require.ErrorAs(t, err, &pe)
	assert.Equal(t, ErrNoTargetSelected, pe.Code)
}

func TestResolveBundleServerless(t *testing.T) {
	ti, err := ResolveTarget(t.Context(), TargetFlags{}, stubCompute{}, BundleTarget{Selected: true, Serverless: true})
	require.NoError(t, err)
	assert.Equal(t, "serverless/serverless-v4", ti.EnvKey)
}

func TestValidateTargetFlagsMutuallyExclusive(t *testing.T) {
	assert.Error(t, ValidateTargetFlags(TargetFlags{Cluster: "a", Serverless: "v4"}))
	assert.NoError(t, ValidateTargetFlags(TargetFlags{Cluster: "a"}))
}
