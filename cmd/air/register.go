package aircmd

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"path/filepath"
	"strings"
	"time"

	"github.com/databricks/cli/cmd/root"
	"github.com/databricks/cli/libs/cmdctx"
	"github.com/databricks/cli/libs/cmdio"
	"github.com/databricks/cli/libs/flags"
	"github.com/databricks/cli/libs/shellquote"
	"github.com/databricks/databricks-sdk-go"
	"github.com/spf13/cobra"
)

// defaultRegisterTimeout bounds the wait for a checkpoint's weights to be copied.
const defaultRegisterTimeout = time.Hour

// registerResult is the JSON payload for `air register`.
type registerResult struct {
	Status         string        `json:"status"`
	DryRun         bool          `json:"dry_run,omitempty"`
	Plan           *registerPlan `json:"plan,omitempty"`
	UCModel        string        `json:"uc_model_name,omitempty"`
	ModelVersion   string        `json:"model_version,omitempty"`
	ModelID        string        `json:"model_id,omitempty"`
	IdempotencyKey string        `json:"idempotency_key,omitempty"`
	// EnvironmentStagingFailed is set when the version's serving environment
	// couldn't be prepared ahead of time, so an endpoint can be slower to start.
	EnvironmentStagingFailed bool `json:"environment_staging_failed,omitempty"`
}

// registerPlan is the registration a dry run reports: what serves the checkpoint,
// command or dedicated, and the request that would register it, without a key.
type registerPlan struct {
	Variant string                      `json:"variant"`
	Request registerModelPackageRequest `json:"request"`
}

// registerOptions are the `air register` flags.
type registerOptions struct {
	file        string
	overrides   []string
	modelSource string
	ucModel     string
	// flagsSet records which of the model source and model flags were passed, so an
	// empty value (say, from an unset shell variable) fails instead of falling back to
	// the file.
	flagsSet map[string]bool
	dryRun   bool
	// idempotencyKeySet records an explicit --idempotency-key, so an empty value
	// (say, from an unset shell variable) fails instead of registering anew.
	idempotencyKeySet  bool
	idempotencyKey     string
	retryFailedStaging bool
	timeout            time.Duration
}

func newRegisterCommand() *cobra.Command {
	return newRegisterCommandWithPollInterval(registrationPollInterval)
}

func newRegisterCommandWithPollInterval(pollInterval time.Duration) *cobra.Command {
	var opts registerOptions
	cmd := &cobra.Command{
		Use:   "register",
		Args:  root.NoArgs,
		Short: "Register a checkpoint as a model version in Unity Catalog",
		Long: `Register a checkpoint in a Unity Catalog volume as a new version of a Unity
Catalog model, ready to serve with databricks air serve.

Describe the registration in a YAML file (see --file), or name the checkpoint and
the model with --model-source and --uc-model-name. A file with engine registers a
checkpoint served by your command; without engine, Databricks serves the
checkpoint with dedicated inference.

air register copies the checkpoint's weights into the version and waits until the
version is ready. Registration is idempotent: run air register again with the
idempotency key it prints to continue an interrupted registration. To look up a
YAML field, pass its path to -h as a separate argument:

  databricks air register -h config                 all fields, with examples
  databricks air register -h config.engine          one section`,
		Example: `  # Register a checkpoint described in a file
  databricks air register -f register.yaml

  # Register a checkpoint for dedicated inference
  databricks air register --model-source /Volumes/main/weights/gemma4_e4b --uc-model-name main.serving.gemma4

  # Continue a registration that was interrupted
  databricks air register -f register.yaml --idempotency-key 3f6c0b52-4c7e-4d0b-9b8e-0f7ad6a4c2d1`,
		// Hidden until the registration API is available in every workspace.
		Hidden: true,
	}
	cmd.SetHelpFunc(schemaHelp(registerSchema, registerCommandPath))

	// List flags in declaration order (most used first) rather than alphabetically.
	cmd.Flags().SortFlags = false
	cmd.Flags().StringVarP(&opts.file, "file", "f", "", "Path to the registration YAML config")
	cmd.Flags().StringVar(&opts.modelSource, modelSourceFlag, "", "Volume directory of the checkpoint, overriding the file's model_source")
	cmd.Flags().StringVar(&opts.ucModel, ucModelNameFlag, "", "Unity Catalog model to register a version of, overriding the file's uc_model_name")
	cmd.Flags().StringArrayVar(&opts.overrides, "override", nil, "Override a YAML field, e.g. engine.environment.version=6 (repeatable)")
	cmd.Flags().BoolVar(&opts.dryRun, "dry-run", false, "Validate the config, and show the plan, without changing anything")
	cmd.Flags().StringVar(&opts.idempotencyKey, "idempotency-key", "", "Continue the registration that used this key, instead of registering another version")
	cmd.Flags().BoolVar(&opts.retryFailedStaging, "retry-failed-staging", false, "Copy the checkpoint's weights again after copying them failed")
	cmd.Flags().DurationVar(&opts.timeout, "timeout", defaultRegisterTimeout, "How long to wait for the version to be ready")

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
		opts.flagsSet = changedFlags(cmd, modelSourceFlag, ucModelNameFlag)
		return renderModelError(cmd, runRegister(cmd, opts, pollInterval))
	}
	return cmd
}

