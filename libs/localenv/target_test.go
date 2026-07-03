package localenv

import (
	"context"
	"errors"
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
	assert.Equal(t, "serverless", ti.Source)
	assert.Equal(t, "v4", ti.ServerlessVersion)
	assert.Equal(t, "serverless/serverless-v4", ti.EnvKey)
}

func TestResolveClusterFlag(t *testing.T) {
	c := stubCompute{clusterVersion: "15.4.x-scala2.12"}
	ti, err := ResolveTarget(t.Context(), TargetFlags{Cluster: "abc"}, c, BundleTarget{})
	require.NoError(t, err)
	assert.Equal(t, "cluster", ti.Source)
	assert.Equal(t, "15.4.x-scala2.12", ti.SparkVersion)
	assert.Equal(t, "dbr/15.4.x-scala2.12", ti.EnvKey)
	assert.Equal(t, "abc", ti.ClusterID)
}

func TestResolveClusterFlagError(t *testing.T) {
	c := stubCompute{clusterErr: errors.New("cluster not found")}
	_, err := ResolveTarget(t.Context(), TargetFlags{Cluster: "abc"}, c, BundleTarget{})
	var pe *PipelineError
	require.ErrorAs(t, err, &pe)
	assert.Equal(t, ErrResolve, pe.Code)
}

func TestResolveBundleNothingSelected(t *testing.T) {
	_, err := ResolveTarget(t.Context(), TargetFlags{}, stubCompute{}, BundleTarget{Selected: false})
	var pe *PipelineError
	require.ErrorAs(t, err, &pe)
	assert.Equal(t, ErrNoTarget, pe.Code)
}

func TestResolveBundleServerless(t *testing.T) {
	ti, err := ResolveTarget(t.Context(), TargetFlags{}, stubCompute{}, BundleTarget{Selected: true, Serverless: true})
	require.NoError(t, err)
	assert.Equal(t, "bundle", ti.Source)
	assert.Equal(t, "serverless/serverless-v4", ti.EnvKey)
}

// jobStubCompute returns distinct values for the first (sparkVersion) and third
// (recorded version) results of GetJobSparkVersion so the classic-compute branch
// can be checked against the documented contract (it must use the first).
type jobStubCompute struct {
	sparkVersion string
	isServerless bool
	version      string
}

func (jobStubCompute) GetClusterSparkVersion(_ context.Context, _ string) (string, error) {
	return "", nil
}

func (s jobStubCompute) GetJobSparkVersion(_ context.Context, _ string) (string, bool, string, error) {
	return s.sparkVersion, s.isServerless, s.version, nil
}

func TestResolveJobClassicUsesSparkVersionReturn(t *testing.T) {
	// Contract: for a classic-compute job the Spark version is the FIRST return.
	// The third "recorded version" return differs here to catch use of the wrong one.
	c := jobStubCompute{sparkVersion: "15.4.x-scala2.12", isServerless: false, version: "wrong-recorded"}
	ti, err := ResolveTarget(t.Context(), TargetFlags{Job: "42"}, c, BundleTarget{})
	require.NoError(t, err)
	assert.Equal(t, "job", ti.Source)
	assert.Equal(t, "15.4.x-scala2.12", ti.SparkVersion)
	assert.Equal(t, "dbr/15.4.x-scala2.12", ti.EnvKey)
}

func TestValidateTargetFlagsMutuallyExclusive(t *testing.T) {
	assert.Error(t, ValidateTargetFlags(TargetFlags{Cluster: "a", Serverless: "v4"}))
	assert.NoError(t, ValidateTargetFlags(TargetFlags{Cluster: "a"}))
}
