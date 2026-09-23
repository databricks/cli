package pipelines

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"net/http"
	"os"
	"slices"
	"strings"
	"time"

	"github.com/databricks/cli/cmd/bundle/utils"
	"github.com/databricks/cli/cmd/root"
	"github.com/databricks/cli/libs/auth"
	"github.com/databricks/cli/libs/cmdctx"
	"github.com/databricks/cli/libs/cmdgroup"
	"github.com/databricks/cli/libs/flags"
	databricks "github.com/databricks/databricks-sdk-go"
	"github.com/databricks/databricks-sdk-go/client"
	"github.com/databricks/databricks-sdk-go/service/pipelines"
	"github.com/spf13/cobra"
)

// startTestUpdateRequest is the StartUpdate body for a test_only update.
//
// The pipelines API field test_only is PUBLIC_UNDOCUMENTED,
// so it is absent from the generated Go SDK's pipelines.StartUpdate struct and
// cannot be set through w.Pipelines.StartUpdate. As an interim measure this
// command POSTs the StartUpdate body directly with test_only=true. The real fix
// is bumping the field to visibility: PUBLIC (see the M4 worklog / CP_API_Design).
type startTestUpdateRequest struct {
	TestOnly      bool     `json:"test_only"`
	FullRefresh   bool     `json:"full_refresh,omitempty"`
	TestSelection []string `json:"test_selection,omitempty"`
	PytestArgs    []string `json:"pytest_args,omitempty"`
}

// testUpdateOptions collects the request-shaping inputs of a test run. It keeps
// buildStartTestUpdateRequest's signature stable as fields are added.
type testUpdateOptions struct {
	// testSelection is the union of --select/-s selectors. A selector is a pytest
	// positional argument: a test file, a directory, or a node id (path or
	// path::nodeid). Paths and node ids are the same input to pytest, so both the
	// --select flag and the deprecated --tests alias feed this single collection
	// (mirroring dbt's --select and the CP TestExecutionSpec's single selectors
	// list). Maps to the (branch-only) CP test_selection field.
	testSelection []string
	// pytestArgs are the raw arguments after -- (only -k / -m this milestone).
	pytestArgs  []string
	fullRefresh bool
}

// buildStartTestUpdateRequest constructs the StartUpdate body for a test run.
// It always sets test_only=true. test_selection carries the --select selectors;
// pytest_args carries the post-`--` passthrough; both require the (branch-only)
// CP fields to be served.
func buildStartTestUpdateRequest(opts testUpdateOptions) startTestUpdateRequest {
	return startTestUpdateRequest{
		TestOnly:      true,
		FullRefresh:   opts.fullRefresh,
		TestSelection: opts.testSelection,
		PytestArgs:    opts.pytestArgs,
	}
}

// parsedTestArgs is the validated invocation of the test command, independent of
// any bundle or network I/O. Everything here is decided purely from the raw args
// and flag values, so it is unit-testable without a workspace or bundle.
type parsedTestArgs struct {
	// PipelineKey is the lone positional (a bundle resource key), "" when omitted.
	PipelineKey string
	// PipelineIDFlag is --pipeline-id, the non-bundle direct pipeline id.
	PipelineIDFlag string
	// TestSelection is the de-duplicated union of --select/-s selectors (files,
	// directories, or node ids).
	TestSelection []string
	// PytestArgs are the arguments after -- (validated against the M4 allow-list).
	PytestArgs []string
	// Exclude are the --exclude node ids (reserved; see the reservation note).
	Exclude []string
}

// splitPipelineTestArgs separates the positional arguments (before --) from the
// pytest passthrough arguments (after --). dashPos is cmd.ArgsLenAtDash(): -1
// when no -- is present, otherwise the count of args preceding the --.
func splitPipelineTestArgs(args []string, dashPos int) (positionals, pytestArgs []string) {
	if dashPos < 0 {
		return args, nil
	}
	return args[:dashPos], args[dashPos:]
}

