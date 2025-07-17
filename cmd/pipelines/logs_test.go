package pipelines

import (
	"testing"

	"github.com/stretchr/testify/assert"
)

func TestBuildInFilter(t *testing.T) {
	tests := []struct {
		name     string
		field    string
		values   []string
		expected string
	}{
		{
			name:     "empty values",
			field:    "level",
			values:   []string{},
			expected: "",
		},
		{
			name:     "single value",
			field:    "level",
			values:   []string{"ERROR"},
			expected: "level = 'ERROR'",
		},
		{
			name:     "multiple values",
			field:    "level",
			values:   []string{"ERROR", "WARN"},
			expected: "level in ('ERROR', 'WARN')",
		},
		{
			name:     "three values",
			field:    "event_type",
			values:   []string{"update_progress", "flow_progress", "pipeline_started"},
			expected: "event_type in ('update_progress', 'flow_progress', 'pipeline_started')",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := buildInFilter(tt.field, tt.values)
			assert.Equal(t, tt.expected, result)
		})
	}
}
