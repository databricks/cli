package run

import (
	"context"
	"fmt"
	"log"
	"time"

	"github.com/databricks/bricks/bundle"
	"github.com/databricks/bricks/bundle/config/resources"
	"github.com/databricks/bricks/bundle/deploy/terraform"
	"github.com/databricks/bricks/bundle/phases"
	parent "github.com/databricks/bricks/cmd/bundle"
	"github.com/databricks/databricks-sdk-go/service/pipelines"
	"github.com/spf13/cobra"
)

type pipelineRunner struct {
	bundle *bundle.Bundle
}

func (r *pipelineRunner) run(ctx context.Context, p *resources.Pipeline) error {
	w := r.bundle.WorkspaceClient()
	_, err := w.Pipelines.GetPipelineByPipelineId(ctx, p.ID)
	if err != nil {
		log.Printf("[WARN] Cannot get pipeline: %s", err)
		return err
	}

	res, err := w.Pipelines.StartUpdate(ctx, pipelines.StartUpdate{
		PipelineId: p.ID,
	})
	if err != nil {
		return err
	}

	updateID := res.UpdateId

	// Log the pipeline update URL as soon as it is available.
	url := fmt.Sprintf("%s/#joblist/pipelines/%s/updates/%s", w.Config.Host, p.ID, updateID)
	log.Printf("[INFO] Pipeline update available at %s", url)

	// Poll update for completion and post status.
	// Note: there is no "StartUpdateAndWait" wrapper for this API.
	var prevState *pipelines.UpdateInfoState
	for {
		update, err := w.Pipelines.GetUpdateByPipelineIdAndUpdateId(ctx, p.ID, updateID)
		if err != nil {
			return err
		}

		// Log only if the current state is different from the previous state.
		state := update.Update.State
		if prevState == nil || *prevState != state {
			log.Printf("[INFO] Pipeline update status: %s", state)
			prevState = &state
		}

		if state == pipelines.UpdateInfoStateCanceled {
			return fmt.Errorf("pipeline state: %s", state)
		}
		if state == pipelines.UpdateInfoStateFailed {
			return fmt.Errorf("pipeline state: %s", state)
		}
		if state == pipelines.UpdateInfoStateCompleted {
			return nil
		}

		time.Sleep(time.Second)
	}
}

var pipelineCmd = &cobra.Command{
	Use:   "pipeline [flags] NAME...",
	Short: "Run a pipeline update",
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
		var pipelines []*resources.Pipeline
		for _, name := range args {
			pipeline, ok := b.Config.Resources.Pipelines[name]
			if !ok {
				return fmt.Errorf("pipeline not found: %s", name)
			}
			pipelines = append(pipelines, &pipeline)
		}

		// Run sequentially.
		runner := &pipelineRunner{bundle: b}
		for _, pipeline := range pipelines {
			err := runner.run(cmd.Context(), pipeline)
			if err != nil {
				return err
			}
		}

		return nil
	},
}

func init() {
	AddCommand(pipelineCmd)
}
