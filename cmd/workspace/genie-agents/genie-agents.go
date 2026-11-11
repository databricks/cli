// Code generated from OpenAPI specs by Databricks SDK Generator. DO NOT EDIT.

package genie_agents

import (
	"context"
	"errors"
	"fmt"
	"strings"

	"github.com/databricks/cli/cmd/root"
	"github.com/databricks/cli/libs/cmdctx"
	"github.com/databricks/cli/libs/cmdio"
	"github.com/databricks/cli/libs/flags"
	"github.com/databricks/databricks-sdk-go/common/types/fieldmask"
	"github.com/databricks/databricks-sdk-go/service/genieagents"
	"github.com/spf13/cobra"
)

// Slice with functions to override default command behavior.
// Functions can be added from the `init()` function in manually curated files in this directory.
var cmdOverrides []func(*cobra.Command)

// Spec declares the command for this service.
var Spec = root.CommandSpec{
	Use:     "genie-agents",
	Short:   `Genie provides a no-code experience for business users, powered by AI/BI.`,
	GroupID: "dashboards",

	// This service is being previewed; hide from help output.
	Hidden: true,
	Build:  buildCommand,
}

func buildCommand(ctx context.Context, cmd *cobra.Command) {
	cmd.Long = `Genie provides a no-code experience for business users, powered by AI/BI.
  Analysts set up Genie Agents that business users can use to ask questions
  using natural language. Genie uses data registered to Unity Catalog and
  requires at least CAN USE permission on a Pro or Serverless SQL warehouse.
  Also, Databricks Assistant must be enabled.`
	cmd.RunE = root.ReportUnknownSubcommand

	cmd.Annotations = map[string]string{
		"launch_stage":         "PRIVATE_PREVIEW",
		"launch_stage_display": "Private Preview",
	}

	// Add methods
	cmd.AddCommand(newCreateGenieAgent())
	cmd.AddCommand(newDeleteGenieAgent())
	cmd.AddCommand(newGetGenieAgent())
	cmd.AddCommand(newListGenieAgents())
	cmd.AddCommand(newUpdateGenieAgent())

	// Apply optional overrides to this command.
	for _, fn := range cmdOverrides {
		fn(cmd)
	}
}

// start create-genie-agent command

// Slice with functions to override default command behavior.
// Functions can be added from the `init()` function in manually curated files in this directory.
var createGenieAgentOverrides []func(
	*cobra.Command,
	*genieagents.CreateGenieAgentRequest,
)

