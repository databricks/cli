package mutator_test

import (
	"testing"

	"github.com/databricks/cli/ucm"
	"github.com/databricks/cli/ucm/config/mutator"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestInheritCatalogTags(t *testing.T) {
	tests := []struct {
		name     string
		yaml     string
		schema   string
		wantTags map[string]string
	}{
		{
			name: "catalog tags inherited by default",
			yaml: `
ucm: {name: t}
resources:
  catalogs:
    c1:
      name: c1
      tags: {cost_center: "1", data_owner: a}
      schemas:
        s1: {name: s1}
`,
			schema:   "s1",
			wantTags: map[string]string{"cost_center": "1", "data_owner": "a"},
		},
		{
			name: "schema tag overrides parent on conflict",
			yaml: `
ucm: {name: t}
resources:
  catalogs:
    c1:
      name: c1
      tags: {cost_center: "1", data_owner: a}
      schemas:
        s1: {name: s1, tags: {data_owner: b}}
`,
			schema:   "s1",
			wantTags: map[string]string{"cost_center": "1", "data_owner": "b"},
		},
		{
			name: "tag_inherit false opts out",
			yaml: `
ucm: {name: t}
resources:
  catalogs:
    c1:
      name: c1
      tags: {cost_center: "1"}
      schemas:
        s1: {name: s1, tag_inherit: false}
`,
			schema:   "s1",
			wantTags: nil,
		},
		{
			name: "flat schema inherits from its named catalog",
			yaml: `
ucm: {name: t}
resources:
  catalogs:
    c1: {name: c1, tags: {cost_center: "1"}}
  schemas:
    s1: {name: s1, catalog: c1}
`,
			schema:   "s1",
			wantTags: map[string]string{"cost_center": "1"},
		},
	}
	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			u := loadUcm(t, tc.yaml)
			diags := ucm.ApplySeq(t.Context(), u,
				mutator.FlattenNestedResources(),
				mutator.InheritCatalogTags(),
			)
			require.NoError(t, diags.Error())

			got := u.Config.Resources.Schemas[tc.schema]
			require.NotNil(t, got)
			if tc.wantTags == nil {
				assert.Empty(t, got.Tags)
			} else {
				assert.Equal(t, tc.wantTags, got.Tags)
			}
		})
	}
}
