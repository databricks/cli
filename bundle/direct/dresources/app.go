package dresources

import (
	"context"
	"errors"
	"fmt"
	"net/http"
	"slices"
	"strings"
	"time"

	"github.com/databricks/cli/bundle/appdeploy"
	"github.com/databricks/cli/bundle/config/resources"
	"github.com/databricks/cli/bundle/deployplan"
	"github.com/databricks/cli/libs/cmdio"
	"github.com/databricks/cli/libs/log"
	"github.com/databricks/cli/libs/retry"
	"github.com/databricks/cli/libs/structs/structpath"
	"github.com/databricks/databricks-sdk-go"
	"github.com/databricks/databricks-sdk-go/apierr"
	"github.com/databricks/databricks-sdk-go/marshal"
	"github.com/databricks/databricks-sdk-go/retries"
	"github.com/databricks/databricks-sdk-go/service/apps"
)

// AppState is the state type for App resources. source_code_path and git_source come from
// apps.App; config, lifecycle, thumbnail and source_code_digest are DABs-only additions
// persisted in state.
// thumbnail is the base64-encoded image; state keeps only its hash (hashed_fields in apps.yml).
// source_code_digest is the digest of the local files behind source_code_path as of the last
// app deployment the bundle made, set while lifecycle.started is true. A change to it is what
// plans an update, and with it a new app deployment, when only the source files changed.
type AppState struct {
	apps.App
	Config           *resources.AppConfig `json:"config,omitempty"`
	Lifecycle        *StateLifecycle      `json:"lifecycle,omitempty"`
	Thumbnail        string               `json:"thumbnail,omitempty"`
	SourceCodeDigest string               `json:"source_code_digest,omitempty"`
}

// AppRemote extends apps.App with the same DABs-only fields as AppState so they
// appear in RemoteType and can be used for $resource resolution and drift detection.
type AppRemote struct {
	apps.App
	Config    *resources.AppConfig `json:"config,omitempty"`
	Lifecycle *StateLifecycle      `json:"lifecycle,omitempty"`
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
		App:              input.App,
		Config:           input.Config,
		Lifecycle:        nil,
		Thumbnail:        input.ThumbnailContent,
		SourceCodeDigest: input.SourceCodeDigest,
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
		App:              remote.App,
		Config:           remote.Config,
		Lifecycle:        remote.Lifecycle,
		Thumbnail:        "",
		SourceCodeDigest: "",
	}
}

func (r *ResourceApp) DoRead(ctx context.Context, id string) (*AppRemote, error) {
	app, err := r.client.Apps.GetByName(ctx, id)
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
		// config (command/env) is not part of apps.App, so it is read back from the
		// active deployment for drift detection. source_code_path and git_source are
		// classified input_only, so their remote value is ignored (see
		// apps.generated.yml) and does not need populating here.
		remote.Config = deploymentToAppConfig(app.ActiveDeployment)
	}
	return remote, nil
}

// appRequestBody returns app for the Create/Update request body. An empty source_code_path
// is never sent: the SDK's JSON unmarshal (used when loading the plan) adds every present
// basic-type field to ForceSendFields, which would otherwise force "source_code_path": ""
// into the body, and the API rejects that (see #6401, fixed in #6404). ForceSendFields is
// cloned so the plan entry is not modified.
func appRequestBody(app apps.App) apps.App {
	if app.SourceCodePath == "" {
		app.ForceSendFields = slices.DeleteFunc(slices.Clone(app.ForceSendFields), func(s string) bool {
			return s == "SourceCodePath"
		})
	}
	return app
}

// appUpdateBody returns config.App for the Update request. UpdateApp rejects source_code_path
// (400 INVALID_PARAMETER_VALUE: invalid update mask), so it is sent on Create only and is not
// in UpdateMaskFields; a changed source_code_path reaches the app through the Deploy API
// (see manageLifecycle). git_source is accepted on Update and stays in the body.
func appUpdateBody(config *AppState) apps.App {
	app := config.App
	app.SourceCodePath = ""
	return appRequestBody(app)
}