func newCreateGenieAgent() *cobra.Command {
	cmd := &cobra.Command{}

	var createGenieAgentReq genieagents.CreateGenieAgentRequest
	createGenieAgentReq.GenieAgent = genieagents.GenieAgent{}
	var createGenieAgentJson flags.JsonFlag

	cmd.Flags().Var(&createGenieAgentJson, "json", `either inline JSON string or @path/to/file.json with request body`)

	cmd.Flags().StringVar(&createGenieAgentReq.GenieAgentId, "genie-agent-id", createGenieAgentReq.GenieAgentId, `The ID for the new Genie Agent: 32 lowercase hexadecimal characters (a UUID without dashes), not all zeros.`)
	cmd.Flags().StringVar(&createGenieAgentReq.GenieAgent.Description, "description", createGenieAgentReq.GenieAgent.Description, `Description of the Genie Agent.`)
	cmd.Flags().StringVar(&createGenieAgentReq.GenieAgent.Name, "name", createGenieAgentReq.GenieAgent.Name, `The resource name of the Genie Agent, in the form agents/{genie_agent_id}.`)
	cmd.Flags().StringVar(&createGenieAgentReq.GenieAgent.ParentPath, "parent-path", createGenieAgentReq.GenieAgent.ParentPath, `The workspace folder to create the Genie Agent in.`)

	cmd.Use = "create-genie-agent DISPLAY_NAME DEFINITION WAREHOUSE_ID"
	cmd.Short = `Create Genie Agent.`
	cmd.Long = `Create Genie Agent.

  Creates a Genie Agent.

  Arguments:
    DISPLAY_NAME: Display name of the Genie Agent. Must be unique within its folder
      (effective_parent_path): a create, rename, or move that would collide with
      an existing Genie Agent's name in the target folder is rejected with
      ALREADY_EXISTS.

      The maximum length is 63 characters. Requests with a longer value are
      rejected with INVALID_PARAMETER_VALUE.
    DEFINITION: The definition of the Genie Agent: its sample questions, data sources,
      instructions, and benchmarks. Must set at least one field. Returned on get
      and on list with GENIE_AGENT_VIEW_FULL. Create and update return a
      submitted definition unchanged; one the service would store differently,
      or with a column config the workspace can't keep (unknown column, or a
      display name while those are disabled), is rejected with
      INVALID_PARAMETER_VALUE.
    WAREHOUSE_ID: Warehouse associated with the Genie Agent.`

	cmd.Annotations = make(map[string]string)
	cmd.Annotations["launch_stage"] = "PRIVATE_PREVIEW"
	cmd.Annotations["launch_stage_display"] = "Private Preview"

	cmd.Args = func(cmd *cobra.Command, args []string) error {
		if cmd.Flags().Changed("json") {
			err := root.ExactArgs(0)(cmd, args)
			if err != nil {
				return errors.New("when --json flag is specified, no positional arguments are allowed. Provide 'display_name', 'definition', 'warehouse_id' in your JSON input")
			}
			return nil
		}
		check := root.ExactArgs(3)
		return check(cmd, args)
	}

	cmd.PreRunE = root.MustWorkspaceClient
	cmd.RunE = func(cmd *cobra.Command, args []string) (err error) {
		ctx := cmd.Context()
		w := cmdctx.WorkspaceClient(ctx)

		if cmd.Flags().Changed("json") {
			diags := createGenieAgentJson.Unmarshal(&createGenieAgentReq.GenieAgent)
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
			createGenieAgentReq.GenieAgent.DisplayName = args[0]
		}
		if !cmd.Flags().Changed("json") {
			_, err = fmt.Sscan(args[1], &createGenieAgentReq.GenieAgent.Definition)
			if err != nil {
				return fmt.Errorf("invalid DEFINITION: %s", args[1])
			}

		}
		if !cmd.Flags().Changed("json") {
			createGenieAgentReq.GenieAgent.WarehouseId = args[2]
		}

		response, err := w.GenieAgents.CreateGenieAgent(ctx, createGenieAgentReq)
		if err != nil {
			return err
		}

		return cmdio.Render(ctx, response)
	}

	// Disable completions since they are not applicable.
	// Can be overridden by manual implementation in `override.go`.
	cmd.ValidArgsFunction = cobra.NoFileCompletions

	// Apply optional overrides to this command.
	for _, fn := range createGenieAgentOverrides {
		fn(cmd, &createGenieAgentReq)
	}

	return cmd
}

// start delete-genie-agent command

// Slice with functions to override default command behavior.
// Functions can be added from the `init()` function in manually curated files in this directory.
var deleteGenieAgentOverrides []func(
	*cobra.Command,
	*genieagents.DeleteGenieAgentRequest,
)

func newDeleteGenieAgent() *cobra.Command {
	cmd := &cobra.Command{}

	var deleteGenieAgentReq genieagents.DeleteGenieAgentRequest

	cmd.Flags().StringVar(&deleteGenieAgentReq.Etag, "etag", deleteGenieAgentReq.Etag, `Etag from a previous read.`)

	cmd.Use = "delete-genie-agent NAME"
	cmd.Short = `Delete Genie Agent.`
	cmd.Long = `Delete Genie Agent.

  Deletes a Genie Agent, moving it to trash.

  Arguments:
    NAME: The resource name of the Genie Agent to delete, in the form
      agents/{genie_agent_id}.`

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

		deleteGenieAgentReq.Name = args[0]

		err = w.GenieAgents.DeleteGenieAgent(ctx, deleteGenieAgentReq)
		if err != nil {
			return err
		}
		return nil
	}

	// Disable completions since they are not applicable.
	// Can be overridden by manual implementation in `override.go`.
	cmd.ValidArgsFunction = cobra.NoFileCompletions

	// Apply optional overrides to this command.
	for _, fn := range deleteGenieAgentOverrides {
		fn(cmd, &deleteGenieAgentReq)
	}

	return cmd
}

// start get-genie-agent command

// Slice with functions to override default command behavior.
// Functions can be added from the `init()` function in manually curated files in this directory.
var getGenieAgentOverrides []func(
	*cobra.Command,
	*genieagents.GetGenieAgentRequest,
)

func newGetGenieAgent() *cobra.Command {
	cmd := &cobra.Command{}

	var getGenieAgentReq genieagents.GetGenieAgentRequest

	cmd.Use = "get-genie-agent NAME"
	cmd.Short = `Get Genie Agent.`
	cmd.Long = `Get Genie Agent.

  Gets a Genie Agent.

  Arguments:
    NAME: The resource name of the Genie Agent to retrieve, in the form
      agents/{genie_agent_id}.`

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

		getGenieAgentReq.Name = args[0]

		response, err := w.GenieAgents.GetGenieAgent(ctx, getGenieAgentReq)
		if err != nil {
			return err
		}

		return cmdio.Render(ctx, response)
	}

	// Disable completions since they are not applicable.
	// Can be overridden by manual implementation in `override.go`.
	cmd.ValidArgsFunction = cobra.NoFileCompletions

	// Apply optional overrides to this command.
	for _, fn := range getGenieAgentOverrides {
		fn(cmd, &getGenieAgentReq)
	}

	return cmd
}

