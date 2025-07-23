package pipelines

import (
	"testing"
)

func TestBuildFieldFilter(t *testing.T) {
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
			expected: "level in ('ERROR')",
		},
		{
			name:     "multiple values with spaces",
			field:    "level",
			values:   []string{"ERROR", "METRICS"},
			expected: "level in ('ERROR', 'METRICS')",
		},
		{
			name:     "event types multiple values",
			field:    "event_type",
			values:   []string{"update_progress", "flow_progress"},
			expected: "event_type in ('update_progress', 'flow_progress')",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := buildFieldFilter(tt.field, tt.values)
			if result != tt.expected {
				t.Errorf("buildFieldFilter(%q, %v) = %q, want %q", tt.field, tt.values, result, tt.expected)
			}
		})
	}
}

func TestBuildPipelineEventFilter(t *testing.T) {
	tests := []struct {
		name       string
		updateId   string
		levels     []string
		eventTypes []string
		expected   string
	}{
		{
			name:       "no filters",
			updateId:   "",
			levels:     []string{},
			eventTypes: []string{},
			expected:   "",
		},
		{
			name:       "update id only",
			updateId:   "update-1",
			levels:     []string{},
			eventTypes: []string{},
			expected:   "update_id = 'update-1'",
		},
		{
			name:       "multiple filters",
			updateId:   "update-1",
			levels:     []string{"ERROR", "METRICS"},
			eventTypes: []string{"update_progress"},
			expected:   "update_id = 'update-1' AND level in ('ERROR', 'METRICS') AND event_type in ('update_progress')",
		},
		{
			name:       "event types with multiple values",
			updateId:   "update-2",
			levels:     []string{"INFO"},
			eventTypes: []string{"update_progress", "flow_progress"},
			expected:   "update_id = 'update-2' AND level in ('INFO') AND event_type in ('update_progress', 'flow_progress')",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := buildPipelineEventFilter(tt.updateId, tt.levels, tt.eventTypes)
			if result != tt.expected {
				t.Errorf("buildPipelineEventFilter(%q, %v, %v) = %q, want %q", tt.updateId, tt.levels, tt.eventTypes, result, tt.expected)
			}
		})
	}
}
