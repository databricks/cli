package pipelines

import (
	"context"
	"errors"
	"fmt"
	"io"
	"net/http"
	"os"
	"os/signal"
	"path"
	"path/filepath"
	"strings"
	"syscall"
	"time"

	"github.com/databricks/cli/bundle"
	"github.com/databricks/cli/bundle/config"
	"github.com/databricks/cli/bundle/resources"
	"github.com/databricks/cli/cmd/bundle/utils"
	"github.com/databricks/cli/cmd/root"
	"github.com/databricks/cli/libs/auth"
	"github.com/databricks/cli/libs/cmdctx"
	"github.com/databricks/cli/libs/cmdgroup"
	"github.com/databricks/cli/libs/flags"
	"github.com/databricks/cli/libs/logdiag"
	"github.com/databricks/cli/libs/shellquote"
	databricks "github.com/databricks/databricks-sdk-go"
	"github.com/databricks/databricks-sdk-go/client"
	"github.com/databricks/databricks-sdk-go/service/pipelines"
	"github.com/spf13/cobra"
	"github.com/spf13/pflag"
)

// startTestUpdateRequest is the Start Update body of a test run.
// test_only and test_options are not in the Go SDK yet, so the request is sent directly.
type startTestUpdateRequest struct {
	TestOnly    bool         `json:"test_only"`
	TestOptions *testOptions `json:"test_options,omitempty"`
}

type testOptions struct {
	TestSelection []string `json:"test_selection,omitempty"`
	PytestArgs    []string `json:"pytest_args,omitempty"`
}

func newStartTestUpdateRequest(selection, pytestArgs []string) startTestUpdateRequest {
	req := startTestUpdateRequest{TestOnly: true}
	if len(selection) > 0 || len(pytestArgs) > 0 {
		req.TestOptions = &testOptions{TestSelection: selection, PytestArgs: pytestArgs}
	}
	return req
}

const pytestArgHint = "pass pytest arguments with --pytest-arg, for example --pytest-arg=-k --pytest-arg='orders and not slow'"

// validatePytestArgs accepts only -k and -m, each with a value in the next argument or attached to it.
func validatePytestArgs(args []string) error {
	for i := 0; i < len(args); i++ {
		arg := args[i]
		switch {
		case arg == "-k" || arg == "-m":
			if i == len(args)-1 {
				return fmt.Errorf("pytest argument %s requires a value: pass it in the next --pytest-arg", arg)
			}
			i++
		case strings.HasPrefix(arg, "-k") || strings.HasPrefix(arg, "-m"):
		case !strings.HasPrefix(arg, "-"):
			return fmt.Errorf("unsupported pytest argument %q: pass test paths and test IDs as TEST_SELECTION arguments", arg)
		default:
			return fmt.Errorf("unsupported pytest argument %q: only -k and -m are supported", arg)
		}
	}
	return nil
}

// testFlagErrorFunc adds guidance when pytest's -k or -m is passed as a CLI flag.
func testFlagErrorFunc(cmd *cobra.Command, err error) error {
	if notExist, ok := errors.AsType[*pflag.NotExistError](err); ok {
		if name := notExist.GetSpecifiedName(); name == "k" || name == "m" {
			return fmt.Errorf("-%s is a pytest argument: %s", name, pytestArgHint)
		}
	}
	return cmd.Parent().FlagErrorFunc()(cmd, err)
}

// resolveTestSelections converts relative selection paths with toWorkspacePath and keeps
// the pytest test ID after "::" unchanged. Paths that start with "/" are workspace paths.
func resolveTestSelections(selections []string, toWorkspacePath func(string) (string, error)) ([]string, error) {
	var resolved []string
	for _, selection := range selections {
		p, testID, _ := strings.Cut(selection, "::")
		if p == "" {
			return nil, fmt.Errorf("test selection %q has no path", selection)
		}
		if strings.HasPrefix(p, "/") {
			p = path.Clean(p)
		} else {
			var err error
			p, err = toWorkspacePath(p)
			if err != nil {
				return nil, fmt.Errorf("cannot resolve test selection %q: %w", selection, err)
			}
		}
		if strings.Contains(selection, "::") {
			p += "::" + testID
		}
		resolved = append(resolved, p)
	}
	return resolved, nil
}

