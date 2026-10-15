// Code generated from OpenAPI specs by Databricks SDK Generator. DO NOT EDIT.

package agent_bricks

import (
	"errors"

	"github.com/databricks/cli/cmd/root"
	"github.com/databricks/cli/libs/cmdctx"
	"github.com/databricks/cli/libs/cmdio"
	"github.com/databricks/cli/libs/flags"
	"github.com/spf13/cobra"

	customllms "github.com/databricks/cli/libs/sdktypes/customllms/v1"
	sdkcustomllms "github.com/databricks/sdk-go/customllms/v1"
)

// Slice with functions to override default command behavior.
// Functions can be added from the `init()` function in manually curated files in this directory.
var cmdOverrides []func(*cobra.Command)

func New() *cobra.Command {
	cmd := &cobra.Command{
		Use:   "agent-bricks",
		Short: `The Custom LLMs service manages state and powers the UI for the Custom LLM product.`,
		Long: `The Custom LLMs service manages state and powers the UI for the Custom LLM
  product.`,
		GroupID: "agentbricks",

		// This service is being previewed; hide from help output.
		Hidden: true,
		RunE:   root.ReportUnknownSubcommand,
	}

	cmd.Annotations = make(map[string]string)
	cmd.Annotations["launch_stage"] = "PRIVATE_PREVIEW"
	cmd.Annotations["launch_stage_display"] = "Private Preview"

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
	*customllms.CancelCustomLlmOptimizationRunRequest,
)

