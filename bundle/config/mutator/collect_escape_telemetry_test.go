package mutator_test

import (
	"testing"

	"github.com/databricks/cli/bundle"
	"github.com/databricks/cli/bundle/config"
	"github.com/databricks/cli/bundle/config/mutator"
	"github.com/databricks/cli/libs/telemetry/protos"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestCollectEscapeTelemetry(t *testing.T) {
	tests := []struct {
		name     string
		value    string
		expected []protos.BoolMapEntry
	}{
		{
			name:     "no escape patterns",
			value:    "hello world",
			expected: nil,
		},
		{
			name:  "double dollar",
			value: "prefix-$${foo}",
			expected: []protos.BoolMapEntry{
				{Key: "config_has_double_dollar", Value: true},
			},
		},
		{
			name:  "backslash dollar",
			value: "prefix-\\${foo}",
			expected: []protos.BoolMapEntry{
				{Key: "config_has_backslash_dollar", Value: true},
			},
		},
		{
			name:  "both patterns",
			value: "$${a}-\\${b}",
			expected: []protos.BoolMapEntry{
				{Key: "config_has_double_dollar", Value: true},
				{Key: "config_has_backslash_dollar", Value: true},
			},
		},
		{
			name:     "single dollar is not matched",
			value:    "cost is $100",
			expected: nil,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			b := &bundle.Bundle{
				Config: config.Root{
					Bundle: config.Bundle{
						Name: tt.value,
					},
				},
			}

			diags := bundle.Apply(t.Context(), b, mutator.CollectEscapeTelemetry())
			require.Empty(t, diags)
			assert.Equal(t, tt.expected, b.Metrics.BoolValues)
		})
	}
}
