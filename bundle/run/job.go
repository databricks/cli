package run

import (
	"context"
	"fmt"
	"log"
	"strconv"
	"time"

	"github.com/databricks/bricks/bundle"
	"github.com/databricks/bricks/bundle/config/resources"
	"github.com/databricks/databricks-sdk-go/retries"
	"github.com/databricks/databricks-sdk-go/service/jobs"
)

// Default timeout for waiting for a job run to complete.
var jobRunTimeout time.Duration = 2 * time.Hour

type jobRunner struct {
	key

	bundle *bundle.Bundle
	job    *resources.Job
}

func (r *jobRunner) Run(ctx context.Context) error {
	jobID, err := strconv.ParseInt(r.job.ID, 10, 64)
	if err != nil {
		return fmt.Errorf("job ID is not an integer: %s", r.job.ID)
	}

	var prefix = fmt.Sprintf("[INFO] [%s]", r.Key())
	var prevState *jobs.RunState

	// This function is called each time the function below polls the run status.
	update := func(info *retries.Info[jobs.Run]) {
		state := info.Info.State
		if state == nil {
			return
		}
		// Log the job run URL as soon as it is available.
		if prevState == nil {
			log.Printf("%s Run available at %s", prefix, info.Info.RunPageUrl)
		}
		if prevState == nil || prevState.LifeCycleState != state.LifeCycleState {
			log.Printf("%s Run status: %s", prefix, info.Info.State.LifeCycleState)
			prevState = state
		}
	}

	w := r.bundle.WorkspaceClient()
	run, err := w.Jobs.RunNowAndWait(ctx, jobs.RunNow{
		JobId: jobID,
	}, retries.Timeout[jobs.Run](jobRunTimeout), update)
	if err != nil {
		return err
	}

	switch run.State.ResultState {
	// The run was canceled at user request.
	case jobs.RunResultStateCanceled:
		log.Printf("%s Run was cancelled!", prefix)
		return fmt.Errorf("run canceled: %s", run.State.StateMessage)

	// The task completed with an error.
	case jobs.RunResultStateFailed:
		log.Printf("%s Run has failed!", prefix)
		return fmt.Errorf("run failed: %s", run.State.StateMessage)

	// The task completed successfully.
	case jobs.RunResultStateSuccess:
		log.Printf("%s Run has completed successfully!", prefix)
		return nil

	// The run was stopped after reaching the timeout.
	case jobs.RunResultStateTimedout:
		log.Printf("%s Run has timed out!", prefix)
		return fmt.Errorf("run timed out: %s", run.State.StateMessage)
	}

	return err
}
