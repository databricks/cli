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

// registerModelPackagePath registers a model whose files Model Serving writes. It
// runs as the caller and is idempotent per key: a repeat request with the same key
// and inputs continues the registration, and finalizes a checkpoint's version once
// its weights are staged. Staging never finalizes the version by itself.
const registerModelPackagePath = "/api/2.0/serving-endpoints:registerModelPackage"

// stagingStatusPath reports the recorded staging progress of a model version, a
// read.
const stagingStatusPath = "/api/2.0/serving-endpoints:checkModelDeploymentStagingStatus"

// registrationPollInterval is how often air deploy checks on a checkpoint's
// weights while they are staged.
const registrationPollInterval = 15 * time.Second

// ModelDeploymentStagingStatus states air deploy acts on.
const (
	// stagingStateInProgress: an upload is queued, running or waiting to be retried.
	stagingStateInProgress = "IN_PROGRESS"
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

// registerUntilReady registers req's version and waits until it is ready to
// serve, calling progress with every registration response and staging update.
// The other sources are ready after the first request. A checkpoint's version
// becomes ready once its weights are staged: while they copy, it polls the staging
// status, a read, then repeats the registration, which finalizes the version. On
// error it returns the last response it got, nil before the first one, and
// errStagingFailed when staging failed.
func registerUntilReady(ctx context.Context, w *databricks.WorkspaceClient, req registerModelPackageRequest, interval time.Duration, progress func(*registerModelPackageResponse)) (*registerModelPackageResponse, error) {
	apiClient, err := client.New(w.Config)
	if err != nil {
		return nil, fmt.Errorf("failed to create API client: %w", err)
	}
	p := modelServingPoller{api: apiClient, headers: auth.WorkspaceIDHeaders(w.Config), interval: interval}
	var last *registerModelPackageResponse
	for {
		var resp registerModelPackageResponse
		if err := p.post(ctx, registerModelPackagePath, req, &resp); err != nil {
			return last, err
		}
		last = &resp
		progress(&resp)
		if resp.Ready {
			return &resp, nil
		}
		if resp.weightsState() == stagingStateFailed {
			return &resp, errStagingFailed
		}
		// Only the first request restages a failed checkpoint: repeating it would
		// restart a staging that keeps failing instead of reporting the failure.
		req.RetryFailedStaging = false
		if err := p.waitForStaging(ctx, &resp, progress); err != nil {
			return &resp, err
		}
	}
}

// modelServingPoller sends the requests of a registration that can take many
// minutes.
type modelServingPoller struct {
	api      *client.DatabricksClient
	headers  map[string]string
	interval time.Duration
}

// waitForStaging polls the staging status of resp's version, updating resp, until
// its weights are no longer being copied.
func (p modelServingPoller) waitForStaging(ctx context.Context, resp *registerModelPackageResponse, progress func(*registerModelPackageResponse)) error {
	statusReq := stagingStatusRequest{Model: resp.RegisteredModel, ModelVersion: resp.ModelVersion}
	for {
		if err := sleepOrCancel(ctx, p.interval); err != nil {
			return err
		}
		if resp.weightsState() != stagingStateInProgress {
			return nil
		}
		var status stagingStatusResponse
		if err := p.post(ctx, stagingStatusPath, statusReq, &status); err != nil {
			return err
		}
		resp.StagingStatus = status.Status
		progress(resp)
		if resp.weightsState() != stagingStateInProgress {
			return nil
		}
	}
}

// post sends a request, repeating it after a transient failure the SDK doesn't
// retry: a wait of many minutes outlasts the SDK's own retries. Both requests air
// deploy polls with are safe to repeat, the registration because it is
// idempotent per key.
func (p modelServingPoller) post(ctx context.Context, path string, body, out any) error {
	_, err := retry.With(ctx, retry.IsTransientRead, func() (struct{}, error) {
		return struct{}{}, p.api.Do(ctx, http.MethodPost, path, p.headers, nil, body, out)
	})
	return err
}
