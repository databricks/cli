package progress

import (
	"fmt"

	"github.com/databricks/cli/libs/workspaceurls"
)

type PipelineUpdateUrlEvent struct {
	Type       string `json:"type"`
	UpdateId   string `json:"update_id"`
	PipelineId string `json:"pipeline_id"`
	Url        string `json:"url"`
}

func NewPipelineUpdateUrlEvent(host, updateId, pipelineId string) *PipelineUpdateUrlEvent {
	return &PipelineUpdateUrlEvent{
		Type:       "pipeline_update_url",
		UpdateId:   updateId,
		PipelineId: pipelineId,
		Url:        fmt.Sprintf("%s/%s", host, workspaceurls.PipelineUpdatePath(pipelineId, updateId)),
	}
}

func (event *PipelineUpdateUrlEvent) String() string {
	return fmt.Sprintf("Update URL: %s\n", event.Url)
}
