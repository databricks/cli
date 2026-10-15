package config_tests

import (
	"testing"

	"github.com/databricks/cli/libs/utils"

	"github.com/stretchr/testify/assert"
)

func TestOverrideJobParametersDev(t *testing.T) {
	b := loadTarget(t, "./override_job_parameters", "development")
	assert.Equal(t, "job", utils.Deref(b.Config.Resources.Jobs["foo"].Name))

	p := b.Config.Resources.Jobs["foo"].Parameters
	assert.Len(t, p, 2)
	assert.Equal(t, "foo", utils.Deref(p[0].Name))
	assert.Equal(t, "v2", utils.Deref(p[0].Default))
	assert.Equal(t, "bar", utils.Deref(p[1].Name))
	assert.Equal(t, "v1", utils.Deref(p[1].Default))
}

func TestOverrideJobParametersStaging(t *testing.T) {
	b := loadTarget(t, "./override_job_parameters", "staging")
	assert.Equal(t, "job", utils.Deref(b.Config.Resources.Jobs["foo"].Name))

	p := b.Config.Resources.Jobs["foo"].Parameters
	assert.Len(t, p, 2)
	assert.Equal(t, "foo", utils.Deref(p[0].Name))
	assert.Equal(t, "v1", utils.Deref(p[0].Default))
	assert.Equal(t, "bar", utils.Deref(p[1].Name))
	assert.Equal(t, "v2", utils.Deref(p[1].Default))
}
