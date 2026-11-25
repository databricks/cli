package dresources

import (
	"testing"

	"github.com/databricks/cli/bundle/config/resources"
	"github.com/stretchr/testify/assert"
)

func TestClusterPolicyPrepareState(t *testing.T) {
	tests := []struct {
		name       string
		definition any
		want       string
	}{
		{
			// The normal post-mutator case: definition is already a JSON string.
			name:       "string definition is copied into state",
			definition: `{"spark_version":{"type":"fixed"}}`,
			want:       `{"spark_version":{"type":"fixed"}}`,
		},
		{
			// ConfigureClusterPolicyDefinition guarantees a string, so a non-string
			// is ignored rather than reaching the API.
			name:       "non-string definition is ignored",
			definition: map[string]any{"spark_version": "fixed"},
			want:       "",
		},
		{
			name:       "absent definition leaves state empty",
			definition: nil,
			want:       "",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			input := &resources.ClusterPolicy{Definition: tt.definition}
			input.Name = "my_policy"

			got := (*ResourceClusterPolicy)(nil).PrepareState(input)

			assert.Equal(t, tt.want, got.Definition)
			assert.Equal(t, "my_policy", got.Name)
		})
	}
}
