package aircmd

import (
	"context"
	"errors"
	"fmt"
	"net/http"
	"slices"
	"time"

	"github.com/databricks/cli/libs/auth"
	"github.com/databricks/cli/libs/retry"
	"github.com/databricks/databricks-sdk-go"
	"github.com/databricks/databricks-sdk-go/client"
)

// registerModelPackagePath registers a model whose files Model Serving writes and
// starts copying a checkpoint's weights into the version. It runs as the caller and
// is idempotent per key: a repeat request with the same key and inputs continues
// the registration. It leaves the version pending.
const registerModelPackagePath = "/api/2.0/serving-endpoints:registerModelPackage"

// stagingStatusPath reports the recorded staging progress of a model version, a
// read.
const stagingStatusPath = "/api/2.0/serving-endpoints:checkModelDeploymentStagingStatus"

// finalizeModelPackagePath finalizes a registered version once its weights are
// copied, so it can be served. It is idempotent: a finalized version is returned
// as it is.
const finalizeModelPackagePath = "/api/2.0/serving-endpoints:finalizeModelPackage"

// registrationPollInterval is how often air register checks on a checkpoint's
// weights while they are staged.
const registrationPollInterval = 15 * time.Second

// ModelDeploymentStagingStatus states air register acts on.
const (
	// stagingStateNotStaged: no staging record exists for the version.
	stagingStateNotStaged = "NOT_STAGED"
	// stagingStateInProgress: an upload is queued, running or waiting to be retried.
	stagingStateInProgress = "IN_PROGRESS"
	// stagingStateSucceeded: the upload is recorded as done.
	stagingStateSucceeded = "SUCCEEDED"
	// stagingStateFailed: the upload failed and won't be retried automatically.
	stagingStateFailed = "FAILED"
)

// registerModelPackageRequest is the RegisterModelPackageRequest body: the fields of
// the registration YAML, with the key and the retry choice. Without Engine, the
// checkpoint is served with dedicated inference.
type registerModelPackageRequest struct {
	ModelSource        string              `json:"model_source"`
	UCModel            string              `json:"uc_model"`
	Engine             *modelPackageEngine `json:"engine,omitempty"`
	IdempotencyKey     string              `json:"idempotency_key,omitempty"`
	RetryFailedStaging bool                `json:"retry_failed_staging,omitempty"`
}

// modelPackageEngine is the command that serves a checkpoint, the task it serves
// and the environment it runs on.
type modelPackageEngine struct {
	Environment modelPackageEnvironment `json:"environment"`
	Command     string                  `json:"command"`
	Task        string                  `json:"task,omitempty"`
}

// modelPackageEnvironment is a serverless environment version, such as 6 or
// databricks_ai_v6, and the packages to install.
type modelPackageEnvironment struct {
	Version      string   `json:"version"`
	Dependencies []string `json:"dependencies,omitempty"`
}

// registerModelPackageResponse is the RegisterModelPackageResponse body.
type registerModelPackageResponse struct {
	ModelID       string              `json:"model_id"`
	UCModel       string              `json:"uc_model"`
	ModelVersion  string              `json:"model_version"`
	Ready         bool                `json:"ready"`
	StagingStatus *modelStagingStatus `json:"staging_status,omitempty"`
}

type modelStagingStatus struct {
	State               string                 `json:"state,omitempty"`
	ModelArtifact       *artifactStagingStatus `json:"model_artifact,omitempty"`
	EnvironmentArtifact *artifactStagingStatus `json:"environment_artifact,omitempty"`
}

type artifactStagingStatus struct {
	State string `json:"state,omitempty"`
}

// finalizeModelPackageRequest is the FinalizeModelPackageRequest body.
type finalizeModelPackageRequest struct {
	UCModel      string `json:"uc_model"`
	ModelVersion string `json:"model_version"`
}

// finalizeModelPackageResponse is the FinalizeModelPackageResponse body, the
// fields air register reads.
type finalizeModelPackageResponse struct {
	ModelVersion  string              `json:"model_version"`
	StagingStatus *modelStagingStatus `json:"staging_status,omitempty"`
}

// stagingStatusRequest is the CheckModelDeploymentStagingStatusRequest body.
type stagingStatusRequest struct {
	Model        string `json:"model"`
	ModelVersion string `json:"model_version"`
}

// stagingStatusResponse is the CheckModelDeploymentStagingStatusResponse body.
type stagingStatusResponse struct {
	Status *modelStagingStatus `json:"status,omitempty"`
}

// weightsState is the staging state of a checkpoint's weights, or "" before a
// response reports it.
func (r *registerModelPackageResponse) weightsState() string {
	if r.StagingStatus == nil || r.StagingStatus.ModelArtifact == nil {
		return ""
	}
	return r.StagingStatus.ModelArtifact.State
}

// environmentState is the staging state of the serving environment, or "" before
// a response reports it.
func (r *registerModelPackageResponse) environmentState() string {
	if r.StagingStatus == nil || r.StagingStatus.EnvironmentArtifact == nil {
		return ""
	}
	return r.StagingStatus.EnvironmentArtifact.State
}

// errStagingFailed reports that Model Serving couldn't copy a checkpoint's weights
// into the version. Rerunning with --retry-failed-staging stages them again.
var errStagingFailed = errors.New("staging failed")

// finalizeError is a failed request to finalize the registered version.
// Rerunning with the same key registers the same version again and retries it.
type finalizeError struct{ err error }

func (e *finalizeError) Error() string { return e.err.Error() }

func (e *finalizeError) Unwrap() error { return e.err }

