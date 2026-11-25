package statemgmt

import (
	"context"
	"fmt"
	"sync"

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

type runningCheckRegistration struct {
	resourceKey string
	isRunning   isRunningFunc
}

var runningCheckRegistrations []runningCheckRegistration

// registerRunningCheck registers isRunning as the check whether a deployed resource of
// the type under resourceKey is running. Deploy fails if one is and
// bundle.deployment.fail_on_active_runs is set. Call it from init() in the file of that
// resource type. A type registers at most one check. The checks of all resources run
// concurrently and the first failure is returned, so their order does not matter.
func registerRunningCheck(resourceKey string, isRunning isRunningFunc) {
	runningCheckRegistrations = append(runningCheckRegistrations, runningCheckRegistration{resourceKey: resourceKey, isRunning: isRunning})
}

type runningCheck struct {
	// singularName names the resource type in the error for a running resource.
	singularName string
	isRunning    isRunningFunc
}

var runningChecks = sync.OnceValue(func() map[string]runningCheck {
	return runningChecksByKey(runningCheckRegistrations)
})

// runningChecksByKey returns the registered checks by resource type key. It panics on a
// registration for a key that is not a resource type and on a key that is registered
// twice, because each of these is a mistake in the code that registers.
func runningChecksByKey(registrations []runningCheckRegistration) map[string]runningCheck {
	checks := map[string]runningCheck{}
	for _, r := range registrations {
		d, ok := config.ResourceTypeAs[config.ConfigResource](r.resourceKey)
		if !ok {
			panic(fmt.Sprintf("running check registered for unknown resource type %q", r.resourceKey))
		}
		if _, ok := checks[r.resourceKey]; ok {
			panic(fmt.Sprintf("running check registered more than once for resource type %q", r.resourceKey))
		}
		checks[r.resourceKey] = runningCheck{singularName: d.ResourceDescription().SingularName, isRunning: r.isRunning}
	}
	return checks
}

func checkAnyResourceRunning(ctx context.Context, w *databricks.WorkspaceClient, state ExportedResourcesMap) error {
	errs, errCtx := errgroup.WithContext(ctx)

	for resourceKey, attrs := range state {
		id := attrs.ID
		if id == "" {
			continue
		}

		check, ok := runningChecks()[config.GetResourceTypeFromKey(resourceKey)]
		if !ok {
			continue
		}

		errs.Go(func() error {
			isRunning, err := check.isRunning(errCtx, w, id)
			if err != nil {
				return err
			}
			if isRunning {
				return &ErrResourceIsRunning{resourceType: check.singularName, resourceId: id}
			}
			return nil
		})
	}

	return errs.Wait()
}