// validatePytestPassthrough enforces the M4 allow-list for arguments after --:
// only -k and -m (with their value, in the separate `-k EXPR` or attached
// `-kEXPR` form) are accepted. Any other token is rejected so an unsupported
// pytest flag fails fast rather than being forwarded and silently ignored.
func validatePytestPassthrough(args []string) error {
	expectValue := false
	for _, a := range args {
		if expectValue {
			expectValue = false
			continue
		}
		switch {
		case a == "-k" || a == "-m":
			// Value follows as the next token.
			expectValue = true
		case strings.HasPrefix(a, "-k") || strings.HasPrefix(a, "-m"):
			// Attached form, e.g. -ksmoke or -mslow; value is part of the token.
		default:
			return fmt.Errorf("unsupported pytest argument %q after --: this milestone only accepts -k and -m", a)
		}
	}
	return nil
}

// combineSelection concatenates the --select selectors and the values from the
// deprecated --tests alias into one selection list (before de-duplication), so
// the two flags fold into a single collection.
func combineSelection(selectFlag, deprecatedTests []string) []string {
	out := make([]string, 0, len(selectFlag)+len(deprecatedTests))
	out = append(out, selectFlag...)
	out = append(out, deprecatedTests...)
	return out
}

// unionNodeIDs de-duplicates repeated --select/-s values, preserving first-seen
// order so the request and any error messages are deterministic.
func unionNodeIDs(ids []string) []string {
	seen := make(map[string]struct{}, len(ids))
	var out []string
	for _, id := range ids {
		if _, ok := seen[id]; ok {
			continue
		}
		seen[id] = struct{}{}
		out = append(out, id)
	}
	return out
}

// parseTestArgs validates and folds the raw cobra args plus repeatable flag
// values into a parsedTestArgs. It performs every check that does not need a
// bundle: the single-positional rule (the sole positional is ALWAYS
// PIPELINE_KEY), the pytest passthrough allow-list, and PIPELINE_KEY /
// --pipeline-id mutual exclusion.
func parseTestArgs(args []string, dashPos int, testSelection, exclude []string, pipelineIDFlag string) (parsedTestArgs, error) {
	positionals, pytestArgs := splitPipelineTestArgs(args, dashPos)
	if len(positionals) > 1 {
		return parsedTestArgs{}, fmt.Errorf("accepts at most one PIPELINE_KEY argument, received %d: %v", len(positionals), positionals)
	}
	if err := validatePytestPassthrough(pytestArgs); err != nil {
		return parsedTestArgs{}, err
	}

	key := ""
	if len(positionals) == 1 {
		key = positionals[0]
	}
	if key != "" && pipelineIDFlag != "" {
		return parsedTestArgs{}, errors.New("PIPELINE_KEY and --pipeline-id are mutually exclusive; pass only one")
	}

	return parsedTestArgs{
		PipelineKey:    key,
		PipelineIDFlag: pipelineIDFlag,
		TestSelection:  unionNodeIDs(testSelection),
		PytestArgs:     pytestArgs,
		Exclude:        exclude,
	}, nil
}

// resolutionMode describes how the target pipeline is identified.
type resolutionMode int

const (
	// resolveByID uses --pipeline-id directly (works in or out of a bundle).
	resolveByID resolutionMode = iota
	// resolveByKey looks up the PIPELINE_KEY positional in the bundle.
	resolveByKey
	// resolveByInference selects the sole pipeline in the bundle.
	resolveByInference
)

// resolvePipelineMode encodes the pipeline-resolution decision table:
//   - --pipeline-id set              -> resolve by id;
//   - no bundle, no --pipeline-id    -> error (outside a bundle it is required);
//   - in bundle, PIPELINE_KEY given  -> resolve by key;
//   - in bundle, nothing given       -> infer the sole pipeline (which now works
//     even with --select selectors, since those are flags, not positionals).
func resolvePipelineMode(inBundle bool, p parsedTestArgs) (resolutionMode, error) {
	if p.PipelineIDFlag != "" {
		return resolveByID, nil
	}
	if !inBundle {
		return 0, errors.New("outside a bundle, --pipeline-id is required to identify the pipeline to test")
	}
	if p.PipelineKey != "" {
		return resolveByKey, nil
	}
	return resolveByInference, nil
}

