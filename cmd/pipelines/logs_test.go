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
		startTime  string
		endTime    string
		expected   string
	}{
		{
			name:     "no filters",
			expected: "",
		},
		{
			name:     "update id only",
			updateId: "update-1",
			expected: "update_id = 'update-1'",
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
		{
			name:       "start time only",
			updateId:   "",
			levels:     []string{},
			eventTypes: []string{},
			startTime:  "2025-01-15T10:30:00Z",
			expected:   "timestamp >= '2025-01-15T10:30:00Z'",
		},
		{
			name:       "start time and end time",
			updateId:   "update-3",
			levels:     []string{"ERROR"},
			eventTypes: []string{},
			startTime:  "2025-01-15T10:30:00Z",
			endTime:    "2025-01-15T11:30:00Z",
			expected:   "update_id = 'update-3' AND level in ('ERROR') AND timestamp >= '2025-01-15T10:30:00Z' AND timestamp <= '2025-01-15T11:30:00Z'",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := buildPipelineEventFilter(tt.updateId, tt.levels, tt.eventTypes, tt.startTime, tt.endTime)
			if result != tt.expected {
				t.Errorf("buildPipelineEventFilter(%q, %v, %v, %q, %q) = %q, want %q", tt.updateId, tt.levels, tt.eventTypes, tt.startTime, tt.endTime, result, tt.expected)
			}
		})
	}
}

func TestParseAndFormatTimestamp(t *testing.T) {
	tests := []struct {
		name     string
		input    string
		expected string
	}{
		{
			name:     "valid timestamp",
			input:    "2025-08-11T21:46:14Z",
			expected: "2025-08-11T21:46:14.000Z",
		},
		{
			name:     "empty string",
			input:    "",
			expected: "",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result, err := parseAndFormatTimestamp(tt.input)
			if err != nil {
				t.Errorf("unexpected error: %v", err)
			}
			if result != tt.expected {
				t.Errorf("parseAndFormatTimestamp(%q) = %q, want %q", tt.input, result, tt.expected)
			}
		})
	}
}