func newCancelOptimize() *cobra.Command {
	cmd := &cobra.Command{}
	var cancelOptimizeReq customllms.CancelCustomLlmOptimizationRunRequest

	cmd.Use = "cancel-optimize ID"
	cmd.Short = `Cancel a Custom LLM Optimization Run.`
	cmd.Long = `Cancel a Custom LLM Optimization Run.`

	cmd.Annotations = make(map[string]string)
	cmd.Annotations["launch_stage"] = "PRIVATE_PREVIEW"
	cmd.Annotations["launch_stage_display"] = "Private Preview"

	cmd.Args = func(cmd *cobra.Command, args []string) error {
		check := root.ExactArgs(1)
		return check(cmd, args)
	}

	cmd.PreRunE = root.MustWorkspaceClient
	cmd.RunE = func(cmd *cobra.Command, args []string) (err error) {
		ctx := cmd.Context()
		w := cmdctx.WorkspaceClient(ctx)
		_ = w

		cancelOptimizeReq.Id = new(args[0])

		sdkReq, err := customllms.CancelCustomLlmOptimizationRunRequestToSDK(&cancelOptimizeReq)
		if err != nil {
			return err
		}
		err = w.CustomLlms.CancelCustomLlmOptimizationRun(ctx, *sdkReq)
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
	*customllms.CreateCustomLlmRequest,
)

func newCreateCustomLlm() *cobra.Command {
	cmd := &cobra.Command{}
	var createCustomLlmReq customllms.CreateCustomLlmRequest

	var createCustomLlmJson flags.JsonFlag

	cmd.Flags().Var(&createCustomLlmJson, "json", `either inline JSON string or @path/to/file.json with request body`)

	flags.StringPtrVar(cmd.Flags(), &createCustomLlmReq.AgentArtifactPath, "agent-artifact-path", `This will soon be deprecated!! Optional: UC path for agent artifacts.`)
	// TODO: array: datasets
	// TODO: array: guidelines

	cmd.Use = "create-custom-llm NAME INSTRUCTIONS"
	cmd.Short = `Create a Custom LLM.`
	cmd.Long = `Create a Custom LLM.

  Arguments:
    NAME: Name of the custom LLM. Only alphanumeric characters and dashes allowed.
    INSTRUCTIONS: Instructions for the custom LLM to follow`

	cmd.Annotations = make(map[string]string)
	cmd.Annotations["launch_stage"] = "PRIVATE_PREVIEW"
	cmd.Annotations["launch_stage_display"] = "Private Preview"

	cmd.Args = func(cmd *cobra.Command, args []string) error {
		if cmd.Flags().Changed("json") {
			err := root.ExactArgs(0)(cmd, args)
			if err != nil {
				return errors.New("when --json flag is specified, no positional arguments are allowed. Provide 'name', 'instructions' in your JSON input")
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
		_ = w

		if cmd.Flags().Changed("json") {
			diags := createCustomLlmJson.Unmarshal(&createCustomLlmReq)
			if diags.HasError() {
				return diags.Error()
			}
			if len(diags) > 0 {
				err := cmdio.RenderDiagnostics(ctx, diags)
				if err != nil {
					return err
				}
			}
		}
		if !cmd.Flags().Changed("json") {
			createCustomLlmReq.Name = new(args[0])
		}
		if !cmd.Flags().Changed("json") {
			createCustomLlmReq.Instructions = new(args[1])
		}

		sdkReq, err := customllms.CreateCustomLlmRequestToSDK(&createCustomLlmReq)
		if err != nil {
			return err
		}
		response, err := w.CustomLlms.CreateCustomLlm(ctx, *sdkReq)
		if err != nil {
			return err
		}

		out, err := customllms.CustomLlmFromSDK(response)
		if err != nil {
			return err
		}
		return cmdio.Render(ctx, out)
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
	*sdkcustomllms.DeleteCustomLlmRequest,
)

func newDeleteCustomLlm() *cobra.Command {
	cmd := &cobra.Command{}
	var deleteCustomLlmReq sdkcustomllms.DeleteCustomLlmRequest

	cmd.Use = "delete-custom-llm ID"
	cmd.Short = `Delete a Custom LLM.`
	cmd.Long = `Delete a Custom LLM.

  Arguments:
    ID: The id of the custom llm`

	cmd.Annotations = make(map[string]string)
	cmd.Annotations["launch_stage"] = "PRIVATE_PREVIEW"
	cmd.Annotations["launch_stage_display"] = "Private Preview"

	cmd.Args = func(cmd *cobra.Command, args []string) error {
		check := root.ExactArgs(1)
		return check(cmd, args)
	}

	cmd.PreRunE = root.MustWorkspaceClient
	cmd.RunE = func(cmd *cobra.Command, args []string) (err error) {
		ctx := cmd.Context()
		w := cmdctx.WorkspaceClient(ctx)
		_ = w

		deleteCustomLlmReq.Id = new(args[0])

		sdkReq := &deleteCustomLlmReq
		err = w.CustomLlms.DeleteCustomLlm(ctx, *sdkReq)
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
	*sdkcustomllms.GetCustomLlmRequest,
)

func newGetCustomLlm() *cobra.Command {
	cmd := &cobra.Command{}
	var getCustomLlmReq sdkcustomllms.GetCustomLlmRequest

	cmd.Use = "get-custom-llm ID"
	cmd.Short = `Get a Custom LLM.`
	cmd.Long = `Get a Custom LLM.

  Arguments:
    ID: The id of the custom llm`

	cmd.Annotations = make(map[string]string)
	cmd.Annotations["launch_stage"] = "PRIVATE_PREVIEW"
	cmd.Annotations["launch_stage_display"] = "Private Preview"

	cmd.Args = func(cmd *cobra.Command, args []string) error {
		check := root.ExactArgs(1)
		return check(cmd, args)
	}

	cmd.PreRunE = root.MustWorkspaceClient
	cmd.RunE = func(cmd *cobra.Command, args []string) (err error) {
		ctx := cmd.Context()
		w := cmdctx.WorkspaceClient(ctx)
		_ = w

		getCustomLlmReq.Id = new(args[0])

		sdkReq := &getCustomLlmReq
		response, err := w.CustomLlms.GetCustomLlm(ctx, *sdkReq)
		if err != nil {
			return err
		}

		out, err := customllms.CustomLlmFromSDK(response)
		if err != nil {
			return err
		}
		return cmdio.Render(ctx, out)
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
	*customllms.StartCustomLlmOptimizationRunRequest,
)

func newStartOptimize() *cobra.Command {
	cmd := &cobra.Command{}
	var startOptimizeReq customllms.StartCustomLlmOptimizationRunRequest

	cmd.Use = "start-optimize ID"
	cmd.Short = `Start a Custom LLM Optimization Run.`
	cmd.Long = `Start a Custom LLM Optimization Run.

  Arguments:
    ID: The Id of the tile.`

	cmd.Annotations = make(map[string]string)
	cmd.Annotations["launch_stage"] = "PRIVATE_PREVIEW"
	cmd.Annotations["launch_stage_display"] = "Private Preview"

	cmd.Args = func(cmd *cobra.Command, args []string) error {
		check := root.ExactArgs(1)
		return check(cmd, args)
	}

	cmd.PreRunE = root.MustWorkspaceClient
	cmd.RunE = func(cmd *cobra.Command, args []string) (err error) {
		ctx := cmd.Context()
		w := cmdctx.WorkspaceClient(ctx)
		_ = w

		startOptimizeReq.Id = new(args[0])

		sdkReq, err := customllms.StartCustomLlmOptimizationRunRequestToSDK(&startOptimizeReq)
		if err != nil {
			return err
		}
		response, err := w.CustomLlms.StartCustomLlmOptimizationRun(ctx, *sdkReq)
		if err != nil {
			return err
		}

		out, err := customllms.CustomLlmFromSDK(response)
		if err != nil {
			return err
		}
		return cmdio.Render(ctx, out)
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
	*customllms.UpdateCustomLlmRequest,
)

func newUpdateCustomLlm() *cobra.Command {
	cmd := &cobra.Command{}
	var updateCustomLlmReq customllms.UpdateCustomLlmRequest

	var updateCustomLlmJson flags.JsonFlag

	cmd.Flags().Var(&updateCustomLlmJson, "json", `either inline JSON string or @path/to/file.json with request body`)

	cmd.Use = "update-custom-llm ID"
	cmd.Short = `Update a Custom LLM.`
	cmd.Long = `Update a Custom LLM.

  Arguments:
    ID: The id of the custom llm`

	cmd.Annotations = make(map[string]string)
	cmd.Annotations["launch_stage"] = "PRIVATE_PREVIEW"
	cmd.Annotations["launch_stage_display"] = "Private Preview"

	cmd.Args = func(cmd *cobra.Command, args []string) error {
		check := root.ExactArgs(1)
		return check(cmd, args)
	}

	cmd.PreRunE = root.MustWorkspaceClient
	cmd.RunE = func(cmd *cobra.Command, args []string) (err error) {
		ctx := cmd.Context()
		w := cmdctx.WorkspaceClient(ctx)
		_ = w

		if cmd.Flags().Changed("json") {
			diags := updateCustomLlmJson.Unmarshal(&updateCustomLlmReq)
			if diags.HasError() {
				return diags.Error()
			}
			if len(diags) > 0 {
				err := cmdio.RenderDiagnostics(ctx, diags)
				if err != nil {
					return err
				}
			}
		} else {
			return errors.New("please provide command input in JSON format by specifying the --json flag")
		}
		updateCustomLlmReq.Id = new(args[0])

		sdkReq, err := customllms.UpdateCustomLlmRequestToSDK(&updateCustomLlmReq)
		if err != nil {
			return err
		}
		response, err := w.CustomLlms.UpdateCustomLlm(ctx, *sdkReq)
		if err != nil {
			return err
		}

		out, err := customllms.CustomLlmFromSDK(response)
		if err != nil {
			return err
		}
		return cmdio.Render(ctx, out)
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

// end service AgentBricks
