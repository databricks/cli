package appdeploy

import (
	"context"
	"fmt"
	"slices"
	"time"

	"github.com/databricks/cli/bundle/config/resources"
	"github.com/databricks/cli/bundle/internal/poll"
	"github.com/databricks/cli/libs/cmdio"
	"github.com/databricks/cli/libs/sdkclients"
	"github.com/databricks/cli/libs/sdktypes/apps/v1"
	"github.com/databricks/cli/libs/utils"
	sdkapps "github.com/databricks/sdk-go/apps/v1"
)

// appOperationTimeout is how long app start, stop and deployment are awaited.
const appOperationTimeout = 20 * time.Minute

func logProgress(ctx context.Context, msg string) {
	if msg == "" {
		return
	}
	cmdio.LogString(ctx, "✓ "+msg)
}

// BuildDeployment constructs an AppDeployment from the app's source code path, inline config and git source.
func BuildDeployment(sourcePath string, config *resources.AppConfig, gitSource *apps.GitSource) (*sdkapps.AppDeployment, error) {
	// GitRepository is not supported in the Deploy API, only as part of Create, so we need to remove it.
	if gitSource != nil {
		gitSource.GitRepository = nil
	}
	deployment := apps.AppDeployment{
		Mode:      apps.AppDeployment_Mode_Snapshot,
		GitSource: gitSource,
	}
	if sourcePath != "" {
		deployment.SourceCodePath = new(sourcePath)
	}

	if config != nil {
		if len(config.Command) > 0 {
			deployment.Command = config.Command
		}

		if len(config.Env) > 0 {
			deployment.EnvVars = make([]apps.EnvVar, len(config.Env))
			for i, env := range config.Env {
				deployment.EnvVars[i] = apps.EnvVar{
					Name:      new(env.Name),
					Value:     nonEmpty(env.Value),
					ValueFrom: nonEmpty(env.ValueFrom),
				}
			}
		}
	}

	return apps.AppDeploymentToSDK(&deployment)
}

func nonEmpty(s string) *string {
	if s == "" {
		return nil
	}
	return new(s)
}

// WaitForDeploymentToComplete waits for active and pending deployments on an app to finish.
func WaitForDeploymentToComplete(ctx context.Context, w *sdkclients.Clients, app *sdkapps.App) error {
	appName := utils.Deref(app.Name)
	if app.ActiveDeployment != nil &&
		app.ActiveDeployment.Status != nil &&
		app.ActiveDeployment.Status.State == apps.AppDeployment_State_InProgress {
		logProgress(ctx, "Waiting for the active deployment to complete...")
		_, err := WaitDeploymentSucceeded(ctx, w, appName, utils.Deref(app.ActiveDeployment.DeploymentId), nil)
		if err != nil {
			return err
		}
		logProgress(ctx, "Active deployment is completed!")
	}

	if app.PendingDeployment != nil &&
		app.PendingDeployment.Status != nil &&
		app.PendingDeployment.Status.State == apps.AppDeployment_State_InProgress {
		logProgress(ctx, "Waiting for the pending deployment to complete...")
		_, err := WaitDeploymentSucceeded(ctx, w, appName, utils.Deref(app.PendingDeployment.DeploymentId), nil)
		if err != nil {
			return err
		}
		logProgress(ctx, "Pending deployment is completed!")
	}

	return nil
}

// Deploy deploys the app using the provided deployment request.
// If another deployment is in progress, it waits for it to complete and retries.
func Deploy(ctx context.Context, w *sdkclients.Clients, appName string, deployment *sdkapps.AppDeployment) error {
	req := sdkapps.CreateAppDeploymentRequest{
		AppName:       new(appName),
		AppDeployment: deployment,
	}
	wait, err := w.Apps.CreateAppDeployment(ctx, req)
	if err != nil {
		existingApp, getErr := w.Apps.GetApp(ctx, sdkapps.GetAppRequest{Name: new(appName)})
		if getErr != nil {
			return fmt.Errorf("failed to get app %s: %w", appName, getErr)
		}

		if waitErr := WaitForDeploymentToComplete(ctx, w, existingApp); waitErr != nil {
			return waitErr
		}

		wait, err = w.Apps.CreateAppDeployment(ctx, req)
		if err != nil {
			return err
		}
	}

	_, err = WaitDeploymentSucceeded(ctx, w, appName, wait.GetDeploymentId(), func(ad *sdkapps.AppDeployment) {
		if ad.Status == nil {
			return
		}
		logProgress(ctx, utils.Deref(ad.Status.Message))
	})
	return err
}