// registerAndFinalize registers req's version and makes it ready to serve,
// calling progress with the registration response and every staging update. The
// checkpoint's weights are copied into the version after registering: while they
// are, it polls the staging status, a read. It then finalizes the version; the
// server checks that the version is ready to finalize. A version that is already
// ready, such as one an earlier run finalized, is returned as it is. Each
// response's staging status replaces the last one, and a status without both
// artifacts' states is an error, not progress. On error it returns the last
// response it got, nil before the first one; errStagingFailed when staging
// failed, and a *finalizeError when finalizing did.
func registerAndFinalize(ctx context.Context, w *databricks.WorkspaceClient, req registerModelPackageRequest, interval time.Duration, progress func(*registerModelPackageResponse)) (*registerModelPackageResponse, error) {
	apiClient, err := client.New(w.Config)
	if err != nil {
		return nil, withCode(codeInternalError, false, fmt.Errorf("failed to create API client: %w", err))
	}
	p := modelServingPoller{api: apiClient, headers: auth.WorkspaceIDHeaders(w.Config), interval: interval}
	var resp registerModelPackageResponse
	if err := p.post(ctx, registerModelPackagePath, req, &resp); err != nil {
		return nil, err
	}
	if resp.ModelVersion == "" {
		return nil, withCode(codeInternalError, true, fmt.Errorf("registering a version of %s returned no version", req.UCModel))
	}
	if !modelVersionRe().MatchString(resp.ModelVersion) {
		return nil, withCode(codeInternalError, true, fmt.Errorf("registering a version of %s returned %q, which isn't a version number", req.UCModel, stripControl(resp.ModelVersion)))
	}
	if err := checkStagingStatus(resp.StagingStatus, req.UCModel, resp.ModelVersion); err != nil {
		return nil, err
	}
	progress(&resp)
	if resp.Ready {
		return &resp, nil
	}
	if err := p.waitForWeights(ctx, req.UCModel, &resp, progress); err != nil {
		return &resp, err
	}
	if resp.weightsState() == stagingStateFailed {
		return &resp, errStagingFailed
	}
	var finalized finalizeModelPackageResponse
	finalizeReq := finalizeModelPackageRequest{UCModel: req.UCModel, ModelVersion: resp.ModelVersion}
	if err := p.post(ctx, finalizeModelPackagePath, finalizeReq, &finalized); err != nil {
		return &resp, &finalizeError{err}
	}
	if finalized.ModelVersion != resp.ModelVersion {
		return &resp, &finalizeError{withCode(codeInternalError, true, fmt.Errorf("the server finalized version %q instead", stripControl(finalized.ModelVersion)))}
	}
	if err := checkStagingStatus(finalized.StagingStatus, req.UCModel, resp.ModelVersion); err != nil {
		return &resp, &finalizeError{err}
	}
	resp.Ready = true
	resp.StagingStatus = finalized.StagingStatus
	return &resp, nil
}

// knownStagingStates are the artifact states the staging status API reports.
var knownStagingStates = []string{stagingStateNotStaged, stagingStateInProgress, stagingStateSucceeded, stagingStateFailed}

// checkStagingStatus checks that status, from a response about version of model,
// reports a known state for both the weights and the serving environment, as the
// server always does. Anything else is a response air register can't act on, so it
// fails rather than being read as progress until the timeout.
func checkStagingStatus(status *modelStagingStatus, model, version string) error {
	var problem string
	switch {
	case status == nil:
		problem = "it has no staging status"
	case status.ModelArtifact == nil || status.ModelArtifact.State == "":
		problem = "its staging status has no state for the weights"
	case !slices.Contains(knownStagingStates, status.ModelArtifact.State):
		problem = fmt.Sprintf("its staging status has an unknown state %q for the weights", stripControl(status.ModelArtifact.State))
	case status.EnvironmentArtifact == nil || status.EnvironmentArtifact.State == "":
		problem = "its staging status has no state for the serving environment"
	case !slices.Contains(knownStagingStates, status.EnvironmentArtifact.State):
		problem = fmt.Sprintf("its staging status has an unknown state %q for the serving environment", stripControl(status.EnvironmentArtifact.State))
	default:
		return nil
	}
	return withCode(codeInternalError, true, fmt.Errorf("air register can't read Model Serving's response about version %s of %s: %s", stripControl(version), model, problem))
}

// modelServingPoller sends the requests of a registration that can take many
// minutes.
type modelServingPoller struct {
	api      *client.DatabricksClient
	headers  map[string]string
	interval time.Duration
}

// waitForWeights polls the staging status of resp's version of model, replacing
// resp's status with each one, until its weights are copied, failed to copy, or
// have no staging record. resp's status was checked, and so is each new one.
func (p modelServingPoller) waitForWeights(ctx context.Context, model string, resp *registerModelPackageResponse, progress func(*registerModelPackageResponse)) error {
	statusReq := stagingStatusRequest{Model: model, ModelVersion: resp.ModelVersion}
	for resp.weightsState() == stagingStateInProgress {
		if err := sleepOrCancel(ctx, p.interval); err != nil {
			return err
		}
		var status stagingStatusResponse
		if err := p.post(ctx, stagingStatusPath, statusReq, &status); err != nil {
			return err
		}
		if err := checkStagingStatus(status.Status, model, resp.ModelVersion); err != nil {
			return err
		}
		resp.StagingStatus = status.Status
		progress(resp)
	}
	return nil
}

// post sends a request, repeating it after a transient failure the SDK doesn't
// retry: a wait of many minutes outlasts the SDK's own retries. Every request air
// register sends here is safe to repeat: the registration is idempotent per key,
// the status is a read, and finalizing a finalized version returns it.
func (p modelServingPoller) post(ctx context.Context, path string, body, out any) error {
	_, err := retry.With(ctx, retry.IsTransientRead, func() (struct{}, error) {
		return struct{}{}, p.api.Do(ctx, http.MethodPost, path, p.headers, nil, body, out)
	})
	return err
}
