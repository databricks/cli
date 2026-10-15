package dresources

import (
	"context"
	"fmt"
	"strings"
	"time"

	"github.com/databricks/cli/libs/apierrs"

	"github.com/databricks/cli/bundle/appdeploy"
	"github.com/databricks/cli/bundle/config/resources"
	"github.com/databricks/cli/bundle/deployplan"
	"github.com/databricks/cli/bundle/direct/internal/retries"
	"github.com/databricks/cli/libs/sdkclients"
	apps "github.com/databricks/cli/libs/sdktypes/apps/v1"
	"github.com/databricks/cli/libs/structs/structpath"
	"github.com/databricks/cli/libs/utils"
	sdkapps "github.com/databricks/sdk-go/apps/v1"
)

// AppState is the state type for App resources. source_code_path and git_source come from
// apps.App; config and lifecycle are DABs-only additions persisted in state.
type AppState struct {
	apps.App
	Config    *resources.AppConfig `json:"config,omitempty"`
	Lifecycle *StateLifecycle      `json:"lifecycle,omitempty"`
}

// AppRemote extends apps.App with the same DABs-only fields as AppState so they
// appear in RemoteType and can be used for $resource resolution and drift detection.
type AppRemote struct {
	apps.App
	Config    *resources.AppConfig `json:"config,omitempty"`
	Lifecycle *StateLifecycle      `json:"lifecycle,omitempty"`
}

type ResourceApp struct {
	client *sdkclients.Clients
}

func (*ResourceApp) New(client *sdkclients.Clients) *ResourceApp {
	return &ResourceApp{client: client}
}

func (*ResourceApp) PrepareState(input *resources.App) *AppState {
	s := &AppState{
		App:       input.App,
		Config:    input.Config,
		Lifecycle: nil,
	}
	if input.Lifecycle != nil && input.Lifecycle.Started != nil {
		s.Lifecycle = &StateLifecycle{Started: input.Lifecycle.Started}
	}
	return s
}

// RemapState maps the remote AppRemote to AppState for diff comparison.
// DoRead populates config, git_source, and source_code_path from the active
// deployment when one exists, enabling drift detection for out-of-band redeploys.
// Started is derived from compute status so the planner can detect start/stop changes.
func (*ResourceApp) RemapState(remote *AppRemote) *AppState {
	return &AppState{
		App:       remote.App,
		Config:    remote.Config,
		Lifecycle: remote.Lifecycle,
	}
}

func (r *ResourceApp) getApp(ctx context.Context, name string) (*sdkapps.App, error) {
	return r.client.Apps.GetApp(ctx, sdkapps.GetAppRequest{Name: new(name)})
}

func (r *ResourceApp) DoRead(ctx context.Context, id string) (*AppRemote, error) {
	app, err := r.getApp(ctx, id)
	if err != nil {
		return nil, err
	}
	return makeAppRemote(app)
}