func (r *ResourceApp) DoCreate(ctx context.Context, config *AppState) (string, *AppRemote, error) {
	// Start app compute only when lifecycle.started=true is explicit.
	// For nil (omitted) or false, use no_compute=true (do not start compute).
	noCompute := config.Lifecycle == nil || config.Lifecycle.Started == nil || !*config.Lifecycle.Started
	request := apps.CreateAppRequest{
		App:             appRequestBody(config.App),
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

	// The thumbnail is uploaded before the state is saved, so a failed upload is not recorded
	// and the next deploy retries it. Failing here would leave the new app out of the state.
	if config.Thumbnail != "" {
		if err := r.updateThumbnail(ctx, app.Name, config.Thumbnail); err != nil {
			cmdio.LogString(ctx, fmt.Sprintf("Warning: failed to upload the thumbnail of app %s, the next deploy retries it: %s", app.Name, err))
			config.Thumbnail = ""
		}
	}

	return app.Name, nil, nil
}

func (r *ResourceApp) updateThumbnail(ctx context.Context, name, thumbnail string) error {
	_, err := r.client.Apps.UpdateAppThumbnail(ctx, apps.UpdateAppThumbnailRequest{
		Name:         name,
		AppThumbnail: &apps.AppThumbnail{Thumbnail: thumbnail, ForceSendFields: nil},
	})
	return err
}

var appThumbnailPath = structpath.MustParsePath("thumbnail")

// UpdateMaskFields lists the App fields UpdateApp may change. source_code_path is absent
// because the API rejects it in the update mask; see appUpdateBody.
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
	"git_source",
	"telemetry_export_destinations",
}

// appUpdateMask returns the update mask for an app update request. compute_size is a
// backend default (see configs/apps.yml), so when the bundle does not declare it the field
// is left out of the mask. Naming it while the body omits it asks the backend to reset the
// size to its default on every deploy, even when the app was resized out of band.
func appUpdateMask(app *apps.App) string {
	fields := UpdateMaskFields
	if app.ComputeSize == "" {
		fields = slices.DeleteFunc(slices.Clone(UpdateMaskFields), func(f string) bool {
			return f == "compute_size"
		})
	}
	return strings.Join(fields, ",")
}

func (r *ResourceApp) DoUpdate(ctx context.Context, id string, config *AppState, entry *PlanEntry) (*AppRemote, error) {
	return nil, withMissingServicePrincipalHint(r.doUpdate(ctx, id, config, entry), id)
}

func (r *ResourceApp) doUpdate(ctx context.Context, id string, config *AppState, entry *PlanEntry) error {
	alreadyStarted := remoteIsStarted(entry)
	// git_source is sent as part of the App body (in the update mask). source_code_path and
	// config (command/env) are deploy-only on update and applied via manageLifecycle; see
	// appUpdateBody.
	if hasAppChanges(entry) {
		app := appUpdateBody(config)
		request := apps.AsyncUpdateAppRequest{
			App:        &app,
			AppName:    id,
			UpdateMask: appUpdateMask(&app),
		}
		var updateWaiter *apps.WaitGetUpdateAppSucceeded[apps.AppUpdate]
		settled, err := r.retryWhileComputeSettles(ctx, id, func() error {
			var err error
			updateWaiter, err = r.client.Apps.CreateUpdate(ctx, request)
			return err
		})
		if err != nil {
			return err
		}
		// The plan-time compute state is stale if the compute moved while we waited.
		if settled != nil {
			alreadyStarted = !isComputeStopped(settled)
		}

		response, err := updateWaiter.Get()
		if err != nil {
			return err
		}

		if response.Status.State != apps.AppUpdateUpdateStatusUpdateStateSucceeded {
			return fmt.Errorf("update status: %s: %s", response.Status.State, response.Status.Message)
		}
	}

	if entry.Changes.HasChange(appThumbnailPath) {
		var err error
		if config.Thumbnail == "" {
			err = r.client.Apps.DeleteAppThumbnail(ctx, apps.DeleteAppThumbnailRequest{Name: id})
		} else {
			err = r.updateThumbnail(ctx, id, config.Thumbnail)
		}
		if err != nil {
			return err
		}
	}

	return r.manageLifecycle(ctx, id, config, alreadyStarted)
}

// missingServicePrincipalMessage is part of the backend error for an app whose service
// principal is gone. The backend reports it as a generic 400 BAD_REQUEST or as a compute
// ERROR status message, so there is no error code or sentinel to match on. Such an app
// cannot be updated or started again, only recreated.
const missingServicePrincipalMessage = "service principal does not exist or is deleted"

