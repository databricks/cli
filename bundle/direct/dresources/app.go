package dresources

import (
	"context"
	"errors"
	"fmt"
	"slices"
	"strings"
	"time"

	"github.com/databricks/cli/bundle/appdeploy"
	"github.com/databricks/cli/bundle/config/resources"
	"github.com/databricks/cli/bundle/deployplan"
	"github.com/databricks/databricks-sdk-go"
	"github.com/databricks/databricks-sdk-go/apierr"
	"github.com/databricks/databricks-sdk-go/marshal"
	"github.com/databricks/databricks-sdk-go/retries"
	"github.com/databricks/databricks-sdk-go/service/apps"
)

// AppStateLifecycle holds lifecycle settings persisted in state.
type AppStateLifecycle struct {
	Started *bool `json:"started,omitempty"`
}

// AppState is the state type for App resources. It extends apps.App with fields
// needed for app deployments (Apps.Deploy) that are not part of the remote state.
type AppState struct {
	apps.App
	SourceCodePath string               `json:"source_code_path,omitempty"`
	Config         *resources.AppConfig `json:"config,omitempty"`
	GitSource      *apps.GitSource      `json:"git_source,omitempty"`
	Lifecycle      *AppStateLifecycle   `json:"lifecycle,omitempty"`
}

// AppRemote extends apps.App with lifecycle.started so that it appears in
// RemoteType and can be used for $resource resolution.
type AppRemote struct {
	apps.App
	Lifecycle *AppStateLifecycle `json:"lifecycle,omitempty"`
}

// Custom marshalers needed because embedded apps.App has its own MarshalJSON
// which would otherwise take over and ignore the additional fields.
func (s *AppState) UnmarshalJSON(b []byte) error {
	return marshal.Unmarshal(b, s)
}

func (s AppState) MarshalJSON() ([]byte, error) {
	return marshal.Marshal(s)
}

func (r *AppRemote) UnmarshalJSON(b []byte) error {
	return marshal.Unmarshal(b, r)
}

func (r AppRemote) MarshalJSON() ([]byte, error) {
	return marshal.Marshal(r)
}

type ResourceApp struct {
	client *databricks.WorkspaceClient
}

func (*ResourceApp) New(client *databricks.WorkspaceClient) *ResourceApp {
	return &ResourceApp{client: client}
}

func (*ResourceApp) PrepareState(input *resources.App) *AppState {
	s := &AppState{
		App:            input.App,
		SourceCodePath: input.SourceCodePath,
		Config:         input.Config,
		GitSource:      input.GitSource,
		Lifecycle:      nil,
	}
	if input.Lifecycle.Started != nil {
		s.Lifecycle = &AppStateLifecycle{Started: input.Lifecycle.Started}
	}
	return s
}

// RemapState maps the remote AppRemote to AppState for diff comparison.
// Deploy-only fields (SourceCodePath, Config, GitSource) are not in remote state,
// so they default to zero values, which prevents false drift detection.
// Started is derived from compute status so the planner can detect start/stop changes.
func (*ResourceApp) RemapState(remote *AppRemote) *AppState {
	started := !isComputeStopped(&remote.App)
	return &AppState{
		App:            remote.App,
		SourceCodePath: "",
		Config:         nil,
		GitSource:      nil,
		Lifecycle:      &AppStateLifecycle{Started: &started},
	}
}

func (r *ResourceApp) DoRead(ctx context.Context, id string) (*AppRemote, error) {
	app, err := r.client.Apps.GetByName(ctx, id)
	if err != nil {
		return nil, err
	}
	started := !isComputeStopped(app)
	return &AppRemote{App: *app, Lifecycle: &AppStateLifecycle{Started: &started}}, nil
}

func (r *ResourceApp) DoCreate(ctx context.Context, config *AppState) (string, *AppRemote, error) {
	// Start app compute only when lifecycle.started=true is explicit.
	// For nil (omitted) or false, use no_compute=true (do not start compute).
	noCompute := config.Lifecycle == nil || config.Lifecycle.Started == nil || !*config.Lifecycle.Started
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

func (r *ResourceApp) DoUpdate(ctx context.Context, id string, config *AppState, entry *PlanEntry) (*AppRemote, error) {
	// Build update mask excluding local-only fields that have no counterpart in the API.
	var maskPaths []string
	for path, change := range entry.Changes {
		if change.Action == deployplan.Update && !localOnlyFields[path] {
			maskPaths = append(maskPaths, path)
		}
	}
	slices.Sort(maskPaths)
	updateMask := strings.Join(maskPaths, ",")

	if updateMask != "" {
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
	}

	if config.Lifecycle == nil || config.Lifecycle.Started == nil {
		return nil, nil
	}

	// The planner computes the remote started value in RemapState based on compute status,
	// so changes["lifecycle.started"].Action == Update means the compute state differs from the desired state.
	startedChange := entry.Changes["lifecycle.started"]

	if *config.Lifecycle.Started {
		// lifecycle.started=true: ensure the app compute is running and deploy the latest code.
		if startedChange != nil && startedChange.Action == deployplan.Update {
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
		if startedChange != nil && startedChange.Action == deployplan.Update {
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
	"source_code_path":  true,
	"config":            true,
	"git_source":        true,
	"lifecycle":         true,
	"lifecycle.started": true,
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

func (r *ResourceApp) WaitAfterCreate(ctx context.Context, config *AppState) (*AppRemote, error) {
	return r.waitForApp(ctx, r.client, config.Name)
}

// waitForApp waits for the app to reach the target state. The target state is either ACTIVE or STOPPED.
// Apps with no_compute set to true will reach the STOPPED state, otherwise they will reach the ACTIVE state.
// We can't use the default waiter from SDK because it only waits on ACTIVE state but we need also STOPPED state.
// Ideally this should be done in Go SDK but currently only ACTIVE is marked as terminal state
// so this would need to be addressed by Apps service team first in their proto.
func (r *ResourceApp) waitForApp(ctx context.Context, w *databricks.WorkspaceClient, name string) (*AppRemote, error) {
	retrier := retries.New[apps.App](retries.WithTimeout(-1), retries.WithRetryFunc(shouldRetry))
	app, err := retrier.Run(ctx, func(ctx context.Context) (*apps.App, error) {
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
	if err != nil {
		return nil, err
	}
	started := !isComputeStopped(app)
	return &AppRemote{App: *app, Lifecycle: &AppStateLifecycle{Started: &started}}, nil
}