// start list-genie-agents command

// Slice with functions to override default command behavior.
// Functions can be added from the `init()` function in manually curated files in this directory.
var listGenieAgentsOverrides []func(
	*cobra.Command,
	*genieagents.ListGenieAgentsRequest,
)

func newListGenieAgents() *cobra.Command {
	cmd := &cobra.Command{}

	var listGenieAgentsReq genieagents.ListGenieAgentsRequest
	// Registered for all paginated methods. Validated at call time in the
	// method-call template. Paginated list methods never have Wait or LRO
	// branches, so the method-call path is always reached.
	var listGenieAgentsLimit int

	cmd.Flags().IntVar(&listGenieAgentsReq.PageSize, "page-size", listGenieAgentsReq.PageSize, `Maximum number of Genie Agents to return per page.`)
	cmd.Flags().Var(&listGenieAgentsReq.View, "view", `The fields to return for each Genie Agent. Supported values: [GENIE_AGENT_VIEW_BASIC, GENIE_AGENT_VIEW_FULL]`)

	// Limit flag for total result capping.
	cmd.Flags().IntVar(&listGenieAgentsLimit, "limit", 0, `Maximum number of results to return.`)

	// Hidden pagination flags (internal API parameters).
	cmd.Flags().StringVar(&listGenieAgentsReq.PageToken, "page-token", listGenieAgentsReq.PageToken, `Pagination token.`)
	cmd.Flags().Lookup("page-token").Hidden = true

	cmd.Use = "list-genie-agents"
	cmd.Short = `List Genie Agents.`
	cmd.Long = `List Genie Agents.

  Lists the Genie Agents in the workspace. Genie Agents created from an
  integration (for example, from a dashboard) are not listed.`

	cmd.Annotations = make(map[string]string)
	cmd.Annotations["launch_stage"] = "PRIVATE_PREVIEW"
	cmd.Annotations["launch_stage_display"] = "Private Preview"

	cmd.Args = func(cmd *cobra.Command, args []string) error {
		check := root.ExactArgs(0)
		return check(cmd, args)
	}

	cmd.PreRunE = root.MustWorkspaceClient
	cmd.RunE = func(cmd *cobra.Command, args []string) (err error) {
		ctx := cmd.Context()
		w := cmdctx.WorkspaceClient(ctx)

		response := w.GenieAgents.ListGenieAgents(ctx, listGenieAgentsReq)
		if listGenieAgentsLimit < 0 {
			return fmt.Errorf("--limit must be a non-negative integer, got %d", listGenieAgentsLimit)
		}
		if listGenieAgentsLimit > 0 {
			ctx = cmdio.WithLimit(ctx, listGenieAgentsLimit)
		}

		return cmdio.RenderIterator(ctx, response)
	}

	// Disable completions since they are not applicable.
	// Can be overridden by manual implementation in `override.go`.
	cmd.ValidArgsFunction = cobra.NoFileCompletions

	// Apply optional overrides to this command.
	for _, fn := range listGenieAgentsOverrides {
		fn(cmd, &listGenieAgentsReq)
	}

	return cmd
}

// start update-genie-agent command

// Slice with functions to override default command behavior.
// Functions can be added from the `init()` function in manually curated files in this directory.
var updateGenieAgentOverrides []func(
	*cobra.Command,
	*genieagents.UpdateGenieAgentRequest,
)

