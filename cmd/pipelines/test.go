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
// path under the bundle's workspace file path.
func bundleWorkspacePath(cwd, syncRoot, workspaceFilePath, localPath string) (string, error) {
	if !filepath.IsAbs(localPath) {
		localPath = filepath.Join(cwd, localPath)
	}
	rel, err := filepath.Rel(syncRoot, localPath)
	if err != nil || rel == ".." || strings.HasPrefix(rel, ".."+string(filepath.Separator)) {
		return "", fmt.Errorf("%s is outside the synchronized bundle files in %s", localPath, syncRoot)
	}
	return path.Join(workspaceFilePath, filepath.ToSlash(rel)), nil
}

// rootPathWorkspacePath resolves a relative path against the pipeline's root_path.
func rootPathWorkspacePath(rootPath, relPath string) (string, error) {
	if rootPath == "" {
		return "", errors.New("the pipeline has no root_path; use an absolute workspace path such as /Workspace/Users/someone@example.com/tests")
	}
	return path.Join(rootPath, filepath.ToSlash(relPath)), nil
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

	var rootPath *string
	selections, err := resolveTestSelections(args, func(p string) (string, error) {
		if rootPath == nil {
			pipeline, err := w.Pipelines.Get(ctx, pipelines.GetPipelineRequest{PipelineId: pipelineID})
			if err != nil {
				return "", err
			}
			rootPath = new(string)
			if pipeline.Spec != nil {
				*rootPath = pipeline.Spec.RootPath
			}
		}
		return rootPathWorkspacePath(*rootPath, p)
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
	suggestPipelineDeploy(ctx, cmd)

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
	selections, err := resolveTestSelections(selectionArgs, func(p string) (string, error) {
		return bundleWorkspacePath(cwd, b.SyncRootPath, b.Config.Workspace.FilePath, p)
	})
	if err != nil {
		return testTarget{}, err
	}
	return testTarget{key: key, pipelineID: pipelineID, selections: selections, w: b.WorkspaceClient(ctx)}, nil
}

func testCommand() *cobra.Command {
	cmd := &cobra.Command{
		Use:   "test [PIPELINE_KEY] [TEST_SELECTION...] [flags] [--pytest-arg=ARG ...]",
		Short: "Run the tests of a pipeline",
		Long: `Run the tests of a pipeline in a test-only update.

PIPELINE_KEY is the unique name of the pipeline, as defined in its YAML file.
It is required when test selections are given. If there is only one pipeline
in the project and no arguments are given, it is selected automatically.
Outside a project, identify the pipeline with --pipeline-id; all positional
arguments are then test selections.

TEST_SELECTION is a test directory, file, or pytest test ID, for example
tests/test_orders.py::test_backfill. Relative paths are converted to their
deployed workspace paths, or outside a project resolved against the pipeline's
root_path. Paths that start with / are workspace paths. Without a selection,
all of the pipeline's tests run.

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
