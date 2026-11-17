package aircmd

import (
	"context"
	"errors"
	"fmt"
	"net/http"
	"path/filepath"
	"regexp"
	"strings"
	"sync"
	"time"

	"github.com/databricks/cli/cmd/root"
	"github.com/databricks/cli/libs/cmdctx"
	"github.com/databricks/cli/libs/cmdio"
	"github.com/databricks/cli/libs/flags"
	"github.com/databricks/cli/libs/shellquote"
	"github.com/databricks/databricks-sdk-go"
	"github.com/databricks/databricks-sdk-go/apierr"
	"github.com/databricks/databricks-sdk-go/retries"
	"github.com/databricks/databricks-sdk-go/service/serving"
	"github.com/spf13/cobra"
)

// defaultDeployTimeout bounds the whole wait: staging a checkpoint's weights and
// starting the endpoint.
const defaultDeployTimeout = time.Hour

// Errors air deploy wraps when it stops waiting. The server keeps working, and
// the error says how to continue.
var (
	errDeployInterrupted = errors.New("air deploy was interrupted")
	errDeployTimedOut    = errors.New("air deploy timed out")
)

// JSON error codes of air deploy's own failures; API errors keep the server's.
const (
	codeInvalidArgs   = "INVALID_ARGS"
	codeInterrupted   = "INTERRUPTED"
	codeTimeout       = "TIMEOUT"
	codeStagingFailed = "STAGING_FAILED"
	// codeEndpointNotReady: the endpoint serves the version but isn't ready, which
	// running air deploy again doesn't change.
	codeEndpointNotReady = "ENDPOINT_NOT_READY"
	codeInternalError    = "INTERNAL_ERROR"
)

// apiErrorCodeRe matches an error code the server sent. The SDK fills ErrorCode
// with "" or "<nil>" when the response body has none.
var apiErrorCodeRe = sync.OnceValue(func() *regexp.Regexp { return regexp.MustCompile(`^[A-Z][A-Z_]*$`) })

// deployError carries the JSON error code and retryability of a failure whose
// message is err's.
type deployError struct {
	code      string
	retryable bool
	err       error
}

func (e *deployError) Error() string { return e.err.Error() }

func (e *deployError) Unwrap() error { return e.err }

func withCode(code string, retryable bool, err error) error {
	return &deployError{code: code, retryable: retryable, err: err}
}

// deployResult is the JSON payload for `air deploy`.
type deployResult struct {
	Status string `json:"status"`
	DryRun bool   `json:"dry_run,omitempty"`
	// CodeFiles lists, on a dry run of a checkpoint served by model.entry, the
	// files it would send.
	CodeFiles       []string `json:"code_files,omitempty"`
	RegisteredModel string   `json:"registered_model,omitempty"`
	ModelVersion    string   `json:"model_version,omitempty"`
	ModelID         string   `json:"model_id,omitempty"`
	IdempotencyKey  string   `json:"idempotency_key,omitempty"`
	// EnvironmentStagingFailed is set when the version's serving environment
	// couldn't be prepared ahead of time, so the endpoint can be slower to start.
	EnvironmentStagingFailed bool   `json:"environment_staging_failed,omitempty"`
	Endpoint                 string `json:"endpoint,omitempty"`
	EndpointURL              string `json:"endpoint_url,omitempty"`
}

// deployOptions are the `air deploy` flags.
type deployOptions struct {
	file      string
	overrides []string
	dryRun    bool
	// idempotencyKeySet records an explicit --idempotency-key, so an empty value
	// (say, from an unset shell variable) fails instead of registering anew.
	idempotencyKeySet  bool
	idempotencyKey     string
	retryFailedStaging bool
	timeout            time.Duration
}

func newDeployCommand() *cobra.Command {
	return newDeployCommandWithPollInterval(registrationPollInterval)
}

