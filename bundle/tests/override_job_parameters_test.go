package config_tests

import (
	"testing"

	"github.com/stretchr/testify/assert"
)

func TestOverrideJobParametersDev(t *testing.T) {
	b := loadTarget(t, "./override_job_parameters", "development")
	assert.Equal(t, "job", b.Config.Resources.Jobs["foo"].Name)

	p := b.Config.Resources.Jobs["foo"].Parameters
	assert.Len(t, p, 2)
	assert.Equal(t, "foo", p[0].Name)
	assert.Equal(t, "v2", p[0].Default)
	assert.Equal(t, "bar", p[1].Name)
	assert.Equal(t, "v1", p[1].Default)
}

func TestOverrideJobParametersStaging(t *testing.T) {
	b := loadTarget(t, "./override_job_parameters", "staging")
	assert.Equal(t, "job", b.Config.Resources.Jobs["foo"].Name)

	p := b.Config.Resources.Jobs["foo"].Parameters
	assert.Len(t, p, 2)
	assert.Equal(t, "foo", p[0].Name)
	assert.Equal(t, "v1", p[0].Default)
	assert.Equal(t, "bar", p[1].Name)
	assert.Equal(t, "v2", p[1].Default)
}
