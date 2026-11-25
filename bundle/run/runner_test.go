package run

import (
	"testing"

	"github.com/databricks/cli/bundle/config/resources"
	refs "github.com/databricks/cli/bundle/resources"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestRunner_IsRunnable(t *testing.T) {
	assert.True(t, IsRunnable(refs.Reference{Resource: &resources.Job{}}))
	assert.True(t, IsRunnable(refs.Reference{Resource: &resources.Pipeline{}}))
	assert.True(t, IsRunnable(refs.Reference{Resource: &resources.App{}}))
	assert.False(t, IsRunnable(refs.Reference{Resource: &resources.MlflowModel{}}))
	assert.False(t, IsRunnable(refs.Reference{Resource: &resources.MlflowExperiment{}}))
}

func TestRunner_ToRunner(t *testing.T) {
	runner, err := ToRunner(nil, refs.Reference{KeyWithType: "jobs.my_job", Resource: &resources.Job{}})
	require.NoError(t, err)
	assert.IsType(t, &jobRunner{}, runner)
	assert.Equal(t, "jobs.my_job", runner.Key())

	_, err = ToRunner(nil, refs.Reference{Resource: &resources.MlflowModel{}})
	assert.EqualError(t, err, "unsupported resource type: *resources.MlflowModel")
}
