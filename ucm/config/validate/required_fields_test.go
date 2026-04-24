package validate_test

import (
	"strings"
	"testing"

	"github.com/databricks/cli/libs/diag"
	"github.com/databricks/cli/ucm"
	"github.com/databricks/cli/ucm/config"
	"github.com/databricks/cli/ucm/config/validate"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// loadUcm parses raw YAML and returns a fresh Ucm backed by it.
func loadUcm(t *testing.T, yaml string) *ucm.Ucm {
	t.Helper()
	cfg, diags := config.LoadFromBytes("/test/ucm.yml", []byte(yaml))
	require.NoError(t, diags.Error())
	return &ucm.Ucm{Config: *cfg}
}

func summaries(ds diag.Diagnostics) []string {
	out := make([]string, 0, len(ds))
	for _, d := range ds {
		out = append(out, d.Summary)
	}
	return out
}

func hasSummary(ds diag.Diagnostics, substr string) bool {
	for _, s := range summaries(ds) {
		if strings.Contains(s, substr) {
			return true
		}
	}
	return false
}

func TestRequiredFields(t *testing.T) {
	tests := []struct {
		name          string
		yaml          string
		wantSummaries []string
		wantEmpty     bool
	}{
		{
			name: "valid fixture has no diagnostics",
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
			name: "catalog missing name",
			yaml: `
ucm: {name: t}
resources:
  catalogs:
    c1: {}
`,
			wantSummaries: []string{`catalog "c1": required field "name"`},
		},
		{
			name: "schema missing name and catalog",
			yaml: `
ucm: {name: t}
resources:
  schemas:
    s1: {}
`,
			wantSummaries: []string{
				`schema "s1": required field "name"`,
				`schema "s1": required field "catalog"`,
			},
		},
		{
			name: "grant missing principal and privileges",
			yaml: `
ucm: {name: t}
resources:
  grants:
    g1:
      securable: {type: catalog, name: c1}
`,
			wantSummaries: []string{
				`grant "g1": required field "principal"`,
				`grant "g1": required field "privileges"`,
			},
		},
		{
			name: "grant missing securable fields",
			yaml: `
ucm: {name: t}
resources:
  grants:
    g1: {principal: p, privileges: [SELECT]}
`,
			wantSummaries: []string{
				`required field "securable.type"`,
				`required field "securable.name"`,
			},
		},
		{
			name: "storage credential missing identity",
			yaml: `
ucm: {name: t}
resources:
  storage_credentials:
    sc1: {name: sc1}
`,
			wantSummaries: []string{`exactly one of aws_iam_role`},
		},
		{
			name: "storage credential with two identities",
			yaml: `
ucm: {name: t}
resources:
  storage_credentials:
    sc1:
      name: sc1
      aws_iam_role: {role_arn: arn}
      azure_managed_identity: {access_connector_id: id}
`,
			wantSummaries: []string{`exactly one of aws_iam_role`},
		},
		{
			name: "external location missing url and credential_name",
			yaml: `
ucm: {name: t}
resources:
  external_locations:
    el1: {name: el1}
`,
			wantSummaries: []string{
				`required field "url"`,
				`required field "credential_name"`,
			},
		},
		{
			name: "volume missing volume_type",
			yaml: `
ucm: {name: t}
resources:
  volumes:
    v1: {name: v1, catalog_name: c, schema_name: s}
`,
			wantSummaries: []string{`required field "volume_type"`},
		},
		{
			name: "connection missing connection_type and options",
			yaml: `
ucm: {name: t}
resources:
  connections:
    cn1: {name: cn1}
`,
			wantSummaries: []string{
				`required field "connection_type"`,
				`required field "options"`,
			},
		},
	}
	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			u := loadUcm(t, tc.yaml)
			diags := ucm.Apply(t.Context(), u, validate.RequiredFields())
			if tc.wantEmpty {
				assert.Empty(t, diags, "summaries=%v", summaries(diags))
				return
			}
			require.NotEmpty(t, diags)
			for _, want := range tc.wantSummaries {
				assert.True(t, hasSummary(diags, want),
					"want summary containing %q, got %v", want, summaries(diags))
			}
			for _, d := range diags {
				assert.Equal(t, diag.Error, d.Severity)
			}
		})
	}
}
