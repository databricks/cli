package tnresources

import (
	"context"
	"fmt"

	"github.com/databricks/cli/bundle/config/resources"
	"github.com/databricks/cli/libs/log"
	"github.com/databricks/databricks-sdk-go"
	"github.com/databricks/databricks-sdk-go/retries"
	"github.com/databricks/databricks-sdk-go/service/apps"
)

type ResourceApp struct {
	client *databricks.WorkspaceClient
}

func (*ResourceApp) New(client *databricks.WorkspaceClient) *ResourceApp {
	return &ResourceApp{client: client}
}

func (*ResourceApp) PrepareConfig(input *resources.App) *apps.App {
	return &input.App
}

func (r *ResourceApp) DoRefresh(ctx context.Context, id string) (*apps.App, error) {
	return r.client.Apps.GetByName(ctx, id)
}

func (r *ResourceApp) DoCreate(ctx context.Context, config *apps.App) (string, error) {
	request := apps.CreateAppRequest{
		App:             *config,
		NoCompute:       true,
		ForceSendFields: nil,
	}
	waiter, err := r.client.Apps.Create(ctx, request)
	if err != nil {
		return "", err
	}
	return waiter.Response.Name, nil
}

func (r *ResourceApp) DoUpdate(ctx context.Context, id string, config *apps.App) error {
	request := apps.UpdateAppRequest{
		App:  *config,
		Name: id,
	}
	response, err := r.client.Apps.Update(ctx, request)
	if err != nil {
		return err
	}
	if response.Name != id {
		log.Warnf(ctx, "apps: response contains unexpected name=%#v (expected %#v)", response.Name, id)
	}
	return nil
}

func (r *ResourceApp) DoDelete(ctx context.Context, id string) error {
	_, err := r.client.Apps.DeleteByName(ctx, id)
	return err
}

func (*ResourceApp) RecreateFields() []string {
	return []string{
		".name",
	}
}

func (r *ResourceApp) WaitAfterCreate(ctx context.Context, config *apps.App) error {
	_, err := r.waitForApp(ctx, r.client, config.Name)
	return err
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