// bundleWorkspacePath maps a local path, relative to the working directory, to its deployed
// path under filesRoot, the deployed path of the bundle's synchronized files.
func bundleWorkspacePath(cwd, syncRoot, filesRoot, localPath string) (string, error) {
	if !filepath.IsAbs(localPath) {
		localPath = filepath.Join(cwd, localPath)
	}
	rel, err := filepath.Rel(syncRoot, localPath)
	if err != nil || rel == ".." || strings.HasPrefix(rel, ".."+string(filepath.Separator)) {
		return "", fmt.Errorf("%s is outside the synchronized bundle files in %s", filepath.ToSlash(localPath), filepath.ToSlash(syncRoot))
	}
	return path.Join(filesRoot, filepath.ToSlash(rel)), nil
}

// bundleFilesRoot returns the deployed path of the bundle's synchronized files, as TranslatePaths
// maps resource paths, or "" for an immutable deployment, whose snapshot path the command does not know.
func bundleFilesRoot(b *bundle.Bundle) string {
	switch {
	case b.SyncMode() == config.SyncModeImmutable:
		return ""
	case b.IsSourceLinkedDeployment():
		return b.SyncRootPath
	default:
		return b.Config.Workspace.FilePath
	}
}

// bundleTestSelectionPath maps a relative selection to a workspace path. A path that exists locally,
// relative to the working directory, maps to its deployed path under filesRoot. Otherwise the
// selection is a test ID that the CLI printed, relative to testIDBase, unless it leaves the working
// directory.
func bundleTestSelectionPath(cwd, syncRoot, filesRoot, testIDBase, localPath string) (string, error) {
	rel := path.Clean(filepath.ToSlash(localPath))
	leavesCwd := rel == ".." || strings.HasPrefix(rel, "../")
	if _, err := os.Stat(filepath.Join(cwd, localPath)); err != nil && !leavesCwd && testIDBase != "" {
		return testIDBasePath(testIDBase, localPath)
	}
	if filesRoot == "" {
		return "", errors.New("local test paths are not supported with workspace.sync_mode immutable; pass the test ID that the results show or an absolute workspace path")
	}
	return bundleWorkspacePath(cwd, syncRoot, filesRoot, localPath)
}

// needsTestIDBase reports whether a selection is a relative path, which may be a test ID.
func needsTestIDBase(selections []string) bool {
	for _, selection := range selections {
		if p, _, _ := strings.Cut(selection, "::"); !strings.HasPrefix(p, "/") {
			return true
		}
	}
	return false
}

// testIDBase returns the folder that the service names test cases relative to, or "" if the
// pipeline has neither a root_path nor test sources. pytest runs with this folder as its rootdir:
// the deepest folder that holds the root path and every test source or, without a root path, the
// folder of every test source.
func testIDBase(rootPath string, sources []string) string {
	var folders [][]string
	if rootPath != "" {
		folders = append(folders, pathSegments(normalizeWorkspacePath(rootPath)))
	}
	for _, source := range sources {
		segments := pathSegments(normalizeWorkspacePath(strings.TrimSuffix(source, "/**")))
		if rootPath == "" && len(segments) > 0 {
			segments = segments[:len(segments)-1]
		}
		folders = append(folders, segments)
	}
	if len(folders) == 0 {
		return ""
	}
	common := folders[0]
	for _, folder := range folders[1:] {
		n := 0
		for n < len(common) && n < len(folder) && common[n] == folder[n] {
			n++
		}
		common = common[:n]
	}
	return "/" + strings.Join(common, "/")
}

// normalizeWorkspacePath adds the /Workspace prefix to an absolute path without it, as the service does.
func normalizeWorkspacePath(p string) string {
	if strings.HasPrefix(p, "/") && !strings.HasPrefix(p, "/Workspace") {
		return "/Workspace" + p
	}
	return p
}

func pathSegments(p string) []string {
	var segments []string
	for segment := range strings.SplitSeq(p, "/") {
		if segment != "" {
			segments = append(segments, segment)
		}
	}
	return segments
}

// testIDBasePath resolves a relative path against the folder that test IDs are relative to.
func testIDBasePath(base, relPath string) (string, error) {
	if base == "" {
		return "", errors.New("the pipeline has no root_path or testing.sources; use an absolute workspace path such as /Workspace/Users/someone@example.com/tests")
	}
	return path.Join(base, filepath.ToSlash(relPath)), nil
}

