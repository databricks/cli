package aircmd

import (
	"context"
	"errors"
	"fmt"
	"path/filepath"
	"strconv"
	"strings"
	"time"

	"github.com/databricks/cli/cmd/root"
	"github.com/databricks/cli/libs/cmdctx"
	"github.com/databricks/cli/libs/cmdio"
	"github.com/databricks/cli/libs/flags"
	"github.com/databricks/cli/libs/shellquote"
	"github.com/databricks/databricks-sdk-go"
	"github.com/databricks/databricks-sdk-go/service/catalog"
	"github.com/spf13/cobra"
)

// defaultServeTimeout bounds the wait for the endpoint: for an update in progress
// that the new one waits behind, and with --watch, for the endpoint to be ready.
const defaultServeTimeout = time.Hour

// Statuses of a served version: the endpoint serves it and is ready, or is still
// updating to serve it.
const (
	serveStatusReady    = "READY"
	serveStatusUpdating = "UPDATING"
)

// serveResult is the JSON payload for `air serve`.
type serveResult struct {
	Status       string     `json:"status"`
	DryRun       bool       `json:"dry_run,omitempty"`
	Plan         *servePlan `json:"plan,omitempty"`
	UCModel      string     `json:"uc_model_name,omitempty"`
	ModelVersion string     `json:"model_version,omitempty"`
	Endpoint     string     `json:"endpoint,omitempty"`
	EndpointURL  string     `json:"endpoint_url,omitempty"`
}

// servePlan is what a dry run reports. A dry run makes no request, so the version
// isn't read, and the compute it gets isn't known yet.
type servePlan struct {
	UCModel  string `json:"uc_model_name"`
	Endpoint string `json:"endpoint_name"`
	// ModelVersion is empty when the newest READY version is served, picked when
	// serving.
	ModelVersion string `json:"model_version,omitempty"`
}

// serveOptions are the `air serve` flags.
type serveOptions struct {
	file         string
	overrides    []string
	ucModel      string
	endpoint     string
	modelVersion string
	// flagsSet records which of the model, endpoint and version flags were passed, so
	// an empty value (say, from an unset shell variable) fails instead of falling back
	// to the file or the newest version.
	flagsSet map[string]bool
	watch    bool
	dryRun   bool
	timeout  time.Duration
}

func newServeCommand() *cobra.Command {
	var opts serveOptions
	cmd := &cobra.Command{
		Use:   "serve",
		Args:  root.NoArgs,
		Short: "Serve a model version on a Model Serving endpoint",
		Long: `Serve a version of a Unity Catalog model on a Model Serving endpoint.

Describe the endpoint in a YAML file (see --file), or name the model and the
endpoint with --uc-model-name and --endpoint-name. air serve serves the newest
READY version unless model_version names one, and creates the endpoint if it
doesn't exist. The compute follows from what the version was registered for:
dedicated inference for a checkpoint registered without engine, and replicas for
any other version. Anything the file leaves out of compute takes a default.

air serve returns once the endpoint accepts the update; with --watch it waits
until the endpoint is ready. To look up a YAML field, pass its path to -h as a
separate argument:

  databricks air serve -h config                 all fields, with examples
  databricks air serve -h config.compute         one section`,
		Example: `  # Serve the newest READY version, and wait until the endpoint is ready
  databricks air serve --uc-model-name main.serving.qwen30b_sglang --endpoint-name qwen3-30b --watch

  # Serve a version on the compute a file describes
  databricks air serve -f serve.yaml

  # Validate the config without changing anything
  databricks air serve -f serve.yaml --dry-run`,
		// Hidden until the registration API is available in every workspace.
		Hidden: true,
	}
	cmd.SetHelpFunc(schemaHelp(serveSchema, serveCommandPath))

	// List flags in declaration order (most used first) rather than alphabetically.
	cmd.Flags().SortFlags = false
	cmd.Flags().StringVarP(&opts.file, "file", "f", "", "Path to the serving YAML config")
	cmd.Flags().StringVar(&opts.ucModel, ucModelNameFlag, "", "Unity Catalog model to serve, overriding the file's uc_model_name")
	cmd.Flags().StringVar(&opts.endpoint, endpointNameFlag, "", "Serving endpoint to serve on, overriding the file's endpoint_name")
	cmd.Flags().StringVar(&opts.modelVersion, modelVersionFlag, "", "Version to serve, overriding the file's model_version (default: the newest READY version)")
	cmd.Flags().BoolVar(&opts.watch, "watch", false, "Wait until the endpoint is ready")
	cmd.Flags().StringArrayVar(&opts.overrides, "override", nil, "Override a YAML field, e.g. compute.gpu.max_replicas=4 (repeatable)")
	cmd.Flags().BoolVar(&opts.dryRun, "dry-run", false, "Validate the config without changing anything")
	cmd.Flags().DurationVar(&opts.timeout, "timeout", defaultServeTimeout, "How long to wait for the endpoint")

	// --dry-run never authenticates, so it reports config mistakes without
	// credentials.
	cmd.PreRunE = func(cmd *cobra.Command, args []string) error {
		if opts.dryRun {
			return nil
		}
		return root.MustWorkspaceClient(cmd, args)
	}

	cmd.RunE = func(cmd *cobra.Command, args []string) error {
		opts.flagsSet = changedFlags(cmd, ucModelNameFlag, endpointNameFlag, modelVersionFlag)
		return renderModelError(cmd, runServe(cmd, opts))
	}
	return cmd
}

