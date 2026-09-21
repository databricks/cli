package pipelines

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"net/http"
	"os"
	"slices"
	"time"

	"github.com/databricks/cli/cmd/bundle/utils"
	"github.com/databricks/cli/cmd/root"
	"github.com/databricks/cli/libs/auth"
	"github.com/databricks/cli/libs/cmdgroup"
	"github.com/databricks/cli/libs/flags"
	databricks "github.com/databricks/databricks-sdk-go"
	"github.com/databricks/databricks-sdk-go/client"
	"github.com/databricks/databricks-sdk-go/service/pipelines"
	"github.com/spf13/cobra"
)

// startTestUpdateRequest is the StartUpdate body for a test_only update.
//
// The pipelines API field test_only (StartUpdate field 27) is PUBLIC_UNDOCUMENTED,
// so it is absent from the generated Go SDK's pipelines.StartUpdate struct and
// cannot be set through w.Pipelines.StartUpdate. As an interim measure this
// command POSTs the StartUpdate body directly with test_only=true. The real fix
// is bumping the field to visibility: PUBLIC (see the M4 worklog / CP_API_Design).
type startTestUpdateRequest struct {
	TestOnly    bool     `json:"test_only"`
	FullRefresh bool     `json:"full_refresh,omitempty"`
	TestFiles   []string `json:"test_files,omitempty"`
}

// buildStartTestUpdateRequest constructs the StartUpdate body for a test run.
// It always sets test_only=true. test_files is only sent when --tests is passed
// and requires the (branch-only) CP test_files field to be served.
func buildStartTestUpdateRequest(tests []string, fullRefresh bool) startTestUpdateRequest {
	return startTestUpdateRequest{
		TestOnly:    true,
		FullRefresh: fullRefresh,
		TestFiles:   tests,
	}
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
	queryParams := map[string]string{
		"filter":   fmt.Sprintf("update_id = '%s'", updateID),
		"order_by": "timestamp asc",
	}

	var response struct {
		Events []testPipelineEvent `json:"events"`
	}
	// Matches the fetchAllPipelineEvents call convention: query params are passed
	// in the request position so the SDK client encodes them onto the URL.
	err = apiClient.Do(ctx, http.MethodGet, path, auth.WorkspaceIDHeaders(w.Config), nil, queryParams, &response)
	if err != nil {
		return nil, fmt.Errorf("failed to fetch pipeline events: %w", err)
	}

	return filterTestEvents(response.Events), nil
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

// pollUntilTerminal polls GetUpdate until the update reaches a terminal state,
// the context is cancelled, or the timeout elapses.
func pollUntilTerminal(ctx context.Context, w *databricks.WorkspaceClient, pipelineID, updateID string, interval time.Duration) (pipelines.UpdateInfoState, error) {
	for {
		resp, err := w.Pipelines.GetUpdate(ctx, pipelines.GetUpdateRequest{
			PipelineId: pipelineID,
			UpdateId:   updateID,
		})
		if err != nil {
			return "", err
		}
		if resp.Update == nil {
			return "", fmt.Errorf("no update found with id %s for pipeline %s", updateID, pipelineID)
		}
		if isTerminalUpdateState(resp.Update.State) {
			return resp.Update.State, nil
		}

		select {
		case <-ctx.Done():
			return "", ctx.Err()
		case <-time.After(interval):
		}
	}
}

// renderTestRun renders a folded result in the requested output format and, when
// junitPath is set, additionally writes a JUnit XML report to that file.
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
		return renderTestResultsText(cmd.OutOrStdout(), pipelineID, updateID, result)
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
		Use:   "test [flags] [KEY]",
		Args:  root.MaximumNArgs(1),
		Short: "Run the tests for a pipeline",
		Long: `Run the tests defined for the pipeline identified by KEY.
KEY is the unique name of the pipeline to test, as defined in its YAML file.
If there is only one pipeline in the project, KEY is optional and the pipeline will be auto-selected.

Triggers a test-only pipeline update, waits for it to complete, and reports the
per-test-case results and a summary read from the pipeline event log.`,
	}

	var tests []string
	var fullRefresh bool
	var noWait bool
	var timeout time.Duration
	var junitXML string

	testGroup := cmdgroup.NewFlagGroup("Pipeline Test")
	testGroup.FlagSet().StringSliceVar(&tests, "tests", nil, "Test file or directory to run (repeatable). Requires the pipeline to support test_files.")
	testGroup.FlagSet().BoolVar(&fullRefresh, "full-refresh", false, "Perform a full graph reset and recompute before running tests.")
	testGroup.FlagSet().BoolVar(&noWait, "no-wait", false, "Start the test update and return immediately without waiting for results.")
	testGroup.FlagSet().DurationVar(&timeout, "timeout", 0, "Maximum time to wait for the test update to complete (e.g. 30m). 0 means no timeout.")
	testGroup.FlagSet().StringVar(&junitXML, "junit-xml", "", "Write a JUnit XML report of the per-case results to this file.")

	wrappedCmd := cmdgroup.NewCommandWithGroupFlag(cmd)
	wrappedCmd.AddFlagGroup(testGroup)

	cmd.RunE = func(cmd *cobra.Command, args []string) error {
		b, err := utils.ProcessBundle(cmd, utils.ProcessOptions{
			InitIDs: true,
		})
		if err != nil {
			return err
		}
		ctx := cmd.Context()

		key, err := resolvePipelineArgument(ctx, b, args)
		if err != nil {
			return err
		}

		pipelineID, err := resolvePipelineIdFromKey(ctx, b, key)
		if err != nil {
			return err
		}

		w := b.WorkspaceClient(ctx)

		updateID, err := startTestUpdate(ctx, w, pipelineID, buildStartTestUpdateRequest(tests, fullRefresh))
		if err != nil {
			return err
		}

		if noWait {
			fmt.Fprintf(cmd.OutOrStdout(), "Started test-only update %s for pipeline %s\n", updateID, pipelineID)
			return nil
		}

		if timeout > 0 {
			var cancel context.CancelFunc
			ctx, cancel = context.WithTimeout(ctx, timeout)
			defer cancel()
		}

		if _, err := pollUntilTerminal(ctx, w, pipelineID, updateID, 5*time.Second); err != nil {
			return err
		}

		events, err := fetchTestEvents(ctx, w, pipelineID, updateID)
		if err != nil {
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
