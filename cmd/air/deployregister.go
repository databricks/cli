package aircmd

import (
	"context"
	"errors"
	"fmt"
	"net/http"
	"time"

	"github.com/databricks/cli/libs/auth"
	"github.com/databricks/databricks-sdk-go"
	"github.com/databricks/databricks-sdk-go/client"
)

// registerModelPackagePath registers a model whose files Model Serving writes. It
// runs as the caller and is idempotent per key: a repeat request with the same key
// and inputs continues the registration, and finalizes a checkpoint's version once
// its weights are staged.
const registerModelPackagePath = "/api/2.0/serving-endpoints:registerModelPackage"

// registrationPollInterval is how often air deploy repeats the registration while
// a checkpoint's weights are staged.
const registrationPollInterval = 15 * time.Second

// stagingStateFailed is the ModelDeploymentStagingStatus state of an artifact that
// failed to stage and won't be retried automatically.
const stagingStateFailed = "FAILED"

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

// registerUntilReady sends req until the version it registers is ready to serve,
// calling progress with every response. A checkpoint's version becomes ready once
// its weights are staged; the other sources are ready after the first request.
// It returns errStagingFailed, with the last response, when staging the weights
// failed.
func registerUntilReady(ctx context.Context, w *databricks.WorkspaceClient, req registerModelPackageRequest, interval time.Duration, progress func(*registerModelPackageResponse)) (*registerModelPackageResponse, error) {
	apiClient, err := client.New(w.Config)
	if err != nil {
		return nil, fmt.Errorf("failed to create API client: %w", err)
	}
	headers := auth.WorkspaceIDHeaders(w.Config)
	for {
		var resp registerModelPackageResponse
		if err := apiClient.Do(ctx, http.MethodPost, registerModelPackagePath, headers, nil, req, &resp); err != nil {
			return nil, err
		}
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

		timer := time.NewTimer(interval)
		select {
		case <-ctx.Done():
			timer.Stop()
			return &resp, ctx.Err()
		case <-timer.C:
		}
	}
}
