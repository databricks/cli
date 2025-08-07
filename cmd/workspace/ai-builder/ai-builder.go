// Code generated from OpenAPI specs by Databricks SDK Generator. DO NOT EDIT.

package ai_builder

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
		Use:   "ai-builder",
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
	cmd.AddCommand(newCancelOptimize())
	cmd.AddCommand(newCreateCustomLlm())
	cmd.AddCommand(newDeleteCustomLlm())
	cmd.AddCommand(newGetCustomLlm())
	cmd.AddCommand(newStartOptimize())
	cmd.AddCommand(newUpdateCustomLlm())

	// Apply optional overrides to this command.
	for _, fn := range cmdOverrides {
		fn(cmd)
	}

	return cmd
}

// start cancel-optimize command

// Slice with functions to override default command behavior.
// Functions can be added from the `init()` function in manually curated files in this directory.
var cancelOptimizeOverrides []func(
	*cobra.Command,
	*aibuilder.CancelCustomLlmOptimizationRunRequest,
)

func newCancelOptimize() *cobra.Command {
	cmd := &cobra.Command{}

	var cancelOptimizeReq aibuilder.CancelCustomLlmOptimizationRunRequest

	cmd.Use = "cancel-optimize ID"
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

		cancelOptimizeReq.Id = args[0]

		err = w.AiBuilder.CancelOptimize(ctx, cancelOptimizeReq)
		if err != nil {
			return err
		}
		return nil
	}

	// Disable completions since they are not applicable.
	// Can be overridden by manual implementation in `override.go`.
	cmd.ValidArgsFunction = cobra.NoFileCompletions

	// Apply optional overrides to this command.
	for _, fn := range cancelOptimizeOverrides {
		fn(cmd, &cancelOptimizeReq)
	}

	return cmd
}

// start create-custom-llm command

// Slice with functions to override default command behavior.
// Functions can be added from the `init()` function in manually curated files in this directory.
var createCustomLlmOverrides []func(
	*cobra.Command,
	*aibuilder.CreateCustomLlmRequest,
)

func newCreateCustomLlm() *cobra.Command {
	cmd := &cobra.Command{}

	var createCustomLlmReq aibuilder.CreateCustomLlmRequest
	var createCustomLlmJson flags.JsonFlag

	cmd.Flags().Var(&createCustomLlmJson, "json", `either inline JSON string or @path/to/file.json with request body`)

	cmd.Flags().StringVar(&createCustomLlmReq.AgentArtifactPath, "agent-artifact-path", createCustomLlmReq.AgentArtifactPath, `Optional: UC path for agent artifacts.`)
	// TODO: array: datasets
	// TODO: array: guidelines

	cmd.Use = "create-custom-llm NAME INSTRUCTIONS"
	cmd.Short = `Create a Custom LLM.`
	cmd.Long = `Create a Custom LLM.

  Arguments:
    NAME: Name of the custom LLM. Only alphanumeric characters and dashes allowed.
    INSTRUCTIONS: Instructions for the custom LLM to follow`

	cmd.Annotations = make(map[string]string)

	cmd.Args = func(cmd *cobra.Command, args []string) error {
		if cmd.Flags().Changed("json") {
			err := root.ExactArgs(0)(cmd, args)
			if err != nil {
				return fmt.Errorf("when --json flag is specified, no positional arguments are required. Provide 'name', 'instructions' in your JSON input")
			}
			return nil
		}
		check := root.ExactArgs(2)
		return check(cmd, args)
	}

	cmd.PreRunE = root.MustWorkspaceClient
	cmd.RunE = func(cmd *cobra.Command, args []string) (err error) {
		ctx := cmd.Context()
		w := cmdctx.WorkspaceClient(ctx)

		if cmd.Flags().Changed("json") {
			diags := createCustomLlmJson.Unmarshal(&createCustomLlmReq)
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
			createCustomLlmReq.Name = args[0]
		}
		if !cmd.Flags().Changed("json") {
			createCustomLlmReq.Instructions = args[1]
		}

		response, err := w.AiBuilder.CreateCustomLlm(ctx, createCustomLlmReq)
		if err != nil {
			return err
		}
		return cmdio.Render(ctx, response)
	}

	// Disable completions since they are not applicable.
	// Can be overridden by manual implementation in `override.go`.
	cmd.ValidArgsFunction = cobra.NoFileCompletions

	// Apply optional overrides to this command.
	for _, fn := range createCustomLlmOverrides {
		fn(cmd, &createCustomLlmReq)
	}

	return cmd
}

// start delete-custom-llm command

