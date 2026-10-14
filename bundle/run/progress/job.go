package progress

import (
	"fmt"
	"strings"
	"time"

	"github.com/databricks/cli/libs/sdktypes/jobs/v2"
	"github.com/databricks/cli/libs/utils"
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
	result.WriteString(string(event.State.LifeCycleState))

	resultState := string(event.State.ResultState)
	if resultState != "" {
		result.WriteString(" " + resultState)
	}

	stateMessage := utils.Deref(event.State.StateMessage)
	if stateMessage != "" {
		result.WriteString(" " + stateMessage)
	}

	return result.String()
}
