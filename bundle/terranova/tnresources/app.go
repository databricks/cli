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
	config apps.App
}

func NewResourceApp(client *databricks.WorkspaceClient, config *resources.App) (*ResourceApp, error) {
	return &ResourceApp{
		client: client,
		config: config.App,
	}, nil
}

func (r *ResourceApp) Config() any {
	return r.config
}

func (r *ResourceApp) DoCreate(ctx context.Context) (string, error) {
	request := apps.CreateAppRequest{
		App:             r.config,
		NoCompute:       true,
		ForceSendFields: nil,
	}
	waiter, err := r.client.Apps.Create(ctx, request)
	if err != nil {
		return "", err
	}

	// TODO: Store waiter for Wait method

	return waiter.Response.Name, nil
}

func (r *ResourceApp) DoUpdate(ctx context.Context, id string) error {
	request := apps.UpdateAppRequest{
		App:  r.config,
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

func DeleteApp(ctx context.Context, client *databricks.WorkspaceClient, id string) error {
	// TODO: implement app deletion
	return nil
}

func (r *ResourceApp) WaitAfterCreate(ctx context.Context) error {
	_, err := r.waitForApp(ctx, r.client, r.config.Name)
	if err != nil {
		return err
	}
	return nil
}

func (r *ResourceApp) WaitAfterUpdate(ctx context.Context) error {
	// Intentional no-op
	return nil
}

// waitForApp waits for the app to reach the target state. The target state is either ACTIVE or STOPPED.
// Apps with no_compute set to true will reach the STOPPED state, otherwise they will reach the ACTIVE state.
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

// This is copied from the retries package of the databricks-sdk-go. It should be made public,
// but for now, I'm copying it here.
func shouldRetry(err error) bool {
	if err == nil {
		return false
	}
	e := err.(*retries.Err)
	if e == nil {
		return false
	}
	return !e.Halt
}
