package pipelines

import (
	"testing"

	"github.com/databricks/databricks-sdk-go/service/pipelines"
	"github.com/stretchr/testify/assert"
)

func TestFilterEventsByType(t *testing.T) {
	events := []pipelines.PipelineEvent{
		{EventType: "update_progress", Message: "a"},
		{EventType: "flow_progress", Message: "b"},
		{EventType: "update_progress", Message: "c"},
		{EventType: "maintenance_progress", Message: "d"},
	}

	tests := []struct {
		name       string
		eventTypes []string
		wantMsgs   []string
	}{
		{
			name:       "no types returns all events unchanged",
			eventTypes: nil,
			wantMsgs:   []string{"a", "b", "c", "d"},
		},
		{
			name:       "single type",
			eventTypes: []string{"update_progress"},
			wantMsgs:   []string{"a", "c"},
		},
		{
			name:       "multiple types preserve order",
			eventTypes: []string{"flow_progress", "maintenance_progress"},
			wantMsgs:   []string{"b", "d"},
		},
		{
			name:       "unmatched type yields no events",
			eventTypes: []string{"does_not_exist"},
			wantMsgs:   []string{},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := filterEventsByType(events, tt.eventTypes)
			msgs := []string{}
			for _, e := range got {
				msgs = append(msgs, e.Message)
			}
			assert.Equal(t, tt.wantMsgs, msgs)
		})
	}
}