// loadRegisterConfig reads the config from --file, if set, then applies the
// --model-source and --uc-model-name flags over it, and validates the result.
func loadRegisterConfig(ctx context.Context, opts registerOptions) (*registerConfig, error) {
	if err := validateRegisterFlags(opts); err != nil {
		return nil, err
	}
	cfg := &registerConfig{}
	switch {
	case opts.file != "":
		loaded, document, err := loadModelConfig[registerConfig](ctx, opts.file, opts.overrides, registerSchema())
		if err != nil {
			return nil, err
		}
		// YAML reads an engine whose fields are all commented out as no engine, which
		// would register the checkpoint for dedicated inference instead.
		if node, _ := mappingValue(document.Content[0], "engine"); loaded.Engine == nil && node != nil {
			return nil, errors.New("engine is empty: set engine.command and engine.environment, or remove engine to register the checkpoint for dedicated inference")
		}
		cfg = loaded
	case len(opts.overrides) > 0:
		return nil, errors.New("--override applies to a file; set --file, or pass the fields as flags")
	case opts.modelSource == "" || opts.ucModel == "":
		return nil, errors.New("set --file, or both --model-source and --uc-model-name")
	}
	if opts.modelSource != "" {
		cfg.ModelSource = opts.modelSource
	}
	if opts.ucModel != "" {
		cfg.UCModel = opts.ucModel
	}
	return cfg, cfg.validate()
}

// validateRegisterFlags checks the model source and model flags that were passed,
// naming the flag in the error.
func validateRegisterFlags(opts registerOptions) error {
	for name, value := range map[string]string{modelSourceFlag: opts.modelSource, ucModelNameFlag: opts.ucModel} {
		if opts.flagsSet[name] && strings.TrimSpace(value) == "" {
			return fmt.Errorf("--%s cannot be empty", name)
		}
	}
	if opts.modelSource != "" && !strings.HasPrefix(opts.modelSource, loggedModelSourcePrefix) {
		if err := validateVolumeDirectory("--"+modelSourceFlag, opts.modelSource); err != nil {
			return err
		}
	}
	if opts.ucModel != "" {
		return validateUCModelName(opts.ucModel, "--"+ucModelNameFlag)
	}
	return nil
}

func runRegister(cmd *cobra.Command, opts registerOptions, pollInterval time.Duration) error {
	ctx := cmd.Context()
	if opts.timeout <= 0 {
		return withCode(codeInvalidArgs, false, fmt.Errorf("--timeout must be positive, got %s", opts.timeout))
	}
	if opts.idempotencyKeySet {
		if err := validateRegistrationKey(opts.idempotencyKey); err != nil {
			return withCode(codeInvalidArgs, false, err)
		}
	}
	cfg, err := loadRegisterConfig(ctx, opts)
	if err != nil {
		return withCode(codeInvalidArgs, false, err)
	}

	jsonOut := root.OutputType(cmd) == flags.OutputJSON
	if opts.dryRun {
		return renderRegisterDryRun(ctx, cfg, jsonOut)
	}

	w := cmdctx.WorkspaceClient(ctx)
	ctx, cancel := context.WithTimeout(ctx, opts.timeout)
	defer cancel()
	ctx, stop := notifyInterrupt(ctx)
	defer stop()

	key := effectiveIdempotencyToken(opts.idempotencyKey, nil)
	resp, err := registerVersion(ctx, w, cfg, opts, key, pollInterval, !jsonOut)
	if err != nil {
		return err
	}
	result := registerResult{
		Status:                   "READY",
		UCModel:                  cfg.UCModel,
		ModelVersion:             resp.ModelVersion,
		ModelID:                  resp.ModelID,
		IdempotencyKey:           key,
		EnvironmentStagingFailed: resp.environmentState() == stagingStateFailed,
	}
	if jsonOut {
		return renderEnvelope(ctx, result)
	}
	if result.EnvironmentStagingFailed {
		cmdio.LogString(ctx, "Warning: the version's serving environment couldn't be prepared ahead of time, so an endpoint can be slower to start, or fail to start.")
	}
	serve := []string{serveCommandPath, "--" + ucModelNameFlag, shellquote.BashArg(cfg.UCModel), "--" + endpointNameFlag, "<endpoint>", "--" + modelVersionFlag, shellquote.BashArg(resp.ModelVersion)}
	if profile := w.Config.Profile; profile != "" {
		serve = append(serve, "-p", shellquote.BashArg(profile))
	}
	fmt.Fprintf(cmd.OutOrStdout(), "Version %s of %s is ready. To serve it, run:\n  %s\n", stripControl(resp.ModelVersion), cfg.UCModel, strings.Join(serve, " "))
	return nil
}

