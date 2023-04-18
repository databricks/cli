package progress

import (
	"fmt"
	"strings"
	"time"

	"github.com/databricks/databricks-sdk-go/service/jobs"
)

type JobProgressEvent struct {
	Timestamp time.Time     `json:"timestamp"`
	JobId     int64         `json:"job_id"`
	RunId     int64         `json:"run_id"`
	RunName   string        `json:"run_name"`
	State     jobs.RunState `json:"state"`
}

func (event *JobProgressEvent) String() string {
	result := strings.Builder{}
	result.WriteString(event.Timestamp.Format("2006-01-02 15:04:05") + " ")
	result.WriteString(fmt.Sprintf(`"%s"`, event.RunName) + " ")
	result.WriteString(event.State.LifeCycleState.String() + " ")
	if event.State.ResultState.String() != "" {
		result.WriteString(event.State.ResultState.String() + " ")
	}
	result.WriteString(event.State.StateMessage)
	return result.String()
}

func (event *JobProgressEvent) IsInplaceSupported() bool {
	return true
}
