package dresources

import (
	"context"
	"errors"
	"fmt"
	"strings"
	"time"

	"github.com/databricks/cli/bundle/appdeploy"
	"github.com/databricks/cli/bundle/config/resources"
	"github.com/databricks/cli/bundle/deployplan"
	"github.com/databricks/cli/libs/structs/structpath"
	"github.com/databricks/databricks-sdk-go"
	"github.com/databricks/databricks-sdk-go/apierr"
	"github.com/databricks/databricks-sdk-go/retries"
	"github.com/databricks/databricks-sdk-go/service/apps"
)

// AppState is the state type for App resources. It extends apps.App with fields
// needed for app deployments (Apps.Deploy) that are not part of the remote state.
type AppState struct {
	apps.App
	SourceCodePath string               `json:"source_code_path,omitempty"`
	Config         *resources.AppConfig `json:"config,omitempty"`
	GitSource      *apps.GitSource      `json:"git_source,omitempty"`
	Started        *bool                `json:"started,omitempty"`
}

type ResourceApp struct {
	client *databricks.WorkspaceClient
}

func (*ResourceApp) New(client *databricks.WorkspaceClient) *ResourceApp {
	return &ResourceApp{client: client}
}

func (*ResourceApp) PrepareState(input *resources.App) *AppState {
	return &AppState{
		App:            input.App,
		SourceCodePath: input.SourceCodePath,
		Config:         input.Config,
		GitSource:      input.GitSource,
		Started:        input.Lifecycle.Started,
	}
}

// RemapState maps the remote apps.App to AppState for diff comparison.
// Deploy-only fields (SourceCodePath, Config, GitSource) are not in remote state,
// so they default to zero values, which prevents false drift detection.
func (*ResourceApp) RemapState(remote *apps.App) *AppState {
	return &AppState{
		App:            *remote,
		SourceCodePath: "",
		Config:         nil,
		GitSource:      nil,
		Started:        nil,
	}
}

func (r *ResourceApp) DoRead(ctx context.Context, id string) (*apps.App, error) {
	return r.client.Apps.GetByName(ctx, id)
}

func (r *ResourceApp) DoCreate(ctx context.Context, config *AppState) (string, *apps.App, error) {
	// Start app compute only when lifecycle.started=true is explicit.
	// For nil (omitted) or false, use no_compute=true (do not start compute).
	noCompute := !(config.Started != nil && *config.Started)
	request := apps.CreateAppRequest{
		App:             config.App,
		NoCompute:       noCompute,
		ForceSendFields: nil,
	}

	retrier := retries.New[apps.App](retries.WithTimeout(15*time.Minute), retries.WithRetryFunc(shouldRetry))
	app, err := retrier.Run(ctx, func(ctx context.Context) (*apps.App, error) {
		waiter, err := r.client.Apps.Create(ctx, request)
		if err != nil {
			if errors.Is(err, apierr.ErrResourceAlreadyExists) {
				// Check if the app is in DELETING state - only then should we retry
				existingApp, getErr := r.client.Apps.GetByName(ctx, config.Name)
				if getErr != nil {
					// If we can't get the app (e.g., it was just deleted), retry the create
					if apierr.IsMissing(getErr) {
						return nil, retries.Continues("app was deleted, retrying create")
					}
					return nil, retries.Halt(err)
				}
				if existingApp.ComputeStatus != nil && existingApp.ComputeStatus.State == apps.ComputeStateDeleting {
					return nil, retries.Continues("app is deleting, retrying create")
				}
				// App exists and is not being deleted - this is a hard error
				return nil, retries.Halt(err)
			}
			return nil, retries.Halt(err)
		}
		return waiter.Response, nil
	})
	if err != nil {
		return "", nil, err
	}

	return app.Name, nil, nil
}