func newDeployCommandWithPollInterval(pollInterval time.Duration) *cobra.Command {
	var opts deployOptions

	cmd := &cobra.Command{
		Use:   "deploy",
		Args:  root.NoArgs,
		Short: "Register a model in Unity Catalog and serve it on an endpoint",
		Long: `Register a model in Unity Catalog and serve it on a Model Serving endpoint.

The deployment is described by a YAML file (see --file). The model is a
checkpoint (weights in a Unity Catalog volume, served by a command or by a
Python file that builds an MLflow model) or an existing model version.
air deploy registers a new version, waits until it is ready, then creates or
updates the endpoint to serve it and waits until the endpoint is ready.

Registration is idempotent: run air deploy again with the idempotency key it
prints to continue an interrupted deployment. To look up a YAML field, pass
its path to -h as a separate argument:

  databricks air deploy -h config                 all fields
  databricks air deploy -h config.model           one section
  databricks air deploy -h config.model.type      one field`,
		Example: `  # Register a checkpoint and serve it
  databricks air deploy -f deploy.yaml

  # Validate the config, and list the code it would send, without deploying
  databricks air deploy -f deploy.yaml --dry-run

  # Continue a deployment that was interrupted
  databricks air deploy -f deploy.yaml --idempotency-key 3f6c0b52-4c7e-4d0b-9b8e-0f7ad6a4c2d1`,
		// Hidden until the registration API is available in every workspace.
		Hidden: true,
	}

	// cobra passes -h's positional args to the help func before Args/required-flag
	// validation, so a config path documents a field without needing -f.
	cmd.SetHelpFunc(func(c *cobra.Command, args []string) {
		fields := c.Flags().Args()
		if len(fields) == 0 {
			// Parent() is nil for a detached command (unit tests).
			if parent := c.Parent(); parent != nil {
				parent.HelpFunc()(c, args)
				return
			}
			_ = c.Usage()
			return
		}
		field, err := resolveSchemaField(deploySchema(), fields[0])
		if err != nil {
			c.PrintErrln("Error:", err)
			return
		}
		renderConfigField(c.OutOrStdout(), deployCommandPath, field, nil)
	})

	// List flags in declaration order (most used first) rather than alphabetically.
	cmd.Flags().SortFlags = false
	cmd.Flags().StringVarP(&opts.file, "file", "f", "", "Path to the deployment YAML config")
	cmd.Flags().StringArrayVar(&opts.overrides, "override", nil, "Override a YAML field, e.g. compute.workload_size=Medium (repeatable)")
	cmd.Flags().BoolVar(&opts.dryRun, "dry-run", false, "Validate the config, and list the code it would send, without deploying")
	cmd.Flags().StringVar(&opts.idempotencyKey, "idempotency-key", "", "Continue the registration that used this key, instead of registering another version")
	cmd.Flags().BoolVar(&opts.retryFailedStaging, "retry-failed-staging", false, "Copy a checkpoint's weights again after copying them failed")
	cmd.Flags().DurationVar(&opts.timeout, "timeout", defaultDeployTimeout, "How long to wait for the version and the endpoint to be ready")
	_ = cmd.MarkFlagRequired("file")

	// --dry-run never authenticates, so it reports config mistakes without
	// credentials.
	cmd.PreRunE = func(cmd *cobra.Command, args []string) error {
		if opts.dryRun {
			return nil
		}
		return root.MustWorkspaceClient(cmd, args)
	}

	cmd.RunE = func(cmd *cobra.Command, args []string) error {
		opts.idempotencyKeySet = cmd.Flags().Changed("idempotency-key")
		err := runDeploy(cmd, opts, pollInterval)
		if err == nil {
			return nil
		}
		code, retryable := deployErrorCode(err)
		kind := "PERMANENT"
		if retryable {
			kind = "TRANSIENT"
		}
		return renderError(cmd.Context(), cmd, code, kind, retryable, err)
	}
	return cmd
}

