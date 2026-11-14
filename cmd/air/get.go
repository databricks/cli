package aircmd

import (
	"context"
	"errors"
	"fmt"
	"path"
	"strconv"
	"strings"
	"sync"

	"github.com/databricks/cli/cmd/root"
	"github.com/databricks/cli/libs/auth"
	"github.com/databricks/cli/libs/cmdctx"
	"github.com/databricks/cli/libs/flags"
	"github.com/databricks/cli/libs/log"
	"github.com/databricks/databricks-sdk-go"
	"github.com/databricks/databricks-sdk-go/apierr"
	"github.com/databricks/databricks-sdk-go/config"
	"github.com/databricks/databricks-sdk-go/service/iam"
	"github.com/databricks/databricks-sdk-go/service/jobs"
	"github.com/spf13/cobra"
)

// getData is the payload printed by `air get`. The json-tagged fields form
// the machine-readable output; fields tagged `json:"-"` are shown only in the
// human-readable text view.
type getData struct {
	RunID             string  `json:"run_id"`
	Status            string  `json:"status"`
	StartedAt         *string `json:"started_at"`
	DurationSeconds   *int64  `json:"duration_seconds"`
	AttemptNumber     int     `json:"attempt_number"`
	ExperimentName    *string `json:"experiment_name"`
	DashboardURL      string  `json:"dashboard_url"`
	MLflowURL         *string `json:"mlflow_url"`
	TerminationReason *string `json:"termination_reason,omitempty"`
	// StatusMessage is the progress message the text view shows as the status,
	// set only while the run is unfinished.
	StatusMessage *string `json:"status_message,omitempty"`
	// ESTRemainingSeconds and ESTPercentComplete are a best-effort progress
	// estimate for a running run, or null when one can't be made (see
	// estimateTrainingETA).
	ESTRemainingSeconds *int64 `json:"est_remaining_seconds"`
	ESTPercentComplete  *int   `json:"est_percent_complete"`

	// The fields below carry the values of the text view's Configuration and
	// Metadata boxes and its sweep summary. Missing values are null; config-based
	// retry limits require opting in to reading the stored config.
	User            *string `json:"user"`
	AcceleratorType *string `json:"accelerator_type"`
	NumAccelerators *int    `json:"num_accelerators"`
	// EnvironmentVersion and BaseEnvironment come from the run's serverless
	// environment spec; at most one is set.
	EnvironmentVersion *string `json:"environment_version"`
	BaseEnvironment    *string `json:"base_environment"`
	// MaxRetries is the run's retry limit and its source (see resolveMaxRetries).
	MaxRetries *runMaxRetries `json:"max_retries"`
	// Config is the stored run config the Configuration box shows, before display
	// reformatting. ConfigSource names where it came from: "uploaded" (the
	// training_config.yaml `air run` uploads after applying overrides and
	// stripping comments), or "yaml_parameters" / "yaml_parameters_file" (legacy
	// runs, a different schema). A config synthesized for display is not exposed.
	Config       *string `json:"-"`
	ConfigSource *string `json:"-"`
	// MLflowRunName is the MLflow run's name, which the MLflow Run cell shows.
	MLflowRunName *string `json:"mlflow_run_name"`
	// Sweep is set for foreach runs; its table replaces the single-run view.
	Sweep *sweepInfo `json:"sweep"`

	// configBody is the Configuration box body, which may be synthesized for a
	// legacy run without stored parameters.
	configBody string
	// detailsResolved records that enrichRunDetails has run.
	detailsResolved bool

	// The fields below are pre-rendered text-view cells, excluded from JSON.
	// Each shows "N/A" when its value is missing. The styled single-run renderer
	// (render.go) consumes them; the Run ID, Status, and MLflow Run cells it draws
	// are styled and hyperlinked there rather than stored here.
	SubmittedDisplay    string `json:"-"`
	DurationDisplay     string `json:"-"`
	ExperimentDisplay   string `json:"-"`
	UserDisplay         string `json:"-"`
	AcceleratorsDisplay string `json:"-"`
	EnvironmentDisplay  string `json:"-"`
	// ProgressDisplay is the pre-rendered "Progress" cell ("45% · ~2h 15m left"),
	// set only for a running run with an estimable remaining time.
	ProgressDisplay string `json:"-"`
	DisplayStatus   string `json:"-"`
	// TrainingConfigPath is the run's config file, downloaded for the config box.
	TrainingConfigPath string `json:"-"`
}

