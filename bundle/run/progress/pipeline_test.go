package progress

import (
	"github.com/databricks/cli/libs/sdktypes/pipelines/v2"
	"testing"

	"github.com/stretchr/testify/assert"
)

func TestFlowProgressEventToString(t *testing.T) {
	event := ProgressEvent{
		EventType: new("flow_progress"),
		Message:   new("my_message"),
		Level:     pipelines.EventLevel_Info,
		Origin: &pipelines.Origin{
			FlowName:     new("my_flow"),
			PipelineName: new("my_pipeline"),
		},
		Timestamp: new("2023-03-27T23:30:36.122Z"),
	}
	assert.Equal(t, `2023-03-27T23:30:36.122Z flow_progress   INFO "my_message"`, event.String())
}

func TestUpdateProgressEventToString(t *testing.T) {
	event := ProgressEvent{
		EventType: new("update_progress"),
		Message:   new("my_message"),
		Level:     pipelines.EventLevel_Error,
		Origin: &pipelines.Origin{
			FlowName:     new("my_flow"),
			PipelineName: new("my_pipeline"),
		},
		Timestamp: new("2023-03-27T23:30:36.122Z"),
	}
	assert.Equal(t, `2023-03-27T23:30:36.122Z update_progress ERROR "my_message"`, event.String())
}

func TestUpdateErrorEventToString(t *testing.T) {
	event := ProgressEvent{
		EventType: new("update_progress"),
		Message:   new("failed to update pipeline"),
		Level:     pipelines.EventLevel_Error,
		Origin: &pipelines.Origin{
			FlowName:     new("my_flow"),
			PipelineName: new("my_pipeline"),
		},
		Timestamp: new("2023-03-27T23:30:36.122Z"),
		Error: &pipelines.ErrorDetail{
			Exceptions: []pipelines.SerializedException{
				{
					Message: new("parsing error"),
				},
			},
		},
	}
	assert.Equal(t, "2023-03-27T23:30:36.122Z update_progress ERROR \"failed to update pipeline\"\nparsing error", event.String())
}

func TestUpdateErrorIgnoredForWarnEvents(t *testing.T) {
	event := ProgressEvent{
		EventType: new("update_progress"),
		Message:   new("failed to update pipeline"),
		Level:     pipelines.EventLevel_Warn,
		Origin: &pipelines.Origin{
			FlowName:     new("my_flow"),
			PipelineName: new("my_pipeline"),
		},
		Timestamp: new("2023-03-27T23:30:36.122Z"),
		Error: &pipelines.ErrorDetail{
			Exceptions: []pipelines.SerializedException{
				{
					Message: new("THIS IS IGNORED"),
				},
			},
		},
	}
	assert.Equal(t, "2023-03-27T23:30:36.122Z update_progress WARN \"failed to update pipeline\"", event.String())
}
