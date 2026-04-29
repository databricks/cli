// Code generated from OpenAPI specs by Databricks SDK Generator. DO NOT EDIT.

package supervisor_agents

import (
	"fmt"
	"strings"

	"github.com/databricks/cli/cmd/root"
	"github.com/databricks/cli/libs/cmdctx"
	"github.com/databricks/cli/libs/cmdio"
	"github.com/databricks/cli/libs/flags"
	"github.com/databricks/databricks-sdk-go/common/types/fieldmask"
	"github.com/databricks/databricks-sdk-go/service/supervisoragents"
	"github.com/spf13/cobra"
)

// Slice with functions to override default command behavior.
// Functions can be added from the `init()` function in manually curated files in this directory.
var cmdOverrides []func(*cobra.Command)

func New() *cobra.Command {
	cmd := &cobra.Command{
		Use:     "supervisor-agents",
		Short:   `Manage Supervisor Agents and related resources.`,
		Long:    `Manage Supervisor Agents and related resources.`,
		GroupID: "agentbricks",
		RunE:    root.ReportUnknownSubcommand,
	}

	// Add methods
	cmd.AddCommand(newCreateSupervisorAgent())
	cmd.AddCommand(newCreateTool())
	cmd.AddCommand(newDeleteSupervisorAgent())
	cmd.AddCommand(newDeleteTool())
	cmd.AddCommand(newGetSupervisorAgent())
	cmd.AddCommand(newGetTool())
	cmd.AddCommand(newListSupervisorAgents())
	cmd.AddCommand(newListTools())
	cmd.AddCommand(newUpdateSupervisorAgent())
	cmd.AddCommand(newUpdateTool())

	// Apply optional overrides to this command.
	for _, fn := range cmdOverrides {
		fn(cmd)
	}

	return cmd
}

// start create-supervisor-agent command

// Slice with functions to override default command behavior.
// Functions can be added from the `init()` function in manually curated files in this directory.
var createSupervisorAgentOverrides []func(
	*cobra.Command,
	*supervisoragents.CreateSupervisorAgentRequest,
)

