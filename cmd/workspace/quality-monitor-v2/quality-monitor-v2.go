// Code generated from OpenAPI specs by Databricks SDK Generator. DO NOT EDIT.

package quality_monitor_v2

import (
	"context"
	"encoding/json"
	"fmt"
	"time"

	"github.com/databricks/cli/cmd/root"
	"github.com/databricks/cli/libs/cmdctx"
	"github.com/databricks/cli/libs/cmdio"
	"github.com/databricks/cli/libs/flags"
	"github.com/databricks/databricks-sdk-go"
	"github.com/databricks/databricks-sdk-go/retries"
	"github.com/databricks/databricks-sdk-go/service/common/lro"
	"github.com/databricks/databricks-sdk-go/service/qualitymonitorv2"
	"github.com/databricks/databricks-sdk-go/useragent"
	"github.com/spf13/cobra"
)

// Slice with functions to override default command behavior.
// Functions can be added from the `init()` function in manually curated files in this directory.
var cmdOverrides []func(*cobra.Command)

func New() *cobra.Command {
	cmd := &cobra.Command{
		Use:     "quality-monitor-v2",
		Short:   `Manage data quality of UC objects (currently support schema).`,
		Long:    `Manage data quality of UC objects (currently support schema)`,
		GroupID: "qualitymonitorv2",
		Annotations: map[string]string{
			"package": "qualitymonitorv2",
		},
		RunE: root.ReportUnknownSubcommand,
	}

	// Add methods
	cmd.AddCommand(newCreateLibrary())
	cmd.AddCommand(newGetOperation())

	// Apply optional overrides to this command.
	for _, fn := range cmdOverrides {
		fn(cmd)
	}

	return cmd
}

// start create-library command

// pollOperation polls an existing operation until completion, similar to SDK's Wait implementation
func pollOperation(ctx context.Context, w *databricks.WorkspaceClient, operation *qualitymonitorv2.Operation, opts *lro.LroOptions) (*qualitymonitorv2.Library, error) {
	timeout := 20 * time.Minute
	if opts != nil && opts.Timeout > 0 {
		timeout = opts.Timeout
	}

	ctx = useragent.InContext(ctx, "sdk-feature", "long-running")
	return retries.Poll[qualitymonitorv2.Library](ctx, timeout, func() (*qualitymonitorv2.Library, *retries.Err) {
		op, err := w.QualityMonitorV2.GetOperation(ctx, qualitymonitorv2.GetOperationRequest{
			Name: operation.Name,
		})
		if err != nil {
			return nil, retries.Halt(err)
		}

		// Update local operation state
		operation = op

		if !op.Done {
			return nil, retries.Continues("operation still in progress")
		}

		if op.Error != nil {
			var errorMsg string
			if op.Error.Message != "" {
				errorMsg = op.Error.Message
			} else {
				errorMsg = "unknown error"
			}

			if op.Error.ErrorCode != "" {
				errorMsg = fmt.Sprintf("[%s] %s", op.Error.ErrorCode, errorMsg)
			}

			return nil, retries.Halt(fmt.Errorf("operation failed: %s", errorMsg))
		}

		// Operation completed successfully, unmarshal response
		if op.Response == nil {
			return nil, retries.Halt(fmt.Errorf("operation completed but no response available"))
		}

		var library qualitymonitorv2.Library
		err = json.Unmarshal(op.Response, &library)
		if err != nil {
			return nil, retries.Halt(fmt.Errorf("failed to unmarshal library response: %w", err))
		}

		return &library, nil
	})
}

// Slice with functions to override default command behavior.
// Functions can be added from the `init()` function in manually curated files in this directory.
var createLibraryOverrides []func(
	*cobra.Command,
	*qualitymonitorv2.CreateLibraryRequest,
)

