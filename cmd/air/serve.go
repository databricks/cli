package aircmd

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"net/http"
	"path/filepath"
	"regexp"
	"strconv"
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
	"github.com/databricks/databricks-sdk-go/service/catalog"
	"github.com/spf13/cobra"
)

// defaultServeTimeout bounds the whole wait: staging a checkpoint's weights and
// starting the endpoint.
const defaultServeTimeout = time.Hour

// Errors air serve wraps when it stops waiting. The server keeps working, and
// the error says how to continue.
var (
	errServeInterrupted = errors.New("air serve was interrupted")
	errServeTimedOut    = errors.New("air serve timed out")
)

// JSON error codes of air serve's own failures; API errors keep the server's.
const (
	codeInvalidArgs   = "INVALID_ARGS"
	codeInterrupted   = "INTERRUPTED"
	codeTimeout       = "TIMEOUT"
	codeStagingFailed = "STAGING_FAILED"
	// codeEndpointNotReady: the endpoint serves the version but isn't ready, which
	// running air serve again doesn't change.
	codeEndpointNotReady = "ENDPOINT_NOT_READY"
	codeInternalError    = "INTERNAL_ERROR"
)

// apiErrorCodeRe matches an error code the server sent. The SDK fills ErrorCode
// with "" or "<nil>" when the response body has none.
var apiErrorCodeRe = sync.OnceValue(func() *regexp.Regexp { return regexp.MustCompile(`^[A-Z][A-Z_]*$`) })

// serveError carries the JSON error code and retryability of a failure whose
// message is err's.
type serveError struct {
	code      string
	retryable bool
	err       error
}

func (e *serveError) Error() string { return e.err.Error() }

func (e *serveError) Unwrap() error { return e.err }

func withCode(code string, retryable bool, err error) error {
	return &serveError{code: code, retryable: retryable, err: err}
}

// servePlan is the deployment a dry run reports.
type servePlan struct {
	// Variant is the variant the file is read as: command, dedicated or registered.
	Variant string `json:"variant"`
	// Checkpoint is the checkpoint a new version is registered from. It is empty when
	// the version is already registered.
	Checkpoint string `json:"checkpoint,omitempty"`
	// ModelVersion is the version served, empty when a new version is registered, or
	// when the newest READY version is served, picked when serving.
	ModelVersion string `json:"model_version,omitempty"`
	// ServedEntity is the entity the endpoint serves, as the endpoints API takes it.
	ServedEntity any `json:"served_entity"`
}

// serveResult is the JSON payload for `air serve`.
type serveResult struct {
	Status          string     `json:"status"`
	DryRun          bool       `json:"dry_run,omitempty"`
	Plan            *servePlan `json:"plan,omitempty"`
	RegisteredModel string     `json:"registered_model,omitempty"`
	ModelVersion    string     `json:"model_version,omitempty"`
	ModelID         string     `json:"model_id,omitempty"`
	IdempotencyKey  string     `json:"idempotency_key,omitempty"`
	// EnvironmentStagingFailed is set when the version's serving environment
	// couldn't be prepared ahead of time, so the endpoint can be slower to start.
	EnvironmentStagingFailed bool   `json:"environment_staging_failed,omitempty"`
	Endpoint                 string `json:"endpoint,omitempty"`
	EndpointURL              string `json:"endpoint_url,omitempty"`
}