func makeAppRemote(resp *sdkapps.App) (*AppRemote, error) {
	app, err := apps.AppFromSDK(resp)
	if err != nil {
		return nil, err
	}
	started := !isComputeStopped(app)
	remote := &AppRemote{
		App:       *app,
		Config:    nil,
		Lifecycle: &StateLifecycle{Started: &started},
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

// appRequestBody returns config.App with the deploy-only fields cleared. source_code_path
// and git_source are part of apps.App, but DABs applies them through the Deploy API (see
// manageLifecycle), so they must not ride along in create/update bodies.
func appRequestBody(config *AppState) *apps.App {
	app := config.App
	app.SourceCodePath = nil
	app.GitSource = nil
	return &app
}

func (r *ResourceApp) DoCreate(ctx context.Context, config *AppState) (string, *AppRemote, error) {
	// Start app compute only when lifecycle.started=true is explicit.
	// For nil (omitted) or false, use no_compute=true (do not start compute).
	noCompute := config.Lifecycle == nil || config.Lifecycle.Started == nil || !*config.Lifecycle.Started
	request, err := apps.CreateAppRequestToSDK(&apps.CreateAppRequest{
		App:       appRequestBody(config),
		NoCompute: new(noCompute),
	})
	if err != nil {
		return "", nil, err
	}

	name := utils.Deref(config.Name)
	retrier := retries.New[sdkapps.App](retries.WithTimeout(15*time.Minute), retries.WithRetryFunc(shouldRetry))
	_, err = retrier.Run(ctx, func(ctx context.Context) (*sdkapps.App, error) {
		_, err := r.client.Apps.CreateApp(ctx, *request)
		if err != nil {
			if apierrs.Is(err, apierrs.ErrResourceAlreadyExists) {
				// Check if the app is in DELETING state - only then should we retry
				existingApp, getErr := r.getApp(ctx, name)
				if getErr != nil {
					// If we can't get the app (e.g., it was just deleted), retry the create
					if apierrs.IsMissing(getErr) {
						return nil, retries.Continues("app was deleted, retrying create")
					}
					return nil, retries.Halt(err)
				}
				if existingApp.ComputeStatus != nil && existingApp.ComputeStatus.State == sdkapps.ComputeStatus_ComputeState_Deleting {
					return nil, retries.Continues("app is deleting, retrying create")
				}
				// App exists and is not being deleted - this is a hard error
				return nil, retries.Halt(err)
			}
			return nil, retries.Halt(err)
		}
		return nil, nil
	})
	if err != nil {
		return "", nil, err
	}

	return name, nil, nil
}

var UpdateMaskFields = []string{
	"description",
	"budget_policy_id",
	"usage_policy_id",
	"resources",
	"user_api_scopes",
	"compute_size",
	"compute_min_instances",
	"compute_max_instances",
	"git_repository",
	"telemetry_export_destinations",
}

var updateMask = strings.Join(UpdateMaskFields, ",")

func (r *ResourceApp) DoUpdate(ctx context.Context, id string, config *AppState, entry *PlanEntry) (*AppRemote, error) {
	// Deploy-only fields (source_code_path, config, git_source, lifecycle) are excluded
	// from the request body; see appRequestBody.
	if hasAppChanges(entry) {
		request, err := apps.AsyncUpdateAppRequestToSDK(&apps.AsyncUpdateAppRequest{
			App:        appRequestBody(config),
			AppName:    new(id),
			UpdateMask: new(updateMask),
		})
		if err != nil {
			return nil, err
		}
		updateWaiter, err := r.client.Apps.AsyncUpdateApp(ctx, *request)
		if err != nil {
			return nil, err
		}

		response, err := updateWaiter.Wait(ctx)
		if err != nil {
			return nil, err
		}

		var state sdkapps.AppUpdate_UpdateStatus_UpdateState
		var message string
		if response.Status != nil {
			state = response.Status.State
			message = utils.Deref(response.Status.Message)
		}
		if state != sdkapps.AppUpdate_UpdateStatus_UpdateState_Succeeded {
			return nil, fmt.Errorf("update status: %s: %s", state, message)
		}
	}

	return nil, r.manageLifecycle(ctx, id, config, remoteIsStarted(entry))
}

func (r *ResourceApp) manageLifecycle(ctx context.Context, id string, config *AppState, alreadyStarted bool) error {
	if config.Lifecycle == nil || config.Lifecycle.Started == nil {
		return nil
	}

	desiredStarted := *config.Lifecycle.Started
	if desiredStarted {
		// lifecycle.started=true: ensure the app compute is running and deploy the latest code.
		if !alreadyStarted {
			startWaiter, err := r.client.Apps.StartApp(ctx, sdkapps.StartAppRequest{Name: new(id)})
			if err != nil {
				return err
			}
			startedApp, err := startWaiter.Wait(ctx)
			if err != nil {
				return err
			}
			if err := appdeploy.WaitForDeploymentToComplete(ctx, r.client, startedApp); err != nil {
				return err
			}
		}
		deployment, err := appdeploy.BuildDeployment(utils.Deref(config.SourceCodePath), config.Config, config.GitSource)
		if err != nil {
			return err
		}
		if err := appdeploy.Deploy(ctx, r.client, id, deployment); err != nil {
			return err
		}
	} else {
		// lifecycle.started=false: ensure the app compute is stopped.
		if alreadyStarted {
			stopWaiter, err := r.client.Apps.StopApp(ctx, sdkapps.StopAppRequest{Name: new(id)})
			if err != nil {
				return err
			}
			if _, err = stopWaiter.Wait(ctx); err != nil {
				return err
			}
		}
	}

	return nil
}

// hasAppChanges reports whether the plan entry contains any Update changes
// to fields that belong to the App Update API (i.e., not deploy-only fields).
func hasAppChanges(entry *PlanEntry) bool {
	return entry.Changes.HasChangeExcept("source_code_path", "config", "git_source", "lifecycle", "lifecycle.started")
}

// OverrideChangeDesc skips drift on the deploy-only fields (source_code_path, config,
// git_source) while the app has no active deployment. DoRead reads them only from the
// active deployment, so before the first deploy (or once a stop clears it) the remote
// side is empty and the diff is spurious; it applies on the next start (manageLifecycle).
func (*ResourceApp) OverrideChangeDesc(_ context.Context, path *structpath.PathNode, change *ChangeDesc, remote *AppRemote) error {
	// Prefix(1) so a nested diff (e.g. config.command) matches its top-level field.
	switch path.Prefix(1).String() {
	case "source_code_path", "config", "git_source":
		if remote.ActiveDeployment == nil {
			change.Action = deployplan.Skip
			change.Reason = "no active deployment"
		}
	}
	return nil
}

func isComputeStopped(app *apps.App) bool {
	return app.ComputeStatus == nil ||
		app.ComputeStatus.State == apps.ComputeStatus_ComputeState_Stopped ||
		app.ComputeStatus.State == apps.ComputeStatus_ComputeState_Error
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
				Name:      utils.Deref(ev.Name),
				Value:     utils.Deref(ev.Value),
				ValueFrom: utils.Deref(ev.ValueFrom),
			}
		}
	}
	return config
}