// getTemplate is the text-mode layout for a sweep (foreach) run. Single runs are
// drawn by the styled renderer in render.go and never reach this template; it is
// used only when .Data.Sweep is set. It reads from the JSON envelope, so every
// field is reached through ".Data".
const getTemplate = `Sweep Run ID: {{.Data.RunID}}
Status:       {{if .Data.DisplayStatus}}{{.Data.DisplayStatus}}{{else}}{{.Data.Status}}{{end}}
Total:        {{.Data.Sweep.Total}}
Completed:    {{.Data.Sweep.Completed}}
Succeeded:    {{.Data.Sweep.Succeeded}}
Failed:       {{.Data.Sweep.Failed}}
Active:       {{.Data.Sweep.Active}}
{{- if .Data.Sweep.Tasks}}

Sweep Tasks:
{{printf "  %-24s %-14s %-12s %s" "TASK" "RUN ID" "STATUS" "EXPERIMENT"}}
{{- range .Data.Sweep.Tasks}}
{{printf "  %-24s %-14s %-12s %s" .TaskKey .RunID .Status .Experiment}}
{{- end}}
{{- end}}
`

// errNoProfile is the actionable message shown when no credentials are
// configured: no default profile, no --profile (-p), and no auth environment.
var errNoProfile = errors.New("no default profile is set: pass --profile (-p) or configure a default profile in your .databrickscfg")

// authError classifies a workspace-client or Me() probe failure. Only genuinely
// auth-shaped errors surface as UNAUTHENTICATED/PERMANENT: missing profile,
// SDK auth wrappers, or an API 401/403. Anything else (network blip, 429, 5xx)
// is transient and reported as INTERNAL_ERROR/TRANSIENT so the caller can retry.
func authError(ctx context.Context, cmd *cobra.Command, err error) error {
	if errors.Is(err, config.ErrCannotConfigureDefault) {
		return renderError(ctx, cmd, "UNAUTHENTICATED", "PERMANENT", false, errNoProfile)
	}
	if errors.Is(err, apierr.ErrUnauthenticated) || errors.Is(err, apierr.ErrPermissionDenied) {
		return renderError(ctx, cmd, "UNAUTHENTICATED", "PERMANENT", false,
			fmt.Errorf("authentication was not successful: %w", err))
	}
	return renderError(ctx, cmd, "INTERNAL_ERROR", "TRANSIENT", true,
		fmt.Errorf("failed to verify authentication: %w", err))
}

// credentialError tells a bad credential apart from missing access after a call
// fails with 401/403. MustWorkspaceClient only attaches credentials (it does not
// check a PAT server-side), so the Me() probe runs here, only on failure. It
// returns the probe's error when the credentials themselves fail, or nil.
func credentialError(ctx context.Context, w *databricks.WorkspaceClient, err error) error {
	if !errors.Is(err, apierr.ErrUnauthenticated) && !errors.Is(err, apierr.ErrPermissionDenied) {
		return nil
	}
	_, meErr := w.CurrentUser.Me(ctx, iam.MeRequest{})
	return meErr
}

