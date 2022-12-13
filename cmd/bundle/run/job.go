package run

import (
	"context"
	"fmt"
	"log"
	"strconv"
	"time"

	"github.com/databricks/bricks/bundle"
	"github.com/databricks/bricks/bundle/config/resources"
	"github.com/databricks/bricks/bundle/deploy/terraform"
	"github.com/databricks/bricks/bundle/phases"
	parent "github.com/databricks/bricks/cmd/bundle"
	"github.com/databricks/databricks-sdk-go/retries"
	"github.com/databricks/databricks-sdk-go/service/jobs"
	"github.com/spf13/cobra"
)

// Default timeout for waiting for a job run to complete.
var jobRunTimeout time.Duration = 2 * time.Hour

type jobRunner struct {
	bundle *bundle.Bundle
}

func (r *jobRunner) run(ctx context.Context, job *resources.Job) error {
	jobID, err := strconv.ParseInt(job.ID, 10, 64)
	if err != nil {
		return fmt.Errorf("job ID is not an integer: %s", job.ID)
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

var jobCmd = &cobra.Command{
	Use:   "job [flags] NAME...",
	Short: "Run a job",
	Args:  cobra.MinimumNArgs(1),

	PreRunE: parent.ConfigureBundle,
	RunE: func(cmd *cobra.Command, args []string) error {
		b := bundle.Get(cmd.Context())
		err := bundle.Apply(cmd.Context(), b, []bundle.Mutator{
			phases.Initialize(),
			terraform.Initialize(),
			terraform.Load(),
		})
		if err != nil {
			return err
		}

		// Locate and validate all resources for which we have to run something.
		var jobs []*resources.Job
		for _, name := range args {
			job, ok := b.Config.Resources.Jobs[name]
			if !ok {
				return fmt.Errorf("job not found: %s", name)
			}
			jobs = append(jobs, &job)
		}

		// Run sequentially.
		runner := &jobRunner{bundle: b}
		for _, job := range jobs {
			err := runner.run(cmd.Context(), job)
			if err != nil {
				return err
			}
		}

		return nil
	},
}

func init() {
	AddCommand(jobCmd)
}
