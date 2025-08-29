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

	// config is fully resolved snapshot of configuration of the resource; this is what is going to be used for create / update
	// it is populated once by NewResource*
	config apps.App

	// remoteState is remote view of the resource, fetched from the backend. It is updated by all *Refresh methods.
	// It is always a pointer as it's a nil for new resource; it may be different type from config.
	remoteState *apps.App
}

func NewResourceApp(client *databricks.WorkspaceClient, config *resources.App) (*ResourceApp, error) {
	return &ResourceApp{
		client:      client,
		config:      config.App,
		remoteState: nil,
	}, nil
}

func (r *ResourceApp) Config() any {
	return r.config
}

func (r *ResourceApp) RemoteState() any {
	return r.remoteState
}

func (r *ResourceApp) DoRefresh(ctx context.Context, id string) error {
	app, err := r.client.Apps.GetByName(ctx, id)
	if err != nil {
		return err
	}
	r.remoteState = app
	return nil
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
	_, err := client.Apps.DeleteByName(ctx, id)
	return err
}

func (r *ResourceApp) WaitAfterCreateWithRefresh(ctx context.Context) error {
	remoteState, err := r.waitForApp(ctx, r.client, r.config.Name)
	if err != nil {
		return err
	}
	r.remoteState = remoteState
	return nil
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
