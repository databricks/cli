package aircmd

import (
	"context"
	"errors"
	"fmt"
	"net/http"
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

// registrationPollInterval is how often air deploy checks on a checkpoint's
// weights while they are staged.
const registrationPollInterval = 15 * time.Second

// ModelDeploymentStagingStatus states air deploy acts on.
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

// registerModelPackageRequest is the RegisterModelPackageRequest body. Exactly one
// of Checkpoint, Code and SavedModel is set.
type registerModelPackageRequest struct {
	RegisteredModel    string                  `json:"registered_model"`
	IdempotencyKey     string                  `json:"idempotency_key"`
	Checkpoint         *modelPackageCheckpoint `json:"checkpoint,omitempty"`
	Code               *modelPackageCode       `json:"code,omitempty"`
	SavedModel         *modelPackageSavedModel `json:"saved_model,omitempty"`
	ExperimentPath     string                  `json:"experiment_path,omitempty"`
	RetryFailedStaging bool                    `json:"retry_failed_staging,omitempty"`
}

type modelPackageCheckpoint struct {
	VolumePath     string                  `json:"volume_path"`
	ServingCommand string                  `json:"serving_command"`
	Task           string                  `json:"task"`
	Environment    modelPackageEnvironment `json:"environment"`
}

type modelPackageCode struct {
	EntryFile   string                  `json:"entry_file"`
	EntrySource string                  `json:"entry_source"`
	Files       []modelPackageFile      `json:"files,omitempty"`
	Task        string                  `json:"task,omitempty"`
	Environment modelPackageEnvironment `json:"environment"`
}

type modelPackageFile struct {
	Path    string `json:"path"`
	Content string `json:"content"`
}

type modelPackageSavedModel struct {
	VolumePath string `json:"volume_path"`
}

type modelPackageEnvironment struct {
	Runtime      string   `json:"runtime"`
	Dependencies []string `json:"dependencies,omitempty"`
}

// registerModelPackageResponse is the RegisterModelPackageResponse body.
type registerModelPackageResponse struct {
	ModelID         string              `json:"model_id"`
	RegisteredModel string              `json:"registered_model"`
	ModelVersion    string              `json:"model_version"`
	Ready           bool                `json:"ready"`
	StagingStatus   *modelStagingStatus `json:"staging_status,omitempty"`
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
	RegisteredModel string `json:"registered_model"`
	ModelVersion    string `json:"model_version"`
}

// finalizeModelPackageResponse is the FinalizeModelPackageResponse body, the
// fields air deploy reads.
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

// weightsState is the staging state of a checkpoint's weights, or "" when the
// response reports none.
func (r *registerModelPackageResponse) weightsState() string {
	if r.StagingStatus == nil || r.StagingStatus.ModelArtifact == nil {
		return ""
	}
	return r.StagingStatus.ModelArtifact.State
}

// environmentState is the staging state of the serving environment, or "" when the
// response reports none.
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

// registrationRequest builds the request that registers cfg's model under key.
// code is the collected model from code, and is nil for the other sources.
func (c *deployConfig) registrationRequest(key string, code *modelPackageCode) registerModelPackageRequest {
	req := registerModelPackageRequest{
		RegisteredModel: c.RegisteredModel,
		IdempotencyKey:  key,
	}
	if c.MLflowExperimentPath != nil {
		req.ExperimentPath = *c.MLflowExperimentPath
	}
	switch c.Model.Type {
	case modelTypeCheckpoint:
		req.Checkpoint = &modelPackageCheckpoint{
			VolumePath:     *c.Model.Path,
			ServingCommand: *c.Command,
			Task:           *c.Task,
			Environment:    c.servingEnvironment(),
		}
	case modelTypeCode:
		req.Code = code
	case modelTypeSavedModelDir:
		req.SavedModel = &modelPackageSavedModel{VolumePath: *c.Model.Path}
	}
	return req
}

// registerAndFinalize registers req's version and makes it ready to serve,
// calling progress with the registration response and every staging update. A
// checkpoint's weights are copied into the version after registering: while they
// are, it polls the staging status, a read. It then finalizes the version, for
// every source; the server checks that the version is ready to finalize. A
// version that is already ready, such as one an earlier run finalized, is
// returned as it is. On error it returns the last response it got, nil before
// the first one; errStagingFailed when staging failed, and a *finalizeError when
// finalizing did.
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
		return nil, withCode(codeInternalError, true, fmt.Errorf("registering a version of %s returned no version", req.RegisteredModel))
	}
	progress(&resp)
	if resp.Ready {
		return &resp, nil
	}
	if req.Checkpoint != nil {
		if err := p.waitForWeights(ctx, req.RegisteredModel, &resp, progress); err != nil {
			return &resp, err
		}
	}
	if resp.weightsState() == stagingStateFailed {
		return &resp, errStagingFailed
	}
	var finalized finalizeModelPackageResponse
	finalizeReq := finalizeModelPackageRequest{RegisteredModel: req.RegisteredModel, ModelVersion: resp.ModelVersion}
	if err := p.post(ctx, finalizeModelPackagePath, finalizeReq, &finalized); err != nil {
		return &resp, &finalizeError{err}
	}
	if finalized.ModelVersion != resp.ModelVersion {
		return &resp, &finalizeError{withCode(codeInternalError, true, fmt.Errorf("the server finalized version %q instead", stripControl(finalized.ModelVersion)))}
	}
	resp.Ready = true
	resp.StagingStatus = mergeStagingStatus(resp.StagingStatus, finalized.StagingStatus)
	return &resp, nil
}

