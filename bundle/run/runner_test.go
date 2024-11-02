package run

import (
	"testing"

	"github.com/databricks/cli/bundle/config/resources"
	refs "github.com/databricks/cli/bundle/resources"
	"github.com/stretchr/testify/assert"
)

func TestRunner_IsRunnable(t *testing.T) {
	assert.True(t, IsRunnable(refs.Reference{Resource: &resources.Job{}}))
	assert.True(t, IsRunnable(refs.Reference{Resource: &resources.Pipeline{}}))
	assert.False(t, IsRunnable(refs.Reference{Resource: &resources.MlflowModel{}}))
	assert.False(t, IsRunnable(refs.Reference{Resource: &resources.MlflowExperiment{}}))
}
