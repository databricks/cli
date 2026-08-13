package resourcemutator_test

import (
	"testing"

	"github.com/databricks/cli/bundle"
	"github.com/databricks/cli/bundle/config"
	"github.com/databricks/cli/bundle/config/mutator/resourcemutator"
	"github.com/databricks/cli/bundle/config/resources"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestConfigureClusterPolicyDefinition(t *testing.T) {
	tests := []struct {
		name           string
		definition     any
		wantDefinition any
		// wantErr, when non-empty, is a substring expected in the diagnostics.
		wantErr string
	}{
		{
			// Inline maps are marshaled to a compact JSON string with sorted keys
			// so config and state hold an identical string and don't drift.
			name:           "inline map is marshaled to a JSON string",
			definition:     map[string]any{"spark_version": map[string]any{"type": "fixed", "value": "13.3.x"}},
			wantDefinition: `{"spark_version":{"type":"fixed","value":"13.3.x"}}`,
		},
		{
			name:           "inline sequence is marshaled to a JSON string",
			definition:     []any{"a", "b"},
			wantDefinition: `["a","b"]`,
		},
		{
			name:           "inline string is left unchanged",
			definition:     `{"spark_version":{"type":"fixed"}}`,
			wantDefinition: `{"spark_version":{"type":"fixed"}}`,
		},
		{
			name:           "absent definition passes through",
			wantDefinition: nil,
		},
		{
			name:       "non-structured definition is rejected",
			definition: true,
			wantErr:    "definition must be a string, map, or sequence, got bool",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			cp := &resources.ClusterPolicy{Definition: tt.definition}

			b := &bundle.Bundle{
				Config: config.Root{
					Resources: config.Resources{
						ClusterPolicies: map[string]*resources.ClusterPolicy{"pol": cp},
					},
				},
			}

			diags := bundle.ApplySeq(t.Context(), b, resourcemutator.ConfigureClusterPolicyDefinition())

			if tt.wantErr != "" {
				require.Error(t, diags.Error())
				assert.ErrorContains(t, diags.Error(), tt.wantErr)
				return
			}

			require.NoError(t, diags.Error())
			assert.Equal(t, tt.wantDefinition, b.Config.Resources.ClusterPolicies["pol"].Definition)
		})
	}
}
