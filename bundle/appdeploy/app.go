package appdeploy

import (
	"context"
	"fmt"
	"time"

	"github.com/databricks/cli/bundle/config/resources"
	"github.com/databricks/cli/libs/cmdio"
	"github.com/databricks/databricks-sdk-go"
	sdkapps "github.com/databricks/databricks-sdk-go/service/apps"
)

func logProgress(ctx context.Context, msg string) {
	if msg == "" {
		return
	}
	cmdio.LogString(ctx, "✓ "+msg)
}

// BuildDeployment constructs an AppDeployment from the app's source code path, inline config and git source.
func BuildDeployment(sourcePath string, config *resources.AppConfig, gitSource *sdkapps.GitSource) sdkapps.AppDeployment {
	deployment := sdkapps.AppDeployment{
		Mode:           sdkapps.AppDeploymentModeSnapshot,
		SourceCodePath: sourcePath,
		GitSource:      gitSource,
	}

	if config != nil {
		if len(config.Command) > 0 {
			deployment.Command = config.Command
		}

		if len(config.Env) > 0 {
			deployment.EnvVars = make([]sdkapps.EnvVar, len(config.Env))
			for i, env := range config.Env {
				deployment.EnvVars[i] = sdkapps.EnvVar{
					Name:      env.Name,
					Value:     env.Value,
					ValueFrom: env.ValueFrom,
				}
			}
		}
	}

	return deployment
}

// WaitForDeploymentToComplete waits for active and pending deployments on an app to finish.
func WaitForDeploymentToComplete(ctx context.Context, w *databricks.WorkspaceClient, app *sdkapps.App) error {
	if app.ActiveDeployment != nil &&
		app.ActiveDeployment.Status.State == sdkapps.AppDeploymentStateInProgress {
		logProgress(ctx, "Waiting for the active deployment to complete...")
		_, err := w.Apps.WaitGetDeploymentAppSucceeded(ctx, app.Name, app.ActiveDeployment.DeploymentId, 20*time.Minute, nil)
		if err != nil {
			return err
		}
		logProgress(ctx, "Active deployment is completed!")
	}

	if app.PendingDeployment != nil &&
		app.PendingDeployment.Status.State == sdkapps.AppDeploymentStateInProgress {
		logProgress(ctx, "Waiting for the pending deployment to complete...")
		_, err := w.Apps.WaitGetDeploymentAppSucceeded(ctx, app.Name, app.PendingDeployment.DeploymentId, 20*time.Minute, nil)
		if err != nil {
			return err
		}
		logProgress(ctx, "Pending deployment is completed!")
	}

	return nil
}

// Deploy deploys the app using the provided deployment request.
// If another deployment is in progress, it waits for it to complete and retries.
func Deploy(ctx context.Context, w *databricks.WorkspaceClient, appName string, deployment sdkapps.AppDeployment) error {
	wait, err := w.Apps.Deploy(ctx, sdkapps.CreateAppDeploymentRequest{
		AppName:       appName,
		AppDeployment: deployment,
	})
	if err != nil {
		existingApp, getErr := w.Apps.Get(ctx, sdkapps.GetAppRequest{Name: appName})
		if getErr != nil {
			return fmt.Errorf("failed to get app %s: %w", appName, getErr)
		}

		if waitErr := WaitForDeploymentToComplete(ctx, w, existingApp); waitErr != nil {
			return waitErr
		}

		wait, err = w.Apps.Deploy(ctx, sdkapps.CreateAppDeploymentRequest{
			AppName:       appName,
			AppDeployment: deployment,
		})
		if err != nil {
			return err
		}
	}

	_, err = wait.OnProgress(func(ad *sdkapps.AppDeployment) {
		if ad.Status == nil {
			return
		}
		logProgress(ctx, ad.Status.Message)
	}).Get()
	return err
}
