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
	"github.com/databricks/cli/libs/structs/structpath"
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

// AppState is the state type for App resources. It extends apps.App with deployment-related
// fields (source_code_path, config, git_source, lifecycle) that are persisted in state.
type AppState struct {
	apps.App
	SourceCodePath string               `json:"source_code_path,omitempty"`
	Config         *resources.AppConfig `json:"config,omitempty"`
	GitSource      *apps.GitSource      `json:"git_source,omitempty"`
	Lifecycle      *AppStateLifecycle   `json:"lifecycle,omitempty"`
}

// AppRemote extends apps.App with the same deployment fields as AppState so they
// appear in RemoteType and can be used for $resource resolution and drift detection.
type AppRemote struct {
	apps.App
	SourceCodePath string               `json:"source_code_path,omitempty"`
	Config         *resources.AppConfig `json:"config,omitempty"`
	GitSource      *apps.GitSource      `json:"git_source,omitempty"`
	Lifecycle      *AppStateLifecycle   `json:"lifecycle,omitempty"`
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
	if input.Lifecycle != nil && input.Lifecycle.Started != nil {
		s.Lifecycle = &AppStateLifecycle{Started: input.Lifecycle.Started}
	}
	return s
}

// RemapState maps the remote AppRemote to AppState for diff comparison.
// Config, GitSource, and SourceCodePath are populated from the active deployment
// when one exists, enabling drift detection for out-of-band redeploys.
// Started is derived from compute status so the planner can detect start/stop changes.
func (*ResourceApp) RemapState(remote *AppRemote) *AppState {
	started := !isComputeStopped(&remote.App)
	return &AppState{
		App:            remote.App,
		SourceCodePath: remote.SourceCodePath,
		Config:         remote.Config,
		GitSource:      remote.GitSource,
		Lifecycle:      &AppStateLifecycle{Started: &started},
	}
}

func (r *ResourceApp) DoRead(ctx context.Context, id string) (*AppRemote, error) {
	app, err := r.client.Apps.GetByName(ctx, id)
	if err != nil {
		return nil, err
	}
	started := !isComputeStopped(app)
	remote := &AppRemote{
		App:            *app,
		Config:         nil,
		GitSource:      nil,
		SourceCodePath: "",
		Lifecycle:      &AppStateLifecycle{Started: &started},
	}
	if app.ActiveDeployment != nil {
		// The source code path in active deployment is snapshotted version of the source code path in the app.
		// We need to use the default source code path to get the correct source code path for drift detection.
		remote.SourceCodePath = app.DefaultSourceCodePath
		remote.GitSource = app.ActiveDeployment.GitSource
		remote.Config = deploymentToAppConfig(app.ActiveDeployment)
	}
	return remote, nil
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
	// Deploy-only fields (source_code_path, config,
	// git_source, lifecycle) are not part of apps.App and thus excluded from the request body.
	if hasAppChanges(entry) {
		fieldPaths := collectUpdatePathsWithPrefix(entry.Changes, "")
		slices.Sort(fieldPaths)
		for i, fieldPath := range fieldPaths {
			fieldPaths[i] = truncateAtIndex(fieldPath)
		}
		updateMask := strings.Join(fieldPaths, ",")
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
			return nil, fmt.Errorf("update status: %s: %s", response.Status.State, response.Status.Message)
		}
	}

	if config.Lifecycle == nil || config.Lifecycle.Started == nil {
		return nil, nil
	}

	desiredStarted := *config.Lifecycle.Started
	remoteStarted := remoteIsStarted(entry)

	if desiredStarted {
		// lifecycle.started=true: ensure the app compute is running and deploy the latest code.
		if !remoteStarted {
			startWaiter, err := r.client.Apps.Start(ctx, apps.StartAppRequest{Name: id})
			if err != nil {
				return nil, err
			}
			startedApp, err := startWaiter.Get()
			if err != nil {
				return nil, err
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
		if remoteStarted {
			stopWaiter, err := r.client.Apps.Stop(ctx, apps.StopAppRequest{Name: id})
			if err != nil {
				return nil, err
			}
			if _, err = stopWaiter.Get(); err != nil {
				return nil, err
			}
		}
	}

	return nil, nil
}

// deployOnlyFields are AppState fields managed via the Deploy API, not the App Update API.
// They have remote counterparts (populated from active deployment and compute status),
// but must not appear in the App update_mask.
var deployOnlyFields = map[string]bool{
	"source_code_path":  true,
	"config":            true,
	"git_source":        true,
	"lifecycle":         true,
	"lifecycle.started": true,
}

// hasAppChanges reports whether the plan entry contains any Update changes
// to fields that belong to the App Update API (i.e., not deploy-only fields).
func hasAppChanges(entry *PlanEntry) bool {
	for path, change := range entry.Changes {
		if change.Action == deployplan.Update && !deployOnlyFields[truncateAtIndex(path)] {
			return true
		}
	}
	return false
}

// OverrideChangeDesc skips source_code_path drift when the remote value is empty.
// This happens when an app has no deployment yet (DefaultSourceCodePath is unset).
func (*ResourceApp) OverrideChangeDesc(_ context.Context, path *structpath.PathNode, change *ChangeDesc, remote *AppRemote) error {
	if path.String() == "source_code_path" && remote.SourceCodePath == "" {
		change.Action = deployplan.Skip
		change.Reason = "no deployment"
	}
	return nil
}

func isComputeStopped(app *apps.App) bool {
	return app.ComputeStatus == nil ||
		app.ComputeStatus.State == apps.ComputeStateStopped ||
		app.ComputeStatus.State == apps.ComputeStateError
}

// remoteIsStarted reads the compute started state from the plan entry's remote state.
func remoteIsStarted(entry *PlanEntry) bool {
	if entry.RemoteState == nil {
		return false
	}
	remote, ok := entry.RemoteState.(*AppRemote)
	if !ok || remote.Lifecycle == nil || remote.Lifecycle.Started == nil {
		return false
	}
	return *remote.Lifecycle.Started
}

// deploymentToAppConfig extracts an AppConfig from an active deployment.
// Returns nil if the deployment has no command or env vars.
func deploymentToAppConfig(d *apps.AppDeployment) *resources.AppConfig {
	if len(d.Command) == 0 && len(d.EnvVars) == 0 {
		return nil
	}
	config := &resources.AppConfig{
		Command: d.Command,
		Env:     nil,
	}
	if len(d.EnvVars) > 0 {
		config.Env = make([]resources.AppEnvVar, len(d.EnvVars))
		for i, ev := range d.EnvVars {
			config.Env[i] = resources.AppEnvVar{
				Name:      ev.Name,
				Value:     ev.Value,
				ValueFrom: ev.ValueFrom,
			}
		}
	}
	return config
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
	remote := &AppRemote{
		App:            *app,
		Config:         nil,
		GitSource:      nil,
		SourceCodePath: "",
		Lifecycle:      &AppStateLifecycle{Started: &started},
	}
	if app.ActiveDeployment != nil {
		remote.SourceCodePath = app.DefaultSourceCodePath
		remote.GitSource = app.ActiveDeployment.GitSource
		remote.Config = deploymentToAppConfig(app.ActiveDeployment)
	}
	return remote, nil
}
