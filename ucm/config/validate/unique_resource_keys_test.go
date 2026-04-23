package validate_test

import (
	"testing"

	"github.com/databricks/cli/ucm"
	"github.com/databricks/cli/ucm/config/validate"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestUniqueResourceKeys(t *testing.T) {
	tests := []struct {
		name        string
		yaml        string
		wantSummary string
		wantEmpty   bool
	}{
		{
			name: "distinct kinds with distinct keys",
			yaml: `
ucm: {name: t}
resources:
  catalogs:
    c1: {name: c1}
  schemas:
    s1: {name: s1, catalog: c1}
`,
			wantEmpty: true,
		},
		{
			name: "catalog and schema share a key",
			yaml: `
ucm: {name: t}
resources:
  catalogs:
    shared: {name: shared}
  schemas:
    shared: {name: shared, catalog: shared}
`,
			wantSummary: "resource key shared is declared under multiple",
		},
		{
			name: "grant and volume share a key",
			yaml: `
ucm: {name: t}
resources:
  grants:
    foo: {securable: {type: catalog, name: c1}, principal: p, privileges: [SELECT]}
  volumes:
    foo: {name: v, catalog_name: c, schema_name: s, volume_type: MANAGED}
`,
			wantSummary: "resource key foo is declared under multiple",
		},
	}
	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			u := loadUcm(t, tc.yaml)
			diags := ucm.Apply(t.Context(), u, validate.UniqueResourceKeys())
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
