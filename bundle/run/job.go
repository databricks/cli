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

	var prevState *jobs.RunState

	// This function is called each time the function below polls the run status.
	update := func(info *retries.Info[jobs.Run]) {
		state := info.Info.State
		if state == nil {
			return
		}
		// Log the job run URL as soon as it is available.
		if prevState == nil {
			log.Printf("[INFO] Job run available at %s", info.Info.RunPageUrl)
		}
		if prevState == nil || prevState.LifeCycleState != state.LifeCycleState {
			log.Printf("[INFO] Job run status: %s", info.Info.State.LifeCycleState)
			prevState = state
		}
	}

	w := r.bundle.WorkspaceClient()
	_, err = w.Jobs.RunNowAndWait(ctx, jobs.RunNow{
		JobId: jobID,
	}, retries.Timeout[jobs.Run](jobRunTimeout), update)
	return err
}