func (r *ResourceApp) DoUpdate(ctx context.Context, id string, config *AppState, changes Changes) (*apps.App, error) {
	updateMask := strings.Join(collectUpdatePathsWithPrefix(changes, ""), ",")

	request := apps.AsyncUpdateAppRequest{
		App:        &config.App,
		AppName:    id,
		UpdateMask: updateMask,
	}
	updateWaiter, err := r.client.Apps.CreateUpdate(ctx, request)
	if err != nil {
		return nil, err
	}

	response, err := updateWaiter.Get()
	if err != nil {
		return nil, err
	}

	if response.Status.State != apps.AppUpdateUpdateStatusUpdateStateSucceeded {
		return nil, fmt.Errorf("failed to update app %s: %s", id, response.Status.Message)
	}

	if config.Started == nil {
		return nil, nil
	}

	app, err := r.client.Apps.GetByName(ctx, id)
	if err != nil {
		return nil, fmt.Errorf("failed to get app %s: %w", id, err)
	}

	if *config.Started {
		// lifecycle.started=true: ensure the app compute is running and deploy the latest code.
		if isComputeStopped(app) {
			startWaiter, err := r.client.Apps.Start(ctx, apps.StartAppRequest{Name: id})
			if err != nil {
				return nil, fmt.Errorf("failed to start app %s: %w", id, err)
			}
			startedApp, err := startWaiter.Get()
			if err != nil {
				return nil, fmt.Errorf("failed to wait for app %s to start: %w", id, err)
			}
			if err := appdeploy.WaitForDeploymentToComplete(ctx, r.client, startedApp); err != nil {
				return nil, err
			}
		}
		deployment := appdeploy.BuildDeployment(config.SourceCodePath, config.Config, config.GitSource)
		if err := appdeploy.Deploy(ctx, r.client, id, deployment); err != nil {
			return nil, err
		}
	} else {
		// lifecycle.started=false: ensure the app compute is stopped.
		if !isComputeStopped(app) {
			stopWaiter, err := r.client.Apps.Stop(ctx, apps.StopAppRequest{Name: id})
			if err != nil {
				return nil, fmt.Errorf("failed to stop app %s: %w", id, err)
			}
			if _, err = stopWaiter.Get(); err != nil {
				return nil, fmt.Errorf("failed to wait for app %s to stop: %w", id, err)
			}
		}
	}

	return nil, nil
}

// localOnlyFields are AppState fields that have no counterpart in the remote state.
// They must not appear in the App update_mask.
var localOnlyFields = map[string]bool{
	"source_code_path": true,
	"config":           true,
	"git_source":       true,
	"started":          true,
}

func (*ResourceApp) OverrideChangeDesc(_ context.Context, p *structpath.PathNode, change *ChangeDesc, _ *apps.App) error {
	if change.Action == deployplan.Update && localOnlyFields[p.Prefix(1).String()] {
		change.Action = deployplan.Skip
	}
	return nil
}

func isComputeStopped(app *apps.App) bool {
	return app.ComputeStatus == nil ||
		app.ComputeStatus.State == apps.ComputeStateStopped ||
		app.ComputeStatus.State == apps.ComputeStateError
}

func (r *ResourceApp) DoDelete(ctx context.Context, id string) error {
	_, err := r.client.Apps.DeleteByName(ctx, id)
	return err
}

func (r *ResourceApp) WaitAfterCreate(ctx context.Context, config *AppState) (*apps.App, error) {
	return r.waitForApp(ctx, r.client, config.Name)
}

// waitForApp waits for the app to reach the target state. The target state is either ACTIVE or STOPPED.
// Apps with no_compute set to true will reach the STOPPED state, otherwise they will reach the ACTIVE state.
// We can't use the default waiter from SDK because it only waits on ACTIVE state but we need also STOPPED state.
// Ideally this should be done in Go SDK but currently only ACTIVE is marked as terminal state
// so this would need to be addressed by Apps service team first in their proto.
func (r *ResourceApp) waitForApp(ctx context.Context, w *databricks.WorkspaceClient, name string) (*apps.App, error) {
	retrier := retries.New[apps.App](retries.WithTimeout(-1), retries.WithRetryFunc(shouldRetry))
	return retrier.Run(ctx, func(ctx context.Context) (*apps.App, error) {
		app, err := w.Apps.GetByName(ctx, name)
		if err != nil {
			return nil, retries.Halt(err)
		}
		status := app.ComputeStatus.State
		statusMessage := app.ComputeStatus.Message
		switch status {
		case apps.ComputeStateActive, apps.ComputeStateStopped:
			return app, nil
		case apps.ComputeStateError:
			err := fmt.Errorf("failed to reach %s or %s, got %s: %s",
				apps.ComputeStateActive, apps.ComputeStateStopped, status, statusMessage)
			return nil, retries.Halt(err)
		default:
			return nil, retries.Continues(statusMessage)
		}
	})
}
