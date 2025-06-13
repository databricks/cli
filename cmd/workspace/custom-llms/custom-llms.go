// Code generated from OpenAPI specs by Databricks SDK Generator. DO NOT EDIT.

package custom_llms

import (
	"fmt"

	"github.com/databricks/cli/cmd/root"
	"github.com/databricks/cli/libs/cmdctx"
	"github.com/databricks/cli/libs/cmdio"
	"github.com/databricks/cli/libs/flags"
	"github.com/databricks/databricks-sdk-go/service/aibuilder"
	"github.com/spf13/cobra"
)

// Slice with functions to override default command behavior.
// Functions can be added from the `init()` function in manually curated files in this directory.
var cmdOverrides []func(*cobra.Command)

func New() *cobra.Command {
	cmd := &cobra.Command{
		Use:   "custom-llms",
		Short: `The Custom LLMs service manages state and powers the UI for the Custom LLM product.`,
		Long: `The Custom LLMs service manages state and powers the UI for the Custom LLM
  product.`,
		GroupID: "aibuilder",
		Annotations: map[string]string{
			"package": "aibuilder",
		},

		// This service is being previewed; hide from help output.
		Hidden: true,
		RunE:   root.ReportUnknownSubcommand,
	}

	// Add methods
	cmd.AddCommand(newCancel())
	cmd.AddCommand(newCreate())
	cmd.AddCommand(newGet())
	cmd.AddCommand(newUpdate())

	// Apply optional overrides to this command.
	for _, fn := range cmdOverrides {
		fn(cmd)
	}

	return cmd
}

// start cancel command

// Slice with functions to override default command behavior.
// Functions can be added from the `init()` function in manually curated files in this directory.
var cancelOverrides []func(
	*cobra.Command,
	*aibuilder.CancelCustomLlmOptimizationRunRequest,
)

func newCancel() *cobra.Command {
	cmd := &cobra.Command{}

	var cancelReq aibuilder.CancelCustomLlmOptimizationRunRequest

	// TODO: short flags

	cmd.Use = "cancel ID"
	cmd.Short = `Cancel a Custom LLM Optimization Run.`
	cmd.Long = `Cancel a Custom LLM Optimization Run.`

	cmd.Annotations = make(map[string]string)

	cmd.Args = func(cmd *cobra.Command, args []string) error {
		check := root.ExactArgs(1)
		return check(cmd, args)
	}

	cmd.PreRunE = root.MustWorkspaceClient
	cmd.RunE = func(cmd *cobra.Command, args []string) (err error) {
		ctx := cmd.Context()
		w := cmdctx.WorkspaceClient(ctx)

		cancelReq.Id = args[0]

		err = w.CustomLlms.Cancel(ctx, cancelReq)
		if err != nil {
			return err
		}
		return nil
	}

	// Disable completions since they are not applicable.
	// Can be overridden by manual implementation in `override.go`.
	cmd.ValidArgsFunction = cobra.NoFileCompletions

	// Apply optional overrides to this command.
	for _, fn := range cancelOverrides {
		fn(cmd, &cancelReq)
	}

	return cmd
}

// start create command

// Slice with functions to override default command behavior.
// Functions can be added from the `init()` function in manually curated files in this directory.
var createOverrides []func(
	*cobra.Command,
	*aibuilder.StartCustomLlmOptimizationRunRequest,
)

func newCreate() *cobra.Command {
	cmd := &cobra.Command{}

	var createReq aibuilder.StartCustomLlmOptimizationRunRequest

	// TODO: short flags

	cmd.Use = "create ID"
	cmd.Short = `Start a Custom LLM Optimization Run.`
	cmd.Long = `Start a Custom LLM Optimization Run.

  Arguments:
    ID: The Id of the tile.`

	cmd.Annotations = make(map[string]string)

	cmd.Args = func(cmd *cobra.Command, args []string) error {
		check := root.ExactArgs(1)
		return check(cmd, args)
	}

	cmd.PreRunE = root.MustWorkspaceClient
	cmd.RunE = func(cmd *cobra.Command, args []string) (err error) {
		ctx := cmd.Context()
		w := cmdctx.WorkspaceClient(ctx)

		createReq.Id = args[0]

		response, err := w.CustomLlms.Create(ctx, createReq)
		if err != nil {
			return err
		}
		return cmdio.Render(ctx, response)
	}

	// Disable completions since they are not applicable.
	// Can be overridden by manual implementation in `override.go`.
	cmd.ValidArgsFunction = cobra.NoFileCompletions

	// Apply optional overrides to this command.
	for _, fn := range createOverrides {
		fn(cmd, &createReq)
	}

	return cmd
}

