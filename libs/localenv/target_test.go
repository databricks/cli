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
	byNameID       string
	byNameVersion  string
	byNameErr      error
}

func (s stubCompute) GetClusterSparkVersion(_ context.Context, _ string) (string, error) {
	return s.clusterVersion, s.clusterErr
}

func (s stubCompute) GetClusterByName(_ context.Context, _ string) (string, string, error) {
	return s.byNameID, s.byNameVersion, s.byNameErr
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

func TestResolveServerlessFlagBareNumber(t *testing.T) {
	// The documented input is a bare number; it normalizes to the vN env key.
	ti, err := ResolveTarget(t.Context(), TargetFlags{Serverless: "5"}, stubCompute{}, BundleTarget{})
	require.NoError(t, err)
	assert.Equal(t, "serverless/serverless-v5", ti.EnvKey)
}

func TestResolveServerlessFlagRejectsMalformed(t *testing.T) {
	// Malformed values fail fast at resolve (E_RESOLVE) rather than resolving to
	// a bogus env key that only 404s at fetch.
	for _, bad := range []string{"vv5", "v", " 5", "5x", "latest"} {
		_, err := ResolveTarget(t.Context(), TargetFlags{Serverless: bad}, stubCompute{}, BundleTarget{})
		var pe *PipelineError
		require.ErrorAs(t, err, &pe, "input %q should error", bad)
		assert.Equal(t, ErrResolve, pe.Code, "input %q", bad)
	}
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

func TestResolveClusterNameFlag(t *testing.T) {
	// --cluster-name resolves to an ID via the Clusters API, then behaves like
	// --cluster-id: source=cluster, the resolved ID is reported, and the env key
	// derives from the resolved cluster's Spark version.
	c := stubCompute{byNameID: "cid-123", byNameVersion: "15.4.x-scala2.12"}
	ti, err := ResolveTarget(t.Context(), TargetFlags{ClusterName: "my-cluster"}, c, BundleTarget{})
	require.NoError(t, err)
	assert.Equal(t, "cluster", ti.Source)
	assert.Equal(t, "cid-123", ti.ClusterID)
	assert.Equal(t, "15.4.x-scala2.12", ti.SparkVersion)
	assert.Equal(t, "dbr/15.4.x-scala2.12", ti.EnvKey)
}

func TestResolveClusterNameFlagError(t *testing.T) {
	// An unknown or ambiguous name surfaces as E_RESOLVE.
	c := stubCompute{byNameErr: errors.New("there are 2 instances of ClusterDetails named 'dup'")}
	_, err := ResolveTarget(t.Context(), TargetFlags{ClusterName: "dup"}, c, BundleTarget{})
	var pe *PipelineError
	require.ErrorAs(t, err, &pe)
	assert.Equal(t, ErrResolve, pe.Code)
}

func TestResolveClusterIdAndNameMutuallyExclusive(t *testing.T) {
	// The library path rejects setting both --cluster-id and --cluster-name.
	_, err := ResolveTarget(t.Context(), TargetFlags{Cluster: "abc", ClusterName: "xyz"}, stubCompute{}, BundleTarget{})
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
	// The bundle records serverless but no version, so the default stand-in applies.
	ti, err := ResolveTarget(t.Context(), TargetFlags{}, stubCompute{}, BundleTarget{Selected: true, Serverless: true})
	require.NoError(t, err)
	assert.Equal(t, "bundle", ti.Source)
	// Concrete literal, not "serverless-"+defaultServerlessVersion: the default
	// is v5, and asserting the constant against itself would pass for any value.
	assert.Equal(t, "serverless/serverless-v5", ti.EnvKey)
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

func (jobStubCompute) GetClusterByName(_ context.Context, _ string) (string, string, error) {
	return "", "", nil
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

func TestResolveJobServerlessUsesRecordedVersion(t *testing.T) {
	// A serverless job (isServerless=true) pins its serverless version via the
	// third "recorded version" return; ResolveTarget must map it to the matching
	// serverless-vN rather than the classic dbr path.
	c := jobStubCompute{isServerless: true, version: "3"}
	ti, err := ResolveTarget(t.Context(), TargetFlags{Job: "42"}, c, BundleTarget{})
	require.NoError(t, err)
	assert.Equal(t, "job", ti.Source)
	assert.Empty(t, ti.SparkVersion)
	assert.Equal(t, "serverless/serverless-v3", ti.EnvKey)
}

func TestResolveJobServerlessEmptyVersionFallsBackToDefault(t *testing.T) {
	// When the job records no serverless version, ResolveTarget uses the default
	// stand-in, matching the bundle serverless path.
	c := jobStubCompute{isServerless: true, version: ""}
	ti, err := ResolveTarget(t.Context(), TargetFlags{Job: "42"}, c, BundleTarget{})
	require.NoError(t, err)
	// Concrete literal, not "serverless-"+defaultServerlessVersion: the default
	// is v5, and asserting the constant against itself would pass for any value.
	assert.Equal(t, "serverless/serverless-v5", ti.EnvKey)
}

func TestValidateTargetFlagsMutuallyExclusive(t *testing.T) {
	assert.Error(t, ValidateTargetFlags(TargetFlags{Cluster: "a", Serverless: "v4"}))
	assert.NoError(t, ValidateTargetFlags(TargetFlags{Cluster: "a"}))
}

func TestResolveTargetRejectsConflictingFlags(t *testing.T) {
	// ResolveTarget must reject incompatible flags rather than silently taking
	// the first precedence branch, so a library caller bypassing Cobra can't
	// resolve a different target than it asked for.
	_, err := ResolveTarget(t.Context(), TargetFlags{Cluster: "c", Serverless: "v4"}, stubCompute{}, BundleTarget{})
	var pe *PipelineError
	require.ErrorAs(t, err, &pe)
	assert.Equal(t, ErrResolve, pe.Code)
}
