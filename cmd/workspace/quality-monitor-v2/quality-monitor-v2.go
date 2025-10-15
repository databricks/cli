// Code generated from OpenAPI specs by Databricks SDK Generator. DO NOT EDIT.

package quality_monitor_v2

import (
	"fmt"
	"time"

	"github.com/databricks/cli/cmd/root"
	"github.com/databricks/cli/libs/cmdctx"
	"github.com/databricks/cli/libs/cmdio"
	"github.com/databricks/cli/libs/flags"
	"github.com/databricks/databricks-sdk-go/service/common/lro"
	"github.com/databricks/databricks-sdk-go/service/qualitymonitorv2"
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
	var createLibraryTimeout time.Duration

	cmd.Flags().BoolVar(&createLibrarySkipWait, "no-wait", createLibrarySkipWait, `do not wait to reach DONE state`)
	cmd.Flags().DurationVar(&createLibraryTimeout, "timeout", 20*time.Minute, `maximum amount of time to reach DONE state`)

	cmd.Flags().Var(&createLibraryJson, "json", `either inline JSON string or @path/to/file.json with request body`)

	cmd.Flags().BoolVar(&createLibraryReq.Library.ContainsSauce, "contains-sauce", createLibraryReq.Library.ContainsSauce, `Whether the resource contains sauce.`)
	cmd.Flags().Int64Var(&createLibraryReq.Library.LegCount, "leg-count", createLibraryReq.Library.LegCount, `Count of legs in the resource.`)

	cmd.Use = "create-library NAME"
	cmd.Short = `Create a Library Resource.`
	cmd.Long = `Create a Library Resource.

  This is a long-running operation. By default, the command waits for the
  operation to complete. Use --no-wait to return immediately with the raw
  operation details. The operation's 'name' field can then be used to poll for
  completion using the get-operation command.

  Arguments:
    NAME: Name of the Resource. Must contain only: - alphanumeric characters -
      underscores - hyphens
      
      Note: The name must be unique within the scope of the resource.`

	cmd.Annotations = make(map[string]string)

	cmd.Args = func(cmd *cobra.Command, args []string) error {
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

		// Determine which mode to execute based on flags
		switch {
		case createLibrarySkipWait:
			wait, err := w.QualityMonitorV2.CreateLibrary(ctx, createLibraryReq)
			if err != nil {
				return err
			}

			// Return operation immediately without waiting.
			operation, err := w.QualityMonitorV2.GetOperation(ctx, qualitymonitorv2.GetOperationRequest{
				Name: wait.Name(),
			})
			if err != nil {
				return err
			}
			return cmdio.Render(ctx, operation)

		default:
			wait, err := w.QualityMonitorV2.CreateLibrary(ctx, createLibraryReq)
			if err != nil {
				return err
			}

			// Show spinner while waiting for completion.
			spinner := cmdio.Spinner(ctx)
			spinner <- "Waiting for create-library to complete..."

			// Wait for completion.
			opts := &lro.LroOptions{Timeout: createLibraryTimeout}
			response, err := wait.Wait(ctx, opts)
			if err != nil {
				return err
			}
			close(spinner)
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
