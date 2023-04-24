package progress

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
	assert.Equal(t, `2023-03-27T23:30:36.122Z flow_progress   INFO "my_message"`, event.String())
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
	assert.Equal(t, `2023-03-27T23:30:36.122Z update_progress ERROR "my_message"`, event.String())
}

func TestUpdateErrorEventToString(t *testing.T) {
	event := ProgressEvent{
		EventType: "update_progress",
		Message:   "failed to update pipeline",
		Level:     pipelines.EventLevelError,
		Origin: &pipelines.Origin{
			FlowName:     "my_flow",
			PipelineName: "my_pipeline",
		},
		Timestamp: "2023-03-27T23:30:36.122Z",
		Error: &pipelines.ErrorDetail{
			Exceptions: []pipelines.SerializedException{
				{
					Message: "parsing error",
				},
			},
		},
	}
	assert.Equal(t, "2023-03-27T23:30:36.122Z update_progress ERROR \"failed to update pipeline\"\nparsing error", event.String())
}

func TestUpdateErrorIgnoredForWarnEvents(t *testing.T) {
	event := ProgressEvent{
		EventType: "update_progress",
		Message:   "failed to update pipeline",
		Level:     pipelines.EventLevelWarn,
		Origin: &pipelines.Origin{
			FlowName:     "my_flow",
			PipelineName: "my_pipeline",
		},
		Timestamp: "2023-03-27T23:30:36.122Z",
		Error: &pipelines.ErrorDetail{
			Exceptions: []pipelines.SerializedException{
				{
					Message: "THIS IS IGNORED",
				},
			},
		},
	}
	assert.Equal(t, "2023-03-27T23:30:36.122Z update_progress WARN \"failed to update pipeline\"", event.String())
}