// testCommandArgs is the cobra positional-args validator. It reads only the
// arguments before -- so pytest passthrough tokens are never mistaken for
// positionals, and enforces the single-optional-positional (PIPELINE_KEY) form.
func testCommandArgs(cmd *cobra.Command, args []string) error {
	positionals, _ := splitPipelineTestArgs(args, cmd.ArgsLenAtDash())
	if len(positionals) > 1 {
		return &root.InvalidArgsError{
			Message: fmt.Sprintf("accepts at most one PIPELINE_KEY argument, received %d", len(positionals)),
			Command: cmd,
		}
	}
	return nil
}

// terminalUpdateStates are the states after which an update no longer changes.
var terminalUpdateStates = []pipelines.UpdateInfoState{
	pipelines.UpdateInfoStateCompleted,
	pipelines.UpdateInfoStateFailed,
	pipelines.UpdateInfoStateCanceled,
}

// isTerminalUpdateState reports whether an update has reached a terminal state.
func isTerminalUpdateState(state pipelines.UpdateInfoState) bool {
	return slices.Contains(terminalUpdateStates, state)
}

// startTestUpdate POSTs a test_only StartUpdate and returns the new update id.
func startTestUpdate(ctx context.Context, w *databricks.WorkspaceClient, pipelineID string, req startTestUpdateRequest) (string, error) {
	apiClient, err := client.New(w.Config)
	if err != nil {
		return "", fmt.Errorf("failed to create API client: %w", err)
	}

	path := fmt.Sprintf("/api/2.0/pipelines/%s/updates", pipelineID)
	headers := auth.WorkspaceIDHeaders(w.Config)
	headers["Content-Type"] = "application/json"

	var response pipelines.StartUpdateResponse
	err = apiClient.Do(ctx, http.MethodPost, path, headers, map[string]any{}, req, &response)
	if err != nil {
		return "", fmt.Errorf("failed to start test-only update: %w", err)
	}
	if response.UpdateId == "" {
		return "", errors.New("start update returned no update id")
	}
	return response.UpdateId, nil
}

// fetchTestEvents fetches the test_case_progress / test_summary events for an
// update. It reads the raw /events response (rather than the SDK type, which
// drops details) and filters to the two test event types client-side.
func fetchTestEvents(ctx context.Context, w *databricks.WorkspaceClient, pipelineID, updateID string) ([]testPipelineEvent, error) {
	apiClient, err := client.New(w.Config)
	if err != nil {
		return nil, fmt.Errorf("failed to create API client: %w", err)
	}

	path := fmt.Sprintf("/api/2.0/pipelines/%s/events", pipelineID)
	var events []testPipelineEvent
	pageToken := ""
	for {
		queryParams := map[string]string{
			"filter":      fmt.Sprintf("update_id = '%s'", updateID),
			"order_by":    "timestamp asc",
			"max_results": "250",
		}
		if pageToken != "" {
			queryParams["page_token"] = pageToken
		}

		var response struct {
			Events        []testPipelineEvent `json:"events"`
			NextPageToken string              `json:"next_page_token,omitempty"`
		}
		// Matches the fetchAllPipelineEvents call convention: query params are
		// passed in the request position so the SDK client encodes them onto the URL.
		err = apiClient.Do(ctx, http.MethodGet, path, auth.WorkspaceIDHeaders(w.Config), nil, queryParams, &response)
		if err != nil {
			return nil, fmt.Errorf("failed to fetch pipeline events: %w", err)
		}
		events = append(events, filterTestEvents(response.Events)...)

		if response.NextPageToken == "" {
			return events, nil
		}
		if response.NextPageToken == pageToken {
			return nil, errors.New("pipeline events API returned the same page token twice")
		}
		pageToken = response.NextPageToken
	}
}

