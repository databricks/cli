package pipelines

import (
	"context"
	"errors"
	"fmt"
	"regexp"
	"slices"
	"time"

	"github.com/databricks/cli/bundle/run"
	pipelinesCli "github.com/databricks/cli/cmd/pipelines"
	"github.com/databricks/cli/libs/cmdctx"
	"github.com/databricks/databricks-sdk-go"
	"github.com/databricks/databricks-sdk-go/service/pipelines"
	"github.com/spf13/cobra"
)

func init() {
	cmdOverrides = append(cmdOverrides, func(cli *cobra.Command) {
		// all auto-generated commands apart from nonManagementCommands go into 'management' group
		nonManagementCommands := []string{
			// 'stop' command is overloaded as Pipelines API and Pipelines CLI command
			"stop",
			// permission commands are assigned into "permission" group in cmd/cmd.go
			// only if they don't have GroupID set
			"get-permission-levels",
			"get-permissions",
			"set-permissions",
			"update-permissions",
		}

		for _, subCmd := range cli.Commands() {
			if slices.Contains(nonManagementCommands, subCmd.Name()) {
				continue
			}

			if subCmd.GroupID == "" {
				subCmd.GroupID = pipelinesCli.ManagementGroupID
			}
		}

		// main section is populated with commands from Pipelines CLI
		for _, pipelinesCmd := range pipelinesCli.Commands() {
			cli.AddCommand(pipelinesCmd)
		}

		// Add --var flag support (from cli/pipelines/variables.go)
		cli.PersistentFlags().StringSlice("var", []string{}, `set values for variables defined in project config. Example: --var="foo=bar"`)
	})

	// 'stop' command is different in context of bundle vs. management command
	stopOverrides = append(stopOverrides, func(cmd *cobra.Command, req *pipelines.StopRequest) {
		originalRunE := cmd.RunE
		cmd.RunE = func(cmd *cobra.Command, args []string) error {
			// For compatibility, if argument looks like pipeline ID, use API
			if len(args) > 0 && looksLikeUUID(args[0]) {
				return originalRunE(cmd, args)
			}
			// Looks like a bundle key or no args - use Lakeflow stop

			// context is already initialized by workspace command group
			// if we initialize it again, CLI crashes
			opts := pipelinesCli.StopCommandOpts{SkipInitContext: true}

			return pipelinesCli.StopCommand(opts).RunE(cmd, args)
		}

		// Update usage to reflect dual nature
		cmd.Use = "stop [KEY|PIPELINE_ID]"
		cmd.Short = "Stop a pipeline"
		cmd.Long = `Stop a pipeline.

With a bundle KEY: Stops the pipeline identified by KEY from your databricks.yml.
If there is only one pipeline in the bundle, KEY is optional.

With a PIPELINE_ID: Stops the pipeline identified by the UUID using the API.`
	})

	startUpdateOverrides = append(startUpdateOverrides, startUpdateOverride)
}

// startUpdateOverride adds --wait, which blocks until the started update ends and
// returns an error if the update fails or is canceled.
func startUpdateOverride(cmd *cobra.Command, req *pipelines.StartUpdate) {
	var wait bool
	var noWait bool
	var timeout time.Duration
	cmd.Flags().BoolVar(&wait, "wait", false, "Wait for the update to complete. Fails if the update fails or is canceled.")
	cmd.Flags().BoolVar(&noWait, "no-wait", false, "Return as soon as the update starts. This is the default.")
	cmd.Flags().DurationVar(&timeout, "timeout", 0, "With --wait, stop waiting after this duration, for example 100s. The update keeps running.")
	cmd.MarkFlagsMutuallyExclusive("wait", "no-wait")

	cmd.Long += `

  With --wait, the command logs the update's progress until it ends, and fails
  if the update fails or is canceled. Set --timeout to bound the wait below the
  time limit of the caller, for example --wait --timeout 100s for a 2 minute
  limit.`

	originalRunE := cmd.RunE
	cmd.RunE = func(cmd *cobra.Command, args []string) error {
		if timeout < 0 {
			return errors.New("--timeout must not be negative")
		}
		if !wait {
			if cmd.Flags().Changed("timeout") {
				return errors.New("--timeout requires --wait")
			}
			return originalRunE(cmd, args)
		}

		ctx := cmd.Context()
		w := cmdctx.WorkspaceClient(ctx)
		recorder := &startUpdateRecorder{PipelinesInterface: w.Pipelines}
		w.Pipelines = recorder
		err := originalRunE(cmd, args)
		if err != nil {
			return err
		}
		return waitForStartedUpdate(ctx, w, req.PipelineId, recorder.updateID, timeout)
	}
}

// startUpdateRecorder records the ID of the update that the generated command starts,
// so that --wait keeps the generated request handling and output.
type startUpdateRecorder struct {
	pipelines.PipelinesInterface
	updateID string
}

func (r *startUpdateRecorder) StartUpdate(ctx context.Context, req pipelines.StartUpdate) (*pipelines.StartUpdateResponse, error) {
	res, err := r.PipelinesInterface.StartUpdate(ctx, req)
	if err == nil {
		r.updateID = res.UpdateId
	}
	return res, err
}

var errStartUpdateTimeout = errors.New("start-update wait timed out")

// waitForStartedUpdate waits for the update to end. The timeout cancels the wait instead of
// setting a context deadline, because the SDK rate limiter fails a request that would wait
// past a deadline with an error that does not name the timeout.
func waitForStartedUpdate(ctx context.Context, w *databricks.WorkspaceClient, pipelineID, updateID string, timeout time.Duration) error {
	ctx, cancel := context.WithCancelCause(ctx)
	defer cancel(nil)
	if timeout > 0 {
		timer := time.AfterFunc(timeout, func() { cancel(errStartUpdateTimeout) })
		defer timer.Stop()
	}

	err := run.WaitForPipelineUpdate(ctx, w, pipelineID, updateID)
	if !errors.Is(err, context.Canceled) || !errors.Is(context.Cause(ctx), errStartUpdateTimeout) {
		return err
	}
	if errors.Is(err, run.ErrPipelineUpdateFailed) {
		return fmt.Errorf("update %s failed; its error details could not be retrieved within the %s timeout", updateID, timeout)
	}
	return fmt.Errorf("stopped waiting for update %s after %s; the update was not canceled. Check its state with: databricks pipelines get-update %s %s", updateID, timeout, pipelineID, updateID)
}

var uuidRegex = regexp.MustCompile(`^[0-9a-f]{8}-[0-9a-f]{4}-[0-9a-f]{4}-[0-9a-f]{4}-[0-9a-f]{12}$`)

// looksLikeUUID checks if a string matches the UUID format with lowercase hex digits
func looksLikeUUID(s string) bool {
	return uuidRegex.MatchString(s)
}