// start get command

// Slice with functions to override default command behavior.
// Functions can be added from the `init()` function in manually curated files in this directory.
var getOverrides []func(
	*cobra.Command,
	*aibuilder.GetCustomLlmRequest,
)

func newGet() *cobra.Command {
	cmd := &cobra.Command{}

	var getReq aibuilder.GetCustomLlmRequest

	// TODO: short flags

	cmd.Use = "get ID"
	cmd.Short = `Get a Custom LLM.`
	cmd.Long = `Get a Custom LLM.

  Arguments:
    ID: The id of the custom llm`

	cmd.Annotations = make(map[string]string)

	cmd.Args = func(cmd *cobra.Command, args []string) error {
		check := root.ExactArgs(1)
		return check(cmd, args)
	}

	cmd.PreRunE = root.MustWorkspaceClient
	cmd.RunE = func(cmd *cobra.Command, args []string) (err error) {
		ctx := cmd.Context()
		w := cmdctx.WorkspaceClient(ctx)

		getReq.Id = args[0]

		response, err := w.CustomLlms.Get(ctx, getReq)
		if err != nil {
			return err
		}
		return cmdio.Render(ctx, response)
	}

	// Disable completions since they are not applicable.
	// Can be overridden by manual implementation in `override.go`.
	cmd.ValidArgsFunction = cobra.NoFileCompletions

	// Apply optional overrides to this command.
	for _, fn := range getOverrides {
		fn(cmd, &getReq)
	}

	return cmd
}

// start update command

// Slice with functions to override default command behavior.
// Functions can be added from the `init()` function in manually curated files in this directory.
var updateOverrides []func(
	*cobra.Command,
	*aibuilder.UpdateCustomLlmRequest,
)

func newUpdate() *cobra.Command {
	cmd := &cobra.Command{}

	var updateReq aibuilder.UpdateCustomLlmRequest
	var updateJson flags.JsonFlag

	// TODO: short flags
	cmd.Flags().Var(&updateJson, "json", `either inline JSON string or @path/to/file.json with request body`)

	cmd.Use = "update ID"
	cmd.Short = `Update a Custom LLM.`
	cmd.Long = `Update a Custom LLM.

  Arguments:
    ID: The id of the custom llm`

	cmd.Annotations = make(map[string]string)

	cmd.Args = func(cmd *cobra.Command, args []string) error {
		check := root.ExactArgs(1)
		return check(cmd, args)
	}

	cmd.PreRunE = root.MustWorkspaceClient
	cmd.RunE = func(cmd *cobra.Command, args []string) (err error) {
		ctx := cmd.Context()
		w := cmdctx.WorkspaceClient(ctx)

		if cmd.Flags().Changed("json") {
			diags := updateJson.Unmarshal(&updateReq)
			if diags.HasError() {
				return diags.Error()
			}
			if len(diags) > 0 {
				err := cmdio.RenderDiagnosticsToErrorOut(ctx, diags)
				if err != nil {
					return err
				}
			}
		} else {
			return fmt.Errorf("please provide command input in JSON format by specifying the --json flag")
		}
		updateReq.Id = args[0]

		response, err := w.CustomLlms.Update(ctx, updateReq)
		if err != nil {
			return err
		}
		return cmdio.Render(ctx, response)
	}

	// Disable completions since they are not applicable.
	// Can be overridden by manual implementation in `override.go`.
	cmd.ValidArgsFunction = cobra.NoFileCompletions

	// Apply optional overrides to this command.
	for _, fn := range updateOverrides {
		fn(cmd, &updateReq)
	}

	return cmd
}

// end service CustomLlms