// filterTestEvents keeps only test_case_progress and test_summary events. This
// is the client-side event_type filter the M4 design pins to (the server-side
// filter is unreliable for external/PAT callers).
func filterTestEvents(events []testPipelineEvent) []testPipelineEvent {
	var out []testPipelineEvent
	for _, e := range events {
		if e.EventType == testCaseProgressEventType || e.EventType == testSummaryEventType {
			out = append(out, e)
		}
	}
	return out
}

// pollTestEventsUntilTerminal polls both update state and /events. onEvents sees
// the full event snapshot on every iteration and is responsible for
// de-duplicating transitions. The returned events are the terminal snapshot.
func pollTestEventsUntilTerminal(
	ctx context.Context,
	w *databricks.WorkspaceClient,
	pipelineID, updateID string,
	interval time.Duration,
	eventSyncInterval time.Duration,
	eventSyncTimeout time.Duration,
	onEvents func([]testPipelineEvent) error,
) (pipelines.UpdateInfoState, []testPipelineEvent, error) {
	var events []testPipelineEvent
	for {
		resp, err := w.Pipelines.GetUpdate(ctx, pipelines.GetUpdateRequest{
			PipelineId: pipelineID,
			UpdateId:   updateID,
		})
		if err != nil {
			return "", nil, err
		}
		if resp.Update == nil {
			return "", nil, fmt.Errorf("no update found with id %s for pipeline %s", updateID, pipelineID)
		}

		terminal := isTerminalUpdateState(resp.Update.State)
		if resp.Update.State == pipelines.UpdateInfoStateRunning || terminal {
			events, err = fetchTestEvents(ctx, w, pipelineID, updateID)
			if err != nil {
				return "", nil, err
			}
			if err := onEvents(events); err != nil {
				return "", nil, err
			}
		}
		if terminal {
			events, err = waitForCompleteTestEvents(
				ctx,
				w,
				pipelineID,
				updateID,
				events,
				eventSyncInterval,
				eventSyncTimeout,
				onEvents,
			)
			return resp.Update.State, events, err
		}

		select {
		case <-ctx.Done():
			return "", nil, ctx.Err()
		case <-time.After(interval):
		}
	}
}

func waitForCompleteTestEvents(
	ctx context.Context,
	w *databricks.WorkspaceClient,
	pipelineID, updateID string,
	events []testPipelineEvent,
	interval time.Duration,
	timeout time.Duration,
	onEvents func([]testPipelineEvent) error,
) ([]testPipelineEvent, error) {
	if reduceTestEvents(events).checkComplete() == nil || timeout <= 0 {
		return events, nil
	}

	timer := time.NewTimer(timeout)
	defer timer.Stop()
	ticker := time.NewTicker(interval)
	defer ticker.Stop()

	for {
		select {
		case <-ctx.Done():
			return nil, ctx.Err()
		case <-timer.C:
			// Let the caller's completeness barrier return the precise
			// NO_RESULT or EVENTS_INCOMPLETE error using the latest snapshot.
			return events, nil
		case <-ticker.C:
			var err error
			events, err = fetchTestEvents(ctx, w, pipelineID, updateID)
			if err != nil {
				return nil, err
			}
			if err := onEvents(events); err != nil {
				return nil, err
			}
			if reduceTestEvents(events).checkComplete() == nil {
				return events, nil
			}
		}
	}
}

type testSubmissionJSONOutput struct {
	PipelineID string `json:"pipeline_id"`
	UpdateID   string `json:"update_id"`
	Submitted  bool   `json:"submitted"`
}

func renderTestSubmission(cmd *cobra.Command, pipelineID, updateID string) error {
	switch root.OutputType(cmd) {
	case flags.OutputText:
		_, err := fmt.Fprintf(cmd.OutOrStdout(), "Started test-only update %s for pipeline %s\n", updateID, pipelineID)
		return err
	case flags.OutputJSON:
		return json.NewEncoder(cmd.OutOrStdout()).Encode(testSubmissionJSONOutput{
			PipelineID: pipelineID,
			UpdateID:   updateID,
			Submitted:  true,
		})
	default:
		return fmt.Errorf("unknown output type %s", root.OutputType(cmd))
	}
}

