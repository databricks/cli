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
	// GitRepository is not supported in the Deploy API, only as part of Create, so we need to remove it.
	if gitSource != nil {
		gitSource.GitRepository = nil
	}
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
// A deployment that ends in FAILED does not stop the caller: every caller deploys next,
// and the new deployment replaces the failed one. Starting an app whose source_code_path
// was set at Create triggers such a deployment, built from the app.yaml in the source
// rather than from the bundle's inline config.
func WaitForDeploymentToComplete(ctx context.Context, w *databricks.WorkspaceClient, app *sdkapps.App) error {
	if isInProgress(app.ActiveDeployment) {
		logProgress(ctx, "Waiting for the active deployment to complete...")
		succeeded, err := waitForDeployment(ctx, w, app.Name, app.ActiveDeployment.DeploymentId)
		if err != nil {
			return err
		}
		if succeeded {
			logProgress(ctx, "Active deployment is completed!")
		}
	}

	if isInProgress(app.PendingDeployment) {
		logProgress(ctx, "Waiting for the pending deployment to complete...")
		succeeded, err := waitForDeployment(ctx, w, app.Name, app.PendingDeployment.DeploymentId)
		if err != nil {
			return err
		}
		if succeeded {
			logProgress(ctx, "Pending deployment is completed!")
		}
	}

	return nil
}

func isInProgress(d *sdkapps.AppDeployment) bool {
	return d != nil && d.Status != nil && d.Status.State == sdkapps.AppDeploymentStateInProgress
}

// waitForDeployment reports whether the deployment succeeded. A FAILED deployment is
// not an error, see WaitForDeploymentToComplete.
func waitForDeployment(ctx context.Context, w *databricks.WorkspaceClient, appName, deploymentID string) (bool, error) {
	var last *sdkapps.AppDeployment
	_, err := w.Apps.WaitGetDeploymentAppSucceeded(ctx, appName, deploymentID, 20*time.Minute, func(d *sdkapps.AppDeployment) {
		last = d
	})
	if err != nil && last != nil && last.Status != nil && last.Status.State == sdkapps.AppDeploymentStateFailed {
		cmdio.LogString(ctx, fmt.Sprintf("Deployment %s failed (%s), deploying again", deploymentID, last.Status.Message))
		return false, nil
	}
	return err == nil, err
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
