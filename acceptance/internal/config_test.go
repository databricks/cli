package internal

import (
	"testing"

	"github.com/stretchr/testify/assert"
)

func TestExpandEnvMatrix(t *testing.T) {
	tests := []struct {
		name     string
		matrix   map[string][]string
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
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := ExpandEnvMatrix(tt.matrix)
			assert.Equal(t, tt.expected, result)
		})
	}
}