func (r *ResourceApp) DoDelete(ctx context.Context, id string, _ *AppState) error {
	_, err := r.client.Apps.DeleteApp(ctx, sdkapps.DeleteAppRequest{Name: new(id)})
	return err
}

// IsGone treats a DELETING app as already-deleted. The Apps DELETE is
// fire-and-forget: it returns success while the app sits in
// ComputeState=DELETING for up to ~20 minutes, and a GET during that window
// returns the app (not 404), so callers cannot rely on IsMissing. A second
// DELETE while DELETING is rejected with 400, so without this the delete/destroy
// path is not idempotent (acceptance/bundle/invariant/{delete,destroy}_idempotent).
// Consulted both at plan time (bundle_plan.go) and after a failed apply-time
// delete (apply.go deleteConfirmedGone), which covers saved-plan deploys where
// the app enters DELETING only after the plan was computed.
// Still uncovered: a DELETING app hit during a normal deploy/update (not a
// delete) produces a spurious update/skip because plan/apply do not special-case it there.
func (*ResourceApp) IsGone(remote *AppRemote) bool {
	return remote.ComputeStatus != nil && remote.ComputeStatus.State == apps.ComputeStatus_ComputeState_Deleting
}

func (r *ResourceApp) WaitAfterCreate(ctx context.Context, id string, config *AppState) (*AppRemote, error) {
	name := utils.Deref(config.Name)
	remote, err := r.waitForApp(ctx, name)
	if err != nil {
		return nil, err
	}
	alreadyStarted := remote.Lifecycle != nil && remote.Lifecycle.Started != nil && *remote.Lifecycle.Started
	if err := r.manageLifecycle(ctx, name, config, alreadyStarted); err != nil {
		return nil, err
	}
	return remote, nil
}

// waitForApp waits for the app to reach the target state. The target state is either ACTIVE or STOPPED.
// Apps with no_compute set to true will reach the STOPPED state, otherwise they will reach the ACTIVE state.
// We can't use the default waiter from SDK because it only waits on ACTIVE state but we need also STOPPED state.
// Ideally this should be done in Go SDK but currently only ACTIVE is marked as terminal state
// so this would need to be addressed by Apps service team first in their proto.
func (r *ResourceApp) waitForApp(ctx context.Context, name string) (*AppRemote, error) {
	retrier := retries.New[sdkapps.App](retries.WithTimeout(-1), retries.WithRetryFunc(shouldRetry))
	app, err := retrier.Run(ctx, func(ctx context.Context) (*sdkapps.App, error) {
		app, err := r.getApp(ctx, name)
		if err != nil {
			return nil, retries.Halt(err)
		}
		var status sdkapps.ComputeStatus_ComputeState
		var statusMessage string
		if app.ComputeStatus != nil {
			status = app.ComputeStatus.State
			statusMessage = utils.Deref(app.ComputeStatus.Message)
		}
		switch status {
		case sdkapps.ComputeStatus_ComputeState_Active, sdkapps.ComputeStatus_ComputeState_Stopped:
			return app, nil
		case sdkapps.ComputeStatus_ComputeState_Error:
			err := fmt.Errorf("failed to reach %s or %s, got %s: %s",
				sdkapps.ComputeStatus_ComputeState_Active, sdkapps.ComputeStatus_ComputeState_Stopped, status, statusMessage)
			return nil, retries.Halt(err)
		default:
			return nil, retries.Continues(statusMessage)
		}
	})
	if err != nil {
		return nil, err
	}
	return makeAppRemote(app)
}
