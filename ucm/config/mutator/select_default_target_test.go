package mutator_test

import (
	"testing"

	"github.com/databricks/cli/ucm"
	"github.com/databricks/cli/ucm/config"
	"github.com/databricks/cli/ucm/config/mutator"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestSelectDefaultTarget_SingleTargetIsDefault(t *testing.T) {
	u := &ucm.Ucm{}
	cfg, diags := config.LoadFromBytes("/test/ucm.yml", []byte(`
ucm:
  name: acme
targets:
  dev: {}
`))
	require.NoError(t, diags.Error())
	u.Config = *cfg

	diags = ucm.Apply(t.Context(), u, mutator.SelectDefaultTarget())
	require.NoError(t, diags.Error())
	assert.Equal(t, "dev", u.Config.Ucm.Target)
}

func TestSelectDefaultTarget_PicksMarkedDefault(t *testing.T) {
	u := &ucm.Ucm{}
	cfg, diags := config.LoadFromBytes("/test/ucm.yml", []byte(`
ucm:
  name: acme
targets:
  dev:
    default: true
  prod: {}
`))
	require.NoError(t, diags.Error())
	u.Config = *cfg

	diags = ucm.Apply(t.Context(), u, mutator.SelectDefaultTarget())
	require.NoError(t, diags.Error())
	assert.Equal(t, "dev", u.Config.Ucm.Target)
}

func TestSelectDefaultTarget_MultipleDefaultsFails(t *testing.T) {
	u := &ucm.Ucm{}
	cfg, diags := config.LoadFromBytes("/test/ucm.yml", []byte(`
ucm:
  name: acme
targets:
  dev:
    default: true
  prod:
    default: true
`))
	require.NoError(t, diags.Error())
	u.Config = *cfg

	diags = ucm.Apply(t.Context(), u, mutator.SelectDefaultTarget())
	require.Error(t, diags.Error())
	assert.Contains(t, diags.Error().Error(), "multiple targets are marked as default")
}

func TestSelectDefaultTarget_NoDefaultErrors(t *testing.T) {
	u := &ucm.Ucm{}
	cfg, diags := config.LoadFromBytes("/test/ucm.yml", []byte(`
ucm:
  name: acme
targets:
  dev: {}
  prod: {}
`))
	require.NoError(t, diags.Error())
	u.Config = *cfg

	diags = ucm.Apply(t.Context(), u, mutator.SelectDefaultTarget())
	require.Error(t, diags.Error())
	assert.Contains(t, diags.Error().Error(), "please specify target")
}

func TestSelectDefaultTarget_NoTargetsFails(t *testing.T) {
	u := &ucm.Ucm{}
	diags := ucm.Apply(t.Context(), u, mutator.SelectDefaultTarget())
	require.Error(t, diags.Error())
}
