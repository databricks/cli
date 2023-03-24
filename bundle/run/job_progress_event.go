package run

// TODO: Refactor the code into jobs directory in run
// TODO: add descriptions to the output here

import (
	"context"
	"strings"
	"time"

	"github.com/databricks/bricks/libs/progress"
	"github.com/databricks/databricks-sdk-go"
	"github.com/databricks/databricks-sdk-go/service/jobs"
)

type JobEventType string

var EventTypeJobRun = JobEventType("JOB_RUN")
var EventTypeTaskRun = JobEventType("TASK_RUN")

// TODO: embed more information here for progress logging
type JobProgressEvent struct {
	Timestamp time.Time
	JobId     int64
	RunId     int64

	Url            string
	Type           JobEventType
	Source         string
	StateMessage   string
	LifecycleState jobs.RunLifeCycleState
	ResultState    jobs.RunResultState
}

// TODO: we should have a symbol denoting completed but not successful
func (event *JobProgressEvent) EventState() progress.EventState {
	switch event.LifecycleState {
	case jobs.RunLifeCycleStateBlocked, jobs.RunLifeCycleStatePending,
		jobs.RunLifeCycleStateWaitingForRetry, jobs.RunLifeCycleStateSkipped:
		return progress.EventStatePending

	case jobs.RunLifeCycleStateInternalError:
		return progress.EventStateFailed

	case jobs.RunLifeCycleStateRunning, jobs.RunLifeCycleStateTerminating:
		return progress.EventStateRunning

	case jobs.RunLifeCycleStateTerminated:
		switch event.ResultState {
		case jobs.RunResultStateSuccess:
			return progress.EventStateCompleted

		case jobs.RunResultStateCanceled, jobs.RunResultStateTimedout:
			return progress.EventStatePending

		case jobs.RunResultStateFailed:
			return progress.EventStateFailed
		}
	}
	return progress.EventStatePending
}

func (event *JobProgressEvent) JobState() string {
	lifecycleState := event.LifecycleState
	switch lifecycleState {
	case jobs.RunLifeCycleStateWaitingForRetry, jobs.RunLifeCycleStateSkipped:
		return progress.Yellow(lifecycleState)

	case jobs.RunLifeCycleStateInternalError:
		return progress.Red(lifecycleState)

	case jobs.RunLifeCycleStateRunning, jobs.RunLifeCycleStateTerminating,
		jobs.RunLifeCycleStateBlocked, jobs.RunLifeCycleStatePending:
		return progress.Green(lifecycleState)

	case jobs.RunLifeCycleStateTerminated:
		resultState := event.ResultState
		switch resultState {
		case jobs.RunResultStateSuccess:
			return progress.Green(resultState)

		case jobs.RunResultStateCanceled, jobs.RunResultStateTimedout:
			return progress.Yellow(resultState)

		case jobs.RunResultStateFailed:
			return progress.Red(resultState)
		}
	}
	return ""
}

func (event *JobProgressEvent) Content() string {
	result := strings.Builder{}
	result.WriteString(event.Source)
	result.WriteString(" ")
	result.WriteString(string(event.Type))
	result.WriteString(" ")
	result.WriteString(event.JobState())
	result.WriteString(" ")
	result.WriteString(`"` + event.StateMessage + `"`)
	result.WriteString(" ")
	result.WriteString(event.Url)
	return result.String()
}

func (event *JobProgressEvent) IndentLevel() int {
	switch event.Type {
	case EventTypeJobRun:
		return 0
	case EventTypeTaskRun:
		return 1
	}
	return 0
}

func pollStatusCallback(ctx context.Context, w *databricks.WorkspaceClient,
	jobRunId int64) func(runId int64) (bool, error) {
	runLifecycleState := make(map[int64]string)
	logger := NewJobTextLogger(ModeInplace)
	return func(runId int64) (bool, error) {
		jobRun, err := w.Jobs.GetRun(ctx, jobs.GetRun{
			RunId: jobRunId,
		})
		if err != nil {
			return false, err
		}
		lifecycleState := jobRun.State.LifeCycleState
		if lifecycleState == jobs.RunLifeCycleStateTerminated ||
			lifecycleState == jobs.RunLifeCycleStateSkipped ||
			lifecycleState == jobs.RunLifeCycleStateInternalError {
			return true, nil
		}
		runLifecycleState[jobRunId] = jobRun.State.LifeCycleState.String()
		if v, ok := runLifecycleState[jobRunId]; !ok || v != string(lifecycleState) {
			logger.Log(&JobProgressEvent{
				RunId:          runId,
				Url:            jobRun.RunPageUrl,
				Type:           "JOB_RUN",
				LifecycleState: jobRun.State.LifeCycleState,
				StateMessage:   jobRun.State.StateMessage,
				ResultState:    jobRun.State.ResultState,
				Source:         jobRun.RunName,
			})
		}
		runLifecycleState[jobRunId] = lifecycleState.String()
		for _, task := range jobRun.Tasks {
			if v, ok := runLifecycleState[task.RunId]; ok && (v == string(jobs.RunLifeCycleStateTerminated) ||
				v == string(jobs.RunLifeCycleStateInternalError) || v == string(jobs.RunLifeCycleStateSkipped)) {
				continue
			}
			taskRun, err := w.Jobs.GetRun(ctx, jobs.GetRun{
				RunId: task.RunId,
			})
			if err != nil {
				return false, err
			}
			if v, ok := runLifecycleState[task.RunId]; !ok || v != taskRun.State.LifeCycleState.String() {
				logger.Log(&JobProgressEvent{
					RunId:          task.RunId,
					Url:            taskRun.RunPageUrl,
					Type:           "TASK_RUN",
					LifecycleState: taskRun.State.LifeCycleState,
					StateMessage:   taskRun.State.StateMessage,
					ResultState:    taskRun.State.ResultState,
					Source:         taskRun.RunName,
				})
			}
			runLifecycleState[task.RunId] = taskRun.State.LifeCycleState.String()
		}
		return false, nil
	}
}

// TODO: Implement backoff here
func RunWithProgressEvents(ctx context.Context, w *databricks.WorkspaceClient, req jobs.RunNow) (*jobs.Run, error) {
	runNowResponse, err := w.Jobs.RunNow(ctx, req)
	if err != nil {
		return nil, err
	}

	runId := runNowResponse.RunId
	pollStatus := pollStatusCallback(ctx, w, runId)

	// timeout := 20 * time.Minute
	for {
		done, err := pollStatus(runId)
		if err != nil {
			return nil, err
		}
		if done {
			break
		}
	}
	return w.Jobs.GetRun(ctx, jobs.GetRun{
		RunId: runId,
	})
}