// pipelineTestPaths holds the fields of a Get Pipeline response that decide the folder of test IDs.
// The Go SDK has no spec.testing yet, so the response is decoded from the raw API call.
type pipelineTestPaths struct {
	Spec struct {
		RootPath string `json:"root_path"`
		Testing  struct {
			Sources []struct {
				Glob struct {
					Include string `json:"include"`
				} `json:"glob"`
			} `json:"sources"`
		} `json:"testing"`
	} `json:"spec"`
}

// getTestIDBase reads the deployed pipeline and returns the folder that its test IDs are relative to.
func getTestIDBase(ctx context.Context, w *databricks.WorkspaceClient, pipelineID string) (string, error) {
	apiClient, err := client.New(w.Config)
	if err != nil {
		return "", fmt.Errorf("failed to create API client: %w", err)
	}
	var response pipelineTestPaths
	err = apiClient.Do(ctx, http.MethodGet, "/api/2.0/pipelines/"+pipelineID, auth.WorkspaceIDHeaders(w.Config), nil, nil, &response)
	if err != nil {
		return "", err
	}
	var sources []string
	for _, source := range response.Spec.Testing.Sources {
		if source.Glob.Include != "" {
			sources = append(sources, source.Glob.Include)
		}
	}
	return testIDBase(response.Spec.RootPath, sources), nil
}

// startTestUpdate starts a test-only update and returns its ID.
func startTestUpdate(ctx context.Context, w *databricks.WorkspaceClient, pipelineID string, req startTestUpdateRequest) (string, error) {
	apiClient, err := client.New(w.Config)
	if err != nil {
		return "", fmt.Errorf("failed to create API client: %w", err)
	}

	var response pipelines.StartUpdateResponse
	err = apiClient.Do(
		ctx,
		http.MethodPost,
		fmt.Sprintf("/api/2.0/pipelines/%s/updates", pipelineID),
		auth.WorkspaceIDHeaders(w.Config),
		nil,
		req,
		&response,
	)
	if err != nil {
		return "", err
	}
	return response.UpdateId, nil
}

// testTarget is the pipeline a test run is submitted to.
type testTarget struct {
	key        string
	pipelineID string
	selections []string
	w          *databricks.WorkspaceClient
}

func (t testTarget) printHeader(out io.Writer, updateID string) {
	if t.key != "" {
		fmt.Fprintf(out, "Pipeline: %s (%s)\n", t.key, t.pipelineID)
	} else {
		fmt.Fprintf(out, "Pipeline: %s\n", t.pipelineID)
	}
	fmt.Fprintf(out, "Update: %s\n", updateID)
}

// readTestIDBase reads the folder of test IDs from the deployed pipeline, if a selection needs it.
func readTestIDBase(ctx context.Context, w *databricks.WorkspaceClient, pipelineID string, selections []string) (string, error) {
	if !needsTestIDBase(selections) {
		return "", nil
	}
	return getTestIDBase(ctx, w, pipelineID)
}

// resolveTestTargetByID identifies the pipeline by --pipeline-id; all positional arguments are selections.
func resolveTestTargetByID(cmd *cobra.Command, args []string, pipelineID string) (testTarget, error) {
	if cmd.Flag("target").Value.String() == "" {
		cmd.SetContext(root.SkipLoadBundle(cmd.Context()))
	}
	if err := root.MustWorkspaceClient(cmd, args); err != nil {
		return testTarget{}, err
	}
	ctx := cmd.Context()
	w := cmdctx.WorkspaceClient(ctx)

	base, err := readTestIDBase(ctx, w, pipelineID, args)
	if err != nil {
		return testTarget{}, err
	}
	selections, err := resolveTestSelections(args, func(p string) (string, error) {
		return testIDBasePath(base, p)
	})
	if err != nil {
		return testTarget{}, err
	}
	return testTarget{pipelineID: pipelineID, selections: selections, w: w}, nil
}