// mergeStagingStatus is newer, with each artifact it doesn't report kept from
// older.
func mergeStagingStatus(older, newer *modelStagingStatus) *modelStagingStatus {
	if newer == nil {
		return older
	}
	merged := *newer
	if older != nil {
		if merged.ModelArtifact == nil {
			merged.ModelArtifact = older.ModelArtifact
		}
		if merged.EnvironmentArtifact == nil {
			merged.EnvironmentArtifact = older.EnvironmentArtifact
		}
	}
	return &merged
}

// modelServingPoller sends the requests of a registration that can take many
// minutes.
type modelServingPoller struct {
	api      *client.DatabricksClient
	headers  map[string]string
	interval time.Duration
}

// waitForWeights polls the staging status of resp's version of model, updating
// resp, until its weights are copied, failed to copy, or have no staging record.
// A state it doesn't know, or a status without one, is waited out like a copy in
// progress, until the context ends.
func (p modelServingPoller) waitForWeights(ctx context.Context, model string, resp *registerModelPackageResponse, progress func(*registerModelPackageResponse)) error {
	statusReq := stagingStatusRequest{Model: model, ModelVersion: resp.ModelVersion}
	for {
		switch resp.weightsState() {
		case stagingStateSucceeded, stagingStateFailed, stagingStateNotStaged:
			return nil
		}
		if err := sleepOrCancel(ctx, p.interval); err != nil {
			return err
		}
		var status stagingStatusResponse
		if err := p.post(ctx, stagingStatusPath, statusReq, &status); err != nil {
			return err
		}
		resp.StagingStatus = mergeStagingStatus(resp.StagingStatus, status.Status)
		progress(resp)
	}
}

// post sends a request, repeating it after a transient failure the SDK doesn't
// retry: a wait of many minutes outlasts the SDK's own retries. Every request air
// deploy sends here is safe to repeat: the registration is idempotent per key,
// the status is a read, and finalizing a finalized version returns it.
func (p modelServingPoller) post(ctx context.Context, path string, body, out any) error {
	_, err := retry.With(ctx, retry.IsTransientRead, func() (struct{}, error) {
		return struct{}{}, p.api.Do(ctx, http.MethodPost, path, p.headers, nil, body, out)
	})
	return err
}
