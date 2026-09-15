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
		name  string
		field string // "definition" or "policy_family_definition_overrides"
		value any
		want  any
		// wantErr, when non-empty, is a substring expected in the diagnostics.
		wantErr string
	}{
		{
			name:  "definition: inline map is marshaled to a JSON string",
			field: "definition",
			value: map[string]any{"spark_version": map[string]any{"type": "fixed", "value": "13.3.x"}},
			want:  `{"spark_version":{"type":"fixed","value":"13.3.x"}}`,
		},
		{
			name:    "definition: inline sequence is rejected",
			field:   "definition",
			value:   []any{"a", "b"},
			wantErr: "definition must be a string or map, got sequence",
		},
		{
			name:  "definition: inline string is left unchanged",
			field: "definition",
			value: `{"spark_version":{"type":"fixed"}}`,
			want:  `{"spark_version":{"type":"fixed"}}`,
		},
		{
			name:  "definition: absent passes through",
			field: "definition",
			want:  nil,
		},
		{
			name:    "definition: non-structured is rejected",
			field:   "definition",
			value:   true,
			wantErr: "definition must be a string or map, got bool",
		},
		{
			// Number stays a JSON number (30, not "30").
			name:  "overrides: inline map is marshaled to a JSON string",
			field: "policy_family_definition_overrides",
			value: map[string]any{"autotermination_minutes": map[string]any{"type": "fixed", "value": 30}},
			want:  `{"autotermination_minutes":{"type":"fixed","value":30}}`,
		},
		{
			name:  "overrides: inline string is left unchanged",
			field: "policy_family_definition_overrides",
			value: `{"autotermination_minutes":{"type":"fixed"}}`,
			want:  `{"autotermination_minutes":{"type":"fixed"}}`,
		},
		{
			name:    "overrides: non-structured is rejected",
			field:   "policy_family_definition_overrides",
			value:   true,
			wantErr: "policy_family_definition_overrides must be a string or map, got bool",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			cp := &resources.ClusterPolicy{}
			switch tt.field {
			case "policy_family_definition_overrides":
				cp.PolicyFamilyDefinitionOverrides = tt.value
			case "definition":
				cp.Definition = tt.value
			default:
				t.Fatalf("unknown field %q", tt.field)
			}

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
			got := b.Config.Resources.ClusterPolicies["pol"]
			switch tt.field {
			case "policy_family_definition_overrides":
				assert.Equal(t, tt.want, got.PolicyFamilyDefinitionOverrides)
			case "definition":
				assert.Equal(t, tt.want, got.Definition)
			}
		})
	}
}
