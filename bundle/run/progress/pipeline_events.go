package progress

import "fmt"

type UpdateUrlEvent struct {
	Type       string `json:"type"`
	UpdateId   string `json:"update_id"`
	PipelineId string `json:"pipeline_id"`
	Url        string `json:"url"`
}

func NewUpdateUrlEvent(host, updateId, pipelineId string) *UpdateUrlEvent {
	return &UpdateUrlEvent{
		Type:       "update_url",
		UpdateId:   updateId,
		PipelineId: pipelineId,
		Url:        fmt.Sprintf("%s/#joblist/pipelines/%s/updates/%s", host, pipelineId, updateId),
	}
}

func (event *UpdateUrlEvent) String() string {
	return fmt.Sprintf("Update URL: %s\n", event.Url)
}