// resolveTestTargetInBundle identifies the pipeline by its bundle KEY, which must precede any selections.
func resolveTestTargetInBundle(cmd *cobra.Command, args []string, resume bool) (testTarget, error) {
	var key string
	b, err := utils.ProcessBundle(cmd, utils.ProcessOptions{
		PostInitFunc: func(ctx context.Context, b *bundle.Bundle) error {
			var err error
			key, err = resolvePipelineArgument(ctx, b, args[:min(len(args), 1)])
			return err
		},
		ErrorOnEmptyState: true,
		SkipInitContext:   true,
	})
	if err != nil {
		return testTarget{}, err
	}
	ctx := cmd.Context()
	if !resume {
		suggestPipelineDeploy(ctx, cmd)
	}

	if _, err := resources.Lookup(b, key, isPipeline); err != nil {
		return testTarget{}, fmt.Errorf("%w; the first argument must be a pipeline KEY when test selections are given", err)
	}
	pipelineID, err := resolvePipelineIdFromKey(ctx, b, key)
	if err != nil {
		return testTarget{}, err
	}

	cwd, err := os.Getwd()
	if err != nil {
		return testTarget{}, err
	}
	var selectionArgs []string
	if len(args) > 1 {
		selectionArgs = args[1:]
	}
	w := b.WorkspaceClient(ctx)
	base, err := readTestIDBase(ctx, w, pipelineID, selectionArgs)
	if err != nil {
		return testTarget{}, err
	}
	filesRoot := bundleFilesRoot(b)
	selections, err := resolveTestSelections(selectionArgs, func(p string) (string, error) {
		return bundleTestSelectionPath(cwd, b.SyncRootPath, filesRoot, base, p)
	})
	if err != nil {
		return testTarget{}, err
	}
	return testTarget{key: key, pipelineID: pipelineID, selections: selections, w: w}, nil
}

// resumeCommand returns the command that reads the result of the update later.
func (t testTarget) resumeCommand(cmd *cobra.Command, updateID string) string {
	parts := []string{cmd.CommandPath()}
	if t.key != "" {
		parts = append(parts, shellquote.BashArg(t.key))
	} else {
		parts = append(parts, "--pipeline-id", shellquote.BashArg(t.pipelineID))
	}
	// Quoted so that the command can be pasted into a shell as is.
	if target := cmd.Flag("target").Value.String(); target != "" {
		parts = append(parts, "--target", shellquote.BashArg(target))
	}
	if profile := cmd.Flag("profile").Value.String(); profile != "" {
		parts = append(parts, "--profile", shellquote.BashArg(profile))
	}
	// Reading the results again replaces the partial report.
	if flag := cmd.Flag("junit-xml"); flag != nil && flag.Value.String() != "" {
		parts = append(parts, "--junit-xml", shellquote.BashArg(flag.Value.String()))
	}
	return strings.Join(append(parts, "--resume", updateID), " ")
}

const testPollInterval = time.Second

// testWatcher follows a test update and prints its cases as they finish.
type testWatcher struct {
	w          *databricks.WorkspaceClient
	pipelineID string
	updateID   string
	events     *testEventReader
	printer    *testProgressPrinter
	run        *testRun
}

func (tw *testWatcher) addEvents(events []testPipelineEvent) {
	for _, e := range events {
		if c := tw.run.add(e); c != nil {
			tw.printer.caseFinished(c)
		}
	}
}

// watch polls the update until it ends and returns its final state.
func (tw *testWatcher) watch(ctx context.Context) (pipelines.UpdateInfoState, error) {
	lastTimestamp := ""
	for {
		response, err := tw.w.Pipelines.GetUpdate(ctx, pipelines.GetUpdateRequest{
			PipelineId: tw.pipelineID,
			UpdateId:   tw.updateID,
		})
		if err != nil {
			return "", err
		}
		if response.Update == nil {
			return "", fmt.Errorf("no update found with id %s for pipeline %s", tw.updateID, tw.pipelineID)
		}
		state := response.Update.State
		tw.printer.updateState(state)

		// Reading from the last timestamp returns its events again; the run ignores duplicates.
		events, err := tw.events.read(ctx, lastTimestamp)
		if err != nil {
			return "", err
		}
		if len(events) > 0 {
			lastTimestamp = events[len(events)-1].Timestamp
		}
		tw.addEvents(events)

		switch state {
		case pipelines.UpdateInfoStateCompleted, pipelines.UpdateInfoStateFailed, pipelines.UpdateInfoStateCanceled:
			// Events can become readable after later ones, so read all of them once more.
			events, err := tw.events.read(ctx, "")
			if err != nil {
				return "", err
			}
			tw.addEvents(events)
			return state, nil
		default:
		}

		select {
		case <-ctx.Done():
			return "", ctx.Err()
		case <-time.After(testPollInterval):
		}
	}
}

