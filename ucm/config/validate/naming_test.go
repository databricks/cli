package validate_test

import (
	"testing"

	"github.com/databricks/cli/ucm"
	"github.com/databricks/cli/ucm/config/validate"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestNaming(t *testing.T) {
	tests := []struct {
		name        string
		yaml        string
		wantSummary string
		wantEmpty   bool
	}{
		{
			name: "valid key and name",
			yaml: `
ucm: {name: t}
resources:
  catalogs:
    team_alpha: {name: team_alpha}
`,
			wantEmpty: true,
		},
		{
			name: "key with dot rejected",
			yaml: `
ucm: {name: t}
resources:
  catalogs:
    "team.alpha": {name: team_alpha}
`,
			wantSummary: `resource key "team.alpha"`,
		},
		{
			name: "key with slash rejected",
			yaml: `
ucm: {name: t}
resources:
  catalogs:
    "team/alpha": {name: team_alpha}
`,
			wantSummary: `resource key "team/alpha"`,
		},
		{
			name: "key starting with digit rejected",
			yaml: `
ucm: {name: t}
resources:
  catalogs:
    "1team": {name: team}
`,
			wantSummary: `resource key "1team"`,
		},
		{
			name: "key with dash accepted",
			yaml: `
ucm: {name: t}
resources:
  catalogs:
    team-alpha: {name: team_alpha}
`,
			wantEmpty: true,
		},
		{
			name: "UC name containing slash rejected",
			yaml: `
ucm: {name: t}
resources:
  catalogs:
    c1: {name: "bad/name"}
`,
			wantSummary: `name="bad/name"`,
		},
		{
			name: "UC name with whitespace rejected",
			yaml: `
ucm: {name: t}
resources:
  catalogs:
    c1: {name: "bad name"}
`,
			wantSummary: `name="bad name"`,
		},
		{
			name: "external_location key+url are unaffected by URL slashes in url field",
			yaml: `
ucm: {name: t}
resources:
  external_locations:
    el1:
      name: el1
      url: s3://bucket/prefix
      credential_name: sc1
`,
			wantEmpty: true,
		},
	}
	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			u := loadUcm(t, tc.yaml)
			diags := ucm.Apply(t.Context(), u, validate.Naming())
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
