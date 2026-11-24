package resourcemutator_test

import (
	"testing"

	"github.com/databricks/cli/bundle"
	"github.com/databricks/cli/bundle/config"
	"github.com/databricks/cli/bundle/config/mutator/resourcemutator"
	"github.com/databricks/cli/bundle/config/resources"
	"github.com/databricks/cli/libs/knowledge"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestPageFixups(t *testing.T) {
	tests := []struct {
		name     string
		synonyms []string
		want     []string
	}{
		{
			name:     "sorted",
			synonyms: []string{"reversal", "dispute"},
			want:     []string{"dispute", "reversal"},
		},
		{
			name:     "trimmed and deduplicated",
			synonyms: []string{"reversal", " dispute ", "reversal", "dispute"},
			want:     []string{"dispute", "reversal"},
		},
		{
			name:     "unset stays unset",
			synonyms: nil,
			want:     nil,
		},
		{
			name:     "empty stays empty",
			synonyms: []string{},
			want:     []string{},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			b := &bundle.Bundle{
				Config: config.Root{
					Resources: config.Resources{
						Pages: map[string]*resources.Page{
							"my_page": {Page: knowledge.Page{Synonyms: tt.synonyms}},
						},
					},
				},
			}

			diags := bundle.ApplySeq(t.Context(), b, resourcemutator.PageFixups())
			require.NoError(t, diags.Error())
			assert.Equal(t, tt.want, b.Config.Resources.Pages["my_page"].Synonyms)
		})
	}
}
