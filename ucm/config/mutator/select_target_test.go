package mutator_test

import (
	"testing"

	"github.com/databricks/cli/ucm"
	"github.com/databricks/cli/ucm/config"
	"github.com/databricks/cli/ucm/config/mutator"
	"github.com/databricks/cli/ucm/config/resources"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestSelectTarget_MergesOverrides(t *testing.T) {
	_, diags := config.LoadFromBytes("/test/ucm.yml", []byte(`
ucm:
  name: acme
workspace:
  host: https://default.example.com
resources:
  catalogs:
    base:
      name: base
targets:
  dev:
    workspace:
      host: https://dev.example.com
    resources:
      catalogs:
        dev_only:
          name: dev_only
`))
	require.NoError(t, diags.Error())

	u := &ucm.Ucm{}
	cfg, diags := config.LoadFromBytes("/test/ucm.yml", []byte(`
ucm:
  name: acme
workspace:
  host: https://default.example.com
resources:
  catalogs:
    base:
      name: base
targets:
  dev:
    workspace:
      host: https://dev.example.com
    resources:
      catalogs:
        dev_only:
          name: dev_only
`))
	require.NoError(t, diags.Error())
	u.Config = *cfg

	diags = ucm.Apply(t.Context(), u, mutator.SelectTarget("dev"))
	require.NoError(t, diags.Error())

	assert.Equal(t, "dev", u.Config.Ucm.Target)
	assert.Equal(t, "https://dev.example.com", u.Config.Workspace.Host)
	assert.Contains(t, u.Config.Resources.Catalogs, "base")
	assert.Contains(t, u.Config.Resources.Catalogs, "dev_only")
	assert.Nil(t, u.Config.Targets) // cleared post-merge
}

func TestSelectTarget_MissingTargetFails(t *testing.T) {
	u := &ucm.Ucm{
		Config: config.Root{
			Targets: map[string]*config.Target{"dev": {}},
		},
	}
	diags := ucm.Apply(t.Context(), u, mutator.SelectTarget("prod"))
	require.Error(t, diags.Error())
	assert.Contains(t, diags.Error().Error(), "no such target")
}

func TestSelectTarget_NoTargetsFails(t *testing.T) {
	u := &ucm.Ucm{}
	diags := ucm.Apply(t.Context(), u, mutator.SelectTarget("dev"))
	require.Error(t, diags.Error())
}

// Guard that u.Target is populated after the merge (value equality — pointer
// identity is lost because the dyn/typed round-trip during MarkMutatorEntry
// rebuilds the map).
func TestSelectTarget_RecordsTargetSnapshot(t *testing.T) {
	u := &ucm.Ucm{
		Config: config.Root{
			Ucm: config.Ucm{Name: "acme"},
			Resources: config.Resources{
				Catalogs: map[string]*resources.Catalog{"base": {Name: "base"}},
			},
			Targets: map[string]*config.Target{"dev": {Default: true}},
		},
	}
	diags := ucm.Apply(t.Context(), u, mutator.SelectTarget("dev"))
	require.NoError(t, diags.Error())

	require.NotNil(t, u.Target)
	assert.True(t, u.Target.Default)
}