func withMissingServicePrincipalHint(err error, name string) error {
	if err == nil || !strings.Contains(strings.ToLower(err.Error()), missingServicePrincipalMessage) {
		return err
	}
	return fmt.Errorf("%w\n\nThe app cannot be updated or started without its service principal. "+
		"To recreate the app, delete it with \"databricks apps delete %s\", wait for the deletion to finish, "+
		"and run \"databricks bundle deploy\" again, using the same profile and target as for this deploy. "+
		"Deleting the app also deletes its deployment history", err, name)
}

func (r *ResourceApp) manageLifecycle(ctx context.Context, id string, config *AppState, alreadyStarted bool) error {
	if config.Lifecycle == nil || config.Lifecycle.Started == nil {
		return nil
	}

	desiredStarted := *config.Lifecycle.Started
	if desiredStarted {
		// lifecycle.started=true: ensure the app compute is running and deploy the latest code.
		if !alreadyStarted {
			startedApp, err := appdeploy.Start(ctx, r.client, id, nil)
			if err != nil {
				return err
			}
			if err := appdeploy.WaitForDeploymentToComplete(ctx, r.client, startedApp); err != nil {
				return err
			}
		}
		// source_code_path is recorded on the app at Create and git_source at Create/Update,
		// but the Deploy API requires a deployment source, so the deployment carries them
		// too. This is also the only way a changed source_code_path reaches the app.
		deployment := appdeploy.BuildDeployment(config.SourceCodePath, config.Config, config.GitSource)
		if err := appdeploy.Deploy(ctx, r.client, id, deployment); err != nil {
			return err
		}
	} else {
		// lifecycle.started=false: ensure the app compute is stopped.
		if alreadyStarted {
			stopWaiter, err := r.client.Apps.Stop(ctx, apps.StopAppRequest{Name: id})
			if err != nil {
				return err
			}
			if _, err = stopWaiter.Get(); err != nil {
				return err
			}
		}
	}

	return nil
}

// hasAppChanges reports whether the plan entry contains any Update changes to fields that
// belong to the App Update API. source_code_path and config (command/env) are deploy-only
// on update and applied via manageLifecycle; lifecycle drives compute start/stop.
// source_code_digest is not an API field: its change only asks manageLifecycle for a new
// deployment of the changed source files.
func hasAppChanges(entry *PlanEntry) bool {
	return entry.Changes.HasChangeExcept("source_code_path", "config", "lifecycle", "lifecycle.started", "thumbnail", "source_code_digest")
}