func newCreateLibrary() *cobra.Command {
	cmd := &cobra.Command{}

	var createLibraryReq qualitymonitorv2.CreateLibraryRequest
	createLibraryReq.Library = qualitymonitorv2.Library{}
	var createLibraryJson flags.JsonFlag

	var createLibrarySkipWait bool
	var createLibraryWait bool
	var createLibraryStatus bool
	var createLibraryTimeout time.Duration

	cmd.Flags().BoolVar(&createLibrarySkipWait, "no-wait", createLibrarySkipWait, `do not wait to reach DONE state`)
	cmd.Flags().BoolVar(&createLibraryWait, "wait", createLibraryWait, `wait for an existing operation to complete`)
	cmd.Flags().BoolVar(&createLibraryStatus, "status", createLibraryStatus, `get the status of an existing operation`)
	cmd.Flags().DurationVar(&createLibraryTimeout, "timeout", 20*time.Minute, `maximum amount of time to reach DONE state`)

	cmd.Flags().Var(&createLibraryJson, "json", `either inline JSON string or @path/to/file.json with request body`)

	cmd.Flags().BoolVar(&createLibraryReq.Library.ContainsSauce, "contains-sauce", createLibraryReq.Library.ContainsSauce, `Whether the resource contains sauce.`)
	cmd.Flags().Int64Var(&createLibraryReq.Library.LegCount, "leg-count", createLibraryReq.Library.LegCount, `Count of legs in the resource.`)

	cmd.Use = "create-library NAME [OPERATION_NAME]"
	cmd.Short = `Create a Library Resource.`
	cmd.Long = `Create a Library Resource.

  Arguments:
    NAME: Name of the Resource. Must contain only: - alphanumeric characters -
      underscores - hyphens

      Note: The name must be unique within the scope of the resource.
    OPERATION_NAME: (Optional) The name of the operation to wait for or check status.
      Required when using --wait or --status flags.`

	cmd.Annotations = make(map[string]string)

	cmd.Args = func(cmd *cobra.Command, args []string) error {
		// For --wait or --status, we need exactly 1 arg (operation name)
		if cmd.Flags().Changed("wait") || cmd.Flags().Changed("status") {
			check := root.ExactArgs(1)
			return check(cmd, args)
		}

		if cmd.Flags().Changed("json") {
			err := root.ExactArgs(0)(cmd, args)
			if err != nil {
				return fmt.Errorf("when --json flag is specified, no positional arguments are required. Provide 'name' in your JSON input")
			}
			return nil
		}
		check := root.ExactArgs(1)
		return check(cmd, args)
	}

	cmd.PreRunE = root.MustWorkspaceClient
	cmd.RunE = func(cmd *cobra.Command, args []string) (err error) {
		ctx := cmd.Context()
		w := cmdctx.WorkspaceClient(ctx)

		// Determine which mode to execute based on flags
		switch {
		case createLibraryStatus:
			// Status mode: get operation status
			operationName := args[0]
			operation, err := w.QualityMonitorV2.GetOperation(ctx, qualitymonitorv2.GetOperationRequest{
				Name: operationName,
			})
			if err != nil {
				return err
			}
			return cmdio.Render(ctx, operation)

		case createLibraryWait:
			// Wait mode: wait for existing operation
			operationName := args[0]
			// Get the operation first to construct the wait object
			operation, err := w.QualityMonitorV2.GetOperation(ctx, qualitymonitorv2.GetOperationRequest{
				Name: operationName,
			})
			if err != nil {
				return err
			}

			// Manually poll the operation until completion
			opts := &lro.LroOptions{Timeout: createLibraryTimeout}
			result, err := pollOperation(ctx, w, operation, opts)
			if err != nil {
				return err
			}
			return cmdio.Render(ctx, result)

		case createLibrarySkipWait:
			// No-wait mode: start operation and return immediately
			if cmd.Flags().Changed("json") {
				diags := createLibraryJson.Unmarshal(&createLibraryReq.Library)
				if diags.HasError() {
					return diags.Error()
				}
				if len(diags) > 0 {
					err := cmdio.RenderDiagnosticsToErrorOut(ctx, diags)
					if err != nil {
						return err
					}
				}
			}
			if !cmd.Flags().Changed("json") {
				createLibraryReq.Library.Name = args[0]
			}

			wait, err := w.QualityMonitorV2.CreateLibrary(ctx, createLibraryReq)
			if err != nil {
				return err
			}

			// Return operation immediately without waiting
			operation, err := w.QualityMonitorV2.GetOperation(ctx, qualitymonitorv2.GetOperationRequest{
				Name: wait.Name(),
			})
			if err != nil {
				return err
			}
			return cmdio.Render(ctx, operation)

		default:
			// Default mode: start operation and wait for completion
			if cmd.Flags().Changed("json") {
				diags := createLibraryJson.Unmarshal(&createLibraryReq.Library)
				if diags.HasError() {
					return diags.Error()
				}
				if len(diags) > 0 {
					err := cmdio.RenderDiagnosticsToErrorOut(ctx, diags)
					if err != nil {
						return err
					}
				}
			}
			if !cmd.Flags().Changed("json") {
				createLibraryReq.Library.Name = args[0]
			}

			wait, err := w.QualityMonitorV2.CreateLibrary(ctx, createLibraryReq)
			if err != nil {
				return err
			}

			// Wait for completion
			opts := &lro.LroOptions{Timeout: createLibraryTimeout}
			response, err := wait.Wait(ctx, opts)
			if err != nil {
				return err
			}
			return cmdio.Render(ctx, response)
		}
	}

	// Disable completions since they are not applicable.
	// Can be overridden by manual implementation in `override.go`.
	cmd.ValidArgsFunction = cobra.NoFileCompletions

	// Apply optional overrides to this command.
	for _, fn := range createLibraryOverrides {
		fn(cmd, &createLibraryReq)
	}

	return cmd
}

// start get-operation command

// Slice with functions to override default command behavior.
// Functions can be added from the `init()` function in manually curated files in this directory.
var getOperationOverrides []func(
	*cobra.Command,
	*qualitymonitorv2.GetOperationRequest,
)

func newGetOperation() *cobra.Command {
	cmd := &cobra.Command{}

	var getOperationReq qualitymonitorv2.GetOperationRequest

	cmd.Use = "get-operation NAME"
	cmd.Short = `Get a Library Operation.`
	cmd.Long = `Get a Library Operation.

  Arguments:
    NAME: The name of the operation resource.`

	cmd.Annotations = make(map[string]string)

	cmd.Args = func(cmd *cobra.Command, args []string) error {
		check := root.ExactArgs(1)
		return check(cmd, args)
	}

	cmd.PreRunE = root.MustWorkspaceClient
	cmd.RunE = func(cmd *cobra.Command, args []string) (err error) {
		ctx := cmd.Context()
		w := cmdctx.WorkspaceClient(ctx)

		getOperationReq.Name = args[0]

		response, err := w.QualityMonitorV2.GetOperation(ctx, getOperationReq)
		if err != nil {
			return err
		}
		return cmdio.Render(ctx, response)
	}

	// Disable completions since they are not applicable.
	// Can be overridden by manual implementation in `override.go`.
	cmd.ValidArgsFunction = cobra.NoFileCompletions

	// Apply optional overrides to this command.
	for _, fn := range getOperationOverrides {
		fn(cmd, &getOperationReq)
	}

	return cmd
}

// end service QualityMonitorV2