// testRunError returns the error the command exits with after the update ended, or nil
// if every collected test passed or was skipped.
func testRunError(updateID string, state pipelines.UpdateInfoState, run *testRun, incompleteReason string) error {
	if incompleteReason == "" {
		summary := run.summaries[0]
		testsFailed := summary.Total == 0 || summary.Failed > 0 || summary.Errors > 0
		if !testsFailed && state == pipelines.UpdateInfoStateCompleted {
			return nil
		}
		// A test update fails when its tests fail; the printed result explains it, unless the
		// update ended in another way or reports an error of its own. The service's own verdict,
		// such as "[PIPELINE_TEST_FAILED] 1 of the pipeline's tests failed", is not.
		if testsFailed && state == pipelines.UpdateInfoStateFailed && run.fatalError == "" {
			return root.ErrAlreadyPrinted
		}
	} else if state == pipelines.UpdateInfoStateCompleted {
		return root.ErrAlreadyPrinted
	}
	if updateError := run.updateError(); updateError != "" {
		return fmt.Errorf("update %s is %s: %s", updateID, state, updateError)
	}
	return fmt.Errorf("update %s is %s", updateID, state)
}

// testRunProblem describes why a run that ended did not pass for a reason other than
// failed tests, or returns "" otherwise.
func testRunProblem(run *testRun, incompleteReason string, runErr error) string {
	var lines []string
	switch {
	case incompleteReason != "":
		lines = append(lines, fmt.Sprintf("RUN INCOMPLETE: %s.", incompleteReason))
	case run.summaries[0].Total == 0:
		lines = append(lines, "No tests were collected.")
	}
	if runErr != nil && !errors.Is(runErr, root.ErrAlreadyPrinted) {
		lines = append(lines, runErr.Error())
	}
	return strings.Join(lines, "\n")
}

var errTestWaitTimeout = errors.New("test wait timed out")