// validateServeFlags checks the model, endpoint and version flags that were passed,
// naming the flag in the error.
func validateServeFlags(opts serveOptions) error {
	for name, value := range map[string]string{ucModelNameFlag: opts.ucModel, endpointNameFlag: opts.endpoint, modelVersionFlag: opts.modelVersion} {
		if opts.flagsSet[name] && strings.TrimSpace(value) == "" {
			return fmt.Errorf("--%s cannot be empty", name)
		}
	}
	if opts.ucModel != "" {
		if err := validateUCModelName(opts.ucModel, "--"+ucModelNameFlag); err != nil {
			return err
		}
	}
	if opts.endpoint != "" && !endpointNameRe().MatchString(opts.endpoint) {
		return fmt.Errorf("--%s must be a serving endpoint name of letters, digits, '-' and '_', got %q", endpointNameFlag, opts.endpoint)
	}
	if opts.modelVersion != "" && !modelVersionRe().MatchString(opts.modelVersion) {
		return fmt.Errorf("--%s must be a model version number such as 3, got %q", modelVersionFlag, opts.modelVersion)
	}
	return nil
}

// loadServeConfig reads the config from --file, if set, then applies the
// --uc-model-name, --endpoint-name and --model-version flags over it, and validates
// the result.
func loadServeConfig(ctx context.Context, opts serveOptions) (*serveConfig, error) {
	if err := validateServeFlags(opts); err != nil {
		return nil, err
	}
	cfg := &serveConfig{}
	switch {
	case opts.file != "":
		loaded, _, err := loadModelConfig[serveConfig](ctx, opts.file, opts.overrides, serveSchema())
		if err != nil {
			return nil, err
		}
		cfg = loaded
	case len(opts.overrides) > 0:
		return nil, errors.New("--override applies to a file; set --file, or pass the fields as flags")
	case opts.ucModel == "" || opts.endpoint == "":
		return nil, errors.New("set --file, or both --uc-model-name and --endpoint-name")
	}
	if opts.ucModel != "" {
		cfg.UCModel = opts.ucModel
	}
	if opts.endpoint != "" {
		cfg.Endpoint = opts.endpoint
	}
	if opts.modelVersion != "" {
		cfg.ModelVersion = &opts.modelVersion
	}
	return cfg, cfg.validate()
}

