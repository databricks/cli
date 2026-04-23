package validate_test

import (
	"testing"

	"github.com/databricks/cli/ucm"
	"github.com/databricks/cli/ucm/config/mutator"
	"github.com/databricks/cli/ucm/config/validate"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestReferenceClosure(t *testing.T) {
	tests := []struct {
		name         string
		yaml         string
		resolveFirst bool
		wantSummary  string
		wantEmpty    bool
	}{
		{
			name: "resolved reference to existing catalog is fine",
			yaml: `
ucm: {name: t}
resources:
  catalogs:
    c1: {name: c1}
  schemas:
    s1: {name: s1, catalog: "${resources.catalogs.c1.name}"}
`,
			resolveFirst: true,
			wantEmpty:    true,
		},
		{
			name: "unresolved reference to declared resource passes (terraform will handle)",
			yaml: `
ucm: {name: t}
resources:
  catalogs:
    c1: {name: c1}
  schemas:
    s1: {name: s1, catalog: "${resources.catalogs.c1.name}"}
`,
			resolveFirst: false,
			wantEmpty:    true,
		},
		{
			name: "reference to undeclared catalog errors",
			yaml: `
ucm: {name: t}
resources:
  catalogs:
    c1: {name: c1}
  schemas:
    s1: {name: s1, catalog: "${resources.catalogs.missing.name}"}
`,
			resolveFirst: false,
			wantSummary:  `${resources.catalogs.missing.name}`,
		},
		{
			name: "reference to undeclared kind errors",
			yaml: `
ucm: {name: t}
resources:
  catalogs:
    c1: {name: c1}
  schemas:
    s1: {name: s1, catalog: "${resources.volumes.nope.name}"}
`,
			resolveFirst: false,
			wantSummary:  `${resources.volumes.nope.name}`,
		},
		{
			name: "non-resource reference ignored",
			yaml: `
ucm: {name: t}
resources:
  catalogs:
    c1:
      name: c1
      comment: "owned by ${var.team}"
`,
			resolveFirst: false,
			wantEmpty:    true,
		},
		{
			name: "pure reference to bare resources namespace ignored",
			yaml: `
ucm: {name: t}
resources:
  catalogs:
    c1:
      name: c1
      comment: "${workspace.host}"
`,
			resolveFirst: false,
			wantEmpty:    true,
		},
	}
	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			u := loadUcm(t, tc.yaml)
			if tc.resolveFirst {
				diags := ucm.Apply(t.Context(), u, mutator.ResolveResourceReferences())
				require.NoError(t, diags.Error())
			}
			diags := ucm.Apply(t.Context(), u, validate.ReferenceClosure())
			if tc.wantEmpty {
				assert.Empty(t, diags, "summaries=%v", summaries(diags))
				return
			}
			require.NotEmpty(t, diags)
			assert.True(t, hasSummary(diags, tc.wantSummary),
				"want summary containing %q, got %v", tc.wantSummary, summaries(diags))
		})
	}
}

func TestReferenceClosure_AfterResolveCatchesDangling(t *testing.T) {
	u := loadUcm(t, `
ucm: {name: t}
resources:
  catalogs:
    c1: {name: c1}
  schemas:
    s1: {name: s1, catalog: "${resources.catalogs.missing.name}"}
`)
	// ResolveResourceReferences will leave the token in place because the
	// target does not exist; the closure check then errors on it.
	_ = ucm.Apply(t.Context(), u, mutator.ResolveResourceReferences())
	diags := ucm.Apply(t.Context(), u, validate.ReferenceClosure())
	require.NotEmpty(t, diags)
	assert.True(t, hasSummary(diags, "missing"))
}
