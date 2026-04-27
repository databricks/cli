package mutator_test

import (
	"testing"

	"github.com/databricks/cli/libs/logdiag"
	"github.com/databricks/cli/ucm/config/mutator"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestDefaultMutators_RunsChainAndAppliesSideEffects(t *testing.T) {
	u := loadUcm(t, `
ucm:
  name: acme
resources:
  catalogs:
    c1:
      name: c1
      tags:
        owner: alpha
      schemas:
        s1: {name: s1}
`)

	ctx := logdiag.InitContext(t.Context())
	mutator.DefaultMutators(ctx, u)
	require.False(t, logdiag.HasError(ctx))

	// FlattenNestedResources unrolled schemas and cleared the nested map.
	got := u.Config.Resources.Schemas["s1"]
	require.NotNil(t, got)
	assert.Equal(t, "c1", got.CatalogName)
	assert.Nil(t, u.Config.Resources.Catalogs["c1"].Schemas)

	// InheritCatalogTags propagated parent tags down to the schema.
	assert.Equal(t, "alpha", got.Tags["owner"])

	// DefineDefaultTarget added the "default" target since none were declared.
	_, ok := u.Config.Targets["default"]
	assert.True(t, ok)
}
