package aircmd

import (
	"context"
	"errors"
	"fmt"
	"net/http"
	"strings"

	"github.com/databricks/cli/libs/auth"
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

// validationUnavailableError means the backend check could not finish.
type validationUnavailableError struct {
	err       error
	retryable bool
}

func (e *validationUnavailableError) Error() string { return e.err.Error() }
func (e *validationUnavailableError) Unwrap() error { return e.err }

func asValidationUnavailable(err error, retryable bool) error {
	return &validationUnavailableError{
		err:       fmt.Errorf("config validation unavailable: %w", err),
		retryable: retryable,
	}
}

func validationUnavailable(err error) (*validationUnavailableError, bool) {
	return errors.AsType[*validationUnavailableError](err)
}

// preflightValidate checks the config against the backend before any upload, so
// a bad config fails fast with the server's own field-level errors.
//
// It fails open for service unavailability because submission validates the
// config authoritatively. Caller errors and field errors still block.
func preflightValidate(ctx context.Context, w *databricks.WorkspaceClient, cfg *runConfig, commandPath string, containers []submittedContainer, idempotencyToken string) error {
	err := validateConfig(ctx, w, cfg, commandPath, containers, idempotencyToken)
	if _, unavailable := validationUnavailable(err); unavailable {
		return nil
	}
	return err
}

// validateConfig performs the current backend validation without client-side
// availability fallbacks. The legacy response reports field errors only; it
// cannot yet distinguish complete success from skipped backend dependencies.
func validateConfig(ctx context.Context, w *databricks.WorkspaceClient, cfg *runConfig, commandPath string, containers []submittedContainer, idempotencyToken string) error {
	apiClient, err := client.New(w.Config)
	if err != nil {
		return fmt.Errorf("failed to create API client: %w", err)
	}

	var resp validateConfigResponse
	err = apiClient.Do(ctx, http.MethodPost, validateConfigPath, auth.WorkspaceIDHeaders(w.Config), nil, validateConfigRequest(cfg, commandPath, containers, idempotencyToken), &resp)
	if err != nil {
		validationErr := fmt.Errorf("failed to validate config: %w", err)
		if unavailable, retryable := classifyValidationFailure(err); unavailable {
			return asValidationUnavailable(validationErr, retryable)
		}
		return validationErr
	}
	if len(resp.Errors) == 0 {
		return nil
	}
	return errors.New(formatConfigErrors(resp.Errors))
}

// classifyValidationFailure separates service failures from caller errors and
// reports whether retrying the same request may succeed.
func classifyValidationFailure(err error) (unavailable, retryable bool) {
	if errors.Is(err, context.Canceled) {
		return false, false
	}
	if errors.Is(err, context.DeadlineExceeded) {
		return true, true
	}
	apiErr, ok := errors.AsType[*apierr.APIError](err)
	if !ok {
		return true, true
	}
	if apiErr.ErrorCode == "FEATURE_DISABLED" ||
		apiErr.StatusCode == http.StatusNotFound ||
		apiErr.StatusCode == http.StatusNotImplemented {
		return true, false
	}
	if apiErr.StatusCode == http.StatusRequestTimeout ||
		apiErr.StatusCode == http.StatusTooManyRequests ||
		apiErr.StatusCode >= 500 {
		return true, true
	}
	return false, false
}

// validateConfigRequest builds the {task, run_options} body from the user's config. commandPath is
// the workspace path where the command script will be uploaded; the caller computes it before this
// call so the server can validate the real path. `parameters` is intentionally omitted: it is
// free-form nested hyperparameters uploaded as a YAML file at submit, not the proto's string map.
func validateConfigRequest(cfg *runConfig, commandPath string, containers []submittedContainer, idempotencyToken string) map[string]any {
	compute := map[string]any{}
	if cfg.Compute != nil {
		compute["accelerator_type"] = cfg.Compute.AcceleratorType
		compute["accelerator_count"] = cfg.Compute.NumAccelerators
		// Wire field stays provisioned_capacity_id (the backend name); the YAML
		// field is pool_id.
		putOpt(compute, "provisioned_capacity_id", cfg.Compute.PoolID)
	}
	deployment := map[string]any{"compute": compute}
	if len(containers) == 0 {
		deployment["command_path"] = commandPath
	} else {
		raw := make([]any, 0, len(containers))
		for _, container := range containers {
			raw = append(raw, map[string]any{
				"name":                     container.Name,
				"command_path":             container.CommandPath,
				"ranks":                    container.Ranks,
				"unity_catalog_image_path": container.UnityCatalogImagePath,
			})
		}
		deployment["containers"] = raw
	}
	task := map[string]any{
		"experiment":  cfg.ExperimentName,
		"deployments": []any{deployment},
	}
	if cfg.Compute != nil {
		// priority_class rides on the ai_runtime_task (task-level), not the deployment
		// compute where provisioned_capacity_id lives.
		putOpt(task, "priority_class", cfg.Compute.PriorityClass)
	}
	putOpt(task, "mlflow_run", cfg.MLflowRunName)
	putOpt(task, "mlflow_experiment_directory", cfg.MLflowExperimentDirectory)
	putOpt(task, "mlflow_artifact_location", cfg.MLflowArtifactLocation)
	if image := cfg.unityCatalogImagePath(); image != "" {
		task["unity_catalog_image_path"] = image
	}

	return map[string]any{
		"task":        task,
		"run_options": validateConfigRunOptions(cfg, idempotencyToken),
	}
}

// validateConfigRunOptions includes the effective idempotency token and any
// run-level fields the user set.
func validateConfigRunOptions(cfg *runConfig, idempotencyToken string) map[string]any {
	runOptions := map[string]any{"idempotency_token": idempotencyToken}
	putOpt(runOptions, "max_retries", cfg.MaxRetries)
	putOpt(runOptions, "timeout_minutes", cfg.TimeoutMinutes)
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