// serveOptions are the `air serve` flags.
type serveOptions struct {
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

func newServeCommand() *cobra.Command {
	return newServeCommandWithPollInterval(registrationPollInterval)
}

func newServeCommandWithPollInterval(pollInterval time.Duration) *cobra.Command {
	var opts serveOptions

	cmd := &cobra.Command{
		Use:   "serve",
		Args:  root.NoArgs,
		Short: "Register a model in Unity Catalog and serve it on an endpoint",
		Long: `Register a model in Unity Catalog and serve it on a Model Serving endpoint.

The deployment is described by a YAML file (see --file). Its fields say what is
served, and which variant the file is read as:
  checkpoint and command   a checkpoint in a volume, served by a command you give
  checkpoint               a checkpoint served with dedicated inference
  neither                  a model version already registered in Unity Catalog

For a checkpoint, air serve registers a new version, waits until it is ready,
then creates or updates the endpoint to serve it and waits until the endpoint is
ready. For a registered version, it only creates or updates the endpoint.

Registration is idempotent: run air serve again with the idempotency key it
prints to continue an interrupted deployment. To look up a YAML field, pass
its path to -h as a separate argument, and see one YAML example per variant
with -h config:

  databricks air serve -h config                 all fields, with examples
  databricks air serve -h config.compute         one section
  databricks air serve -h config.checkpoint      one field`,
		Example: `  # Register a checkpoint and serve it
  databricks air serve -f serve.yaml

  # Validate the config, and show the plan, without changing anything
  databricks air serve -f serve.yaml --dry-run

  # Continue a deployment that was interrupted
  databricks air serve -f serve.yaml --idempotency-key 3f6c0b52-4c7e-4d0b-9b8e-0f7ad6a4c2d1`,
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
		field, err := resolveSchemaField(serveSchema(), fields[0])
		if err != nil {
			c.PrintErrln("Error:", err)
			return
		}
		renderConfigField(c.OutOrStdout(), serveCommandPath, field, nil)
	})

	// List flags in declaration order (most used first) rather than alphabetically.
	cmd.Flags().SortFlags = false
	cmd.Flags().StringVarP(&opts.file, "file", "f", "", "Path to the deployment YAML config")
	cmd.Flags().StringArrayVar(&opts.overrides, "override", nil, "Override a YAML field, e.g. compute.num_accelerators=4 (repeatable)")
	cmd.Flags().BoolVar(&opts.dryRun, "dry-run", false, "Validate the config, and show the plan, without changing anything")
	cmd.Flags().StringVar(&opts.idempotencyKey, "idempotency-key", "", "Continue the registration that used this key, instead of registering another version")
	cmd.Flags().BoolVar(&opts.retryFailedStaging, "retry-failed-staging", false, "Copy a checkpoint's weights again after copying them failed")
	cmd.Flags().DurationVar(&opts.timeout, "timeout", defaultServeTimeout, "How long to wait for the version and the endpoint to be ready")
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
		err := runServe(cmd, opts, pollInterval)
		if err == nil {
			return nil
		}
		code, retryable := serveErrorCode(err)
		kind := "PERMANENT"
		if retryable {
			kind = "TRANSIENT"
		}
		return renderError(cmd.Context(), cmd, code, kind, retryable, err)
	}
	return cmd
}