// WaitDeploymentSucceeded polls the deployment until it succeeds, calling
// callback (if not nil) after every poll.
func WaitDeploymentSucceeded(ctx context.Context, w *sdkclients.Clients, appName, deploymentID string, callback func(*sdkapps.AppDeployment)) (*sdkapps.AppDeployment, error) {
	return poll.Until(ctx, appOperationTimeout, func(ctx context.Context) (poll.Status[sdkapps.AppDeployment], error) {
		deployment, err := w.Apps.GetAppDeployment(ctx, sdkapps.GetAppDeploymentRequest{
			AppName:      new(appName),
			DeploymentId: new(deploymentID),
		})
		if err != nil {
			return poll.Status[sdkapps.AppDeployment]{}, err
		}
		if callback != nil {
			callback(deployment)
		}
		var status sdkapps.AppDeploymentStatus
		if deployment.Status != nil {
			status = *deployment.Status
		}
		message := utils.Deref(status.Message)
		switch status.State {
		case apps.AppDeployment_State_Succeeded:
			return poll.Status[sdkapps.AppDeployment]{Done: true, Result: deployment}, nil
		case apps.AppDeployment_State_Failed:
			return poll.Status[sdkapps.AppDeployment]{}, fmt.Errorf("failed to reach %s, got %s: %s",
				apps.AppDeployment_State_Succeeded, status.State, message)
		}
		return poll.Status[sdkapps.AppDeployment]{Message: message}, nil
	})
}

// WaitComputeState polls the app until its compute reaches target, failing
// when it reaches one of the failure states. callback (if not nil) is called
// after every poll.
func WaitComputeState(ctx context.Context, w *sdkclients.Clients, appName string, target apps.ComputeStatus_ComputeState, failures []apps.ComputeStatus_ComputeState, callback func(*sdkapps.App)) (*sdkapps.App, error) {
	return poll.Until(ctx, appOperationTimeout, func(ctx context.Context) (poll.Status[sdkapps.App], error) {
		app, err := w.Apps.GetApp(ctx, sdkapps.GetAppRequest{Name: new(appName)})
		if err != nil {
			return poll.Status[sdkapps.App]{}, err
		}
		if callback != nil {
			callback(app)
		}
		var status sdkapps.ComputeStatus
		if app.ComputeStatus != nil {
			status = *app.ComputeStatus
		}
		message := utils.Deref(status.Message)
		if status.State == target {
			return poll.Status[sdkapps.App]{Done: true, Result: app}, nil
		}
		if slices.Contains(failures, status.State) {
			return poll.Status[sdkapps.App]{}, fmt.Errorf("failed to reach %s, got %s: %s", target, status.State, message)
		}
		return poll.Status[sdkapps.App]{Message: message}, nil
	})
}

// WaitAppActive waits for the app compute to become ACTIVE.
func WaitAppActive(ctx context.Context, w *sdkclients.Clients, appName string, callback func(*sdkapps.App)) (*sdkapps.App, error) {
	return WaitComputeState(ctx, w, appName, apps.ComputeStatus_ComputeState_Active,
		[]apps.ComputeStatus_ComputeState{apps.ComputeStatus_ComputeState_Error, apps.ComputeStatus_ComputeState_Stopped}, callback)
}

// WaitAppStopped waits for the app compute to become STOPPED.
func WaitAppStopped(ctx context.Context, w *sdkclients.Clients, appName string, callback func(*sdkapps.App)) (*sdkapps.App, error) {
	return WaitComputeState(ctx, w, appName, apps.ComputeStatus_ComputeState_Stopped,
		[]apps.ComputeStatus_ComputeState{apps.ComputeStatus_ComputeState_Error}, callback)
}
