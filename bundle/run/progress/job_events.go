package progress

import (
	"fmt"
	"strings"
)

type TaskErrorEvent struct {
	TaskKey    string `json:"task_key"`
	Error      string `json:"error"`
	ErrorTrace string `json:"error_trace"`
}

func NewTaskErrorEvent(taskKey, errorMessage, errorTrace string) *TaskErrorEvent {
	return &TaskErrorEvent{
		TaskKey:    taskKey,
		Error:      errorMessage,
		ErrorTrace: errorTrace,
	}
}

func (event *TaskErrorEvent) String() string {
	result := strings.Builder{}
	result.WriteString(fmt.Sprintf("Task %s FAILED:\n", event.TaskKey))
	result.WriteString(event.Error + "\n")
	result.WriteString(event.ErrorTrace + "\n")
	return result.String()
}

func (event *TaskErrorEvent) IsInplaceSupported() bool {
	return false
}

type JobRunUrlEvent struct {
	Type string `json:"type"`
	Url  string `json:"url"`
}

func NewJobRunUrlEvent(url string) *JobRunUrlEvent {
	return &JobRunUrlEvent{
		Type: "job_run_url",
		Url:  url,
	}
}

func (event *JobRunUrlEvent) String() string {
	return fmt.Sprintf("Run URL: %s\n", event.Url)
}

func (event *JobRunUrlEvent) IsInplaceSupported() bool {
	return false
}
