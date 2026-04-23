package mutator_test

import (
	"strings"
	"testing"

	"github.com/databricks/cli/ucm"
	"github.com/databricks/cli/ucm/config/mutator"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestResolveResourceReferences_LiteralsPassThrough(t *testing.T) {
	u := loadUcm(t, `
ucm: {name: t}
resources:
  catalogs:
    sales:
      name: sales_prod
      storage_root: s3://acme-sales/prod
`)
	diags := ucm.Apply(t.Context(), u, mutator.ResolveResourceReferences())
	require.Empty(t, diags, "unexpected diags: %v", summaries(diags))
	assert.Equal(t, "s3://acme-sales/prod", u.Config.Resources.Catalogs["sales"].StorageRoot)
}

func TestResolveResourceReferences_InterpolatesUcmRef(t *testing.T) {
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
	diags := ucm.Apply(t.Context(), u, mutator.ResolveResourceReferences())
	require.Empty(t, diags, "unexpected diags: %v", summaries(diags))
	assert.Equal(t, "sales_cred", u.Config.Resources.Catalogs["sales"].StorageRoot)
}

func TestResolveResourceReferences_UnknownRefErrors(t *testing.T) {
	u := loadUcm(t, `
ucm: {name: t}
resources:
  catalogs:
    sales:
      name: sales_prod
      storage_root: ${resources.storage_credentials.missing.name}
`)
	diags := ucm.Apply(t.Context(), u, mutator.ResolveResourceReferences())
	require.NotEmpty(t, diags, "expected error diagnostic")
	found := false
	for _, s := range summaries(diags) {
		if strings.Contains(s, "resources.storage_credentials.missing") {
			found = true
			break
		}
	}
	assert.True(t, found, "expected diag to mention the missing ref, got %v", summaries(diags))
}

func TestResolveResourceReferences_LeavesNonResourceRefsUntouched(t *testing.T) {
	u := loadUcm(t, `
ucm: {name: t}
resources:
  catalogs:
    sales:
      name: sales_prod
      storage_root: ${var.some_future_var}
`)
	diags := ucm.Apply(t.Context(), u, mutator.ResolveResourceReferences())
	require.Empty(t, diags, "unexpected diags: %v", summaries(diags))
	assert.Equal(t, "${var.some_future_var}", u.Config.Resources.Catalogs["sales"].StorageRoot)
}

func TestResolveVariableReferences_SubstitutesVarShorthand(t *testing.T) {
	u := loadUcm(t, `
ucm: {name: t}
variables:
  catalog_name:
    default: team_alpha
    value: team_alpha
resources:
  catalogs:
    primary:
      name: ${var.catalog_name}
`)
	diags := ucm.Apply(t.Context(), u, mutator.ResolveVariableReferences())
	require.Empty(t, diags, "unexpected diags: %v", summaries(diags))
	assert.Equal(t, "team_alpha", u.Config.Resources.Catalogs["primary"].Name)
}

func TestResolveVariableReferences_ChainedRefs(t *testing.T) {
	u := loadUcm(t, `
ucm: {name: t}
variables:
  base:
    default: alpha
    value: alpha
  derived:
    default: ${var.base}_suffix
    value: ${var.base}_suffix
resources:
  catalogs:
    primary:
      name: ${var.derived}
`)
	diags := ucm.Apply(t.Context(), u, mutator.ResolveVariableReferences())
	require.Empty(t, diags, "unexpected diags: %v", summaries(diags))
	assert.Equal(t, "alpha_suffix", u.Config.Resources.Catalogs["primary"].Name)
}

func TestResolveVariableReferences_UnknownErrors(t *testing.T) {
	u := loadUcm(t, `
ucm: {name: t}
resources:
  catalogs:
    primary:
      name: ${var.missing}
`)
	diags := ucm.Apply(t.Context(), u, mutator.ResolveVariableReferences())
	require.NotEmpty(t, diags, "expected error diagnostic")
	found := false
	for _, s := range summaries(diags) {
		if strings.Contains(s, "var.missing") {
			found = true
			break
		}
	}
	assert.True(t, found, "expected diag to mention missing var, got %v", summaries(diags))
}