func newCreateSupervisorAgent() *cobra.Command {
	cmd := &cobra.Command{}

	var createSupervisorAgentReq supervisoragents.CreateSupervisorAgentRequest
	createSupervisorAgentReq.SupervisorAgent = supervisoragents.SupervisorAgent{}
	var createSupervisorAgentJson flags.JsonFlag

	cmd.Flags().Var(&createSupervisorAgentJson, "json", `either inline JSON string or @path/to/file.json with request body`)

	cmd.Flags().StringVar(&createSupervisorAgentReq.SupervisorAgent.Instructions, "instructions", createSupervisorAgentReq.SupervisorAgent.Instructions, `Optional natural-language instructions for the supervisor agent.`)
	cmd.Flags().StringVar(&createSupervisorAgentReq.SupervisorAgent.Name, "name", createSupervisorAgentReq.SupervisorAgent.Name, `The resource name of the SupervisorAgent.`)

	cmd.Use = "create-supervisor-agent DISPLAY_NAME DESCRIPTION"
	cmd.Short = `Create a Supervisor Agent.`
	cmd.Long = `Create a Supervisor Agent.
  
  Creates a new Supervisor Agent.

  Arguments:
    DISPLAY_NAME: The display name of the Supervisor Agent, unique at workspace level.
    DESCRIPTION: Description of what this agent can do (user-facing).`

	cmd.Annotations = make(map[string]string)

	cmd.Args = func(cmd *cobra.Command, args []string) error {
		if cmd.Flags().Changed("json") {
			err := root.ExactArgs(0)(cmd, args)
			if err != nil {
				return fmt.Errorf("when --json flag is specified, no positional arguments are allowed. Provide 'display_name', 'description' in your JSON input")
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
			diags := createSupervisorAgentJson.Unmarshal(&createSupervisorAgentReq.SupervisorAgent)
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
			createSupervisorAgentReq.SupervisorAgent.DisplayName = args[0]
		}
		if !cmd.Flags().Changed("json") {
			createSupervisorAgentReq.SupervisorAgent.Description = args[1]
		}

		response, err := w.SupervisorAgents.CreateSupervisorAgent(ctx, createSupervisorAgentReq)
		if err != nil {
			return err
		}

		return cmdio.Render(ctx, response)
	}

	// Disable completions since they are not applicable.
	// Can be overridden by manual implementation in `override.go`.
	cmd.ValidArgsFunction = cobra.NoFileCompletions

	// Apply optional overrides to this command.
	for _, fn := range createSupervisorAgentOverrides {
		fn(cmd, &createSupervisorAgentReq)
	}

	return cmd
}

// start create-tool command

// Slice with functions to override default command behavior.
// Functions can be added from the `init()` function in manually curated files in this directory.
var createToolOverrides []func(
	*cobra.Command,
	*supervisoragents.CreateToolRequest,
)

func newCreateTool() *cobra.Command {
	cmd := &cobra.Command{}

	var createToolReq supervisoragents.CreateToolRequest
	createToolReq.Tool = supervisoragents.Tool{}
	var createToolJson flags.JsonFlag

	cmd.Flags().Var(&createToolJson, "json", `either inline JSON string or @path/to/file.json with request body`)

	// TODO: complex arg: app
	// TODO: complex arg: genie_space
	// TODO: complex arg: knowledge_assistant
	cmd.Flags().StringVar(&createToolReq.Tool.Name, "name", createToolReq.Tool.Name, `Full resource name: supervisor-agents/{supervisor_agent_id}/tools/{tool_id}.`)
	// TODO: complex arg: uc_connection
	// TODO: complex arg: uc_function
	// TODO: complex arg: volume

	cmd.Use = "create-tool PARENT TOOL_ID TOOL_TYPE DESCRIPTION"
	cmd.Short = `Create a Tool.`
	cmd.Long = `Create a Tool.
  
  Creates a Tool under a Supervisor Agent. Specify one of "genie_space",
  "knowledge_assistant", "uc_function", "uc_connection", "app", "volume",
  "lakeview_dashboard", "uc_table", "vector_search_index" in the request body.

  Arguments:
    PARENT: Parent resource where this tool will be created. Format:
      supervisor-agents/{supervisor_agent_id}
    TOOL_ID: The ID to use for the tool, which will become the final component of the
      tool's resource name.
    TOOL_TYPE: Tool type. Must be one of: "genie_space", "knowledge_assistant",
      "uc_function", "uc_connection", "app", "volume", "lakeview_dashboard",
      "serving_endpoint", "uc_table", "vector_search_index".
    DESCRIPTION: Description of what this tool does (user-facing).`

	cmd.Annotations = make(map[string]string)

	cmd.Args = func(cmd *cobra.Command, args []string) error {
		if cmd.Flags().Changed("json") {
			err := root.ExactArgs(2)(cmd, args)
			if err != nil {
				return fmt.Errorf("when --json flag is specified, provide only PARENT, TOOL_ID as positional arguments. Provide 'tool_type', 'description' in your JSON input")
			}
			return nil
		}
		check := root.ExactArgs(4)
		return check(cmd, args)
	}

	cmd.PreRunE = root.MustWorkspaceClient
	cmd.RunE = func(cmd *cobra.Command, args []string) (err error) {
		ctx := cmd.Context()
		w := cmdctx.WorkspaceClient(ctx)

		if cmd.Flags().Changed("json") {
			diags := createToolJson.Unmarshal(&createToolReq.Tool)
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
		createToolReq.Parent = args[0]
		createToolReq.ToolId = args[1]
		if !cmd.Flags().Changed("json") {
			createToolReq.Tool.ToolType = args[2]
		}
		if !cmd.Flags().Changed("json") {
			createToolReq.Tool.Description = args[3]
		}

		response, err := w.SupervisorAgents.CreateTool(ctx, createToolReq)
		if err != nil {
			return err
		}

		return cmdio.Render(ctx, response)
	}

	// Disable completions since they are not applicable.
	// Can be overridden by manual implementation in `override.go`.
	cmd.ValidArgsFunction = cobra.NoFileCompletions

	// Apply optional overrides to this command.
	for _, fn := range createToolOverrides {
		fn(cmd, &createToolReq)
	}

	return cmd
}

// start delete-supervisor-agent command

// Slice with functions to override default command behavior.
// Functions can be added from the `init()` function in manually curated files in this directory.
var deleteSupervisorAgentOverrides []func(
	*cobra.Command,
	*supervisoragents.DeleteSupervisorAgentRequest,
)

func newDeleteSupervisorAgent() *cobra.Command {
	cmd := &cobra.Command{}

	var deleteSupervisorAgentReq supervisoragents.DeleteSupervisorAgentRequest

	cmd.Use = "delete-supervisor-agent NAME"
	cmd.Short = `Delete a Supervisor Agent.`
	cmd.Long = `Delete a Supervisor Agent.
  
  Deletes a Supervisor Agent.

  Arguments:
    NAME: The resource name of the Supervisor Agent. Format:
      supervisor-agents/{supervisor_agent_id}`

	cmd.Annotations = make(map[string]string)

	cmd.Args = func(cmd *cobra.Command, args []string) error {
		check := root.ExactArgs(1)
		return check(cmd, args)
	}

	cmd.PreRunE = root.MustWorkspaceClient
	cmd.RunE = func(cmd *cobra.Command, args []string) (err error) {
		ctx := cmd.Context()
		w := cmdctx.WorkspaceClient(ctx)

		deleteSupervisorAgentReq.Name = args[0]

		err = w.SupervisorAgents.DeleteSupervisorAgent(ctx, deleteSupervisorAgentReq)
		if err != nil {
			return err
		}
		return nil
	}

	// Disable completions since they are not applicable.
	// Can be overridden by manual implementation in `override.go`.
	cmd.ValidArgsFunction = cobra.NoFileCompletions

	// Apply optional overrides to this command.
	for _, fn := range deleteSupervisorAgentOverrides {
		fn(cmd, &deleteSupervisorAgentReq)
	}

	return cmd
}

// start delete-tool command

// Slice with functions to override default command behavior.
// Functions can be added from the `init()` function in manually curated files in this directory.
var deleteToolOverrides []func(
	*cobra.Command,
	*supervisoragents.DeleteToolRequest,
)

func newDeleteTool() *cobra.Command {
	cmd := &cobra.Command{}

	var deleteToolReq supervisoragents.DeleteToolRequest

	cmd.Use = "delete-tool NAME"
	cmd.Short = `Delete a Tool.`
	cmd.Long = `Delete a Tool.
  
  Deletes a Tool.

  Arguments:
    NAME: The resource name of the Tool. Format:
      supervisor-agents/{supervisor_agent_id}/tools/{tool_id}`

	cmd.Annotations = make(map[string]string)

	cmd.Args = func(cmd *cobra.Command, args []string) error {
		check := root.ExactArgs(1)
		return check(cmd, args)
	}

	cmd.PreRunE = root.MustWorkspaceClient
	cmd.RunE = func(cmd *cobra.Command, args []string) (err error) {
		ctx := cmd.Context()
		w := cmdctx.WorkspaceClient(ctx)

		deleteToolReq.Name = args[0]

		err = w.SupervisorAgents.DeleteTool(ctx, deleteToolReq)
		if err != nil {
			return err
		}
		return nil
	}

	// Disable completions since they are not applicable.
	// Can be overridden by manual implementation in `override.go`.
	cmd.ValidArgsFunction = cobra.NoFileCompletions

	// Apply optional overrides to this command.
	for _, fn := range deleteToolOverrides {
		fn(cmd, &deleteToolReq)
	}

	return cmd
}

// start get-supervisor-agent command

// Slice with functions to override default command behavior.
// Functions can be added from the `init()` function in manually curated files in this directory.
var getSupervisorAgentOverrides []func(
	*cobra.Command,
	*supervisoragents.GetSupervisorAgentRequest,
)

func newGetSupervisorAgent() *cobra.Command {
	cmd := &cobra.Command{}

	var getSupervisorAgentReq supervisoragents.GetSupervisorAgentRequest

	cmd.Use = "get-supervisor-agent NAME"
	cmd.Short = `Get a Supervisor Agent.`
	cmd.Long = `Get a Supervisor Agent.
  
  Gets a Supervisor Agent.

  Arguments:
    NAME: The resource name of the Supervisor Agent. Format:
      supervisor-agents/{supervisor_agent_id}`

	cmd.Annotations = make(map[string]string)

	cmd.Args = func(cmd *cobra.Command, args []string) error {
		check := root.ExactArgs(1)
		return check(cmd, args)
	}

	cmd.PreRunE = root.MustWorkspaceClient
	cmd.RunE = func(cmd *cobra.Command, args []string) (err error) {
		ctx := cmd.Context()
		w := cmdctx.WorkspaceClient(ctx)

		getSupervisorAgentReq.Name = args[0]

		response, err := w.SupervisorAgents.GetSupervisorAgent(ctx, getSupervisorAgentReq)
		if err != nil {
			return err
		}

		return cmdio.Render(ctx, response)
	}

	// Disable completions since they are not applicable.
	// Can be overridden by manual implementation in `override.go`.
	cmd.ValidArgsFunction = cobra.NoFileCompletions

	// Apply optional overrides to this command.
	for _, fn := range getSupervisorAgentOverrides {
		fn(cmd, &getSupervisorAgentReq)
	}

	return cmd
}

// start get-tool command

// Slice with functions to override default command behavior.
// Functions can be added from the `init()` function in manually curated files in this directory.
var getToolOverrides []func(
	*cobra.Command,
	*supervisoragents.GetToolRequest,
)

func newGetTool() *cobra.Command {
	cmd := &cobra.Command{}

	var getToolReq supervisoragents.GetToolRequest

	cmd.Use = "get-tool NAME"
	cmd.Short = `Get a Tool.`
	cmd.Long = `Get a Tool.
  
  Gets a Tool.

  Arguments:
    NAME: The resource name of the Tool. Format:
      supervisor-agents/{supervisor_agent_id}/tools/{tool_id}`

	cmd.Annotations = make(map[string]string)

	cmd.Args = func(cmd *cobra.Command, args []string) error {
		check := root.ExactArgs(1)
		return check(cmd, args)
	}

	cmd.PreRunE = root.MustWorkspaceClient
	cmd.RunE = func(cmd *cobra.Command, args []string) (err error) {
		ctx := cmd.Context()
		w := cmdctx.WorkspaceClient(ctx)

		getToolReq.Name = args[0]

		response, err := w.SupervisorAgents.GetTool(ctx, getToolReq)
		if err != nil {
			return err
		}

		return cmdio.Render(ctx, response)
	}

	// Disable completions since they are not applicable.
	// Can be overridden by manual implementation in `override.go`.
	cmd.ValidArgsFunction = cobra.NoFileCompletions

	// Apply optional overrides to this command.
	for _, fn := range getToolOverrides {
		fn(cmd, &getToolReq)
	}

	return cmd
}

// start list-supervisor-agents command

// Slice with functions to override default command behavior.
// Functions can be added from the `init()` function in manually curated files in this directory.
var listSupervisorAgentsOverrides []func(
	*cobra.Command,
	*supervisoragents.ListSupervisorAgentsRequest,
)

func newListSupervisorAgents() *cobra.Command {
	cmd := &cobra.Command{}

	var listSupervisorAgentsReq supervisoragents.ListSupervisorAgentsRequest
	// Registered for all paginated methods. Validated at call time in the
	// method-call template. Paginated list methods never have Wait or LRO
	// branches, so the method-call path is always reached.
	var listSupervisorAgentsLimit int

	cmd.Flags().IntVar(&listSupervisorAgentsReq.PageSize, "page-size", listSupervisorAgentsReq.PageSize, `The maximum number of supervisor agents to return.`)

	// Limit flag for total result capping.
	cmd.Flags().IntVar(&listSupervisorAgentsLimit, "limit", 0, `Maximum number of results to return.`)

	// Hidden pagination flags (internal API parameters).
	cmd.Flags().StringVar(&listSupervisorAgentsReq.PageToken, "page-token", listSupervisorAgentsReq.PageToken, `Pagination token.`)
	cmd.Flags().Lookup("page-token").Hidden = true

	cmd.Use = "list-supervisor-agents"
	cmd.Short = `List Supervisor Agents.`
	cmd.Long = `List Supervisor Agents.
  
  Lists Supervisor Agents.`

	cmd.Annotations = make(map[string]string)

	cmd.Args = func(cmd *cobra.Command, args []string) error {
		check := root.ExactArgs(0)
		return check(cmd, args)
	}

	cmd.PreRunE = root.MustWorkspaceClient
	cmd.RunE = func(cmd *cobra.Command, args []string) (err error) {
		ctx := cmd.Context()
		w := cmdctx.WorkspaceClient(ctx)

		response := w.SupervisorAgents.ListSupervisorAgents(ctx, listSupervisorAgentsReq)
		if listSupervisorAgentsLimit < 0 {
			return fmt.Errorf("--limit must be a non-negative integer, got %d", listSupervisorAgentsLimit)
		}
		if listSupervisorAgentsLimit > 0 {
			ctx = cmdio.WithLimit(ctx, listSupervisorAgentsLimit)
		}

		return cmdio.RenderIterator(ctx, response)
	}

	// Disable completions since they are not applicable.
	// Can be overridden by manual implementation in `override.go`.
	cmd.ValidArgsFunction = cobra.NoFileCompletions

	// Apply optional overrides to this command.
	for _, fn := range listSupervisorAgentsOverrides {
		fn(cmd, &listSupervisorAgentsReq)
	}

	return cmd
}

// start list-tools command

// Slice with functions to override default command behavior.
// Functions can be added from the `init()` function in manually curated files in this directory.
var listToolsOverrides []func(
	*cobra.Command,
	*supervisoragents.ListToolsRequest,
)

func newListTools() *cobra.Command {
	cmd := &cobra.Command{}

	var listToolsReq supervisoragents.ListToolsRequest
	// Registered for all paginated methods. Validated at call time in the
	// method-call template. Paginated list methods never have Wait or LRO
	// branches, so the method-call path is always reached.
	var listToolsLimit int

	cmd.Flags().IntVar(&listToolsReq.PageSize, "page-size", listToolsReq.PageSize, ``)

	// Limit flag for total result capping.
	cmd.Flags().IntVar(&listToolsLimit, "limit", 0, `Maximum number of results to return.`)

	// Hidden pagination flags (internal API parameters).
	cmd.Flags().StringVar(&listToolsReq.PageToken, "page-token", listToolsReq.PageToken, `Pagination token.`)
	cmd.Flags().Lookup("page-token").Hidden = true

	cmd.Use = "list-tools PARENT"
	cmd.Short = `List Tools.`
	cmd.Long = `List Tools.
  
  Lists Tools under a Supervisor Agent.

  Arguments:
    PARENT: Parent resource to list from. Format:
      supervisor-agents/{supervisor_agent_id}`

	cmd.Annotations = make(map[string]string)

	cmd.Args = func(cmd *cobra.Command, args []string) error {
		check := root.ExactArgs(1)
		return check(cmd, args)
	}

	cmd.PreRunE = root.MustWorkspaceClient
	cmd.RunE = func(cmd *cobra.Command, args []string) (err error) {
		ctx := cmd.Context()
		w := cmdctx.WorkspaceClient(ctx)

		listToolsReq.Parent = args[0]

		response := w.SupervisorAgents.ListTools(ctx, listToolsReq)
		if listToolsLimit < 0 {
			return fmt.Errorf("--limit must be a non-negative integer, got %d", listToolsLimit)
		}
		if listToolsLimit > 0 {
			ctx = cmdio.WithLimit(ctx, listToolsLimit)
		}

		return cmdio.RenderIterator(ctx, response)
	}

	// Disable completions since they are not applicable.
	// Can be overridden by manual implementation in `override.go`.
	cmd.ValidArgsFunction = cobra.NoFileCompletions

	// Apply optional overrides to this command.
	for _, fn := range listToolsOverrides {
		fn(cmd, &listToolsReq)
	}

	return cmd
}

// start update-supervisor-agent command

// Slice with functions to override default command behavior.
// Functions can be added from the `init()` function in manually curated files in this directory.
var updateSupervisorAgentOverrides []func(
	*cobra.Command,
	*supervisoragents.UpdateSupervisorAgentRequest,
)

func newUpdateSupervisorAgent() *cobra.Command {
	cmd := &cobra.Command{}

	var updateSupervisorAgentReq supervisoragents.UpdateSupervisorAgentRequest
	updateSupervisorAgentReq.SupervisorAgent = supervisoragents.SupervisorAgent{}
	var updateSupervisorAgentJson flags.JsonFlag

	cmd.Flags().Var(&updateSupervisorAgentJson, "json", `either inline JSON string or @path/to/file.json with request body`)

	cmd.Flags().StringVar(&updateSupervisorAgentReq.SupervisorAgent.Instructions, "instructions", updateSupervisorAgentReq.SupervisorAgent.Instructions, `Optional natural-language instructions for the supervisor agent.`)
	cmd.Flags().StringVar(&updateSupervisorAgentReq.SupervisorAgent.Name, "name", updateSupervisorAgentReq.SupervisorAgent.Name, `The resource name of the SupervisorAgent.`)

	cmd.Use = "update-supervisor-agent NAME UPDATE_MASK DISPLAY_NAME DESCRIPTION"
	cmd.Short = `Update a Supervisor Agent.`
	cmd.Long = `Update a Supervisor Agent.
  
  Updates a Supervisor Agent. The fields that are required depend on the paths
  specified in update_mask. Only fields included in the mask will be updated.

  Arguments:
    NAME: The resource name of the SupervisorAgent. Format:
      supervisor-agents/{supervisor_agent_id}
    UPDATE_MASK: Field mask for fields to be updated.
    DISPLAY_NAME: The display name of the Supervisor Agent, unique at workspace level.
    DESCRIPTION: Description of what this agent can do (user-facing).`

	cmd.Annotations = make(map[string]string)

	cmd.Args = func(cmd *cobra.Command, args []string) error {
		if cmd.Flags().Changed("json") {
			err := root.ExactArgs(2)(cmd, args)
			if err != nil {
				return fmt.Errorf("when --json flag is specified, provide only NAME, UPDATE_MASK as positional arguments. Provide 'display_name', 'description' in your JSON input")
			}
			return nil
		}
		check := root.ExactArgs(4)
		return check(cmd, args)
	}

	cmd.PreRunE = root.MustWorkspaceClient
	cmd.RunE = func(cmd *cobra.Command, args []string) (err error) {
		ctx := cmd.Context()
		w := cmdctx.WorkspaceClient(ctx)

		if cmd.Flags().Changed("json") {
			diags := updateSupervisorAgentJson.Unmarshal(&updateSupervisorAgentReq.SupervisorAgent)
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
		updateSupervisorAgentReq.Name = args[0]
		if args[1] != "" {
			updateMaskArray := strings.Split(args[1], ",")
			updateSupervisorAgentReq.UpdateMask = *fieldmask.New(updateMaskArray)
		}
		if !cmd.Flags().Changed("json") {
			updateSupervisorAgentReq.SupervisorAgent.DisplayName = args[2]
		}
		if !cmd.Flags().Changed("json") {
			updateSupervisorAgentReq.SupervisorAgent.Description = args[3]
		}

		response, err := w.SupervisorAgents.UpdateSupervisorAgent(ctx, updateSupervisorAgentReq)
		if err != nil {
			return err
		}

		return cmdio.Render(ctx, response)
	}

	// Disable completions since they are not applicable.
	// Can be overridden by manual implementation in `override.go`.
	cmd.ValidArgsFunction = cobra.NoFileCompletions

	// Apply optional overrides to this command.
	for _, fn := range updateSupervisorAgentOverrides {
		fn(cmd, &updateSupervisorAgentReq)
	}

	return cmd
}

// start update-tool command

// Slice with functions to override default command behavior.
// Functions can be added from the `init()` function in manually curated files in this directory.
var updateToolOverrides []func(
	*cobra.Command,
	*supervisoragents.UpdateToolRequest,
)

func newUpdateTool() *cobra.Command {
	cmd := &cobra.Command{}

	var updateToolReq supervisoragents.UpdateToolRequest
	updateToolReq.Tool = supervisoragents.Tool{}
	var updateToolJson flags.JsonFlag

	cmd.Flags().Var(&updateToolJson, "json", `either inline JSON string or @path/to/file.json with request body`)

	// TODO: complex arg: app
	// TODO: complex arg: genie_space
	// TODO: complex arg: knowledge_assistant
	cmd.Flags().StringVar(&updateToolReq.Tool.Name, "name", updateToolReq.Tool.Name, `Full resource name: supervisor-agents/{supervisor_agent_id}/tools/{tool_id}.`)
	// TODO: complex arg: uc_connection
	// TODO: complex arg: uc_function
	// TODO: complex arg: volume

	cmd.Use = "update-tool NAME UPDATE_MASK TOOL_TYPE DESCRIPTION"
	cmd.Short = `Update a Tool.`
	cmd.Long = `Update a Tool.
  
  Updates a Tool. Only the description field can be updated. To change
  immutable fields such as tool type, spec, or tool ID, delete the tool and
  recreate it.

  Arguments:
    NAME: Full resource name:
      supervisor-agents/{supervisor_agent_id}/tools/{tool_id}
    UPDATE_MASK: Field mask for fields to be updated.
    TOOL_TYPE: Tool type. Must be one of: "genie_space", "knowledge_assistant",
      "uc_function", "uc_connection", "app", "volume", "lakeview_dashboard",
      "serving_endpoint", "uc_table", "vector_search_index".
    DESCRIPTION: Description of what this tool does (user-facing).`

	cmd.Annotations = make(map[string]string)

	cmd.Args = func(cmd *cobra.Command, args []string) error {
		if cmd.Flags().Changed("json") {
			err := root.ExactArgs(2)(cmd, args)
			if err != nil {
				return fmt.Errorf("when --json flag is specified, provide only NAME, UPDATE_MASK as positional arguments. Provide 'tool_type', 'description' in your JSON input")
			}
			return nil
		}
		check := root.ExactArgs(4)
		return check(cmd, args)
	}

	cmd.PreRunE = root.MustWorkspaceClient
	cmd.RunE = func(cmd *cobra.Command, args []string) (err error) {
		ctx := cmd.Context()
		w := cmdctx.WorkspaceClient(ctx)

		if cmd.Flags().Changed("json") {
			diags := updateToolJson.Unmarshal(&updateToolReq.Tool)
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
		updateToolReq.Name = args[0]
		if args[1] != "" {
			updateMaskArray := strings.Split(args[1], ",")
			updateToolReq.UpdateMask = *fieldmask.New(updateMaskArray)
		}
		if !cmd.Flags().Changed("json") {
			updateToolReq.Tool.ToolType = args[2]
		}
		if !cmd.Flags().Changed("json") {
			updateToolReq.Tool.Description = args[3]
		}

		response, err := w.SupervisorAgents.UpdateTool(ctx, updateToolReq)
		if err != nil {
			return err
		}

		return cmdio.Render(ctx, response)
	}

	// Disable completions since they are not applicable.
	// Can be overridden by manual implementation in `override.go`.
	cmd.ValidArgsFunction = cobra.NoFileCompletions

	// Apply optional overrides to this command.
	for _, fn := range updateToolOverrides {
		fn(cmd, &updateToolReq)
	}

	return cmd
}

// end service SupervisorAgents
