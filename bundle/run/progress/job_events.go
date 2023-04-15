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
