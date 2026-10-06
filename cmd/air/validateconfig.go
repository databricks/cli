package aircmd

import (
	"context"
	"errors"
	"fmt"
	"net/http"
	"net/url"
	"strings"
	"time"

	"github.com/databricks/cli/libs/auth"
	"github.com/databricks/databricks-sdk-go"
	"github.com/databricks/databricks-sdk-go/apierr"
	"github.com/databricks/databricks-sdk-go/client"
	"github.com/databricks/databricks-sdk-go/config"
	"github.com/databricks/databricks-sdk-go/httpclient"
)

// validateConfigPath is AiTrainingService's pre-flight: it checks a training
// config server-side and returns the problems, without submitting. Called with a
// raw client.Do because the SDK does not model AiTrainingService.
const validateConfigPath = "/api/2.0/ai-training/config:validate"

const dryRunValidationMaxAttempts = 2

type dryRunValidationAttemptBudgetKey struct{}

func armDryRunValidationAttemptBudget(ctx context.Context) context.Context {
	return context.WithValue(ctx, dryRunValidationAttemptBudgetKey{}, new(int))
}

func allowDryRunValidationRetry(ctx context.Context) bool {
	attempts, ok := ctx.Value(dryRunValidationAttemptBudgetKey{}).(*int)
	if !ok {
		return false
	}
	*attempts++
	return *attempts < dryRunValidationMaxAttempts
}

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
// It preserves the existing submission behavior: a missing endpoint or 5xx
// fails open, while other request failures and field errors block.
func preflightValidate(ctx context.Context, w *databricks.WorkspaceClient, cfg *runConfig, commandPath string, containers []submittedContainer, idempotencyToken string) error {
	apiClient, err := client.New(w.Config)
	if err != nil {
		return fmt.Errorf("failed to create API client: %w", err)
	}
	err = validateConfig(ctx, apiClient, cfg, commandPath, containers, idempotencyToken)
	if endpointUnavailable(err) || serverError(err) {
		return nil
	}
	return err
}

func newDryRunValidationClient(w *databricks.WorkspaceClient) (*client.DatabricksClient, error) {
	clientCfg, err := config.HTTPClientConfigFromConfig(w.Config)
	if err != nil {
		return nil, err
	}
	clientCfg.TransientErrors = nil
	clientCfg.ErrorRetriable = func(ctx context.Context, err error) bool {
		apiErr, ok := errors.AsType[*apierr.APIError](err)
		if !ok {
			return false
		}
		if apiErr.StatusCode != http.StatusTooManyRequests && apiErr.StatusCode != http.StatusServiceUnavailable {
			return false
		}
		return allowDryRunValidationRetry(ctx)
	}
	return client.NewWithClient(w.Config, httpclient.NewApiClient(clientCfg))
}

func validateDryRunConfig(ctx context.Context, w *databricks.WorkspaceClient, cfg *runConfig, commandPath string, containers []submittedContainer, idempotencyToken string, timeout time.Duration) error {
	apiClient, err := newDryRunValidationClient(w)
	if err != nil {
		return fmt.Errorf("failed to create API client: %w", err)
	}

	validationCtx, cancel := context.WithTimeout(ctx, timeout)
	defer cancel()
	validationCtx = armDryRunValidationAttemptBudget(validationCtx)

	err = validateConfig(validationCtx, apiClient, cfg, commandPath, containers, idempotencyToken)
	if unavailable, retryable := classifyValidationFailure(err); unavailable {
		return asValidationUnavailable(err, retryable)
	}
	return err
}

// validateConfig performs the current backend validation without client-side
// availability fallbacks. The legacy response reports field errors only; it
// cannot yet distinguish complete success from skipped backend dependencies.
func validateConfig(ctx context.Context, apiClient *client.DatabricksClient, cfg *runConfig, commandPath string, containers []submittedContainer, idempotencyToken string) error {
	request := validateConfigRequest(ctx, cfg, commandPath, containers, idempotencyToken)
	var resp validateConfigResponse
	err := apiClient.Do(ctx, http.MethodPost, validateConfigPath, auth.WorkspaceIDHeaders(apiClient.Config), nil, request, &resp)
	if rejectsValidateConfigEnvironment(err) {
		delete(request, "environment")
		resp = validateConfigResponse{}
		err = apiClient.Do(ctx, http.MethodPost, validateConfigPath, auth.WorkspaceIDHeaders(apiClient.Config), nil, request, &resp)
	}
	if err != nil {
		return fmt.Errorf("failed to validate config: %w", err)
	}
	if len(resp.Errors) == 0 {
		return nil
	}
	return errors.New(formatConfigErrors(resp.Errors))
}

const unsupportedEnvironmentMessage = "Cannot find field: environment in message databricks.aicomputemanager.ValidateConfigRequest"

// rejectsValidateConfigEnvironment identifies the one old-backend response for
// which retrying without the additive environment field is safe.
func rejectsValidateConfigEnvironment(err error) bool {
	apiErr, ok := errors.AsType[*apierr.APIError](err)
	return ok &&
		apiErr.StatusCode == http.StatusBadRequest &&
		apiErr.ErrorCode == "MALFORMED_REQUEST" &&
		apiErr.Message == unsupportedEnvironmentMessage
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
	if _, ok := errors.AsType[*url.Error](err); ok {
		return true, true
	}
	apiErr, ok := errors.AsType[*apierr.APIError](err)
	if !ok {
		return false, false
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

// validateConfigRequest builds the {task, run_options, environment} body from the user's config. commandPath is
// the workspace path where the command script will be uploaded; the caller computes it before this
// call so the server can validate the real path. `parameters` is intentionally omitted: it is
// free-form nested hyperparameters uploaded as a YAML file at submit, not the proto's string map.
func validateConfigRequest(ctx context.Context, cfg *runConfig, commandPath string, containers []submittedContainer, idempotencyToken string) map[string]any {
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
	runtimeVersion, _ := cfg.runtimeVersion()
	dependencies, _ := cfg.inlineDependencies()

	return map[string]any{
		"task":        task,
		"run_options": validateConfigRunOptions(cfg, idempotencyToken),
		"environment": buildRunEnvironment(dlRuntimeImage(ctx, runtimeVersion), dependencies),
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

// endpointUnavailable reports that the validation endpoint could not answer
// because it is disabled or absent.
func endpointUnavailable(err error) bool {
	apiErr, ok := errors.AsType[*apierr.APIError](err)
	return ok && (apiErr.ErrorCode == "FEATURE_DISABLED" ||
		apiErr.StatusCode == http.StatusNotFound ||
		apiErr.StatusCode == http.StatusNotImplemented)
}

// serverError reports a backend 5xx rather than a caller error.
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
