package run

import (
	"context"
	"errors"
	"fmt"
	"time"

	"github.com/databricks/cli/bundle"
	"github.com/databricks/cli/bundle/config/resources"
	"github.com/databricks/cli/bundle/run/output"
	"github.com/databricks/cli/libs/cmdio"
	"github.com/databricks/databricks-sdk-go"
	"github.com/databricks/databricks-sdk-go/service/apps"
	"github.com/spf13/cobra"
)

func logProgress(ctx context.Context, msg string) {
	if msg == "" {
		return
	}
	cmdio.LogString(ctx, "✓ "+msg)
}

type appRunner struct {
	key

	bundle *bundle.Bundle
	app    *resources.App
}

func (a *appRunner) Name() string {
	if a.app == nil {
		return ""
	}

	return a.app.Name
}

func isAppStopped(app *apps.App) bool {
	return app.ComputeStatus == nil ||
		(app.ComputeStatus.State == apps.ComputeStateStopped || app.ComputeStatus.State == apps.ComputeStateError)
}

func (a *appRunner) Run(ctx context.Context, opts *Options) (output.RunOutput, error) {
	app := a.app
	b := a.bundle
	if app == nil {
		return nil, errors.New("app is not defined")
	}

	logProgress(ctx, "Getting the status of the app "+app.Name)
	w := b.WorkspaceClient()

	// Check the status of the app first.
	createdApp, err := w.Apps.Get(ctx, apps.GetAppRequest{Name: app.Name})
	if err != nil {
		return nil, err
	}

	if createdApp.AppStatus != nil {
		logProgress(ctx, fmt.Sprintf("App is in %s state", createdApp.AppStatus.State))
	}

	if createdApp.ComputeStatus != nil {
		logProgress(ctx, fmt.Sprintf("App compute is in %s state", createdApp.ComputeStatus.State))
	}

	// There could be 2 reasons why the app is not running:
	// 1. The app is new and was never deployed yet.
	// 2. The app was stopped (compute not running).
	// We need to start the app only if the compute is not running.
	if isAppStopped(createdApp) {
		err := a.start(ctx)
		if err != nil {
			return nil, err
		}
	}

	// Deploy the app.
	err = a.deploy(ctx)
	if err != nil {
		return nil, err
	}

	cmdio.LogString(ctx, "You can access the app at "+createdApp.Url)
	return nil, nil
}

func (a *appRunner) start(ctx context.Context) error {
	app := a.app
	b := a.bundle
	w := b.WorkspaceClient()

	logProgress(ctx, "Starting the app "+app.Name)
	wait, err := w.Apps.Start(ctx, apps.StartAppRequest{Name: app.Name})
	if err != nil {
		return err
	}

	startedApp, err := wait.OnProgress(func(p *apps.App) {
		if p.AppStatus == nil {
			return
		}
		logProgress(ctx, "App is starting...")
	}).Get()
	if err != nil {
		return err
	}

	// After the app is started (meaning the compute is running), the API will return the app object with the
	// active and pending deployments fields (if any). If there are active or pending deployments,
	// we need to wait for them to complete before we can do the new deployment.
	// Otherwise, the new deployment will fail.
	err = waitForDeploymentToComplete(ctx, w, startedApp)
	if err != nil {
		return err
	}

	logProgress(ctx, "App is started!")
	return nil
}

func waitForDeploymentToComplete(ctx context.Context, w *databricks.WorkspaceClient, app *apps.App) error {
	// We first wait for the active deployment to complete.
	if app.ActiveDeployment != nil &&
		app.ActiveDeployment.Status.State == apps.AppDeploymentStateInProgress {
		logProgress(ctx, "Waiting for the active deployment to complete...")
		_, err := w.Apps.WaitGetDeploymentAppSucceeded(ctx, app.Name, app.ActiveDeployment.DeploymentId, 20*time.Minute, nil)
		if err != nil {
			return err
		}
		logProgress(ctx, "Active deployment is completed!")
	}

	// Then, we wait for the pending deployment to complete.
	if app.PendingDeployment != nil &&
		app.PendingDeployment.Status.State == apps.AppDeploymentStateInProgress {
		logProgress(ctx, "Waiting for the pending deployment to complete...")
		_, err := w.Apps.WaitGetDeploymentAppSucceeded(ctx, app.Name, app.PendingDeployment.DeploymentId, 20*time.Minute, nil)
		if err != nil {
			return err
		}
		logProgress(ctx, "Pending deployment is completed!")
	}

	return nil
}

func (a *appRunner) deploy(ctx context.Context) error {
	app := a.app
	b := a.bundle
	w := b.WorkspaceClient()

	sourceCodePath := app.SourceCodePath
	wait, err := w.Apps.Deploy(ctx, apps.CreateAppDeploymentRequest{
		AppName: app.Name,
		AppDeployment: apps.AppDeployment{
			Mode:           apps.AppDeploymentModeSnapshot,
			SourceCodePath: sourceCodePath,
		},
	})
	// If deploy returns an error, then there's an active deployment in progress, wait for it to complete.
	// For this we first need to get an app and its acrive and pending deployments and then wait for them.
	if err != nil {
		app, err := w.Apps.Get(ctx, apps.GetAppRequest{Name: app.Name})
		if err != nil {
			return fmt.Errorf("failed to get app %s: %w", app.Name, err)
		}

		err = waitForDeploymentToComplete(ctx, w, app)
		if err != nil {
			return err
		}

		// Now we can try to deploy the app again
		wait, err = w.Apps.Deploy(ctx, apps.CreateAppDeploymentRequest{
			AppName: app.Name,
			AppDeployment: apps.AppDeployment{
				Mode:           apps.AppDeploymentModeSnapshot,
				SourceCodePath: sourceCodePath,
			},
		})
		if err != nil {
			return err
		}
	}

	_, err = wait.OnProgress(func(ad *apps.AppDeployment) {
		if ad.Status == nil {
			return
		}
		logProgress(ctx, ad.Status.Message)
	}).Get()
	if err != nil {
		return err
	}

	return nil
}

func (a *appRunner) Cancel(ctx context.Context) error {
	// We should cancel the app by stopping it.
	app := a.app
	b := a.bundle
	if app == nil {
		return errors.New("app is not defined")
	}

	w := b.WorkspaceClient()

	logProgress(ctx, "Stopping app "+app.Name)
	wait, err := w.Apps.Stop(ctx, apps.StopAppRequest{Name: app.Name})
	if err != nil {
		return err
	}

	_, err = wait.OnProgress(func(p *apps.App) {
		if p.AppStatus == nil {
			return
		}
		logProgress(ctx, p.AppStatus.Message)
	}).Get()

	logProgress(ctx, "App is stopped!")
	return err
}

func (a *appRunner) Restart(ctx context.Context, opts *Options) (output.RunOutput, error) {
	// We should restart the app by just running it again meaning a new app deployment will be done.
	return a.Run(ctx, opts)
}

func (a *appRunner) ParseArgs(args []string, opts *Options) error {
	if len(args) == 0 {
		return nil
	}

	return fmt.Errorf("received %d unexpected positional arguments", len(args))
}

func (a *appRunner) CompleteArgs(args []string, toComplete string) ([]string, cobra.ShellCompDirective) {
	return nil, cobra.ShellCompDirectiveNoFileComp
}