// newGetCommand returns the `air get JOB_RUN_ID` command, which shows status,
// configuration, and timing details for a specific run.
func newGetCommand() *cobra.Command {
	var includeConfig bool
	cmd := &cobra.Command{
		Use:   "get JOB_RUN_ID",
		Args:  root.ExactArgs(1),
		Short: "Show a run's status, configuration, and timing",
		Long: `Show a run's status, configuration, and timing.

JOB_RUN_ID is the job run ID returned by "databricks air run".`,
		Annotations: map[string]string{
			"template": getTemplate,
		},
	}

	cmd.Flags().BoolVar(&includeConfig, "include-config", false, "Include stored config in JSON (downloads over 1 MiB are omitted; may contain plaintext env_variables and parameters)")

	// Resolve and authenticate the workspace client up front so an auth failure
	// fails fast here, before any run status or config is fetched or printed.
	// ErrAlreadyPrinted passes through (it was handled upstream); other failures
	// become an actionable auth error (JSON envelope in -o json mode).
	cmd.PreRunE = func(cmd *cobra.Command, args []string) error {
		err := root.MustWorkspaceClient(cmd, args)
		if err == nil || errors.Is(err, root.ErrAlreadyPrinted) {
			return err
		}
		return authError(cmd.Context(), cmd, err)
	}

	cmd.RunE = func(cmd *cobra.Command, args []string) error {
		ctx := cmd.Context()
		if includeConfig && root.OutputType(cmd) == flags.OutputText {
			return renderError(ctx, cmd, "INVALID_ARGS", "PERMANENT", false, errors.New("--include-config requires -o json"))
		}
		w := cmdctx.WorkspaceClient(ctx)

		runID, err := strconv.ParseInt(args[0], 10, 64)
		if err != nil || runID <= 0 {
			return renderError(ctx, cmd, "INVALID_ARGS", "PERMANENT", false,
				fmt.Errorf("invalid JOB_RUN_ID %q: must be a positive integer", args[0]))
		}

		run, err := w.Jobs.GetRun(ctx, jobs.GetRunRequest{RunId: runID})
		if err != nil {
			if credErr := credentialError(ctx, w, err); credErr != nil {
				return authError(ctx, cmd, credErr)
			}
			// The backend returns this when the run ID is unknown to the user.
			if errors.Is(err, apierr.ErrResourceDoesNotExist) {
				return renderError(ctx, cmd, "NOT_FOUND", "NOT_FOUND", false,
					fmt.Errorf("run %d not found: check the run ID and that it is a job run ID", runID))
			}
			return renderError(ctx, cmd, "INTERNAL_ERROR", "TRANSIENT", true,
				fmt.Errorf("failed to get status for run %d: %w", runID, err))
		}

		// Prefer the configured workspace ID; only look it up when it is unset.
		id, err := auth.ResolveWorkspaceID(ctx, w)
		if err != nil {
			return renderError(ctx, cmd, "INTERNAL_ERROR", "TRANSIENT", true,
				fmt.Errorf("failed to get workspace id for run %d: %w", runID, err))
		}
		workspaceID, err := strconv.ParseInt(id, 10, 64)
		if err != nil {
			return renderError(ctx, cmd, "INVALID_ARGS", "PERMANENT", false,
				fmt.Errorf("configured workspace_id %q is not a numeric workspace ID", id))
		}

		data := buildGetData(run)
		data.DashboardURL = dashboardURL(w.Config.Host, runID, workspaceID)
		taskOutput, taskOutputErr := latestTaskOutput(ctx, w, run.Tasks)
		ids := mlflowIDsFromRetries(ctx, w, run.Tasks, isTerminal(run), taskOutput, taskOutputErr)
		if ids != nil {
			url := mlflowLogsURL(w.Config.Host, ids)
			data.MLflowURL = &url
			// A remaining-time estimate only makes sense while the run is still
			// training; a terminal run is either done or stopped mid-progress.
			if isRunning(run) {
				if eta := estimateTrainingETA(ctx, w, ids.RunID); eta != nil {
					secs := eta.RemainingSeconds
					pct := eta.PercentComplete
					data.ESTRemainingSeconds = &secs
					data.ESTPercentComplete = &pct
					data.ProgressDisplay = eta.detailed()
				}
			}
		}
		statusMessage := ""
		if taskOutput != nil && run.State != nil && run.State.ResultState == "" {
			statusMessage = normalizeStatusMessage(taskOutput.StatusMessage)
		}
		if statusMessage != "" {
			data.StatusMessage = &statusMessage
		}
		if task := findForEachTask(run); task != nil {
			data.Sweep = buildSweepInfo(ctx, w, task)
		} else {
			if genAIComputeTask(run) == nil {
				enrichFromAiRuntimeTask(run, &data)
			}
			enrichCtx := ctx
			if root.OutputType(cmd) != flags.OutputText {
				// A failed read already shows as a null field; keep stderr quiet
				// for callers that merge it into stdout.
				enrichCtx = withQuietEnrichment(ctx)
			}
			enrichRunDetails(enrichCtx, w, run, &data, ids, root.OutputType(cmd) == flags.OutputText || includeConfig)
		}

		if root.OutputType(cmd) != flags.OutputText {
			if includeConfig {
				return renderEnvelope(ctx, struct {
					getData
					Config       *string `json:"config"`
					ConfigSource *string `json:"config_source"`
				}{data, data.Config, data.ConfigSource})
			}
			return renderEnvelope(ctx, data)
		}

		out := cmd.OutOrStdout()
		if data.Sweep != nil {
			// A sweep has no single status, config, or timing, so lead with the
			// job run link and render the foreach summary table (getTemplate).
			fmt.Fprintf(out, "Job Link: %s\n\n", hyperlink(ctx, out, data.DashboardURL, data.DashboardURL))
			return renderEnvelope(ctx, data)
		}
		data.DisplayStatus = detailedDisplayRunStatus(run, statusMessage)

		renderRunText(ctx, out, w, run, &data, ids)
		return nil
	}

	return cmd
}

// enrichFromAiRuntimeTask fills the config path, experiment, and accelerators
// from the run's ai_runtime_task. Best-effort: empty fields leave the existing
// "N/A" fallbacks in place.
func enrichFromAiRuntimeTask(run *jobs.Run, data *getData) {
	task := aiRuntimeTaskOf(run)
	if task == nil {
		return
	}
	if len(task.Deployments) > 0 {
		d := task.Deployments[0]
		if d.CommandPath != "" {
			data.TrainingConfigPath = path.Join(path.Dir(d.CommandPath), trainingConfigName)
		}
		setAccelerators(data, string(d.Compute.AcceleratorType), d.Compute.AcceleratorCount)
	}
	if task.Experiment != "" {
		exp := stripExperimentUserPrefix(task.Experiment)
		data.ExperimentName = &exp
		data.ExperimentDisplay = exp
	}
}

