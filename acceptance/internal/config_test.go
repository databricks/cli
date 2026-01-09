package internal

import (
	"testing"

	"github.com/stretchr/testify/assert"
)

func TestExpandEnvMatrix(t *testing.T) {
	tests := []struct {
		name     string
		matrix   map[string][]string
		exclude  map[string][]string
		expected [][]string
	}{
		{
			name:     "empty matrix",
			matrix:   map[string][]string{},
			expected: [][]string{{}},
		},
		{
			name: "single key with single value",
			matrix: map[string][]string{
				"KEY": {"VALUE"},
			},
			expected: [][]string{
				{"KEY=VALUE"},
			},
		},
		{
			name: "single key with multiple values",
			matrix: map[string][]string{
				"KEY": {"A", "B"},
			},
			expected: [][]string{
				{"KEY=A"},
				{"KEY=B"},
			},
		},
		{
			name: "multiple keys with single values",
			matrix: map[string][]string{
				"KEY1": {"VALUE1"},
				"KEY2": {"VALUE2"},
			},
			expected: [][]string{
				{"KEY1=VALUE1", "KEY2=VALUE2"},
			},
		},
		{
			name: "multiple keys with multiple values",
			matrix: map[string][]string{
				"KEY1": {"A", "B"},
				"KEY2": {"C", "D"},
			},
			expected: [][]string{
				{"KEY1=A", "KEY2=C"},
				{"KEY1=A", "KEY2=D"},
				{"KEY1=B", "KEY2=C"},
				{"KEY1=B", "KEY2=D"},
			},
		},
		{
			name: "keys with empty values are filtered out",
			matrix: map[string][]string{
				"KEY1": {"A", "B"},
				"KEY2": {},
				"KEY3": {"C"},
			},
			expected: [][]string{
				{"KEY1=A", "KEY3=C"},
				{"KEY1=B", "KEY3=C"},
			},
		},
		{
			name: "all keys with empty values",
			matrix: map[string][]string{
				"KEY1": {},
				"KEY2": {},
			},
			expected: [][]string{{}},
		},
		{
			name: "example from documentation",
			matrix: map[string][]string{
				"KEY":   {"A", "B"},
				"OTHER": {"VALUE"},
			},
			expected: [][]string{
				{"KEY=A", "OTHER=VALUE"},
				{"KEY=B", "OTHER=VALUE"},
			},
		},
		{
			name: "exclude single combination",
			matrix: map[string][]string{
				"KEY1": {"A", "B"},
				"KEY2": {"C", "D"},
			},
			exclude: map[string][]string{
				"rule1": {"KEY1=A", "KEY2=C"},
			},
			expected: [][]string{
				{"KEY1=A", "KEY2=D"},
				{"KEY1=B", "KEY2=C"},
				{"KEY1=B", "KEY2=D"},
			},
		},
		{
			name: "exclude multiple combinations",
			matrix: map[string][]string{
				"KEY1": {"A", "B"},
				"KEY2": {"C", "D"},
			},
			exclude: map[string][]string{
				"rule1": {"KEY1=A", "KEY2=C"},
				"rule2": {"KEY1=B", "KEY2=D"},
			},
			expected: [][]string{
				{"KEY1=A", "KEY2=D"},
				{"KEY1=B", "KEY2=C"},
			},
		},
		{
			name: "exclude with terraform and readplan example",
			matrix: map[string][]string{
				"DATABRICKS_BUNDLE_ENGINE": {"terraform", "direct"},
				"READPLAN":                 {"0", "1"},
			},
			exclude: map[string][]string{
				"noplantf": {"READPLAN=1", "DATABRICKS_BUNDLE_ENGINE=terraform"},
			},
			expected: [][]string{
				{"DATABRICKS_BUNDLE_ENGINE=terraform", "READPLAN=0"},
				{"DATABRICKS_BUNDLE_ENGINE=direct", "READPLAN=0"},
				{"DATABRICKS_BUNDLE_ENGINE=direct", "READPLAN=1"},
			},
		},
		{
			name: "exclude rule with subset of keys matches",
			matrix: map[string][]string{
				"KEY1": {"A"},
				"KEY2": {"B"},
				"KEY3": {"C"},
			},
			exclude: map[string][]string{
				"rule1": {"KEY1=A", "KEY2=B"},
			},
			expected: nil,
		},
		{
			name: "exclude rule with more keys than envset does not match",
			matrix: map[string][]string{
				"KEY1": {"A"},
			},
			exclude: map[string][]string{
				"rule1": {"KEY1=A", "KEY2=B"},
			},
			expected: [][]string{
				{"KEY1=A"},
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := ExpandEnvMatrix(tt.matrix, tt.exclude)
			assert.Equal(t, tt.expected, result)
		})
	}
}