func testCommand() *cobra.Command {
	cmd := &cobra.Command{
		Use:   "test [PIPELINE_KEY] [TEST_SELECTION...] [flags] [--pytest-arg=ARG ...]",
		Short: "*Preview* Run the tests of a pipeline",
		Long: `Run the tests of a pipeline in a test-only update.

PIPELINE_KEY is the unique name of the pipeline, as defined in its YAML file.
It is required when test selections are given. If there is only one pipeline
in the project and no arguments are given, it is selected automatically.
Outside a project, identify the pipeline with --pipeline-id; all positional
arguments are then test selections.

TEST_SELECTION is a test directory, file, or pytest test ID, for example
tests/test_orders.py::test_backfill. A relative path that exists locally is
converted to its deployed workspace path. Any other relative path is a test ID
as the results show it: relative to the deepest folder that holds the
pipeline's root_path and its test sources. Paths that start with / are
workspace paths. Without a selection, all of the pipeline's tests run.

Each --pytest-arg passes one argument to pytest, in the order given. Only -k
and -m are supported, for example:

  databricks pipelines test my_pipeline --pytest-arg=-k --pytest-arg='orders and not slow'

The command waits for the results and succeeds only if at least one test ran
and none failed or errored. --no-wait returns after the update starts.
--timeout and Ctrl+C stop waiting but not the update; pass its ID to --resume
to read the results later. To stop the update, run 'pipelines stop'.

--junit-xml writes one entry per test with its outcome and duration. An
incomplete run, a run without tests, or a timeout adds a failing entry that
names the cause.`,
	}

	var pipelineID string
	var pytestArgs []string
	var noWait bool
	var timeout time.Duration
	var resumeUpdateID string
	var junitPath string

	testGroup := cmdgroup.NewFlagGroup("Pipeline Test")
	testGroup.FlagSet().StringVar(&pipelineID, "pipeline-id", "", "ID of the pipeline to test, instead of a PIPELINE_KEY.")
	testGroup.FlagSet().StringArrayVar(&pytestArgs, "pytest-arg", nil, "Argument to pass to pytest (repeatable). Only -k and -m are supported.")
	testGroup.FlagSet().BoolVar(&noWait, "no-wait", false, "Return after the test update starts, without waiting for results.")
	testGroup.FlagSet().DurationVar(&timeout, "timeout", 0, "Stop waiting for results after this duration, for example 30m. The update keeps running.")
	testGroup.FlagSet().StringVar(&resumeUpdateID, "resume", "", "Read the results of an existing test update instead of starting one.")
	testGroup.FlagSet().StringVar(&junitPath, "junit-xml", "", "Write the results as a JUnit XML report to this file, replacing it.")

	wrappedCmd := cmdgroup.NewCommandWithGroupFlag(cmd)
	wrappedCmd.AddFlagGroup(testGroup)

	cmd.SetFlagErrorFunc(testFlagErrorFunc)

	cmd.RunE = func(cmd *cobra.Command, args []string) error {
		if cmd.ArgsLenAtDash() >= 0 {
			return errors.New("the -- separator is not supported: " + pytestArgHint)
		}
		if root.OutputType(cmd) != flags.OutputText {
			return fmt.Errorf("--output %s is not supported by pipelines test", root.OutputType(cmd))
		}
		if err := validatePytestArgs(pytestArgs); err != nil {
			return err
		}
		if timeout < 0 {
			return errors.New("--timeout must not be negative")
		}
		if noWait && timeout > 0 {
			return errors.New("--timeout cannot be used with --no-wait")
		}
		if noWait && junitPath != "" {
			return errors.New("--junit-xml cannot be used with --no-wait because no results are available when the command returns")
		}
		if resumeUpdateID != "" {
			selectionCount := len(args)
			if pipelineID == "" {
				selectionCount = max(len(args)-1, 0)
			}
			switch {
			case noWait:
				return errors.New("--resume cannot be used with --no-wait")
			case len(pytestArgs) > 0:
				return errors.New("--resume cannot be used with --pytest-arg")
			case selectionCount > 0:
				return errors.New("--resume cannot be used with test selections")
			}
		}

		// From here on, every exit writes the report, so that an older report never survives.
		report := &testReport{path: junitPath, out: cmd.OutOrStdout(), suiteName: pipelineID}
		if report.suiteName == "" && len(args) > 0 {
			report.suiteName = args[0]
		}

		var target testTarget
		var err error
		if pipelineID != "" {
			target, err = resolveTestTargetByID(cmd, args, pipelineID)
		} else {
			ctx := logdiag.InitContext(cmd.Context())
			cmd.SetContext(ctx)
			if !bundle.HasRoot(ctx) {
				return errors.New("outside a project, identify the pipeline with --pipeline-id")
			}
			target, err = resolveTestTargetInBundle(cmd, args, resumeUpdateID != "")
		}
		if err != nil {
			return report.failed(err)
		}
		report.suiteName = target.key
		if report.suiteName == "" {
			report.suiteName = target.pipelineID
		}
		// The CLI root doesn't cancel the context on signals, so stop waiting here.
		ctx, stop := signal.NotifyContext(cmd.Context(), os.Interrupt, syscall.SIGTERM)
		defer stop()
		out := cmd.OutOrStdout()

		updateID := resumeUpdateID
		if updateID == "" {
			req := newStartTestUpdateRequest(target.selections, pytestArgs)
			updateID, err = startTestUpdate(ctx, target.w, target.pipelineID, req)
			if err != nil {
				return report.failed(err)
			}
		}
		target.printHeader(out, updateID)
		if noWait {
			return nil
		}

		events, err := newTestEventReader(target.w, target.pipelineID, updateID)
		if err != nil {
			return report.failed(err)
		}
		watcher := &testWatcher{
			w:          target.w,
			pipelineID: target.pipelineID,
			updateID:   updateID,
			events:     events,
			printer:    &testProgressPrinter{out: out},
			run:        newTestRun(),
		}

		waitCtx := ctx
		if timeout > 0 {
			var cancel context.CancelFunc
			waitCtx, cancel = context.WithTimeoutCause(ctx, timeout, errTestWaitTimeout)
			defer cancel()
		}
		state, err := watcher.watch(waitCtx)
		if err != nil {
			if waitCtx.Err() == nil {
				return report.finish(updateID, watcher.run, err.Error(), err)
			}
			resume := target.resumeCommand(cmd, updateID)
			stopErr := fmt.Errorf("stopped waiting for update %s; the update keeps running, to get its results run: %s", updateID, resume)
			if errors.Is(context.Cause(waitCtx), errTestWaitTimeout) {
				stopErr = fmt.Errorf("stopped waiting for update %s after %s; the update keeps running, to get its results run: %s", updateID, timeout, resume)
			}
			return report.finish(updateID, watcher.run, stopErr.Error(), stopErr)
		}

		incompleteReason := watcher.run.incompleteReason(state)
		renderTestResult(out, watcher.run, incompleteReason)
		runErr := testRunError(updateID, state, watcher.run, incompleteReason)
		return report.finish(updateID, watcher.run, testRunProblem(watcher.run, incompleteReason, runErr), runErr)
	}

	return cmd
}