func runServe(cmd *cobra.Command, opts serveOptions) error {
	ctx := cmd.Context()
	if opts.timeout <= 0 {
		return withCode(codeInvalidArgs, false, fmt.Errorf("--timeout must be positive, got %s", opts.timeout))
	}
	cfg, err := loadServeConfig(ctx, opts)
	if err != nil {
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

	version, err := resolveModelVersion(ctx, w, cfg)
	if err != nil {
		return err
	}
	if !jsonOut && cfg.ModelVersion == nil {
		cmdio.LogString(ctx, fmt.Sprintf("Serving version %s, the newest READY version of %s.", version, cfg.UCModel))
	}
	pkg, err := readVersionPackage(ctx, w, cfg.UCModel, version)
	if err != nil {
		return err
	}
	entity, err := cfg.servedEntity(version, pkg)
	if err != nil {
		return withCode(codeInvalidArgs, false, err)
	}

	ready, err := serveVersion(ctx, w, cfg.Endpoint, entity, opts.watch, !jsonOut)
	resume := serveResumeCommand(w.Config.Profile, opts, version)
	if err != nil {
		if stopped := stoppedWaiting(ctx, err); stopped != nil {
			return fmt.Errorf("air serve %w while waiting for endpoint %s, which keeps updating. To continue, run:\n  %s", stopped, cfg.Endpoint, resume)
		}
		if errors.Is(err, errEndpointNotReady) {
			return err
		}
		failed := fmt.Errorf("serving version %s of %s on endpoint %s failed: %w", version, cfg.UCModel, cfg.Endpoint, err)
		// A failure the same request would meet again, such as an endpoint serving
		// another model, gets no retry command.
		if _, retryable := modelErrorCode(err); !retryable {
			return failed
		}
		return fmt.Errorf("%w\n\nTo try again, run:\n  %s", failed, resume)
	}
	result := serveResult{
		Status:       serveStatusUpdating,
		UCModel:      cfg.UCModel,
		ModelVersion: version,
		Endpoint:     cfg.Endpoint,
		EndpointURL:  endpointQueryURL(w.Config.Host, cfg.Endpoint, pkg.task),
	}
	if ready {
		result.Status = serveStatusReady
	}
	if jsonOut {
		return renderEnvelope(ctx, result)
	}
	out := cmd.OutOrStdout()
	if ready {
		fmt.Fprintf(out, "Endpoint %s serves version %s of %s.\nQuery it at: %s\n", cfg.Endpoint, version, cfg.UCModel, result.EndpointURL)
		return nil
	}
	fmt.Fprintf(out, "Endpoint %s is updating to serve version %s of %s, at %s once it's ready. To wait for it, run:\n  %s\n", cfg.Endpoint, version, cfg.UCModel, result.EndpointURL, resume)
	return nil
}

// renderServeDryRun reports what a valid config serves, without changing anything.
func renderServeDryRun(ctx context.Context, cfg *serveConfig, jsonOut bool) error {
	plan := servePlan{UCModel: cfg.UCModel, Endpoint: cfg.Endpoint}
	if cfg.ModelVersion != nil {
		plan.ModelVersion = *cfg.ModelVersion
	}
	if jsonOut {
		return renderEnvelope(ctx, serveResult{Status: "DRY_RUN_OK", DryRun: true, Plan: &plan})
	}
	version := "the newest READY version"
	if plan.ModelVersion != "" {
		version = "version " + plan.ModelVersion
	}
	cmdio.LogString(ctx, fmt.Sprintf("Dry run: would serve %s of %s on endpoint %s, on the compute it was registered for.", version, cfg.UCModel, cfg.Endpoint))
	cmdio.LogString(ctx, fmt.Sprintf("Dry run: configuration for endpoint %q is valid; nothing was changed.", cfg.Endpoint))
	return nil
}

// resolveModelVersion returns the version to serve: the one the config names, else
// the highest-numbered READY version of the model.
func resolveModelVersion(ctx context.Context, w *databricks.WorkspaceClient, cfg *serveConfig) (string, error) {
	if cfg.ModelVersion != nil {
		return *cfg.ModelVersion, nil
	}
	versions, err := w.ModelVersions.ListAll(ctx, catalog.ListModelVersionsRequest{FullName: cfg.UCModel})
	if err != nil {
		return "", fmt.Errorf("listing the versions of %s: %w", cfg.UCModel, err)
	}
	newest := 0
	for _, v := range versions {
		if v.Status == catalog.ModelVersionInfoStatusReady && v.Version > newest {
			newest = v.Version
		}
	}
	if newest == 0 {
		return "", withCode(codeInvalidArgs, false, fmt.Errorf("%s has no READY version to serve; register one with %s, or set model_version", cfg.UCModel, registerCommandPath))
	}
	return strconv.Itoa(newest), nil
}

// serveVersion serves entity on the endpoint, waiting until it's ready when watch
// is set, and showing the endpoint's state on a spinner when show is true. It
// reports whether the endpoint serves entity and is ready.
func serveVersion(ctx context.Context, w *databricks.WorkspaceClient, name string, entity servedEntity, watch, show bool) (bool, error) {
	if !show {
		return serveOnEndpoint(ctx, w, name, entity, watch, nil)
	}
	cmdio.LogString(ctx, fmt.Sprintf("Serving version %s of %s on endpoint %s", entity.EntityVersion, entity.EntityName, name))
	sp := cmdio.NewSpinner(ctx)
	defer sp.Close()
	return serveOnEndpoint(ctx, w, name, entity, watch, func(endpoint *servingEndpoint) {
		if endpoint.State != nil {
			sp.Update(fmt.Sprintf("Waiting for endpoint %s (%s, %s)…", name, endpoint.State.Ready, endpoint.State.ConfigUpdate))
		}
	})
}

// serveResumeCommand is the command that waits for this update: the same config
// (by absolute path, so it runs from any directory), overrides, flags and profile,
// with the version served, so a newer one registered meanwhile isn't picked.
func serveResumeCommand(profile string, opts serveOptions, version string) string {
	args := []string{serveCommandPath}
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
	if opts.ucModel != "" {
		args = append(args, "--uc-model-name", shellquote.BashArg(opts.ucModel))
	}
	if opts.endpoint != "" {
		args = append(args, "--endpoint-name", shellquote.BashArg(opts.endpoint))
	}
	args = append(args, "--model-version", shellquote.BashArg(version), "--watch")
	if profile != "" {
		args = append(args, "-p", shellquote.BashArg(profile))
	}
	return strings.Join(args, " ")
}

// endpointQueryURL is the REST URL that queries the endpoint: an audio model's
// audio route, which takes a multipart upload, or invocations for any other model.
func endpointQueryURL(host, name, task string) string {
	base := strings.TrimSuffix(host, "/") + "/serving-endpoints/" + name
	switch task {
	case taskAudioTranscriptions:
		return base + "/audio/transcriptions"
	case taskAudioTranslations:
		return base + "/audio/translations"
	}
	return base + "/invocations"
}
