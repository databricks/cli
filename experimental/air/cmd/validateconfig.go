package aircmd

import (
	"context"
	"errors"
	"fmt"
	"net/http"
	"strings"

	"github.com/databricks/databricks-sdk-go"
	"github.com/databricks/databricks-sdk-go/apierr"
	"github.com/databricks/databricks-sdk-go/client"
)

// validateConfigPath is AiTrainingService's pre-flight: it checks a training
// config server-side and returns the problems, without submitting. Called with a
// raw client.Do because the SDK does not model AiTrainingService.
const validateConfigPath = "/api/2.0/ai-training/config:validate"

// configFieldError is one problem the server found, addressed to the config
// field that caused it. Mirrors the proto FieldError.
type configFieldError struct {
	Path    string `json:"path"`
	Message string `json:"message"`
	Code    string `json:"code"`
}

type validateConfigResponse struct {
	Errors []configFieldError `json:"errors"`
}

// preflightValidate checks the config against the backend before any upload, so
// a bad config fails fast with the server's own field-level errors.
//
// It fails open: the endpoint is behind a SAFE flag and older workspaces do not
// have it, so a disabled or missing endpoint skips the check and lets submission
// proceed (where the config is validated again, authoritatively). A 5xx is a
// backend problem, not the user's config, so it fails open too. Only a 4xx (the
// server rejected the config) or a populated error list blocks.
func preflightValidate(ctx context.Context, w *databricks.WorkspaceClient, cfg *runConfig, commandPath string) error {
	apiClient, err := client.New(w.Config)
	if err != nil {
		return fmt.Errorf("failed to create API client: %w", err)
	}

	var resp validateConfigResponse
	err = apiClient.Do(ctx, http.MethodPost, validateConfigPath, nil, nil, validateConfigRequest(cfg, commandPath), &resp)
	if err != nil {
		if endpointUnavailable(err) || serverError(err) {
			return nil
		}
		return fmt.Errorf("failed to validate config: %w", err)
	}
	if len(resp.Errors) == 0 {
		return nil
	}
	return errors.New(formatConfigErrors(resp.Errors))
}

// validateConfigRequest builds the {task, run_options} body from the user's config. commandPath is
// the workspace path where the command script will be uploaded; the caller computes it before this
// call so the server can validate the real path. `parameters` is intentionally omitted: it is
// free-form nested hyperparameters uploaded as a YAML file at submit, not the proto's string map.
func validateConfigRequest(cfg *runConfig, commandPath string) map[string]any {
	compute := map[string]any{}
	if cfg.Compute != nil {
		compute["accelerator_type"] = cfg.Compute.AcceleratorType
		compute["accelerator_count"] = cfg.Compute.NumAccelerators
		putOpt(compute, "provisioned_capacity_id", cfg.Compute.ProvisionedCapacityID)
	}
	task := map[string]any{
		"experiment":  cfg.ExperimentName,
		"deployments": []any{map[string]any{"command_path": commandPath, "compute": compute}},
	}
	if cfg.Compute != nil {
		// priority_class rides on the ai_runtime_task (task-level), not the deployment
		// compute where provisioned_capacity_id lives.
		putOpt(task, "priority_class", cfg.Compute.PriorityClass)
	}
	putOpt(task, "mlflow_run", cfg.MLflowRunName)
	putOpt(task, "mlflow_experiment_directory", cfg.MLflowExperimentDirectory)
	putOpt(task, "mlflow_artifact_location", cfg.MLflowArtifactLocation)
	if dockerImageURL := cfg.dockerImageURL(); dockerImageURL != "" {
		task["docker_image_url"] = dockerImageURL
	}

	req := map[string]any{"task": task}
	if runOptions := validateConfigRunOptions(cfg); len(runOptions) > 0 {
		req["run_options"] = runOptions
	}
	return req
}

// validateConfigRunOptions gathers the run-level fields into run_options,
// omitting any the user didn't set.
func validateConfigRunOptions(cfg *runConfig) map[string]any {
	runOptions := map[string]any{}
	putOpt(runOptions, "max_retries", cfg.MaxRetries)
	putOpt(runOptions, "timeout_minutes", cfg.TimeoutMinutes)
	putOpt(runOptions, "idempotency_token", cfg.IdempotencyToken)
	putOpt(runOptions, "usage_policy_name", cfg.UsagePolicyName)
	putOpt(runOptions, "usage_policy_id", cfg.UsagePolicyID)
	if len(cfg.EnvVariables) > 0 {
		runOptions["env_variables"] = cfg.EnvVariables
	}
	if len(cfg.Secrets) > 0 {
		runOptions["secrets"] = cfg.Secrets
	}
	return runOptions
}

// putOpt sets key to the pointer's value only when it is non-nil, so an unset
// config field is left out of the request rather than sent as a zero value.
func putOpt[T any](m map[string]any, key string, value *T) {
	if value != nil {
		m[key] = *value
	}
}

// endpointUnavailable reports whether the failure means the endpoint isn't there
// to answer — the flag is off, or the workspace predates it — as opposed to the
// config being rejected. Those cases fail open.
func endpointUnavailable(err error) bool {
	apiErr, ok := errors.AsType[*apierr.APIError](err)
	return ok && (apiErr.ErrorCode == "FEATURE_DISABLED" ||
		apiErr.StatusCode == http.StatusNotFound ||
		apiErr.StatusCode == http.StatusNotImplemented)
}

// serverError reports whether the failure is a 5xx: a backend problem, not the
// user's config. The SDK already retries the transient subset (503, 429, IO
// errors); a 5xx that still surfaces here fails open, since blocking a submit on
// a backend blip isn't actionable and submit re-validates anyway.
func serverError(err error) bool {
	apiErr, ok := errors.AsType[*apierr.APIError](err)
	return ok && apiErr.StatusCode >= 500
}

// formatConfigErrors renders the field errors as one message, one problem per
// line, each pointing at the config field the user wrote.
func formatConfigErrors(fieldErrors []configFieldError) string {
	var b strings.Builder
	b.WriteString("config validation failed:")
	for _, e := range fieldErrors {
		b.WriteString("\n  ")
		if e.Path != "" {
			b.WriteString(e.Path)
			b.WriteString(": ")
		}
		b.WriteString(e.Message)
	}
	return b.String()
}
