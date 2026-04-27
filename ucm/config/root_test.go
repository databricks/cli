package config_test

import (
	"testing"

	"github.com/databricks/cli/ucm/config"
	"github.com/databricks/cli/ucm/config/variable"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestLoadFromBytes_MinimalUcm(t *testing.T) {
	cfg, diags := config.LoadFromBytes("/test/ucm.yml", []byte(`
ucm:
  name: acme
`))
	require.NoError(t, diags.Error())
	assert.Equal(t, "acme", cfg.Ucm.Name)
}

func TestLoadFromBytes_FullM0Surface(t *testing.T) {
	cfg, diags := config.LoadFromBytes("/test/ucm.yml", []byte(`
ucm:
  name: acme

workspace:
  host: https://example.cloud.databricks.com

account:
  account_id: 0000-0000
  host: https://accounts.cloud.databricks.com

resources:
  catalogs:
    team_alpha:
      name: team_alpha
      comment: alpha catalog
      tags:
        cost_center: "1234"
  schemas:
    bronze:
      catalog_name: team_alpha
      name: bronze
  grants:
    reads:
      securable: { type: catalog, name: team_alpha }
      principal: alpha-readers
      privileges: [USE_CATALOG, SELECT]
  tag_validation_rules:
    r:
      securable_types: [catalog]
      required: [cost_center]

targets:
  dev:
    default: true
  prod:
    workspace:
      host: https://prod.example.com
`))
	require.NoError(t, diags.Error())

	assert.Equal(t, "https://example.cloud.databricks.com", cfg.Workspace.Host)
	assert.Equal(t, "0000-0000", cfg.Account.AccountID)

	require.Contains(t, cfg.Resources.Catalogs, "team_alpha")
	assert.Equal(t, "team_alpha", cfg.Resources.Catalogs["team_alpha"].Name)
	assert.Equal(t, "1234", cfg.Resources.Catalogs["team_alpha"].Tags["cost_center"])

	require.Contains(t, cfg.Resources.Schemas, "bronze")
	assert.Equal(t, "team_alpha", cfg.Resources.Schemas["bronze"].CatalogName)

	require.Contains(t, cfg.Resources.Grants, "reads")
	assert.Equal(t, "catalog", cfg.Resources.Grants["reads"].Securable.Type)
	assert.Equal(t, []string{"USE_CATALOG", "SELECT"}, cfg.Resources.Grants["reads"].Privileges)

	require.Contains(t, cfg.Resources.TagValidationRules, "r")

	require.Contains(t, cfg.Targets, "dev")
	assert.True(t, cfg.Targets["dev"].Default)
}

func TestLoadFromBytes_InvalidYAML(t *testing.T) {
	_, diags := config.LoadFromBytes("/test/ucm.yml", []byte("this:\n  is: [unterminated"))
	require.Error(t, diags.Error())
}

func TestMergeTargetOverrides_WorkspaceHost(t *testing.T) {
	cfg, diags := config.LoadFromBytes("/test/ucm.yml", []byte(`
ucm:
  name: acme
workspace:
  host: https://base.example.com
targets:
  prod:
    workspace:
      host: https://prod.example.com
`))
	require.NoError(t, diags.Error())

	require.NoError(t, cfg.MergeTargetOverrides("prod"))
	assert.Equal(t, "https://prod.example.com", cfg.Workspace.Host)
}

func TestMergeTargetOverrides_VariablesDefault(t *testing.T) {
	cfg, diags := config.LoadFromBytes("/test/ucm.yml", []byte(`
ucm:
  name: acme
variables:
  catalog_name:
    description: UC catalog root
    default: team_alpha
targets:
  prod:
    variables:
      catalog_name:
        default: team_prod
`))
	require.NoError(t, diags.Error())

	require.NoError(t, cfg.MergeTargetOverrides("prod"))
	require.Contains(t, cfg.Variables, "catalog_name")
	assert.Equal(t, "team_prod", cfg.Variables["catalog_name"].Default)
}

func TestInitializeVariables_AssignsValues(t *testing.T) {
	cfg, diags := config.LoadFromBytes("/test/ucm.yml", []byte(`
ucm:
  name: acme
variables:
  foo:
    default: d
  bar:
    default: d2
`))
	require.NoError(t, diags.Error())

	require.NoError(t, cfg.InitializeVariables([]string{"foo=123", "bar=456"}))
	assert.Equal(t, "123", cfg.Variables["foo"].Value)
	assert.Equal(t, "456", cfg.Variables["bar"].Value)
}

func TestInitializeVariables_EqualSignInValue(t *testing.T) {
	cfg, diags := config.LoadFromBytes("/test/ucm.yml", []byte(`
ucm:
  name: acme
variables:
  foo: {}
`))
	require.NoError(t, diags.Error())

	require.NoError(t, cfg.InitializeVariables([]string{"foo=123=567"}))
	assert.Equal(t, "123=567", cfg.Variables["foo"].Value)
}

func TestInitializeVariables_InvalidFormat(t *testing.T) {
	cfg := &config.Root{Variables: map[string]*variable.Variable{"foo": {}}}
	err := cfg.InitializeVariables([]string{"foo"})
	assert.ErrorContains(t, err, "unexpected flag value")
}

func TestInitializeVariables_Undefined(t *testing.T) {
	cfg := &config.Root{Variables: map[string]*variable.Variable{"foo": {}}}
	err := cfg.InitializeVariables([]string{"bar=567"})
	assert.ErrorContains(t, err, "variable bar has not been defined")
}

func TestInitializeVariables_ComplexRejected(t *testing.T) {
	cfg := &config.Root{
		Variables: map[string]*variable.Variable{
			"foo": {Type: variable.VariableTypeComplex},
		},
	}
	err := cfg.InitializeVariables([]string{"foo=bar"})
	assert.ErrorContains(t, err, "setting variables of complex type via --var flag is not supported")
}
