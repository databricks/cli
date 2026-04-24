package mutator_test

import (
	"strings"
	"testing"

	"github.com/databricks/cli/ucm"
	"github.com/databricks/cli/ucm/config/mutator"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestResolveVariableReferencesOnlyResources_ResolvesInsideResources(t *testing.T) {
	u := loadUcm(t, `
ucm: {name: t}
resources:
  storage_credentials:
    sales_cred:
      name: sales_cred
      aws_iam_role:
        role_arn: arn:aws:iam::1:role/uc
  catalogs:
    sales:
      name: sales_prod
      storage_root: ${resources.storage_credentials.sales_cred.name}
`)
	diags := ucm.Apply(t.Context(), u, mutator.ResolveVariableReferencesOnlyResources("resources"))
	require.Empty(t, diags, "unexpected diags: %v", summaries(diags))
	assert.Equal(t, "sales_cred", u.Config.Resources.Catalogs["sales"].StorageRoot)
}

func TestResolveVariableReferencesOnlyResources_IgnoresNonResourceSubtree(t *testing.T) {
	u := loadUcm(t, `
ucm:
  name: t
workspace:
  host: ${ucm.name}
resources:
  catalogs:
    c1:
      name: ${ucm.name}
`)
	diags := ucm.Apply(t.Context(), u, mutator.ResolveVariableReferencesOnlyResources())
	require.Empty(t, diags, "unexpected diags: %v", summaries(diags))
	// Inside resources, the ${ucm.name} ref is resolved.
	assert.Equal(t, "t", u.Config.Resources.Catalogs["c1"].Name)
	// Outside resources, the same ref stays as-is.
	assert.Equal(t, "${ucm.name}", u.Config.Workspace.Host)
}

func TestResolveVariableReferencesOnlyResources_LeavesNonMatchingPrefixAlone(t *testing.T) {
	u := loadUcm(t, `
ucm: {name: t}
resources:
  catalogs:
    c1:
      name: c1
      storage_root: ${var.some_future_var}
`)
	diags := ucm.Apply(t.Context(), u, mutator.ResolveVariableReferencesOnlyResources("resources"))
	require.Empty(t, diags, "unexpected diags: %v", summaries(diags))
	assert.Equal(t, "${var.some_future_var}", u.Config.Resources.Catalogs["c1"].StorageRoot)
}

func TestResolveVariableReferencesOnlyResources_UnknownRefErrors(t *testing.T) {
	u := loadUcm(t, `
ucm: {name: t}
resources:
  catalogs:
    sales:
      name: sales_prod
      storage_root: ${resources.storage_credentials.missing.name}
`)
	diags := ucm.Apply(t.Context(), u, mutator.ResolveVariableReferencesOnlyResources("resources"))
	require.NotEmpty(t, diags, "expected error diagnostic")
	assert.True(t, anyContains(summaries(diags), "resources.storage_credentials.missing"),
		"expected diag to mention the missing ref, got %v", summaries(diags))
}

func TestResolveVariableReferencesWithoutResources_ResolvesOutsideResources(t *testing.T) {
	u := loadUcm(t, `
ucm:
  name: t
workspace:
  host: ${ucm.name}
resources:
  catalogs:
    c1:
      name: ${ucm.name}
`)
	diags := ucm.Apply(t.Context(), u, mutator.ResolveVariableReferencesWithoutResources())
	require.Empty(t, diags, "unexpected diags: %v", summaries(diags))
	// Outside resources, the ref is resolved.
	assert.Equal(t, "t", u.Config.Workspace.Host)
	// Inside resources, the ref stays.
	assert.Equal(t, "${ucm.name}", u.Config.Resources.Catalogs["c1"].Name)
}

func TestResolveVariableReferencesWithoutResources_RespectsExplicitPrefixList(t *testing.T) {
	u := loadUcm(t, `
ucm:
  name: t
variables:
  team:
    default: alpha
    value: alpha
workspace:
  host: ${variables.team.value}
  profile: ${ucm.name}
`)
	// Only "variables" prefix — ${ucm.name} should be left alone.
	diags := ucm.Apply(t.Context(), u, mutator.ResolveVariableReferencesWithoutResources("variables"))
	require.Empty(t, diags, "unexpected diags: %v", summaries(diags))
	assert.Equal(t, "alpha", u.Config.Workspace.Host)
	assert.Equal(t, "${ucm.name}", u.Config.Workspace.Profile)
}

func TestResolveVariableReferencesWithoutResources_SubstitutesVarShorthand(t *testing.T) {
	u := loadUcm(t, `
ucm: {name: t}
variables:
  catalog_name:
    default: team_alpha
    value: team_alpha
workspace:
  profile: ${var.catalog_name}
`)
	diags := ucm.Apply(t.Context(), u, mutator.ResolveVariableReferencesWithoutResources("variables"))
	require.Empty(t, diags, "unexpected diags: %v", summaries(diags))
	assert.Equal(t, "team_alpha", u.Config.Workspace.Profile)
}

func TestResolveVariableReferencesWithoutResources_MultiRoundChained(t *testing.T) {
	u := loadUcm(t, `
ucm: {name: t}
variables:
  base:
    default: alpha
    value: alpha
  derived:
    default: ${var.base}_suffix
    value: ${var.base}_suffix
workspace:
  profile: ${var.derived}
`)
	diags := ucm.Apply(t.Context(), u, mutator.ResolveVariableReferencesWithoutResources("variables"))
	require.Empty(t, diags, "unexpected diags: %v", summaries(diags))
	assert.Equal(t, "alpha_suffix", u.Config.Workspace.Profile)
}

func TestResolveVariableReferencesWithoutResources_UnknownVarErrors(t *testing.T) {
	u := loadUcm(t, `
ucm: {name: t}
workspace:
  profile: ${var.missing}
`)
	diags := ucm.Apply(t.Context(), u, mutator.ResolveVariableReferencesWithoutResources("variables"))
	require.NotEmpty(t, diags, "expected error diagnostic")
	assert.True(t, anyContains(summaries(diags), "var.missing"),
		"expected diag to mention missing var, got %v", summaries(diags))
}

func TestResolveVariableReferencesInLookup_ResolvesInsideLookup(t *testing.T) {
	u := loadUcm(t, `
ucm: {name: t}
variables:
  ms_name:
    default: primary
    value: primary
  ms_id:
    lookup:
      metastore: ${var.ms_name}
`)
	diags := ucm.Apply(t.Context(), u, mutator.ResolveVariableReferencesInLookup())
	require.Empty(t, diags, "unexpected diags: %v", summaries(diags))
	assert.Equal(t, "primary", u.Config.Variables["ms_id"].Lookup.Metastore)
}

func TestResolveVariableReferencesInLookup_LeavesOutsideLookupAlone(t *testing.T) {
	u := loadUcm(t, `
ucm: {name: t}
variables:
  team:
    default: alpha
    value: alpha
workspace:
  profile: ${var.team}
resources:
  catalogs:
    c1:
      name: ${var.team}
`)
	diags := ucm.Apply(t.Context(), u, mutator.ResolveVariableReferencesInLookup())
	require.Empty(t, diags, "unexpected diags: %v", summaries(diags))
	// Refs outside a lookup block are left untouched.
	assert.Equal(t, "${var.team}", u.Config.Workspace.Profile)
	assert.Equal(t, "${var.team}", u.Config.Resources.Catalogs["c1"].Name)
}

func TestResolveVariableReferencesInLookup_NestedLookupErrors(t *testing.T) {
	u := loadUcm(t, `
ucm: {name: t}
variables:
  inner:
    lookup:
      metastore: primary
  outer:
    lookup:
      metastore: ${var.inner}
`)
	diags := ucm.Apply(t.Context(), u, mutator.ResolveVariableReferencesInLookup())
	require.NotEmpty(t, diags, "expected error diagnostic")
	assert.True(t, anyContains(summaries(diags), "lookup"),
		"expected diag to mention lookup, got %v", summaries(diags))
}

func TestResolveVariableReferences_ArtifactsRefErrors(t *testing.T) {
	u := loadUcm(t, `
ucm: {name: t}
resources:
  catalogs:
    c1:
      name: ${artifacts.unknown.path}
`)
	diags := ucm.Apply(t.Context(), u, mutator.ResolveVariableReferencesOnlyResources("resources"))
	require.NotEmpty(t, diags, "expected error diagnostic")
	assert.True(t, anyContains(summaries(diags), "artifacts"),
		"expected diag to mention artifacts, got %v", summaries(diags))
}

func anyContains(ss []string, sub string) bool {
	for _, s := range ss {
		if strings.Contains(s, sub) {
			return true
		}
	}
	return false
}
