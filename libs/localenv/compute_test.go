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

func (s stubCompute) JobTaskEnvironment(_ context.Context, _, _ string) (string, bool, string, error) {
	return "", false, "", nil
}

func TestResolveServerlessFlag(t *testing.T) {
	ti, err := ResolveCompute(t.Context(), ComputeFlags{Serverless: "v4"}, stubCompute{}, BundleTarget{})
	require.NoError(t, err)
	assert.Equal(t, "serverless", ti.Source)
	assert.Equal(t, "v4", ti.ServerlessVersion)
	assert.Equal(t, "serverless/serverless-v4", ti.EnvKey)
}

func TestResolveServerlessFlagBareNumber(t *testing.T) {
	// The documented input is a bare number; it normalizes to the vN env key.
	ti, err := ResolveCompute(t.Context(), ComputeFlags{Serverless: "5"}, stubCompute{}, BundleTarget{})
	require.NoError(t, err)
	assert.Equal(t, "serverless/serverless-v5", ti.EnvKey)
}

func TestResolveServerlessFlagRejectsMalformed(t *testing.T) {
	// Malformed values fail fast at resolve (E_RESOLVE) rather than resolving to
	// a bogus env key that only 404s at fetch.
	for _, bad := range []string{"vv5", "v", " 5", "5x", "latest"} {
		_, err := ResolveCompute(t.Context(), ComputeFlags{Serverless: bad}, stubCompute{}, BundleTarget{})
		var pe *PipelineError
		require.ErrorAs(t, err, &pe, "input %q should error", bad)
		assert.Equal(t, ErrResolve, pe.Code, "input %q", bad)
	}
}

func TestResolveClusterFlag(t *testing.T) {
	c := stubCompute{clusterVersion: "15.4.x-scala2.12"}
	ti, err := ResolveCompute(t.Context(), ComputeFlags{Cluster: "abc"}, c, BundleTarget{})
	require.NoError(t, err)
	assert.Equal(t, "cluster", ti.Source)
	assert.Equal(t, "15.4.x-scala2.12", ti.SparkVersion)
	assert.Equal(t, "dbr/15.4.x-scala2.12", ti.EnvKey)
	assert.Equal(t, "abc", ti.ClusterID)
}

func TestResolveClusterFlagError(t *testing.T) {
	c := stubCompute{clusterErr: errors.New("cluster not found")}
	_, err := ResolveCompute(t.Context(), ComputeFlags{Cluster: "abc"}, c, BundleTarget{})
	var pe *PipelineError
	require.ErrorAs(t, err, &pe)
	assert.Equal(t, ErrResolve, pe.Code)
}

func TestResolveClusterNameFlag(t *testing.T) {
	// --cluster-name resolves to an ID via the Clusters API, then behaves like
	// --cluster-id: source=cluster, the resolved ID is reported, and the env key
	// derives from the resolved cluster's Spark version.
	c := stubCompute{byNameID: "cid-123", byNameVersion: "15.4.x-scala2.12"}
	ti, err := ResolveCompute(t.Context(), ComputeFlags{ClusterName: "my-cluster"}, c, BundleTarget{})
	require.NoError(t, err)
	assert.Equal(t, "cluster", ti.Source)
	assert.Equal(t, "cid-123", ti.ClusterID)
	assert.Equal(t, "15.4.x-scala2.12", ti.SparkVersion)
	assert.Equal(t, "dbr/15.4.x-scala2.12", ti.EnvKey)
}

func TestResolveClusterNameFlagError(t *testing.T) {
	// An unknown or ambiguous name surfaces as E_RESOLVE.
	c := stubCompute{byNameErr: errors.New("there are 2 instances of ClusterDetails named 'dup'")}
	_, err := ResolveCompute(t.Context(), ComputeFlags{ClusterName: "dup"}, c, BundleTarget{})
	var pe *PipelineError
	require.ErrorAs(t, err, &pe)
	assert.Equal(t, ErrResolve, pe.Code)
}

func TestResolveClusterIdAndNameMutuallyExclusive(t *testing.T) {
	// The library path rejects setting both --cluster-id and --cluster-name.
	_, err := ResolveCompute(t.Context(), ComputeFlags{Cluster: "abc", ClusterName: "xyz"}, stubCompute{}, BundleTarget{})
	var pe *PipelineError
	require.ErrorAs(t, err, &pe)
	assert.Equal(t, ErrResolve, pe.Code)
}

func TestResolveBundleNothingSelected(t *testing.T) {
	_, err := ResolveCompute(t.Context(), ComputeFlags{}, stubCompute{}, BundleTarget{Selected: false})
	var pe *PipelineError
	require.ErrorAs(t, err, &pe)
	assert.Equal(t, ErrNoTarget, pe.Code)
}

func TestResolveBundleServerless(t *testing.T) {
	// The bundle records serverless but no version, so the default stand-in applies.
	ti, err := ResolveCompute(t.Context(), ComputeFlags{}, stubCompute{}, BundleTarget{Selected: true, Serverless: true})
	require.NoError(t, err)
	assert.Equal(t, "bundle", ti.Source)
	// Concrete literal, not "serverless-"+defaultServerlessVersion: the default
	// is v5, and asserting the constant against itself would pass for any value.
	assert.Equal(t, "serverless/serverless-v5", ti.EnvKey)
}