// Slice with functions to override default command behavior.
// Functions can be added from the `init()` function in manually curated files in this directory.
var deleteCustomLlmOverrides []func(
	*cobra.Command,
	*aibuilder.DeleteCustomLlmRequest,
)

func newDeleteCustomLlm() *cobra.Command {
	cmd := &cobra.Command{}

	var deleteCustomLlmReq aibuilder.DeleteCustomLlmRequest

	cmd.Use = "delete-custom-llm ID"
	cmd.Short = `Delete a Custom LLM.`
	cmd.Long = `Delete a Custom LLM.

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

		deleteCustomLlmReq.Id = args[0]

		err = w.AiBuilder.DeleteCustomLlm(ctx, deleteCustomLlmReq)
		if err != nil {
			return err
		}
		return nil
	}

	// Disable completions since they are not applicable.
	// Can be overridden by manual implementation in `override.go`.
	cmd.ValidArgsFunction = cobra.NoFileCompletions

	// Apply optional overrides to this command.
	for _, fn := range deleteCustomLlmOverrides {
		fn(cmd, &deleteCustomLlmReq)
	}

	return cmd
}

// start get-custom-llm command

// Slice with functions to override default command behavior.
// Functions can be added from the `init()` function in manually curated files in this directory.
var getCustomLlmOverrides []func(
	*cobra.Command,
	*aibuilder.GetCustomLlmRequest,
)

func newGetCustomLlm() *cobra.Command {
	cmd := &cobra.Command{}

	var getCustomLlmReq aibuilder.GetCustomLlmRequest

	cmd.Use = "get-custom-llm ID"
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

		getCustomLlmReq.Id = args[0]

		response, err := w.AiBuilder.GetCustomLlm(ctx, getCustomLlmReq)
		if err != nil {
			return err
		}
		return cmdio.Render(ctx, response)
	}

	// Disable completions since they are not applicable.
	// Can be overridden by manual implementation in `override.go`.
	cmd.ValidArgsFunction = cobra.NoFileCompletions

	// Apply optional overrides to this command.
	for _, fn := range getCustomLlmOverrides {
		fn(cmd, &getCustomLlmReq)
	}

	return cmd
}

// start start-optimize command

// Slice with functions to override default command behavior.
// Functions can be added from the `init()` function in manually curated files in this directory.
var startOptimizeOverrides []func(
	*cobra.Command,
	*aibuilder.StartCustomLlmOptimizationRunRequest,
)

func newStartOptimize() *cobra.Command {
	cmd := &cobra.Command{}

	var startOptimizeReq aibuilder.StartCustomLlmOptimizationRunRequest

	cmd.Use = "start-optimize ID"
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

		startOptimizeReq.Id = args[0]

		response, err := w.AiBuilder.StartOptimize(ctx, startOptimizeReq)
		if err != nil {
			return err
		}
		return cmdio.Render(ctx, response)
	}

	// Disable completions since they are not applicable.
	// Can be overridden by manual implementation in `override.go`.
	cmd.ValidArgsFunction = cobra.NoFileCompletions

	// Apply optional overrides to this command.
	for _, fn := range startOptimizeOverrides {
		fn(cmd, &startOptimizeReq)
	}

	return cmd
}

// start update-custom-llm command

// Slice with functions to override default command behavior.
// Functions can be added from the `init()` function in manually curated files in this directory.
var updateCustomLlmOverrides []func(
	*cobra.Command,
	*aibuilder.UpdateCustomLlmRequest,
)

func newUpdateCustomLlm() *cobra.Command {
	cmd := &cobra.Command{}

	var updateCustomLlmReq aibuilder.UpdateCustomLlmRequest
	var updateCustomLlmJson flags.JsonFlag

	cmd.Flags().Var(&updateCustomLlmJson, "json", `either inline JSON string or @path/to/file.json with request body`)

	cmd.Use = "update-custom-llm ID"
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
			diags := updateCustomLlmJson.Unmarshal(&updateCustomLlmReq)
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
		updateCustomLlmReq.Id = args[0]

		response, err := w.AiBuilder.UpdateCustomLlm(ctx, updateCustomLlmReq)
		if err != nil {
			return err
		}
		return cmdio.Render(ctx, response)
	}

	// Disable completions since they are not applicable.
	// Can be overridden by manual implementation in `override.go`.
	cmd.ValidArgsFunction = cobra.NoFileCompletions

	// Apply optional overrides to this command.
	for _, fn := range updateCustomLlmOverrides {
		fn(cmd, &updateCustomLlmReq)
	}

	return cmd
}

// end service AiBuilder
