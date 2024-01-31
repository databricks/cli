package deploy

import (
	"context"
	"errors"
	"fmt"
	"strconv"

	"github.com/databricks/cli/bundle"
	"github.com/databricks/cli/libs/cmdio"
	"github.com/databricks/databricks-sdk-go"
	"github.com/databricks/databricks-sdk-go/service/jobs"
	"github.com/databricks/databricks-sdk-go/service/pipelines"
	"github.com/hashicorp/terraform-exec/tfexec"
	tfjson "github.com/hashicorp/terraform-json"
	"golang.org/x/sync/errgroup"
)

type ErrResourceIsRunning struct{}

func (ErrResourceIsRunning) Error() string {
	return "resource is running"
}

type checkRunningResources struct {
}

func (l *checkRunningResources) Name() string {
	return "check-running-resources"
}

func (l *checkRunningResources) Apply(ctx context.Context, b *bundle.Bundle) error {
	if !b.Config.Bundle.FailIfRunning {
		return nil
	}

	tf := b.Terraform
	if tf == nil {
		return fmt.Errorf("terraform not initialized")
	}

	err := tf.Init(ctx, tfexec.Upgrade(true))
	if err != nil {
		return fmt.Errorf("terraform init: %w", err)
	}

	state, err := b.Terraform.Show(ctx)
	if err != nil {
		return err
	}

	isRunning := isAnyResourceRunning(ctx, b.WorkspaceClient(), state)
	if !isRunning {
		return nil
	}

	ans, err := cmdio.AskYesOrNo(ctx, `Some of the bundle resources are still running.
Deploying the bundle can disrupt any jobs or pipelines in progress.
Do you want to continue?`)

	if err != nil {
		return err
	}
	if !ans {
		return errors.New("deployment aborted")
	}

	return nil
}

func CheckRunningResource() *checkRunningResources {
	return &checkRunningResources{}
}

func isAnyResourceRunning(ctx context.Context, w *databricks.WorkspaceClient, state *tfjson.State) bool {
	if state.Values == nil || state.Values.RootModule == nil {
		return false
	}

	errs, errCtx := errgroup.WithContext(ctx)

	for _, resource := range state.Values.RootModule.Resources {
		// Limit to resources.
		if resource.Mode != tfjson.ManagedResourceMode {
			continue
		}

		value, ok := resource.AttributeValues["id"]
		if !ok {
			continue
		}
		id, ok := value.(string)
		if !ok {
			continue
		}

		switch resource.Type {
		case "databricks_job":
			errs.Go(func() error {
				isRunning, err := IsJobRunning(errCtx, w, id)
				// If there's an error retrieving the job, we assume it's not running
				if err != nil {
					return err
				}
				if isRunning {
					return &ErrResourceIsRunning{}
				}
				return nil
			})
		case "databricks_pipeline":
			errs.Go(func() error {
				isRunning, err := IsPipelineRunning(errCtx, w, id)
				// If there's an error retrieving the pipeline, we assume it's not running
				if err != nil {
					return nil
				}
				if isRunning {
					return &ErrResourceIsRunning{}
				}
				return nil
			})
		}
	}

	err := errs.Wait()
	return err != nil
}

func IsJobRunning(ctx context.Context, w *databricks.WorkspaceClient, jobId string) (bool, error) {
	id, err := strconv.Atoi(jobId)
	if err != nil {
		return false, err
	}

	runs, err := w.Jobs.ListRunsAll(ctx, jobs.ListRunsRequest{JobId: int64(id), ActiveOnly: true})
	if err != nil {
		return false, err
	}

	return len(runs) > 0, nil
}

func IsPipelineRunning(ctx context.Context, w *databricks.WorkspaceClient, pipelineId string) (bool, error) {
	resp, err := w.Pipelines.Get(ctx, pipelines.GetPipelineRequest{PipelineId: pipelineId})
	if err != nil {
		return false, err
	}
	switch resp.State {
	case pipelines.PipelineStateIdle, pipelines.PipelineStateFailed, pipelines.PipelineStateDeleted:
		return false, nil
	default:
		return true, nil
	}
}
