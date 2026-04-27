package mutator_test

import (
	"testing"

	"github.com/databricks/cli/ucm"
	"github.com/databricks/cli/ucm/config/engine"
	"github.com/databricks/cli/ucm/config/mutator"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestValidateLifecycleStartedDirectEngineNoDiags(t *testing.T) {
	u := loadUcm(t, `
ucm:
  name: acme
resources:
  catalogs:
    c1:
      name: c1
  schemas:
    s1:
      catalog_name: c1
      name: s1
`)
	diags := ucm.Apply(t.Context(), u, mutator.ValidateLifecycleStarted(engine.EngineDirect))
	require.NoError(t, diags.Error())
	assert.Empty(t, diags)
}

func TestValidateLifecycleStartedTerraformEngineNoDiags(t *testing.T) {
	u := loadUcm(t, `
ucm:
  name: acme
resources:
  catalogs:
    c1:
      name: c1
  volumes:
    v1:
      catalog_name: c1
      schema_name: s1
      name: v1
`)
	diags := ucm.Apply(t.Context(), u, mutator.ValidateLifecycleStarted(engine.EngineTerraform))
	require.NoError(t, diags.Error())
	assert.Empty(t, diags)
}

func TestValidateLifecycleStartedEmptyConfigNoDiags(t *testing.T) {
	u := loadUcm(t, `
ucm:
  name: acme
`)
	diags := ucm.Apply(t.Context(), u, mutator.ValidateLifecycleStarted(engine.EngineTerraform))
	require.NoError(t, diags.Error())
	assert.Empty(t, diags)
}
