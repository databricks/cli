package pipelines

import (
	"context"
	"errors"
	"fmt"
	"io"
	"net/http"
	"os"
	"path"
	"path/filepath"
	"strings"

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
// A value in the next argument cannot start with "-", because pytest reads it as an option.
func validatePytestArgs(args []string) error {
	for i := 0; i < len(args); i++ {
		arg := args[i]
		switch {
		case arg == "-k" || arg == "-m":
			if i == len(args)-1 {
				return fmt.Errorf("pytest argument %s requires a value: pass it in the next --pytest-arg", arg)
			}
			if next := args[i+1]; strings.HasPrefix(next, "-") {
				return fmt.Errorf("pytest argument %s requires a value, not %q: pass it in the next --pytest-arg", arg, next)
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
func resolveTestTargetInBundle(cmd *cobra.Command, args []string) (testTarget, error) {
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

	if _, err := resources.Lookup(b, key, isPipeline); err != nil {
		return testTarget{}, fmt.Errorf("%w; the first argument must be a pipeline KEY when test selections are given", err)
	}
	pipelineID, err := resolvePipelineIdFromKey(ctx, b, key)
	if err != nil {
		return testTarget{}, err
	}
	suggestPipelineDeploy(ctx, cmd)

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

  databricks pipelines test my_pipeline --pytest-arg=-k --pytest-arg='orders and not slow'`,
	}

	var pipelineID string
	var pytestArgs []string

	testGroup := cmdgroup.NewFlagGroup("Pipeline Test")
	testGroup.FlagSet().StringVar(&pipelineID, "pipeline-id", "", "ID of the pipeline to test, instead of a PIPELINE_KEY.")
	testGroup.FlagSet().StringArrayVar(&pytestArgs, "pytest-arg", nil, "Argument to pass to pytest (repeatable). Only -k and -m are supported.")

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
			target, err = resolveTestTargetInBundle(cmd, args)
		}
		if err != nil {
			return err
		}
		ctx := cmd.Context()

		req := newStartTestUpdateRequest(target.selections, pytestArgs)
		updateID, err := startTestUpdate(ctx, target.w, target.pipelineID, req)
		if err != nil {
			return err
		}
		target.printHeader(cmd.OutOrStdout(), updateID)
		return nil
	}

	return cmd
}
