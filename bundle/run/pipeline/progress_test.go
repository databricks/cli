package pipeline

import (
	"testing"

	"github.com/databricks/databricks-sdk-go/service/pipelines"
	"github.com/stretchr/testify/assert"
)

func TestFlowProgressEventToString(t *testing.T) {
	event := ProgressEvent{
		EventType: "flow_progress",
		Message:   "my_message",
		Level:     pipelines.EventLevelInfo,
		Origin: &pipelines.Origin{
			FlowName:     "my_flow",
			PipelineName: "my_pipeline",
		},
		Timestamp: "2023-03-27T23:30:36.122Z",
	}
	assert.Equal(t, "2023-03-27T23:30:36.122Z flow_progress my_flow INFO my_message", event.String())
}

func TestUpdateProgressEventToString(t *testing.T) {
	event := ProgressEvent{
		EventType: "update_progress",
		Message:   "my_message",
		Level:     pipelines.EventLevelError,
		Origin: &pipelines.Origin{
			FlowName:     "my_flow",
			PipelineName: "my_pipeline",
		},
		Timestamp: "2023-03-27T23:30:36.122Z",
	}
	assert.Equal(t, "2023-03-27T23:30:36.122Z update_progress my_pipeline ERROR my_message", event.String())
}