// aiRuntimeTaskOf returns the run's first ai_runtime_task, or nil.
func aiRuntimeTaskOf(run *jobs.Run) *jobs.AiRuntimeTask {
	if len(run.Tasks) == 0 {
		return nil
	}
	return run.Tasks[0].AiRuntimeTask
}

// buildGetData extracts the fields we display from a run. The text-view cells
// are pre-rendered here with their "N/A" fallbacks; the styled renderer adds the
// hyperlinks and colors once the dashboard and MLflow identifiers are known.
func buildGetData(run *jobs.Run) getData {
	data := getData{
		RunID:             strconv.FormatInt(run.RunId, 10),
		Status:            displayRunStatus(run),
		DisplayStatus:     displayRunStatus(run),
		TerminationReason: terminationReason(run),
		StartedAt:         startedAt(run),
		DurationSeconds:   durationSeconds(run),
		AttemptNumber:     latestAttemptNumber(run),
		ExperimentName:    experimentName(run),
	}
	data.SubmittedDisplay = submittedDisplay(run)
	data.DurationDisplay = na
	if data.DurationSeconds != nil {
		data.DurationDisplay = formatDuration(*data.DurationSeconds)
	}
	data.ExperimentDisplay = na
	if data.ExperimentName != nil {
		data.ExperimentDisplay = *data.ExperimentName
	}
	data.User = nilIfEmpty(run.CreatorUserName)
	data.UserDisplay = orNA(run.CreatorUserName)
	data.AcceleratorsDisplay = na
	accelType, count := jobCompute(run)
	setAccelerators(&data, accelType, count)
	// EnvironmentDisplay is resolved at render time: the serverless environment
	// version needs a raw GetRun read (the typed SDK Run drops it), so it is not
	// filled here alongside the fields read straight off the run.
	data.EnvironmentDisplay = na
	return data
}

// setAccelerators records GPU compute in the same nullable shape as air list.
func setAccelerators(data *getData, accelType string, count int) {
	data.AcceleratorType = nilIfEmpty(accelType)
	if count != 0 {
		data.NumAccelerators = new(count)
	} else {
		data.NumAccelerators = nil
	}
	data.AcceleratorsDisplay = orNA(acceleratorLabel(accelType, count))
}

// enrichRunDetails resolves the single-run values that need requests beyond the
// typed GetRun: the environment (raw GetRun), the stored config (download) and
// the retry limit and the MLflow run name. When readConfig is false, only a
// server-reported retry limit is available. Each read is best-effort.
func enrichRunDetails(ctx context.Context, w *databricks.WorkspaceClient, run *jobs.Run, data *getData, ids *mlflowIdentifiers, readConfig bool) {
	data.detailsResolved = true

	// The reads are independent, so run them concurrently.
	var (
		wg            sync.WaitGroup
		version, base string
		body          string
		raw           []byte
		mlflowName    string
	)
	wg.Go(func() { version, base = aiRuntimeEnvironment(ctx, w, run.RunId) })
	if readConfig {
		wg.Go(func() { body, raw = resolveConfigYAML(ctx, w, run, data) })
	}
	if ids != nil {
		wg.Go(func() { mlflowName = fetchMLflowRunName(ctx, w, ids.RunID) })
	}
	wg.Wait()

	data.EnvironmentVersion = nilIfEmpty(version)
	data.BaseEnvironment = nilIfEmpty(base)
	data.EnvironmentDisplay = orNA(version)
	data.configBody = body
	if len(raw) > 0 {
		// Normalize CRLF so the machine-readable output is identical for the
		// same config uploaded from Windows and Unix checkouts.
		cfg := strings.ReplaceAll(string(raw), "\r\n", "\n")
		data.Config = &cfg
		data.ConfigSource = configSource(run, data)
	}
	data.MaxRetries = resolveMaxRetries(run, raw)
	data.MLflowRunName = nilIfEmpty(mlflowName)
}

// configSource names where resolveConfigYAML read a stored config from.
func configSource(run *jobs.Run, data *getData) *string {
	source := "uploaded"
	if task := genAIComputeTask(run); task != nil && data.TrainingConfigPath == "" {
		source = "yaml_parameters_file"
		if task.YamlParameters != "" {
			source = "yaml_parameters"
		}
	}
	return &source
}

type quietEnrichmentKey struct{}

// withQuietEnrichment demotes best-effort enrichment failures from warnings to
// debug logs.
func withQuietEnrichment(ctx context.Context) context.Context {
	return context.WithValue(ctx, quietEnrichmentKey{}, true)
}

func enrichmentWarnf(ctx context.Context, format string, v ...any) {
	if quiet, _ := ctx.Value(quietEnrichmentKey{}).(bool); quiet {
		log.Debugf(ctx, format, v...)
		return
	}
	log.Warnf(ctx, format, v...)
}