// jobStubCompute stubs JobTaskEnvironment. It records the jobID and taskKey it
// was called with (so tests can assert the "<job-id>.<task-key>" split) and
// returns configurable results, including an error for the bare-job / unknown-key
// paths. The sparkVersion/version fields are distinct so the classic branch can
// be checked against the contract (it must use the Spark-version return).
type jobStubCompute struct {
	sparkVersion string
	isServerless bool
	version      string
	err          error

	gotJobID   string
	gotTaskKey string
}

func (*jobStubCompute) GetClusterSparkVersion(_ context.Context, _ string) (string, error) {
	return "", nil
}

func (*jobStubCompute) GetClusterByName(_ context.Context, _ string) (string, string, error) {
	return "", "", nil
}

func (s *jobStubCompute) JobTaskEnvironment(_ context.Context, jobID, taskKey string) (string, bool, string, error) {
	s.gotJobID = jobID
	s.gotTaskKey = taskKey
	if s.err != nil {
		return "", false, "", s.err
	}
	return s.sparkVersion, s.isServerless, s.version, nil
}

func TestResolveJobTaskClassicUsesSparkVersion(t *testing.T) {
	// A classic task resolves to its cluster's Spark version → dbr/ env key.
	c := &jobStubCompute{sparkVersion: "15.4.x-scala2.12"}
	ti, err := ResolveCompute(t.Context(), ComputeFlags{JobTask: "42.ingest"}, c, BundleTarget{})
	require.NoError(t, err)
	assert.Equal(t, "job", ti.Source)
	assert.Equal(t, "15.4.x-scala2.12", ti.SparkVersion)
	assert.Equal(t, "dbr/15.4.x-scala2.12", ti.EnvKey)
	// The value is split on the first dot into job ID and task key.
	assert.Equal(t, "42", c.gotJobID)
	assert.Equal(t, "ingest", c.gotTaskKey)
}

func TestResolveJobTaskSplitsOnFirstDot(t *testing.T) {
	// Task keys may themselves contain dots, so only the first dot separates the
	// job ID from the task key.
	c := &jobStubCompute{sparkVersion: "15.4.x-scala2.12"}
	_, err := ResolveCompute(t.Context(), ComputeFlags{JobTask: "42.a.b.c"}, c, BundleTarget{})
	require.NoError(t, err)
	assert.Equal(t, "42", c.gotJobID)
	assert.Equal(t, "a.b.c", c.gotTaskKey)
}

func TestResolveJobTaskServerlessUsesTaskVersion(t *testing.T) {
	// A serverless task's environment version is read directly (no fallback):
	// it maps to the matching serverless-vN, not the classic dbr path.
	c := &jobStubCompute{isServerless: true, version: "3"}
	ti, err := ResolveCompute(t.Context(), ComputeFlags{JobTask: "42.transform"}, c, BundleTarget{})
	require.NoError(t, err)
	assert.Equal(t, "job", ti.Source)
	assert.Empty(t, ti.SparkVersion)
	assert.Equal(t, "serverless/serverless-v3", ti.EnvKey)
}

func TestResolveJobTaskMissingKeyIsUsageError(t *testing.T) {
	// A bare "<job-id>" (no task key) is E_USAGE, not E_RESOLVE: the user must
	// pick a task. The stub reports the enumerate request via ErrTaskKeyRequired.
	c := &jobStubCompute{err: &ErrTaskKeyRequired{JobID: "42", TaskKeys: []string{"ingest", "transform"}}}
	_, err := ResolveCompute(t.Context(), ComputeFlags{JobTask: "42"}, c, BundleTarget{})
	var pe *PipelineError
	require.ErrorAs(t, err, &pe)
	assert.Equal(t, ErrUsage, pe.Code)
	assert.Empty(t, c.gotTaskKey, "a bare job ID yields an empty task key")
}

func TestResolveJobTaskUnknownKeyIsResolveError(t *testing.T) {
	// An unknown task key is a genuine resolve failure (E_RESOLVE), distinct from
	// the missing-key usage error above.
	c := &jobStubCompute{err: errors.New(`job 42 has no task "nope" (available: ingest)`)}
	_, err := ResolveCompute(t.Context(), ComputeFlags{JobTask: "42.nope"}, c, BundleTarget{})
	var pe *PipelineError
	require.ErrorAs(t, err, &pe)
	assert.Equal(t, ErrResolve, pe.Code)
}

func TestValidateComputeFlagsMutuallyExclusive(t *testing.T) {
	assert.Error(t, ValidateComputeFlags(ComputeFlags{Cluster: "a", Serverless: "v4"}))
	assert.NoError(t, ValidateComputeFlags(ComputeFlags{Cluster: "a"}))
}

func TestResolveComputeRejectsConflictingFlags(t *testing.T) {
	// ResolveCompute must reject incompatible flags rather than silently taking
	// the first precedence branch, so a library caller bypassing Cobra can't
	// resolve a different target than it asked for.
	_, err := ResolveCompute(t.Context(), ComputeFlags{Cluster: "c", Serverless: "v4"}, stubCompute{}, BundleTarget{})
	var pe *PipelineError
	require.ErrorAs(t, err, &pe)
	assert.Equal(t, ErrResolve, pe.Code)
}
