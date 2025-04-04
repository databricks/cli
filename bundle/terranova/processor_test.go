package terranova

import (
	"testing"

	"github.com/databricks/cli/libs/dyn"
	"github.com/stretchr/testify/assert"
)

func TestApplyMove(t *testing.T) {
	tests := []struct {
		name     string
		input    map[string]any
		fields   []string
		target   string
		expected map[string]any
	}{
		{
			name: "basic move",
			input: map[string]any{
				"job_id": 123,
				"field1": "hello",
				"field2": "world",
			},
			fields: []string{"field1", "field2"},
			target: "data",
			expected: map[string]any{
				"job_id": 123,
				"data": map[string]any{
					"field1": "hello",
					"field2": "world",
				},
			},
		},
		{
			name: "move subset of fields",
			input: map[string]any{
				"id":     1,
				"name":   "test",
				"email":  "test@example.com",
				"active": true,
			},
			fields: []string{"name", "email"},
			target: "contact",
			expected: map[string]any{
				"id":     1,
				"active": true,
				"contact": map[string]any{
					"name":  "test",
					"email": "test@example.com",
				},
			},
		},
		{
			name: "move with non-existent fields",
			input: map[string]any{
				"id":   1,
				"name": "test",
			},
			fields: []string{"name", "missing"},
			target: "data",
			expected: map[string]any{
				"id": 1,
				"data": map[string]any{
					"name": "test",
				},
			},
		},
		{
			name:   "empty input",
			input:  map[string]any{},
			fields: []string{"field1"},
			target: "data",
			expected: map[string]any{
				"data": map[string]any{},
			},
		},
		{
			name: "negated field syntax",
			input: map[string]any{
				"job_id": 123,
				"field1": "hello",
				"field2": "world",
			},
			fields: []string{"!job_id"},
			target: "data",
			expected: map[string]any{
				"job_id": 123,
				"data": map[string]any{
					"field1": "hello",
					"field2": "world",
				},
			},
		},
		{
			name: "multiple negated fields",
			input: map[string]any{
				"id":     1,
				"name":   "test",
				"email":  "test@example.com",
				"active": true,
			},
			fields: []string{"!id", "!active"},
			target: "contact",
			expected: map[string]any{
				"id":     1,
				"active": true,
				"contact": map[string]any{
					"name":  "test",
					"email": "test@example.com",
				},
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			move := &Move{
				Fields: tt.fields,
				Target: tt.target,
			}

			input := dyn.NewValue(tt.input, nil)
			result, err := move.ApplyMove(input)

			assert.NoError(t, err)
			assert.Equal(t, tt.expected, result.AsAny())
		})
	}
}