func newUpdateGenieAgent() *cobra.Command {
	cmd := &cobra.Command{}

	var updateGenieAgentReq genieagents.UpdateGenieAgentRequest
	updateGenieAgentReq.GenieAgent = genieagents.GenieAgent{}
	var updateGenieAgentJson flags.JsonFlag

	cmd.Flags().Var(&updateGenieAgentJson, "json", `either inline JSON string or @path/to/file.json with request body`)

	cmd.Flags().StringVar(&updateGenieAgentReq.Etag, "etag", updateGenieAgentReq.Etag, `Etag from a previous read.`)
	cmd.Flags().StringVar(&updateGenieAgentReq.GenieAgent.Description, "description", updateGenieAgentReq.GenieAgent.Description, `Description of the Genie Agent.`)
	cmd.Flags().StringVar(&updateGenieAgentReq.GenieAgent.Name, "name", updateGenieAgentReq.GenieAgent.Name, `The resource name of the Genie Agent, in the form agents/{genie_agent_id}.`)
	cmd.Flags().StringVar(&updateGenieAgentReq.GenieAgent.ParentPath, "parent-path", updateGenieAgentReq.GenieAgent.ParentPath, `The workspace folder to create the Genie Agent in.`)

	cmd.Use = "update-genie-agent NAME UPDATE_MASK DISPLAY_NAME DEFINITION WAREHOUSE_ID"
	cmd.Short = `Update Genie Agent.`
	cmd.Long = `Update Genie Agent.

  Updates a Genie Agent. Only the fields named in update_mask are applied.

  Arguments:
    NAME: The resource name of the Genie Agent, in the form
      agents/{genie_agent_id}. Ignored on create.
    UPDATE_MASK: The set of fields to update. The mask is required and must name at least
      one field; an empty or absent mask is rejected with
      INVALID_PARAMETER_VALUE, as is a path that is not an updatable field.

      A field named in the mask takes the value from genie_agent, even when it
      is unset there: an unset description is cleared; an unset
      display_name, warehouse_id, or definition is rejected with
      INVALID_PARAMETER_VALUE; an unset parent_path is cleared without moving
      the Genie Agent. definition is replaced as a whole; paths within it are
      not supported.

      The special value * replaces the whole Genie Agent, as if every
      updatable field were named. It must be the only path in the mask. Prefer
      naming fields explicitly: with *, any field added to Genie Agent later
      is also replaced, so a client unaware of it may clear it.
    DISPLAY_NAME: Display name of the Genie Agent. Must be unique within its folder
      (effective_parent_path): a create, rename, or move that would collide with
      an existing Genie Agent's name in the target folder is rejected with
      ALREADY_EXISTS.

      The maximum length is 63 characters. Requests with a longer value are
      rejected with INVALID_PARAMETER_VALUE.
    DEFINITION: The definition of the Genie Agent: its sample questions, data sources,
      instructions, and benchmarks. Must set at least one field. Returned on get
      and on list with GENIE_AGENT_VIEW_FULL. Create and update return a
      submitted definition unchanged; one the service would store differently,
      or with a column config the workspace can't keep (unknown column, or a
      display name while those are disabled), is rejected with
      INVALID_PARAMETER_VALUE.
    WAREHOUSE_ID: Warehouse associated with the Genie Agent.`

	cmd.Annotations = make(map[string]string)
	cmd.Annotations["launch_stage"] = "PRIVATE_PREVIEW"
	cmd.Annotations["launch_stage_display"] = "Private Preview"

	cmd.Args = func(cmd *cobra.Command, args []string) error {
		if cmd.Flags().Changed("json") {
			err := root.ExactArgs(2)(cmd, args)
			if err != nil {
				return errors.New("when --json flag is specified, provide only NAME, UPDATE_MASK as positional arguments. Provide 'display_name', 'definition', 'warehouse_id' in your JSON input")
			}
			return nil
		}
		check := root.ExactArgs(5)
		return check(cmd, args)
	}

	cmd.PreRunE = root.MustWorkspaceClient
	cmd.RunE = func(cmd *cobra.Command, args []string) (err error) {
		ctx := cmd.Context()
		w := cmdctx.WorkspaceClient(ctx)

		if cmd.Flags().Changed("json") {
			diags := updateGenieAgentJson.Unmarshal(&updateGenieAgentReq.GenieAgent)
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
		updateGenieAgentReq.Name = args[0]
		if args[1] != "" {
			updateMaskArray := strings.Split(args[1], ",")
			updateGenieAgentReq.UpdateMask = *fieldmask.New(updateMaskArray)
		}
		if !cmd.Flags().Changed("json") {
			updateGenieAgentReq.GenieAgent.DisplayName = args[2]
		}
		if !cmd.Flags().Changed("json") {
			_, err = fmt.Sscan(args[3], &updateGenieAgentReq.GenieAgent.Definition)
			if err != nil {
				return fmt.Errorf("invalid DEFINITION: %s", args[3])
			}

		}
		if !cmd.Flags().Changed("json") {
			updateGenieAgentReq.GenieAgent.WarehouseId = args[4]
		}

		response, err := w.GenieAgents.UpdateGenieAgent(ctx, updateGenieAgentReq)
		if err != nil {
			return err
		}

		return cmdio.Render(ctx, response)
	}

	// Disable completions since they are not applicable.
	// Can be overridden by manual implementation in `override.go`.
	cmd.ValidArgsFunction = cobra.NoFileCompletions

	// Apply optional overrides to this command.
	for _, fn := range updateGenieAgentOverrides {
		fn(cmd, &updateGenieAgentReq)
	}

	return cmd
}

// end service GenieAgents