// OverrideChangeDesc skips drift on config (command/env) while the app has no active
// deployment. config is read only from the active deployment (DoRead), so before the
// first deploy (or once a stop clears it) the remote side is empty and the diff is
// spurious; it applies on the next start (manageLifecycle).
func (*ResourceApp) OverrideChangeDesc(_ context.Context, path *structpath.PathNode, change *ChangeDesc, remote *AppRemote) error {
	// Prefix(1) so a nested diff (e.g. config.command) matches its top-level field.
	if path.Prefix(1).String() == "config" {
		if remote.ActiveDeployment == nil {
			change.Action = deployplan.Skip
			change.Reason = "no active deployment"
		}
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

func (r *ResourceApp) DoDelete(ctx context.Context, id string, _ *AppState) error {
	_, err := r.retryWhileComputeSettles(ctx, id, func() error {
		_, err := r.client.Apps.DeleteByName(ctx, id)
		return err
	})
	if errors.Is(err, errAppGoneWhileSettling) {
		return nil
	}
	return err
}

// computeUpdatingErrorMessages are the 400 BAD_REQUEST messages the Apps API returns for an
// update or delete while the compute is UPDATING, as seen in cloud test runs:
//   - "Cannot update app X as its compute is in UPDATING state. App compute needs to be ACTIVE or STOPPED to update."
//   - "Cannot delete app X as it is not terminal with state UPDATING, and was updated less than 20 minutes ago. ..."
//
// The DELETING variant of the delete message is left out: IsGone and deleteConfirmedGone
// handle a delete of a DELETING app, and waiting would not help.
var computeUpdatingErrorMessages = []string{
	"compute is in UPDATING state",
	"not terminal with state UPDATING,",
}

func isComputeUpdatingError(err error) bool {
	apiErr, ok := errors.AsType[*apierr.APIError](err)
	if !ok || apiErr.StatusCode != http.StatusBadRequest {
		return false
	}
	return slices.ContainsFunc(computeUpdatingErrorMessages, func(m string) bool {
		return strings.Contains(apiErr.Message, m)
	})
}

// errAppGoneWhileSettling reports that the app disappeared while waiting for its compute.
var errAppGoneWhileSettling = errors.New("app is gone")

// computeSettleTimeout bounds one wait for the app compute to leave a transitional state.
const computeSettleTimeout = 20 * time.Minute

// retryWhileComputeSettles calls fn and, if the Apps API rejected it because the compute is
// UPDATING, waits for the compute to settle and calls fn again (see retry.With for the attempt
// limit and the pause between attempts). The compute can enter UPDATING on its own shortly
// after a create or update returned. It returns the app as last read while waiting, or nil if
// it did not wait, so a caller can refresh what it knew about the compute state.
//
// If the wait itself fails, the error stays the API's 400 for callers that classify it (for
// example retry.IsTransient in apply.go), with the wait failure only in the text. If the app
// is gone, the error wraps errAppGoneWhileSettling and the 404.
func (r *ResourceApp) retryWhileComputeSettles(ctx context.Context, name string, fn func() error) (*apps.App, error) {
	var settled *apps.App
	var waitErr error
	_, err := retry.With(ctx, func(ctx context.Context, err error) bool {
		if !isComputeUpdatingError(err) {
			return false
		}
		log.Warnf(ctx, "%s", computeUpdatingRetryMessage(name))
		settled, waitErr = r.waitForComputeToSettle(ctx, name)
		return waitErr == nil
	}, func() (struct{}, error) {
		return struct{}{}, fn()
	})
	if err != nil && waitErr != nil {
		if apierr.IsMissing(waitErr) {
			return nil, fmt.Errorf("%w: %w", errAppGoneWhileSettling, waitErr)
		}
		return nil, fmt.Errorf("%w (waiting for the app compute to settle failed: %s)", err, waitErr.Error())
	}
	return settled, err
}

// computeUpdatingRetryMessage is the warning logged before waiting for the compute. Like
// libs/retry, it says "retrying after" so that acceptance tests suppress it
// (SuppressLines.warn_retry): whether a cloud run hits the race is timing dependent.
func computeUpdatingRetryMessage(name string) string {
	return fmt.Sprintf("app %s compute is UPDATING: retrying after it settles (waiting up to %s)", name, computeSettleTimeout)
}

func isComputeTransitional(app *apps.App) bool {
	if app.ComputeStatus == nil {
		return false
	}
	switch app.ComputeStatus.State {
	case apps.ComputeStateUpdating, apps.ComputeStateStarting, apps.ComputeStateStopping:
		return true
	default:
		return false
	}
}

// waitForComputeToSettle polls the app until its compute leaves a transitional state.
func (r *ResourceApp) waitForComputeToSettle(ctx context.Context, name string) (*apps.App, error) {
	retrier := retries.New[apps.App](retries.WithTimeout(computeSettleTimeout), retries.WithRetryFunc(shouldRetry))
	return retrier.Run(ctx, func(ctx context.Context) (*apps.App, error) {
		app, err := r.client.Apps.GetByName(ctx, name)
		if err != nil {
			return nil, retries.Halt(err)
		}
		if isComputeTransitional(app) {
			return nil, retries.Continues(fmt.Sprintf("app compute is %s", app.ComputeStatus.State))
		}
		return app, nil
	})
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
	return remote.ComputeStatus != nil && remote.ComputeStatus.State == apps.ComputeStateDeleting
}

func (r *ResourceApp) WaitAfterCreate(ctx context.Context, id string, config *AppState) (*AppRemote, error) {
	remote, err := r.waitAfterCreate(ctx, config)
	return remote, withMissingServicePrincipalHint(err, config.Name)
}

func (r *ResourceApp) waitAfterCreate(ctx context.Context, config *AppState) (*AppRemote, error) {
	remote, err := r.waitForApp(ctx, r.client, config.Name)
	if err != nil {
		return nil, err
	}
	alreadyStarted := remote.Lifecycle != nil && remote.Lifecycle.Started != nil && *remote.Lifecycle.Started
	if err := r.manageLifecycle(ctx, config.Name, config, alreadyStarted); err != nil {
		return nil, err
	}
	return remote, nil
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
		App:       *app,
		Config:    nil,
		Lifecycle: &StateLifecycle{Started: &started},
	}
	if app.ActiveDeployment != nil {
		remote.Config = deploymentToAppConfig(app.ActiveDeployment)
	}
	return remote, nil
}
