package mutator_test

import (
	"testing"

	"github.com/databricks/cli/ucm"
	"github.com/databricks/cli/ucm/config/engine"
	"github.com/databricks/cli/ucm/config/mutator"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestValidateDirectOnlyResourcesDirectEngineNoDiags(t *testing.T) {
	u := loadUcm(t, `
ucm:
  name: acme
resources:
  catalogs:
    c1:
      name: c1
  external_locations:
    e1:
      name: e1
      url: s3://bucket/path
      credential_name: cred
`)
	diags := ucm.Apply(t.Context(), u, mutator.ValidateDirectOnlyResources(engine.EngineDirect))
	require.NoError(t, diags.Error())
	assert.Empty(t, diags)
}

func TestValidateDirectOnlyResourcesTerraformEngineNoDiags(t *testing.T) {
	u := loadUcm(t, `
ucm:
  name: acme
resources:
  catalogs:
    c1:
      name: c1
  external_locations:
    e1:
      name: e1
      url: s3://bucket/path
      credential_name: cred
`)
	diags := ucm.Apply(t.Context(), u, mutator.ValidateDirectOnlyResources(engine.EngineTerraform))
	require.NoError(t, diags.Error())
	assert.Empty(t, diags)
}

func TestValidateDirectOnlyResourcesEmptyConfigNoDiags(t *testing.T) {
	u := loadUcm(t, `
ucm:
  name: acme
`)
	diags := ucm.Apply(t.Context(), u, mutator.ValidateDirectOnlyResources(engine.EngineTerraform))
	require.NoError(t, diags.Error())
	assert.Empty(t, diags)
}
