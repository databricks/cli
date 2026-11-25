package statemgmt

import (
	"context"
	"fmt"

	"github.com/databricks/cli/bundle"
	"github.com/databricks/cli/bundle/config"
	"github.com/databricks/cli/libs/diag"
	"github.com/databricks/databricks-sdk-go"
	"golang.org/x/sync/errgroup"
)

type ErrResourceIsRunning struct {
	resourceType string
	resourceId   string
}

func (e ErrResourceIsRunning) Error() string {
	return fmt.Sprintf("%s %s is running", e.resourceType, e.resourceId)
}

type checkRunningResources struct{}

func (l *checkRunningResources) Name() string {
	return "check-running-resources"
}

func (l *checkRunningResources) Apply(ctx context.Context, b *bundle.Bundle) diag.Diagnostics {
	if !config.IsExplicitlyEnabled(b.Config.Bundle.Deployment.FailOnActiveRuns) {
		return nil
	}

	w := b.WorkspaceClient(ctx)
	err := checkAnyResourceRunning(ctx, w, b.DeploymentBundle.ExportState(ctx))
	if err != nil {
		return diag.FromErr(err)
	}
	return nil
}

func CheckRunningResource() bundle.Mutator {
	return &checkRunningResources{}
}

func checkAnyResourceRunning(ctx context.Context, w *databricks.WorkspaceClient, state ExportedResourcesMap) error {
	errs, errCtx := errgroup.WithContext(ctx)

	for resourceKey, attrs := range state {
		id := attrs.ID
		if id == "" {
			continue
		}

		resourceType := config.GetResourceTypeFromKey(resourceKey)

		if resourceType == "jobs" {
			errs.Go(func() error {
				isRunning, err := IsJobRunning(errCtx, w, id)
				if err != nil {
					return err
				}
				if isRunning {
					return &ErrResourceIsRunning{resourceType: "job", resourceId: id}
				}
				return nil
			})
		}

		if resourceType == "pipelines" {
			errs.Go(func() error {
				isRunning, err := IsPipelineRunning(errCtx, w, id)
				if err != nil {
					return err
				}
				if isRunning {
					return &ErrResourceIsRunning{resourceType: "pipeline", resourceId: id}
				}
				return nil
			})
		}
	}

	return errs.Wait()
}
