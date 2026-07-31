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
			name:     "single dollar is not matched",
			value:    "cost is $100",
			expected: nil,
		},
		{
			name:  "double dollar with brace",
			value: "prefix-$${foo}",
			expected: []protos.BoolMapEntry{
				{Key: "config_has_double_dollar_brace", Value: true},
			},
		},
		{
			name:  "double dollar without brace",
			value: "price is $$5",
			expected: []protos.BoolMapEntry{
				{Key: "config_has_double_dollar", Value: true},
			},
		},
		{
			name:  "double dollar at end of string",
			value: "ends with $$",
			expected: []protos.BoolMapEntry{
				{Key: "config_has_double_dollar", Value: true},
			},
		},
		{
			name:  "backslash dollar with brace",
			value: "prefix-\\${foo}",
			expected: []protos.BoolMapEntry{
				{Key: "config_has_backslash_dollar_brace", Value: true},
			},
		},
		{
			name:  "backslash dollar without brace",
			value: "price is \\$5",
			expected: []protos.BoolMapEntry{
				{Key: "config_has_backslash_dollar", Value: true},
			},
		},
		{
			name:  "backslash dollar at end of string",
			value: "ends with \\$",
			expected: []protos.BoolMapEntry{
				{Key: "config_has_backslash_dollar", Value: true},
			},
		},
		{
			name:  "all four patterns",
			value: "$${a}-$$b-\\${c}-\\$d",
			expected: []protos.BoolMapEntry{
				{Key: "config_has_double_dollar_brace", Value: true},
				{Key: "config_has_double_dollar", Value: true},
				{Key: "config_has_backslash_dollar_brace", Value: true},
				{Key: "config_has_backslash_dollar", Value: true},
			},
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
