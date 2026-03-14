package dresources

import (
	"context"
	"errors"
	"fmt"
	"strings"
	"time"

	"github.com/databricks/cli/bundle/config/resources"
	"github.com/databricks/databricks-sdk-go"
	"github.com/databricks/databricks-sdk-go/apierr"
	"github.com/databricks/databricks-sdk-go/retries"
	"github.com/databricks/databricks-sdk-go/service/apps"
)

type ResourceApp struct {
	client *databricks.WorkspaceClient
}

func (*ResourceApp) New(client *databricks.WorkspaceClient) *ResourceApp {
	return &ResourceApp{client: client}
}

func (*ResourceApp) PrepareState(input *resources.App) *apps.App {
	return &input.App
}

func (r *ResourceApp) DoRead(ctx context.Context, id string) (*apps.App, error) {
	return r.client.Apps.GetByName(ctx, id)
}

func (r *ResourceApp) DoCreate(ctx context.Context, config *apps.App) (string, *apps.App, error) {
	request := apps.CreateAppRequest{
		App:             *config,
		NoCompute:       true,
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

func (r *ResourceApp) DoUpdate(ctx context.Context, id string, config *apps.App, changes Changes) (*apps.App, error) {
	updateMask := strings.Join(collectUpdatePathsWithPrefix(changes, ""), ",")

	request := apps.AsyncUpdateAppRequest{
		App:        config,
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
	return nil, nil
}

func (r *ResourceApp) DoDelete(ctx context.Context, id string) error {
	_, err := r.client.Apps.DeleteByName(ctx, id)
	return err
}

func (r *ResourceApp) WaitAfterCreate(ctx context.Context, config *apps.App) (*apps.App, error) {
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
