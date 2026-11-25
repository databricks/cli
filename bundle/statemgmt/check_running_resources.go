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

// isRunningFunc reports whether the deployed resource with the given ID is running.
type isRunningFunc func(ctx context.Context, w *databricks.WorkspaceClient, id string) (bool, error)

// runningChecks holds, per resource type, the check whether a deployed resource of that
// type is running. Deploy fails if one is and bundle.deployment.fail_on_active_runs is
// set. The checks of all resources run concurrently and the first failure is returned,
// so their order does not matter.
var runningChecks = config.NewResourceTypeRegistry[isRunningFunc]("running check")

func checkAnyResourceRunning(ctx context.Context, w *databricks.WorkspaceClient, state ExportedResourcesMap) error {
	errs, errCtx := errgroup.WithContext(ctx)

	for resourceKey, attrs := range state {
		id := attrs.ID
		if id == "" {
			continue
		}

		resourceType := config.GetResourceTypeFromKey(resourceKey)
		isRunning, ok := runningChecks.Get(resourceType)
		if !ok {
			continue
		}
		d, _ := config.ResourceTypeAs[config.ConfigResource](resourceType)
		singularName := d.ResourceDescription().SingularName

		errs.Go(func() error {
			running, err := isRunning(errCtx, w, id)
			if err != nil {
				return err
			}
			if running {
				return &ErrResourceIsRunning{resourceType: singularName, resourceId: id}
			}
			return nil
		})
	}

	return errs.Wait()
}