// renderTestRun renders the final result after per-case transitions have been
// streamed and, when junitPath is set, writes a JUnit XML report.
func renderTestRun(cmd *cobra.Command, pipelineID, updateID string, result *testRunResult, junitPath string) error {
	if junitPath != "" {
		f, err := os.Create(junitPath)
		if err != nil {
			return fmt.Errorf("failed to create JUnit file: %w", err)
		}
		defer f.Close()
		if err := renderJUnitXML(f, pipelineID, updateID, result); err != nil {
			return err
		}
	}

	switch root.OutputType(cmd) {
	case flags.OutputText:
		return renderStreamedTestResultsFooter(cmd.OutOrStdout(), result)
	case flags.OutputJSON:
		enc := json.NewEncoder(cmd.OutOrStdout())
		enc.SetIndent("", "  ")
		return enc.Encode(result.toJSONOutput(pipelineID, updateID))
	default:
		return fmt.Errorf("unknown output type %s", root.OutputType(cmd))
	}
}

func testCommand() *cobra.Command {
	cmd := &cobra.Command{
		Use:   "test [PIPELINE_KEY] [flags] [-- PYTEST_ARG...]",
		Args:  testCommandArgs,
		Short: "Run the tests for a pipeline",
		Long: `Run the tests defined for the pipeline identified by PIPELINE_KEY.
PIPELINE_KEY is the unique name of the pipeline to test, as defined in its YAML file.
It is the only positional argument; select individual test cases with --select/-s.
If there is only one pipeline in the project, PIPELINE_KEY is optional and the
pipeline is auto-selected. Outside a bundle, pass --pipeline-id instead.

Triggers a test-only pipeline update, waits for it to complete, and reports the
per-test-case results and a summary read from the pipeline event log.`,
	}

	var selectFlag []string
	var deprecatedTests []string
	var exclude []string
	var pipelineIDFlag string
	var fullRefresh bool
	var noWait bool
	var timeout time.Duration
	var junitXML string

	testGroup := cmdgroup.NewFlagGroup("Pipeline Test")
	fs := testGroup.FlagSet()
	// --select mirrors dbt's selector flag. -s is free (not a global/root
	// shorthand; the taken ones are -o/-p/-t/-h), so it is wired here.
	fs.StringSliceVarP(&selectFlag, "select", "s", nil, "Pytest selector to run (repeatable; unioned across repeats): a test file, directory, or node id (path or path::nodeid). Restricts the run to the matching cases.")
	fs.StringSliceVar(&exclude, "exclude", nil, "Pytest node id to exclude (repeatable). Reserved: not yet supported in M4.")
	fs.StringVar(&pipelineIDFlag, "pipeline-id", "", "Pipeline ID to test directly, without a bundle. Mutually exclusive with PIPELINE_KEY.")
	// --tests is a deprecated alias of --select. Test files and node ids are the
	// same pytest positional input, so they fold into one selection collection.
	fs.StringSliceVar(&deprecatedTests, "tests", nil, "Deprecated alias of --select.")
	_ = fs.MarkDeprecated("tests", "use --select instead")
	fs.BoolVar(&fullRefresh, "full-refresh", false, "Perform a full graph reset and recompute before running tests.")
	fs.BoolVar(&noWait, "no-wait", false, "Start the test update and return immediately without waiting for results.")
	fs.DurationVar(&timeout, "timeout", 0, "Maximum time to wait for the test update to complete (e.g. 30m). 0 means no timeout.")
	fs.StringVar(&junitXML, "junit-xml", "", "Write a JUnit XML report of the per-case results to this file.")

	wrappedCmd := cmdgroup.NewCommandWithGroupFlag(cmd)
	wrappedCmd.AddFlagGroup(testGroup)

	cmd.RunE = func(cmd *cobra.Command, args []string) error {
		if noWait && junitXML != "" {
			return errors.New("--no-wait and --junit-xml cannot be used together")
		}

		selection := combineSelection(selectFlag, deprecatedTests)
		parsed, err := parseTestArgs(args, cmd.ArgsLenAtDash(), selection, exclude, pipelineIDFlag)
		if err != nil {
			return err
		}
		if len(parsed.Exclude) > 0 {
			// RESERVED: --exclude is defined so the surface is stable, but wiring
			// it to pytest --deselect interacts with the -k/-m-only passthrough
			// allow-list and the (still unbuilt) CP test_selection field, so it is
			// rejected rather than half-implemented this milestone.
			return errors.New("--exclude is reserved and not yet supported in M4 (it will map to pytest --deselect once the pipeline test-selection surface lands)")
		}

		opts := testUpdateOptions{
			testSelection: parsed.TestSelection,
			pytestArgs:    parsed.PytestArgs,
			fullRefresh:   fullRefresh,
		}

		var (
			ctx        context.Context
			w          *databricks.WorkspaceClient
			pipelineID string
		)

		if parsed.PipelineIDFlag != "" {
			// resolveByID: direct pipeline id, no bundle key lookup. Works both
			// inside and outside a bundle. MustWorkspaceClient sets up auth and
			// the workspace client on the command context.
			if err := root.MustWorkspaceClient(cmd, args); err != nil {
				return err
			}
			ctx = cmd.Context()
			w = cmdctx.WorkspaceClient(ctx)
			pipelineID = parsed.PipelineIDFlag
		} else {
			// Bundle path: ProcessBundle requires (and thus proves) a bundle, so a
			// missing bundle here surfaces as "outside a bundle, --pipeline-id is
			// required" would in resolvePipelineMode. A lone positional is always
			// the PIPELINE_KEY; when omitted the sole pipeline is inferred.
			b, err := utils.ProcessBundle(cmd, utils.ProcessOptions{
				InitIDs: true,
			})
			if err != nil {
				return err
			}
			ctx = cmd.Context()

			var positionals []string
			if parsed.PipelineKey != "" {
				positionals = []string{parsed.PipelineKey}
			}
			key, err := resolvePipelineArgument(ctx, b, positionals)
			if err != nil {
				return err
			}
			pipelineID, err = resolvePipelineIdFromKey(ctx, b, key)
			if err != nil {
				return err
			}
			w = b.WorkspaceClient(ctx)
		}

		updateID, err := startTestUpdate(ctx, w, pipelineID, buildStartTestUpdateRequest(opts))
		if err != nil {
			return err
		}

		if noWait {
			return renderTestSubmission(cmd, pipelineID, updateID)
		}

		if timeout > 0 {
			var cancel context.CancelFunc
			ctx, cancel = context.WithTimeout(ctx, timeout)
			defer cancel()
		}

		progressWriter := cmd.OutOrStdout()
		if root.OutputType(cmd) == flags.OutputJSON {
			progressWriter = cmd.ErrOrStderr()
		}
		progress := newTestProgressPrinter(progressWriter, pipelineID, updateID)

		_, events, err := pollTestEventsUntilTerminal(
			ctx,
			w,
			pipelineID,
			updateID,
			5*time.Second,
			time.Second,
			30*time.Second,
			progress.consume,
		)
		if err != nil {
			if errors.Is(err, context.DeadlineExceeded) {
				return fmt.Errorf("timed out waiting for pipeline test update %s: %w", updateID, err)
			}
			return err
		}

		result := reduceTestEvents(events)
		if err := result.checkComplete(); err != nil {
			return err
		}

		if err := renderTestRun(cmd, pipelineID, updateID, result, junitXML); err != nil {
			return err
		}

		if result.failed() {
			return fmt.Errorf("%d of %d pipeline tests failed", result.Summary.Failed+result.Summary.Errors, result.Summary.Total)
		}
		return nil
	}

	return cmd
}