func runServe(cmd *cobra.Command, opts serveOptions, pollInterval time.Duration) error {
	ctx := cmd.Context()
	cfg, err := loadServeConfig(ctx, opts.file, opts.overrides)
	if err != nil {
		return withCode(codeInvalidArgs, false, err)
	}
	if err := validateServeFlags(cfg, opts); err != nil {
		return withCode(codeInvalidArgs, false, err)
	}

	jsonOut := root.OutputType(cmd) == flags.OutputJSON
	if opts.dryRun {
		return renderServeDryRun(ctx, cfg, jsonOut)
	}

	w := cmdctx.WorkspaceClient(ctx)
	ctx, cancel := context.WithTimeout(ctx, opts.timeout)
	defer cancel()
	ctx, stop := notifyInterrupt(ctx)
	defer stop()

	result := serveResult{RegisteredModel: cfg.RegisteredModel, Endpoint: cfg.Endpoint}
	if cfg.registers() {
		// Fail on an endpoint air serve won't change before copying any weights.
		if err := checkEndpoint(ctx, w, cfg.Endpoint, cfg.RegisteredModel, cfg.variant() == variantDedicated); err != nil {
			if stopped := serveStopped(ctx, err); stopped != nil {
				return fmt.Errorf("%w while checking endpoint %s, before registering anything", stopped, cfg.Endpoint)
			}
			return err
		}
		result.IdempotencyKey = effectiveIdempotencyToken(opts.idempotencyKey, cfg.IdempotencyToken)
		resp, err := registerDeployment(ctx, w, cfg, opts, result.IdempotencyKey, pollInterval, !jsonOut)
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
		version, err := resolveModelVersion(ctx, w, cfg)
		if err != nil {
			return err
		}
		result.ModelVersion = version
		if !jsonOut && cfg.ModelVersion == nil {
			cmdio.LogString(ctx, fmt.Sprintf("Serving version %s, the newest READY version of %s.", version, cfg.RegisteredModel))
		}
	}

	entity, err := cfg.servedEntity(result.ModelVersion)
	if err != nil {
		return withCode(codeInvalidArgs, false, err)
	}
	_, err = serveVersion(ctx, w, cfg.Endpoint, entity, !jsonOut)
	if err != nil {
		resume := serveResumeCommand(w.Config.Profile, opts, result.IdempotencyKey)
		if stopped := serveStopped(ctx, err); stopped != nil {
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

// plan is the deployment a dry run reports. A dry run makes no request, so the version
// a new registration produces isn't known, and a registered model without
// model_version serves its newest READY version, picked when serving.
func (c *serveConfig) plan() (servePlan, error) {
	p := servePlan{Variant: c.variant()}
	if c.registers() {
		p.Checkpoint = *c.Checkpoint
	}
	if c.ModelVersion != nil {
		p.ModelVersion = *c.ModelVersion
	}
	entity, err := c.servedEntity(p.ModelVersion)
	if err != nil {
		return servePlan{}, err
	}
	if entity.Dedicated != nil {
		p.ServedEntity = entity.dedicatedJSON()
	} else {
		p.ServedEntity = entity.servingInput()
	}
	return p, nil
}

// renderServeDryRun reports the plan of a valid config, without changing anything.
func renderServeDryRun(ctx context.Context, cfg *serveConfig, jsonOut bool) error {
	plan, err := cfg.plan()
	if err != nil {
		return withCode(codeInvalidArgs, false, err)
	}
	if jsonOut {
		return renderEnvelope(ctx, serveResult{Status: "DRY_RUN_OK", DryRun: true, Plan: &plan})
	}
	cmdio.LogString(ctx, fmt.Sprintf("Dry run: %s.", cfg.readAs()))
	switch {
	case plan.Checkpoint != "":
		cmdio.LogString(ctx, fmt.Sprintf("Dry run: would register a new version of %s from %s, and serve it.", cfg.RegisteredModel, plan.Checkpoint))
	case plan.ModelVersion != "":
		cmdio.LogString(ctx, fmt.Sprintf("Dry run: would serve version %s of %s.", plan.ModelVersion, cfg.RegisteredModel))
	default:
		cmdio.LogString(ctx, fmt.Sprintf("Dry run: would serve the newest READY version of %s, picked when serving.", cfg.RegisteredModel))
	}
	served, err := json.MarshalIndent(plan.ServedEntity, "  ", "  ")
	if err != nil {
		return withCode(codeInternalError, false, err)
	}
	cmdio.LogString(ctx, fmt.Sprintf("Dry run: would serve it on endpoint %s as:\n  %s", cfg.Endpoint, served))
	cmdio.LogString(ctx, fmt.Sprintf("Dry run: configuration for endpoint %q is valid; nothing was changed.", cfg.Endpoint))
	return nil
}

// resolveModelVersion returns the version of a registered model to serve: model_version
// when the config sets it, else the highest-numbered READY version of the model.
func resolveModelVersion(ctx context.Context, w *databricks.WorkspaceClient, cfg *serveConfig) (string, error) {
	if cfg.ModelVersion != nil {
		return *cfg.ModelVersion, nil
	}
	versions, err := w.ModelVersions.ListAll(ctx, catalog.ListModelVersionsRequest{FullName: cfg.RegisteredModel})
	if err != nil {
		return "", fmt.Errorf("listing the versions of %s: %w", cfg.RegisteredModel, err)
	}
	newest := 0
	for _, v := range versions {
		if v.Status == catalog.ModelVersionInfoStatusReady && v.Version > newest {
			newest = v.Version
		}
	}
	if newest == 0 {
		return "", withCode(codeInvalidArgs, false, fmt.Errorf("%s has no READY version to serve; register and finalize a version, or set model_version", cfg.RegisteredModel))
	}
	return strconv.Itoa(newest), nil
}

// registerDeployment registers the version and makes it ready to serve, showing
// progress on a spinner when show is true. Its errors say how to continue.
func registerDeployment(ctx context.Context, w *databricks.WorkspaceClient, cfg *serveConfig, opts serveOptions, key string, pollInterval time.Duration, show bool) (*registerModelPackageResponse, error) {
	// Printed to stderr in every output mode, before the first request, so a run
	// that is killed can still be continued with this key.
	cmdio.LogString(ctx, fmt.Sprintf("Registering a version of %s with idempotency key %s", cfg.RegisteredModel, key))
	req := cfg.registrationRequest(key)
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
	resume := serveResumeCommand(w.Config.Profile, opts, key)
	if errors.Is(err, errStagingFailed) {
		return nil, withCode(codeStagingFailed, true, fmt.Errorf("copying the weights in %s into version %s of %s failed. Check that you can read every file, then run:\n  %s --retry-failed-staging", *cfg.Checkpoint, stripControl(resp.ModelVersion), cfg.RegisteredModel, resume))
	}
	if stopped := serveStopped(ctx, err); stopped != nil {
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
	if _, retryable := serveErrorCode(err); retryable {
		return nil, fmt.Errorf("%w\n\nTo try again, run:\n  %s", failed, resume)
	}
	return nil, failed
}

// serveStopped returns why air serve stopped waiting, or nil if it didn't. The
// SDK's endpoint poller reports a canceled context as its own timeout, so the
// context decides between Ctrl-C and --timeout.
func serveStopped(ctx context.Context, err error) error {
	switch {
	case errors.Is(ctx.Err(), context.Canceled):
		return errServeInterrupted
	case errors.Is(ctx.Err(), context.DeadlineExceeded):
		return errServeTimedOut
	}
	if _, ok := errors.AsType[*retries.ErrTimedOut](err); ok {
		return errServeTimedOut
	}
	return nil
}

// serveErrorCode classifies a failed deployment for the JSON error envelope.
// Failures that aren't the user's input or the server's answer, such as a lost
// connection or a failed endpoint update, are worth retrying with the printed
// command.
func serveErrorCode(err error) (code string, retryable bool) {
	switch {
	case errors.Is(err, errServeInterrupted):
		return codeInterrupted, true
	case errors.Is(err, errServeTimedOut):
		return codeTimeout, true
	}
	if coded, ok := errors.AsType[*serveError](err); ok {
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

// validateServeFlags rejects flags the config's variant can't use.
func validateServeFlags(cfg *serveConfig, opts serveOptions) error {
	if opts.timeout <= 0 {
		return fmt.Errorf("--timeout must be positive, got %s", opts.timeout)
	}
	if opts.idempotencyKeySet {
		if !cfg.registers() {
			return fmt.Errorf("--idempotency-key applies only when air serve registers a version; %s", cfg.readAs())
		}
		if err := validateServeIdempotencyKey(opts.idempotencyKey, "--idempotency-key"); err != nil {
			return err
		}
	}
	if opts.retryFailedStaging && !cfg.registers() {
		return fmt.Errorf("--retry-failed-staging applies only to a checkpoint; %s", cfg.readAs())
	}
	return nil
}

// registrationProgress describes a registration response on the spinner.
func registrationProgress(cfg *serveConfig, resp *registerModelPackageResponse) string {
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
func serveVersion(ctx context.Context, w *databricks.WorkspaceClient, name string, entity servedEntity, show bool) (*servingEndpoint, error) {
	if !show {
		return serveOnEndpoint(ctx, w, name, entity, nil)
	}
	cmdio.LogString(ctx, fmt.Sprintf("Serving version %s of %s on endpoint %s", entity.EntityVersion, entity.EntityName, name))
	sp := cmdio.NewSpinner(ctx)
	defer sp.Close()
	return serveOnEndpoint(ctx, w, name, entity, func(endpoint *servingEndpoint) {
		if endpoint.State != nil {
			sp.Update(fmt.Sprintf("Waiting for endpoint %s (%s, %s)…", name, endpoint.State.Ready, endpoint.State.ConfigUpdate))
		}
	})
}

// serveResumeCommand is the command that continues this deployment: the same
// config (by absolute path, so it runs from any directory), overrides and
// profile, with the registration's key when it has one. The overrides are part
// of the request the key was first used with, so they must be repeated.
func serveResumeCommand(profile string, opts serveOptions, key string) string {
	file := opts.file
	if abs, err := filepath.Abs(file); err == nil {
		file = abs
	}
	args := []string{serveCommandPath, "-f", shellquote.BashArg(filepath.ToSlash(file))}
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