// renderRegisterDryRun reports the registration of a valid config, without
// changing anything.
func renderRegisterDryRun(ctx context.Context, cfg *registerConfig, jsonOut bool) error {
	plan := registerPlan{Variant: cfg.variant(), Request: cfg.registrationRequest("")}
	if jsonOut {
		return renderEnvelope(ctx, registerResult{Status: "DRY_RUN_OK", DryRun: true, Plan: &plan})
	}
	served := "its command"
	if plan.Variant == variantDedicated {
		served = "dedicated inference"
	}
	described, err := json.MarshalIndent(plan, "  ", "  ")
	if err != nil {
		return withCode(codeInternalError, false, err)
	}
	cmdio.LogString(ctx, fmt.Sprintf("Dry run: would register a new version of %s from %s, served by %s:\n  %s", cfg.UCModel, cfg.ModelSource, served, described))
	cmdio.LogString(ctx, "Dry run: configuration is valid; nothing was changed.")
	return nil
}

// registerVersion registers the version and makes it ready to serve, showing
// progress on a spinner when show is true. Its errors say how to continue.
func registerVersion(ctx context.Context, w *databricks.WorkspaceClient, cfg *registerConfig, opts registerOptions, key string, pollInterval time.Duration, show bool) (*registerModelPackageResponse, error) {
	// Printed to stderr in every output mode, before the first request, so a run
	// that is killed can still be continued with this key.
	cmdio.LogString(ctx, fmt.Sprintf("Registering a version of %s with idempotency key %s", cfg.UCModel, key))
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
	resume := registerResumeCommand(w.Config.Profile, opts, key)
	if errors.Is(err, errStagingFailed) {
		return nil, withCode(codeStagingFailed, true, fmt.Errorf("copying the weights in %s into version %s of %s failed. Check that you can read every file, then run:\n  %s --retry-failed-staging", cfg.ModelSource, stripControl(resp.ModelVersion), cfg.UCModel, resume))
	}
	if stopped := stoppedWaiting(ctx, err); stopped != nil {
		continuing := ""
		switch {
		case resp == nil:
		case resp.weightsState() == stagingStateSucceeded:
			continuing = " The weights are copied, so continuing finalizes the version."
		case resp.weightsState() != "":
			continuing = " Copying the weights continues meanwhile."
		}
		return nil, fmt.Errorf("air register %w while registering a version of %s.%s To continue, run:\n  %s", stopped, cfg.UCModel, continuing, resume)
	}
	if _, ok := errors.AsType[*finalizeError](err); ok {
		return nil, fmt.Errorf("finalizing version %s of %s failed: %w\n\nTo try again, run:\n  %s", stripControl(resp.ModelVersion), cfg.UCModel, err, resume)
	}
	failed := fmt.Errorf("registering a version of %s with idempotency key %s failed: %w", cfg.UCModel, key, err)
	// A retryable failure is retried with the same key, so no other version is
	// registered.
	if _, retryable := modelErrorCode(err); retryable {
		return nil, fmt.Errorf("%w\n\nTo try again, run:\n  %s", failed, resume)
	}
	return nil, failed
}

// registrationProgress describes a registration response on the spinner.
func registrationProgress(cfg *registerConfig, resp *registerModelPackageResponse) string {
	state := resp.weightsState()
	switch state {
	case "":
		return fmt.Sprintf("Registering version %s of %s…", stripControl(resp.ModelVersion), cfg.UCModel)
	case stagingStateSucceeded:
		return fmt.Sprintf("Finalizing version %s of %s…", stripControl(resp.ModelVersion), cfg.UCModel)
	}
	return fmt.Sprintf("Copying the weights into version %s of %s (%s)…", stripControl(resp.ModelVersion), cfg.UCModel, stripControl(state))
}

// registerResumeCommand is the command that continues this registration: the same
// config (by absolute path, so it runs from any directory), overrides, flags and
// profile, with the registration's key. The inputs are part of the request the key
// was first used with, so they must be repeated.
func registerResumeCommand(profile string, opts registerOptions, key string) string {
	args := []string{registerCommandPath}
	if opts.file != "" {
		file := opts.file
		if abs, err := filepath.Abs(file); err == nil {
			file = abs
		}
		args = append(args, "-f", shellquote.BashArg(filepath.ToSlash(file)))
	}
	for _, override := range opts.overrides {
		args = append(args, "--override", shellquote.BashArg(override))
	}
	if opts.modelSource != "" {
		args = append(args, "--model-source", shellquote.BashArg(opts.modelSource))
	}
	if opts.ucModel != "" {
		args = append(args, "--uc-model-name", shellquote.BashArg(opts.ucModel))
	}
	args = append(args, "--idempotency-key", shellquote.BashArg(key))
	if profile != "" {
		args = append(args, "-p", shellquote.BashArg(profile))
	}
	return strings.Join(args, " ")
}