func runDeploy(cmd *cobra.Command, opts deployOptions, pollInterval time.Duration) error {
	ctx := cmd.Context()
	cfg, err := loadDeployConfig(ctx, opts.file, opts.overrides)
	if err != nil {
		return withCode(codeInvalidArgs, false, err)
	}
	if err := validateDeployFlags(cfg, opts); err != nil {
		return withCode(codeInvalidArgs, false, err)
	}

	var code *modelPackagePythonModel
	if cfg.Model.Entry != nil {
		code, err = collectEntryCode(ctx, cfg, opts.file)
		if err != nil {
			return withCode(codeInvalidArgs, false, err)
		}
	}

	jsonOut := root.OutputType(cmd) == flags.OutputJSON
	if opts.dryRun {
		return renderDeployDryRun(ctx, cfg, code, jsonOut)
	}

	w := cmdctx.WorkspaceClient(ctx)
	ctx, cancel := context.WithTimeout(ctx, opts.timeout)
	defer cancel()
	ctx, stop := notifyInterrupt(ctx)
	defer stop()

	result := deployResult{RegisteredModel: cfg.RegisteredModel, Endpoint: cfg.Endpoint}
	if cfg.registers() {
		// Fail on an endpoint air deploy won't change before copying any weights.
		if err := checkEndpoint(ctx, w, cfg.Endpoint, cfg.RegisteredModel); err != nil {
			if stopped := deployStopped(ctx, err); stopped != nil {
				return fmt.Errorf("%w while checking endpoint %s, before registering anything", stopped, cfg.Endpoint)
			}
			return err
		}
		result.IdempotencyKey = effectiveIdempotencyToken(opts.idempotencyKey, cfg.IdempotencyToken)
		resp, err := registerDeployment(ctx, w, cfg, opts, code, result.IdempotencyKey, pollInterval, !jsonOut)
		if err != nil {
			return err
		}
		result.ModelVersion, result.ModelID = resp.ModelVersion, resp.ModelID
		result.EnvironmentStagingFailed = resp.environmentState() == stagingStateFailed
		if !jsonOut {
			cmdio.LogString(ctx, fmt.Sprintf("Version %s of %s is ready.", stripControl(resp.ModelVersion), cfg.RegisteredModel))
			if result.EnvironmentStagingFailed {
				cmdio.LogString(ctx, "Warning: the version's serving environment couldn't be prepared ahead of time, so the endpoint can be slower to start, or fail to start.")
			}
		}
	} else {
		result.ModelVersion = *cfg.Model.Version
	}

	entity := cfg.servedEntity(result.ModelVersion)
	_, err = serveVersion(ctx, w, cfg.Endpoint, entity, !jsonOut)
	if err != nil {
		resume := deployResumeCommand(w.Config.Profile, opts, result.IdempotencyKey)
		if stopped := deployStopped(ctx, err); stopped != nil {
			return fmt.Errorf("%w while waiting for endpoint %s, which keeps updating. To continue, run:\n  %s", stopped, cfg.Endpoint, resume)
		}
		if errors.Is(err, errEndpointNotReady) {
			return err
		}
		return fmt.Errorf("serving version %s of %s on endpoint %s failed: %w\n\nTo try again without registering another version, run:\n  %s", result.ModelVersion, cfg.RegisteredModel, cfg.Endpoint, err, resume)
	}
	result.Status = "READY"
	result.EndpointURL = endpointQueryURL(w.Config.Host, cfg.Endpoint, cfg.Task)
	if jsonOut {
		return renderEnvelope(ctx, result)
	}
	out := cmd.OutOrStdout()
	fmt.Fprintf(out, "Endpoint %s serves version %s of %s.\n", cfg.Endpoint, result.ModelVersion, cfg.RegisteredModel)
	fmt.Fprintln(out, "Query it at: "+result.EndpointURL)
	return nil
}

// renderDeployDryRun reports a valid config and, for a checkpoint served by
// model.entry, the files a deployment would send.
func renderDeployDryRun(ctx context.Context, cfg *deployConfig, code *modelPackagePythonModel, jsonOut bool) error {
	var files []string
	if code != nil {
		files = codeFilePaths(code)
	}
	if jsonOut {
		return renderEnvelope(ctx, deployResult{Status: "DRY_RUN_OK", DryRun: true, CodeFiles: files})
	}
	if code != nil {
		cmdio.LogString(ctx, fmt.Sprintf("Dry run: model.entry would send %d file(s):", len(files)))
		for _, f := range files {
			cmdio.LogString(ctx, "  "+f)
		}
	}
	cmdio.LogString(ctx, fmt.Sprintf("Dry run: configuration for endpoint %q is valid; not deploying.", cfg.Endpoint))
	return nil
}

