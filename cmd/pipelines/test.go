package pipelines

import (
	"context"
	"errors"
	"fmt"
	"io"
	"net/http"
	"path/filepath"
	"strings"
	"uuid"

	"github.com/databricks/cli/cmd/root"
	"github.com/databricks/cli/libs/auth"
	"github.com/databricks/cli/libs/cmdctx"
	"github.com/databricks/cli/libs/cmdgroup"
	"github.com/databricks/cli/libs/env"
	"github.com/databricks/cli/libs/flags"
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

// pipelinesTestEnvVar enables the experimental pipelines test command.
const pipelinesTestEnvVar = "DATABRICKS_CLI_PIPELINES_TEST"

const pytestArgHint = "pass pytest arguments with --pytest-arg, for example --pytest-arg=-k --pytest-arg='orders and not slow'"

// validatePytestArgs accepts only -k and -m, each followed by its value in the next argument, as the API
// requires. The value cannot be empty or start with "-", because pytest reads it as an option.
func validatePytestArgs(args []string) error {
	for i := 0; i < len(args); i++ {
		arg := args[i]
		switch {
		case strings.TrimSpace(arg) == "":
			return errors.New("--pytest-arg requires a non-empty value")
		case arg == "-k" || arg == "-m":
			if i == len(args)-1 {
				return fmt.Errorf("pytest argument %s requires a value: pass it in the next --pytest-arg", arg)
			}
			next := args[i+1]
			if strings.TrimSpace(next) == "" {
				return fmt.Errorf("pytest argument %s requires a non-empty value", arg)
			}
			if strings.HasPrefix(next, "-") {
				return fmt.Errorf("pytest argument %s requires a value, not %q: pass it in the next --pytest-arg", arg, next)
			}
			i++
		case strings.HasPrefix(arg, "-k") || strings.HasPrefix(arg, "-m"):
			option, value := arg[:2], strings.TrimPrefix(arg[2:], "=")
			return fmt.Errorf("pytest argument %q must be split: pass --pytest-arg=%s --pytest-arg=%s", arg, option, shellquote.BashArg(value))
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
// the pytest test ID after "::" unchanged. Paths that start with "/" are workspace paths and are
// passed as typed: the service checks them and reports errors about what the user wrote.
func resolveTestSelections(selections []string, toWorkspacePath func(string) (string, error)) ([]string, error) {
	var resolved []string
	for _, selection := range selections {
		p, testID, _ := strings.Cut(selection, "::")
		if p == "" {
			return nil, fmt.Errorf("test selection %q has no path", selection)
		}
		if !strings.HasPrefix(p, "/") {
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

// hasRelativeSelection reports whether a selection is a relative path.
func hasRelativeSelection(selections []string) bool {
	for _, selection := range selections {
		if p, _, _ := strings.Cut(selection, "::"); !strings.HasPrefix(p, "/") {
			return true
		}
	}
	return false
}

// rootPathSelection appends a relative path to the pipeline's root_path without cleaning it,
// so that the service checks the path as the user typed it.
func rootPathSelection(rootPath, relPath string) (string, error) {
	if rootPath == "" {
		return "", errors.New("the pipeline has no root_path; use an absolute workspace path such as /Workspace/Users/someone@example.com/tests")
	}
	return strings.TrimSuffix(rootPath, "/") + "/" + filepath.ToSlash(relPath), nil
}

// getRootPath reads the root_path of the deployed pipeline.
func getRootPath(ctx context.Context, w *databricks.WorkspaceClient, pipelineID string) (string, error) {
	pipeline, err := w.Pipelines.Get(ctx, pipelines.GetPipelineRequest{PipelineId: pipelineID})
	if err != nil {
		return "", err
	}
	if pipeline.Spec == nil {
		return "", nil
	}
	return pipeline.Spec.RootPath, nil
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
	pipelineID string
	selections []string
	w          *databricks.WorkspaceClient
}

func (t testTarget) printHeader(out io.Writer, updateID string) {
	fmt.Fprintf(out, "Pipeline: %s\n", t.pipelineID)
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

	var rootPath string
	if hasRelativeSelection(args) {
		var err error
		rootPath, err = getRootPath(ctx, w, pipelineID)
		if err != nil {
			return testTarget{}, err
		}
	}
	selections, err := resolveTestSelections(args, func(p string) (string, error) {
		return rootPathSelection(rootPath, p)
	})
	if err != nil {
		return testTarget{}, err
	}
	return testTarget{pipelineID: pipelineID, selections: selections, w: w}, nil
}

func testCommand() *cobra.Command {
	cmd := &cobra.Command{
		Use:   "test [flags] [TEST_SELECTION...]",
		Short: "*Preview* Run the tests of a pipeline",
		Long: `Run the tests of a pipeline in a test-only update.

This command is experimental. Set DATABRICKS_CLI_PIPELINES_TEST=true to enable it.

Identify the pipeline with --pipeline-id.

TEST_SELECTION is a test directory, file, or pytest test ID, for example
tests/test_orders.py::test_backfill. A relative path is relative to the
pipeline's root_path. Paths that start with / are workspace paths. Without a
selection, all of the pipeline's tests run.

Each --pytest-arg passes one argument to pytest, in the order given. Only -k
and -m are supported, each followed by its value in the next --pytest-arg, for
example:

  databricks pipelines test --pipeline-id <id> --pytest-arg=-k --pytest-arg='orders and not slow'`,
	}

	var pipelineID string
	var pytestArgs []string

	testGroup := cmdgroup.NewFlagGroup("Pipeline Test")
	testGroup.FlagSet().StringVar(&pipelineID, "pipeline-id", "", "ID of the pipeline to test.")
	testGroup.FlagSet().StringArrayVar(&pytestArgs, "pytest-arg", nil, "Argument to pass to pytest (repeatable). Only -k and -m are supported.")

	wrappedCmd := cmdgroup.NewCommandWithGroupFlag(cmd)
	wrappedCmd.AddFlagGroup(testGroup)

	cmd.SetFlagErrorFunc(testFlagErrorFunc)

	cmd.RunE = func(cmd *cobra.Command, args []string) error {
		if enabled, _ := env.GetBool(cmd.Context(), pipelinesTestEnvVar); !enabled {
			return fmt.Errorf("pipelines test is experimental. Set %s=true to enable it", pipelinesTestEnvVar)
		}
		if cmd.ArgsLenAtDash() >= 0 {
			return errors.New("the -- separator is not supported: " + pytestArgHint)
		}
		if root.OutputType(cmd) != flags.OutputText {
			return fmt.Errorf("--output %s is not supported by pipelines test", root.OutputType(cmd))
		}
		if err := validatePytestArgs(pytestArgs); err != nil {
			return err
		}
		if pipelineID == "" {
			return errors.New("identify the pipeline with --pipeline-id")
		}
		id, err := uuid.Parse(pipelineID)
		if err != nil {
			return fmt.Errorf("--pipeline-id %q is not a pipeline ID", pipelineID)
		}

		target, err := resolveTestTargetByID(cmd, args, id.String())
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
