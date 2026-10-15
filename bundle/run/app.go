package run

import (
	"context"
	"errors"
	"fmt"

	"github.com/databricks/cli/bundle"
	"github.com/databricks/cli/bundle/appdeploy"
	"github.com/databricks/cli/bundle/config/resources"
	"github.com/databricks/cli/bundle/run/output"
	"github.com/databricks/cli/libs/cmdio"
	"github.com/databricks/cli/libs/sdktypes/apps/v1"
	"github.com/databricks/cli/libs/utils"
	sdkapps "github.com/databricks/sdk-go/apps/v1"
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

	return utils.Deref(a.app.Name)
}

func isAppComputeStopped(app *sdkapps.App) bool {
	return app.ComputeStatus == nil ||
		(app.ComputeStatus.State == apps.ComputeStatus_ComputeState_Stopped || app.ComputeStatus.State == apps.ComputeStatus_ComputeState_Error)
}

func (a *appRunner) Run(ctx context.Context, opts *Options) (output.RunOutput, error) {
	app := a.app
	b := a.bundle
	if app == nil {
		return nil, errors.New("app is not defined")
	}

	appName := utils.Deref(app.Name)
	logProgress(ctx, "Getting the status of the app "+appName)
	w := b.WorkspaceClient(ctx)

	// Check the status of the app first.
	createdApp, err := w.Apps.GetApp(ctx, sdkapps.GetAppRequest{Name: new(appName)})
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
	if isAppComputeStopped(createdApp) {
		err := a.start(ctx)
		if err != nil {
			return nil, err
		}
	}

	// If the app is starting, we need to wait for it to be active before we can deploy it.
	if isAppComputeStarting(createdApp) {
		_, err := appdeploy.WaitAppActive(ctx, w, appName, nil)
		if err != nil {
			return nil, err
		}
	}

	// Deploy the app.
	err = a.deploy(ctx)
	if err != nil {
		return nil, err
	}

	cmdio.LogString(ctx, "You can access the app at "+utils.Deref(createdApp.Url))
	return nil, nil
}

func isAppComputeStarting(app *sdkapps.App) bool {
	return app.ComputeStatus != nil && app.ComputeStatus.State == apps.ComputeStatus_ComputeState_Starting
}

func (a *appRunner) start(ctx context.Context) error {
	app := a.app
	b := a.bundle
	w := b.WorkspaceClient(ctx)

	appName := utils.Deref(app.Name)
	logProgress(ctx, "Starting the app "+appName)
	_, err := w.Apps.StartApp(ctx, sdkapps.StartAppRequest{Name: new(appName)})
	if err != nil {
		return err
	}

	startedApp, err := appdeploy.WaitAppActive(ctx, w, appName, func(p *sdkapps.App) {
		if p.AppStatus == nil {
			return
		}
		logProgress(ctx, "App is starting...")
	})
	if err != nil {
		return err
	}

	// After the app is started (meaning the compute is running), the API will return the app object with the
	// active and pending deployments fields (if any). If there are active or pending deployments,
	// we need to wait for them to complete before we can do the new deployment.
	// Otherwise, the new deployment will fail.
	err = appdeploy.WaitForDeploymentToComplete(ctx, w, startedApp)
	if err != nil {
		return err
	}

	logProgress(ctx, "App is started!")
	return nil
}

func (a *appRunner) deploy(ctx context.Context) error {
	w := a.bundle.WorkspaceClient(ctx)
	// source_code_path and config already have their ${resources.*} references resolved
	// against the deployed state during run setup (see
	// DeploymentBundle.ResolveConfigAgainstState), so they are usable as-is.
	deployment, err := appdeploy.BuildDeployment(utils.Deref(a.app.SourceCodePath), a.app.Config, a.app.GitSource)
	if err != nil {
		return err
	}
	return appdeploy.Deploy(ctx, w, utils.Deref(a.app.Name), deployment)
}

func (a *appRunner) Cancel(ctx context.Context) error {
	// We should cancel the app by stopping it.
	app := a.app
	b := a.bundle
	if app == nil {
		return errors.New("app is not defined")
	}

	w := b.WorkspaceClient(ctx)

	appName := utils.Deref(app.Name)
	logProgress(ctx, "Stopping app "+appName)
	_, err := w.Apps.StopApp(ctx, sdkapps.StopAppRequest{Name: new(appName)})
	if err != nil {
		return err
	}

	_, err = appdeploy.WaitAppStopped(ctx, w, appName, func(p *sdkapps.App) {
		if p.AppStatus == nil {
			return
		}
		logProgress(ctx, utils.Deref(p.AppStatus.Message))
	})

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