// registerDeployment registers the version and makes it ready to serve, showing
// progress on a spinner when show is true. Its errors say how to continue.
func registerDeployment(ctx context.Context, w *databricks.WorkspaceClient, cfg *deployConfig, opts deployOptions, code *modelPackagePythonModel, key string, pollInterval time.Duration, show bool) (*registerModelPackageResponse, error) {
	// Printed to stderr in every output mode, before the first request, so a run
	// that is killed can still be continued with this key.
	cmdio.LogString(ctx, fmt.Sprintf("Registering a version of %s with idempotency key %s", cfg.RegisteredModel, key))
	req, err := cfg.registrationRequest(key, code)
	if err != nil {
		return nil, withCode(codeInvalidArgs, false, err)
	}
	req.RetryFailedStaging = opts.retryFailedStaging

	progress := func(*registerModelPackageResponse) {}
	if show {
		sp := cmdio.NewSpinner(ctx)
		defer sp.Close()
		progress = func(resp *registerModelPackageResponse) {
			sp.Update(registrationProgress(cfg, resp))
		}
	}
	resp, err := registerAndFinalize(ctx, w, req, pollInterval, progress)
	if err == nil {
		return resp, nil
	}
	resume := deployResumeCommand(w.Config.Profile, opts, key)
	if errors.Is(err, errStagingFailed) {
		return nil, withCode(codeStagingFailed, true, fmt.Errorf("copying the weights in %s into version %s of %s failed. Check that you can read every file, then run:\n  %s --retry-failed-staging", *cfg.Model.Path, stripControl(resp.ModelVersion), cfg.RegisteredModel, resume))
	}
	if stopped := deployStopped(ctx, err); stopped != nil {
		continuing := ""
		switch {
		case resp == nil:
		case resp.weightsState() == stagingStateSucceeded:
			continuing = " The weights are copied, so continuing finalizes the version."
		case resp.weightsState() != "":
			continuing = " Copying the weights continues meanwhile."
		}
		return nil, fmt.Errorf("%w while registering a version of %s.%s To continue, run:\n  %s", stopped, cfg.RegisteredModel, continuing, resume)
	}
	if _, ok := errors.AsType[*finalizeError](err); ok {
		return nil, fmt.Errorf("finalizing version %s of %s failed: %w\n\nTo try again, run:\n  %s", stripControl(resp.ModelVersion), cfg.RegisteredModel, err, resume)
	}
	failed := fmt.Errorf("registering a version of %s with idempotency key %s failed: %w", cfg.RegisteredModel, key, err)
	// A retryable failure is retried with the same key, so no other version is
	// registered.
	if _, retryable := deployErrorCode(err); retryable {
		return nil, fmt.Errorf("%w\n\nTo try again, run:\n  %s", failed, resume)
	}
	return nil, failed
}

// deployStopped returns why air deploy stopped waiting, or nil if it didn't. The
// SDK's endpoint poller reports a canceled context as its own timeout, so the
// context decides between Ctrl-C and --timeout.
func deployStopped(ctx context.Context, err error) error {
	switch {
	case errors.Is(ctx.Err(), context.Canceled):
		return errDeployInterrupted
	case errors.Is(ctx.Err(), context.DeadlineExceeded):
		return errDeployTimedOut
	}
	if _, ok := errors.AsType[*retries.ErrTimedOut](err); ok {
		return errDeployTimedOut
	}
	return nil
}

// deployErrorCode classifies a failed deployment for the JSON error envelope.
// Failures that aren't the user's input or the server's answer, such as a lost
// connection or a failed endpoint update, are worth retrying with the printed
// command.
func deployErrorCode(err error) (code string, retryable bool) {
	switch {
	case errors.Is(err, errDeployInterrupted):
		return codeInterrupted, true
	case errors.Is(err, errDeployTimedOut):
		return codeTimeout, true
	}
	if coded, ok := errors.AsType[*deployError](err); ok {
		return coded.code, coded.retryable
	}
	if apiErr, ok := errors.AsType[*apierr.APIError](err); ok {
		retryable = apiErr.StatusCode == http.StatusTooManyRequests || apiErr.StatusCode >= http.StatusInternalServerError
		if !apiErrorCodeRe().MatchString(apiErr.ErrorCode) {
			return codeInternalError, retryable
		}
		return apiErr.ErrorCode, retryable
	}
	return codeInternalError, true
}

