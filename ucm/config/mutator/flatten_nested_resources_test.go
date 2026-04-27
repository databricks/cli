package mutator_test

import (
	"strings"
	"testing"

	"github.com/databricks/cli/ucm"
	"github.com/databricks/cli/ucm/config/mutator"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestFlattenNestedResources(t *testing.T) {
	tests := []struct {
		name          string
		yaml          string
		wantDiagSubs  []string
		wantSchema    map[string]string // schemaName → expected catalog
		wantGrantKind map[string]string // grantName → expected securable kind
		wantGrantName map[string]string // grantName → expected securable name
	}{
		{
			name: "bare nested schema injects catalog",
			yaml: `
ucm: {name: t}
resources:
  catalogs:
    c1:
      name: c1
      schemas:
        s1: {name: s1}
`,
			wantSchema: map[string]string{"s1": "c1"},
		},
		{
			name: "nested schema with matching catalog does not error",
			yaml: `
ucm: {name: t}
resources:
  catalogs:
    c1:
      name: c1
      schemas:
        s1: {name: s1, catalog_name: c1}
`,
			wantSchema: map[string]string{"s1": "c1"},
		},
		{
			name: "nested schema with conflicting catalog errors",
			yaml: `
ucm: {name: t}
resources:
  catalogs:
    c1:
      name: c1
      schemas:
        s1: {name: s1, catalog_name: other}
`,
			wantDiagSubs: []string{"conflicts with parent"},
			wantSchema:   map[string]string{"s1": "other"},
		},
		{
			name: "nested grant at catalog level injects securable",
			yaml: `
ucm: {name: t}
resources:
  catalogs:
    c1:
      name: c1
      grants:
        g1: {principal: p, privileges: [USE_CATALOG]}
`,
			wantGrantKind: map[string]string{"g1": "catalog"},
			wantGrantName: map[string]string{"g1": "c1"},
		},
		{
			name: "nested grant at schema level injects securable",
			yaml: `
ucm: {name: t}
resources:
  catalogs:
    c1:
      name: c1
      schemas:
        s1:
          name: s1
          grants:
            g1: {principal: p, privileges: [SELECT]}
`,
			wantGrantKind: map[string]string{"g1": "schema"},
			wantGrantName: map[string]string{"g1": "s1"},
			wantSchema:    map[string]string{"s1": "c1"},
		},
		{
			name: "flat-vs-nested schema collision errors",
			yaml: `
ucm: {name: t}
resources:
  schemas:
    s1: {name: s1, catalog_name: c1}
  catalogs:
    c1:
      name: c1
      schemas:
        s1: {name: s1}
`,
			wantDiagSubs: []string{"declared both as a flat entry and nested"},
		},
	}
	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			u := loadUcm(t, tc.yaml)
			diags := ucm.Apply(t.Context(), u, mutator.FlattenNestedResources())

			for _, sub := range tc.wantDiagSubs {
				found := false
				for _, s := range summaries(diags) {
					if strings.Contains(s, sub) {
						found = true
						break
					}
				}
				assert.True(t, found, "expected diag containing %q, got %v", sub, summaries(diags))
			}
			if len(tc.wantDiagSubs) == 0 {
				require.Empty(t, diags, "unexpected diags: %v", summaries(diags))
			}

			for name, want := range tc.wantSchema {
				got := u.Config.Resources.Schemas[name]
				require.NotNil(t, got, "schema %q missing", name)
				assert.Equal(t, want, got.CatalogName, "schema %q catalog", name)
			}
			for name, want := range tc.wantGrantKind {
				got := u.Config.Resources.Grants[name]
				require.NotNil(t, got, "grant %q missing", name)
				assert.Equal(t, want, got.Securable.Type)
			}
			for name, want := range tc.wantGrantName {
				got := u.Config.Resources.Grants[name]
				require.NotNil(t, got, "grant %q missing", name)
				assert.Equal(t, want, got.Securable.Name)
			}

			// Nested maps must be cleared after flatten.
			for _, c := range u.Config.Resources.Catalogs {
				assert.Nil(t, c.Schemas, "catalog.Schemas should be cleared")
				assert.Nil(t, c.Grants, "catalog.Grants should be cleared")
			}
			for _, s := range u.Config.Resources.Schemas {
				assert.Nil(t, s.Grants, "schema.Grants should be cleared")
			}
		})
	}
}