// validateDeployFlags rejects flags the config's model source can't use.
func validateDeployFlags(cfg *deployConfig, opts deployOptions) error {
	if opts.timeout <= 0 {
		return fmt.Errorf("--timeout must be positive, got %s", opts.timeout)
	}
	if opts.idempotencyKeySet {
		if !cfg.registers() {
			return errors.New("--idempotency-key applies only when air deploy registers a version; model.type is registered")
		}
		if err := validateDeployIdempotencyKey(opts.idempotencyKey, "--idempotency-key"); err != nil {
			return err
		}
	}
	if opts.retryFailedStaging && cfg.Model.Type != modelTypeCheckpoint {
		return fmt.Errorf("--retry-failed-staging applies only to a checkpoint; model.type is %s", cfg.Model.Type)
	}
	return nil
}

// registrationProgress describes a registration response on the spinner.
func registrationProgress(cfg *deployConfig, resp *registerModelPackageResponse) string {
	state := resp.weightsState()
	switch state {
	case "":
		return fmt.Sprintf("Registering version %s of %s…", stripControl(resp.ModelVersion), cfg.RegisteredModel)
	case stagingStateSucceeded:
		return fmt.Sprintf("Finalizing version %s of %s…", stripControl(resp.ModelVersion), cfg.RegisteredModel)
	}
	return fmt.Sprintf("Copying the weights into version %s of %s (%s)…", stripControl(resp.ModelVersion), cfg.RegisteredModel, stripControl(state))
}

// serveVersion serves entity on the endpoint, showing the endpoint's state on a
// spinner when show is true.
func serveVersion(ctx context.Context, w *databricks.WorkspaceClient, name string, entity serving.ServedEntityInput, show bool) (*serving.ServingEndpointDetailed, error) {
	if !show {
		return serveOnEndpoint(ctx, w, name, entity, nil)
	}
	cmdio.LogString(ctx, fmt.Sprintf("Serving version %s of %s on endpoint %s", entity.EntityVersion, entity.EntityName, name))
	sp := cmdio.NewSpinner(ctx)
	defer sp.Close()
	return serveOnEndpoint(ctx, w, name, entity, func(endpoint *serving.ServingEndpointDetailed) {
		if endpoint.State != nil {
			sp.Update(fmt.Sprintf("Waiting for endpoint %s (%s, %s)…", name, endpoint.State.Ready, endpoint.State.ConfigUpdate))
		}
	})
}

// deployResumeCommand is the command that continues this deployment: the same
// config (by absolute path, so it runs from any directory), overrides and
// profile, with the registration's key when it has one. The overrides are part
// of the request the key was first used with, so they must be repeated.
func deployResumeCommand(profile string, opts deployOptions, key string) string {
	file := opts.file
	if abs, err := filepath.Abs(file); err == nil {
		file = abs
	}
	args := []string{deployCommandPath, "-f", shellquote.BashArg(filepath.ToSlash(file))}
	for _, override := range opts.overrides {
		args = append(args, "--override", shellquote.BashArg(override))
	}
	if key != "" {
		args = append(args, "--idempotency-key", shellquote.BashArg(key))
	}
	if profile != "" {
		args = append(args, "-p", shellquote.BashArg(profile))
	}
	return strings.Join(args, " ")
}

// endpointQueryURL is the REST URL that queries the endpoint: an audio model's
// audio route, which takes a multipart upload, or invocations for any other model.
func endpointQueryURL(host, name string, task *string) string {
	base := strings.TrimSuffix(host, "/") + "/serving-endpoints/" + name
	if task != nil {
		switch *task {
		case taskAudioTranscriptions:
			return base + "/audio/transcriptions"
		case taskAudioTranslations:
			return base + "/audio/translations"
		}
	}
	return base + "/invocations"
}
