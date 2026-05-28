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
		Use:   "supervisor-agents",
		Short: `*Beta* Manage Supervisor Agents and related resources.`,
		Long: `This command is in Beta and may change without notice.

Manage Supervisor Agents and related resources.`,
		GroupID: "agentbricks",
		RunE:    root.ReportUnknownSubcommand,
	}

	cmd.Annotations = make(map[string]string)
	cmd.Annotations["launch_stage"] = "PUBLIC_BETA"
	cmd.Annotations["launch_stage_display"] = "Beta"

	// Add methods
	cmd.AddCommand(newCreateExample())
	cmd.AddCommand(newCreateSupervisorAgent())
	cmd.AddCommand(newCreateTool())
	cmd.AddCommand(newDeleteExample())
	cmd.AddCommand(newDeleteSupervisorAgent())
	cmd.AddCommand(newDeleteTool())
	cmd.AddCommand(newGetExample())
	cmd.AddCommand(newGetPermissionLevels())
	cmd.AddCommand(newGetPermissions())
	cmd.AddCommand(newGetSupervisorAgent())
	cmd.AddCommand(newGetTool())
	cmd.AddCommand(newListExamples())
	cmd.AddCommand(newListSupervisorAgents())
	cmd.AddCommand(newListTools())
	cmd.AddCommand(newSetPermissions())
	cmd.AddCommand(newUpdateExample())
	cmd.AddCommand(newUpdatePermissions())
	cmd.AddCommand(newUpdateSupervisorAgent())
	cmd.AddCommand(newUpdateTool())

	// Apply optional overrides to this command.
	for _, fn := range cmdOverrides {
		fn(cmd)
	}

	return cmd
}

// start create-example command

// Slice with functions to override default command behavior.
// Functions can be added from the `init()` function in manually curated files in this directory.
var createExampleOverrides []func(
	*cobra.Command,
	*supervisoragents.CreateExampleRequest,
)

func newCreateExample() *cobra.Command {
	cmd := &cobra.Command{}

	var createExampleReq supervisoragents.CreateExampleRequest
	createExampleReq.Example = supervisoragents.Example{}
	var createExampleJson flags.JsonFlag

	cmd.Flags().Var(&createExampleJson, "json", `either inline JSON string or @path/to/file.json with request body`)

	cmd.Flags().StringVar(&createExampleReq.Example.Name, "name", createExampleReq.Example.Name, `Full resource name: supervisor-agents/{supervisor_agent_id}/examples/{example_id}.`)

	cmd.Use = "create-example PARENT QUESTION GUIDELINES"
	cmd.Short = `*Beta* Create an example for a Supervisor Agent.`
	cmd.Long = `This command is in Beta and may change without notice.

Create an example for a Supervisor Agent.

  Creates an example for a Supervisor Agent.

  Arguments:
    PARENT: Parent resource where this example will be created. Format:
      supervisor-agents/{supervisor_agent_id}
    QUESTION: The example question.
    GUIDELINES: Guidelines for answering the question.`

	cmd.Annotations = make(map[string]string)
	cmd.Annotations["launch_stage"] = "PUBLIC_BETA"
	cmd.Annotations["launch_stage_display"] = "Beta"

	cmd.Args = func(cmd *cobra.Command, args []string) error {
		if cmd.Flags().Changed("json") {
			err := root.ExactArgs(1)(cmd, args)
			if err != nil {
				return fmt.Errorf("when --json flag is specified, provide only PARENT as positional arguments. Provide 'question', 'guidelines' in your JSON input")
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
			diags := createExampleJson.Unmarshal(&createExampleReq.Example)
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
		createExampleReq.Parent = args[0]
		if !cmd.Flags().Changed("json") {
			createExampleReq.Example.Question = args[1]
		}
		if !cmd.Flags().Changed("json") {
			_, err = fmt.Sscan(args[2], &createExampleReq.Example.Guidelines)
			if err != nil {
				return fmt.Errorf("invalid GUIDELINES: %s", args[2])
			}

		}

		response, err := w.SupervisorAgents.CreateExample(ctx, createExampleReq)
		if err != nil {
			return err
		}

		return cmdio.Render(ctx, response)
	}

	// Disable completions since they are not applicable.
	// Can be overridden by manual implementation in `override.go`.
	cmd.ValidArgsFunction = cobra.NoFileCompletions

	// Apply optional overrides to this command.
	for _, fn := range createExampleOverrides {
		fn(cmd, &createExampleReq)
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

	cmd.Flags().StringVar(&createSupervisorAgentReq.SupervisorAgent.Description, "description", createSupervisorAgentReq.SupervisorAgent.Description, `Description of what this agent can do (user-facing).`)
	cmd.Flags().StringVar(&createSupervisorAgentReq.SupervisorAgent.Instructions, "instructions", createSupervisorAgentReq.SupervisorAgent.Instructions, `Optional natural-language instructions for the supervisor agent.`)
	cmd.Flags().StringVar(&createSupervisorAgentReq.SupervisorAgent.Name, "name", createSupervisorAgentReq.SupervisorAgent.Name, `The resource name of the SupervisorAgent.`)

	cmd.Use = "create-supervisor-agent DISPLAY_NAME"
	cmd.Short = `*Beta* Create a Supervisor Agent.`
	cmd.Long = `This command is in Beta and may change without notice.

Create a Supervisor Agent.

  Creates a new Supervisor Agent.

  Arguments:
    DISPLAY_NAME: The display name of the Supervisor Agent, unique at workspace level.`

	cmd.Annotations = make(map[string]string)
	cmd.Annotations["launch_stage"] = "PUBLIC_BETA"
	cmd.Annotations["launch_stage_display"] = "Beta"

	cmd.Args = func(cmd *cobra.Command, args []string) error {
		if cmd.Flags().Changed("json") {
			err := root.ExactArgs(0)(cmd, args)
			if err != nil {
				return fmt.Errorf("when --json flag is specified, no positional arguments are allowed. Provide 'display_name' in your JSON input")
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
	cmd.Flags().StringVar(&createToolReq.Tool.Description, "description", createToolReq.Tool.Description, `Description of what this tool does (user-facing).`)
	// TODO: complex arg: genie_space
	// TODO: complex arg: knowledge_assistant
	cmd.Flags().StringVar(&createToolReq.Tool.Name, "name", createToolReq.Tool.Name, `Full resource name: supervisor-agents/{supervisor_agent_id}/tools/{tool_id}.`)
	// TODO: complex arg: uc_connection
	// TODO: complex arg: uc_function
	// TODO: complex arg: volume

	cmd.Use = "create-tool PARENT TOOL_ID TOOL_TYPE"
	cmd.Short = `*Beta* Create a Tool.`
	cmd.Long = `This command is in Beta and may change without notice.

Create a Tool.

  Creates a Tool under a Supervisor Agent. Specify one of "genie_space",
  "knowledge_assistant", "uc_function", "uc_connection", "app", "volume",
  "dashboard", "table", "vector_search_index", "catalog", "schema",
  "supervisor_agent", "web_search" in the request body. The legacy values
  "lakeview_dashboard" and "uc_table" are also accepted and remain equivalent to
  "dashboard" and "table" respectively.

  Arguments:
    PARENT: Parent resource where this tool will be created. Format:
      supervisor-agents/{supervisor_agent_id}
    TOOL_ID: The ID to use for the tool, which will become the final component of the
      tool's resource name.
    TOOL_TYPE: Tool type. Must be one of: "genie_space", "knowledge_assistant",
      "uc_function", "uc_connection", "app", "volume", "dashboard",
      "serving_endpoint", "table", "vector_search_index", "catalog", "schema",
      "supervisor_agent", "web_search". The legacy values "lakeview_dashboard"
      and "uc_table" are also accepted and remain equivalent to "dashboard" and
      "table" respectively.`

	cmd.Annotations = make(map[string]string)
	cmd.Annotations["launch_stage"] = "PUBLIC_BETA"
	cmd.Annotations["launch_stage_display"] = "Beta"

	cmd.Args = func(cmd *cobra.Command, args []string) error {
		if cmd.Flags().Changed("json") {
			err := root.ExactArgs(2)(cmd, args)
			if err != nil {
				return fmt.Errorf("when --json flag is specified, provide only PARENT, TOOL_ID as positional arguments. Provide 'tool_type' in your JSON input")
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

// start delete-example command

// Slice with functions to override default command behavior.
// Functions can be added from the `init()` function in manually curated files in this directory.
var deleteExampleOverrides []func(
	*cobra.Command,
	*supervisoragents.DeleteExampleRequest,
)

func newDeleteExample() *cobra.Command {
	cmd := &cobra.Command{}

	var deleteExampleReq supervisoragents.DeleteExampleRequest

	cmd.Use = "delete-example NAME"
	cmd.Short = `*Beta* Delete an example from a Supervisor Agent.`
	cmd.Long = `This command is in Beta and may change without notice.

Delete an example from a Supervisor Agent.

  Deletes an example from a Supervisor Agent.

  Arguments:
    NAME: The resource name of the example to delete. Format:
      supervisor-agents/{supervisor_agent_id}/examples/{example_id}`

	cmd.Annotations = make(map[string]string)
	cmd.Annotations["launch_stage"] = "PUBLIC_BETA"
	cmd.Annotations["launch_stage_display"] = "Beta"

	cmd.Args = func(cmd *cobra.Command, args []string) error {
		check := root.ExactArgs(1)
		return check(cmd, args)
	}

	cmd.PreRunE = root.MustWorkspaceClient
	cmd.RunE = func(cmd *cobra.Command, args []string) (err error) {
		ctx := cmd.Context()
		w := cmdctx.WorkspaceClient(ctx)

		deleteExampleReq.Name = args[0]

		err = w.SupervisorAgents.DeleteExample(ctx, deleteExampleReq)
		if err != nil {
			return err
		}
		return nil
	}

	// Disable completions since they are not applicable.
	// Can be overridden by manual implementation in `override.go`.
	cmd.ValidArgsFunction = cobra.NoFileCompletions

	// Apply optional overrides to this command.
	for _, fn := range deleteExampleOverrides {
		fn(cmd, &deleteExampleReq)
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
	cmd.Short = `*Beta* Delete a Supervisor Agent.`
	cmd.Long = `This command is in Beta and may change without notice.

Delete a Supervisor Agent.

  Deletes a Supervisor Agent.

  Arguments:
    NAME: The resource name of the Supervisor Agent. Format:
      supervisor-agents/{supervisor_agent_id}`

	cmd.Annotations = make(map[string]string)
	cmd.Annotations["launch_stage"] = "PUBLIC_BETA"
	cmd.Annotations["launch_stage_display"] = "Beta"

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
	cmd.Short = `*Beta* Delete a Tool.`
	cmd.Long = `This command is in Beta and may change without notice.

Delete a Tool.

  Deletes a Tool.

  Arguments:
    NAME: The resource name of the Tool. Format:
      supervisor-agents/{supervisor_agent_id}/tools/{tool_id}`

	cmd.Annotations = make(map[string]string)
	cmd.Annotations["launch_stage"] = "PUBLIC_BETA"
	cmd.Annotations["launch_stage_display"] = "Beta"

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

// start get-example command

// Slice with functions to override default command behavior.
// Functions can be added from the `init()` function in manually curated files in this directory.
var getExampleOverrides []func(
	*cobra.Command,
	*supervisoragents.GetExampleRequest,
)

func newGetExample() *cobra.Command {
	cmd := &cobra.Command{}

	var getExampleReq supervisoragents.GetExampleRequest

	cmd.Use = "get-example NAME"
	cmd.Short = `*Beta* Get an example from a Supervisor Agent.`
	cmd.Long = `This command is in Beta and may change without notice.

Get an example from a Supervisor Agent.

  Gets an example from a Supervisor Agent.

  Arguments:
    NAME: The resource name of the example. Format:
      supervisor-agents/{supervisor_agent_id}/examples/{example_id}`

	cmd.Annotations = make(map[string]string)
	cmd.Annotations["launch_stage"] = "PUBLIC_BETA"
	cmd.Annotations["launch_stage_display"] = "Beta"

	cmd.Args = func(cmd *cobra.Command, args []string) error {
		check := root.ExactArgs(1)
		return check(cmd, args)
	}

	cmd.PreRunE = root.MustWorkspaceClient
	cmd.RunE = func(cmd *cobra.Command, args []string) (err error) {
		ctx := cmd.Context()
		w := cmdctx.WorkspaceClient(ctx)

		getExampleReq.Name = args[0]

		response, err := w.SupervisorAgents.GetExample(ctx, getExampleReq)
		if err != nil {
			return err
		}

		return cmdio.Render(ctx, response)
	}

	// Disable completions since they are not applicable.
	// Can be overridden by manual implementation in `override.go`.
	cmd.ValidArgsFunction = cobra.NoFileCompletions

	// Apply optional overrides to this command.
	for _, fn := range getExampleOverrides {
		fn(cmd, &getExampleReq)
	}

	return cmd
}

// start get-permission-levels command

// Slice with functions to override default command behavior.
// Functions can be added from the `init()` function in manually curated files in this directory.
var getPermissionLevelsOverrides []func(
	*cobra.Command,
	*supervisoragents.GetSupervisorAgentPermissionLevelsRequest,
)

func newGetPermissionLevels() *cobra.Command {
	cmd := &cobra.Command{}

	var getPermissionLevelsReq supervisoragents.GetSupervisorAgentPermissionLevelsRequest

	cmd.Use = "get-permission-levels SUPERVISOR_AGENT_ID"
	cmd.Short = `*Beta* Get supervisor agent permission levels.`
	cmd.Long = `This command is in Beta and may change without notice.

Get supervisor agent permission levels.

  Gets the permission levels that a user can have on an object.

  Arguments:
    SUPERVISOR_AGENT_ID: The supervisor agent for which to get or manage permissions.`

	cmd.Annotations = make(map[string]string)
	cmd.Annotations["launch_stage"] = "PUBLIC_BETA"
	cmd.Annotations["launch_stage_display"] = "Beta"

	cmd.Args = func(cmd *cobra.Command, args []string) error {
		check := root.ExactArgs(1)
		return check(cmd, args)
	}

	cmd.PreRunE = root.MustWorkspaceClient
	cmd.RunE = func(cmd *cobra.Command, args []string) (err error) {
		ctx := cmd.Context()
		w := cmdctx.WorkspaceClient(ctx)

		getPermissionLevelsReq.SupervisorAgentId = args[0]

		response, err := w.SupervisorAgents.GetPermissionLevels(ctx, getPermissionLevelsReq)
		if err != nil {
			return err
		}

		return cmdio.Render(ctx, response)
	}

	// Disable completions since they are not applicable.
	// Can be overridden by manual implementation in `override.go`.
	cmd.ValidArgsFunction = cobra.NoFileCompletions

	// Apply optional overrides to this command.
	for _, fn := range getPermissionLevelsOverrides {
		fn(cmd, &getPermissionLevelsReq)
	}

	return cmd
}

// start get-permissions command

// Slice with functions to override default command behavior.
// Functions can be added from the `init()` function in manually curated files in this directory.
var getPermissionsOverrides []func(
	*cobra.Command,
	*supervisoragents.GetSupervisorAgentPermissionsRequest,
)

func newGetPermissions() *cobra.Command {
	cmd := &cobra.Command{}

	var getPermissionsReq supervisoragents.GetSupervisorAgentPermissionsRequest

	cmd.Use = "get-permissions SUPERVISOR_AGENT_ID"
	cmd.Short = `*Beta* Get supervisor agent permissions.`
	cmd.Long = `This command is in Beta and may change without notice.

Get supervisor agent permissions.

  Gets the permissions of a supervisor agent. Supervisor agents can inherit
  permissions from their root object.

  Arguments:
    SUPERVISOR_AGENT_ID: The supervisor agent for which to get or manage permissions.`

	cmd.Annotations = make(map[string]string)
	cmd.Annotations["launch_stage"] = "PUBLIC_BETA"
	cmd.Annotations["launch_stage_display"] = "Beta"

	cmd.Args = func(cmd *cobra.Command, args []string) error {
		check := root.ExactArgs(1)
		return check(cmd, args)
	}

	cmd.PreRunE = root.MustWorkspaceClient
	cmd.RunE = func(cmd *cobra.Command, args []string) (err error) {
		ctx := cmd.Context()
		w := cmdctx.WorkspaceClient(ctx)

		getPermissionsReq.SupervisorAgentId = args[0]

		response, err := w.SupervisorAgents.GetPermissions(ctx, getPermissionsReq)
		if err != nil {
			return err
		}

		return cmdio.Render(ctx, response)
	}

	// Disable completions since they are not applicable.
	// Can be overridden by manual implementation in `override.go`.
	cmd.ValidArgsFunction = cobra.NoFileCompletions

	// Apply optional overrides to this command.
	for _, fn := range getPermissionsOverrides {
		fn(cmd, &getPermissionsReq)
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
	cmd.Short = `*Beta* Get a Supervisor Agent.`
	cmd.Long = `This command is in Beta and may change without notice.

Get a Supervisor Agent.

  Gets a Supervisor Agent.

  Arguments:
    NAME: The resource name of the Supervisor Agent. Format:
      supervisor-agents/{supervisor_agent_id}`

	cmd.Annotations = make(map[string]string)
	cmd.Annotations["launch_stage"] = "PUBLIC_BETA"
	cmd.Annotations["launch_stage_display"] = "Beta"

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
	cmd.Short = `*Beta* Get a Tool.`
	cmd.Long = `This command is in Beta and may change without notice.

Get a Tool.

  Gets a Tool.

  Arguments:
    NAME: The resource name of the Tool. Format:
      supervisor-agents/{supervisor_agent_id}/tools/{tool_id}`

	cmd.Annotations = make(map[string]string)
	cmd.Annotations["launch_stage"] = "PUBLIC_BETA"
	cmd.Annotations["launch_stage_display"] = "Beta"

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

// start list-examples command

// Slice with functions to override default command behavior.
// Functions can be added from the `init()` function in manually curated files in this directory.
var listExamplesOverrides []func(
	*cobra.Command,
	*supervisoragents.ListExamplesRequest,
)

func newListExamples() *cobra.Command {
	cmd := &cobra.Command{}

	var listExamplesReq supervisoragents.ListExamplesRequest
	// Registered for all paginated methods. Validated at call time in the
	// method-call template. Paginated list methods never have Wait or LRO
	// branches, so the method-call path is always reached.
	var listExamplesLimit int

	cmd.Flags().IntVar(&listExamplesReq.PageSize, "page-size", listExamplesReq.PageSize, `The maximum number of examples to return.`)

	// Limit flag for total result capping.
	cmd.Flags().IntVar(&listExamplesLimit, "limit", 0, `Maximum number of results to return.`)

	// Hidden pagination flags (internal API parameters).
	cmd.Flags().StringVar(&listExamplesReq.PageToken, "page-token", listExamplesReq.PageToken, `Pagination token.`)
	cmd.Flags().Lookup("page-token").Hidden = true

	cmd.Use = "list-examples PARENT"
	cmd.Short = `*Beta* List examples for a Supervisor Agent.`
	cmd.Long = `This command is in Beta and may change without notice.

List examples for a Supervisor Agent.

  Lists examples under a Supervisor Agent.

  Arguments:
    PARENT: Parent resource to list from. Format:
      supervisor-agents/{supervisor_agent_id}`

	cmd.Annotations = make(map[string]string)
	cmd.Annotations["launch_stage"] = "PUBLIC_BETA"
	cmd.Annotations["launch_stage_display"] = "Beta"

	cmd.Args = func(cmd *cobra.Command, args []string) error {
		check := root.ExactArgs(1)
		return check(cmd, args)
	}

	cmd.PreRunE = root.MustWorkspaceClient
	cmd.RunE = func(cmd *cobra.Command, args []string) (err error) {
		ctx := cmd.Context()
		w := cmdctx.WorkspaceClient(ctx)

		listExamplesReq.Parent = args[0]

		response := w.SupervisorAgents.ListExamples(ctx, listExamplesReq)
		if listExamplesLimit < 0 {
			return fmt.Errorf("--limit must be a non-negative integer, got %d", listExamplesLimit)
		}
		if listExamplesLimit > 0 {
			ctx = cmdio.WithLimit(ctx, listExamplesLimit)
		}

		return cmdio.RenderIterator(ctx, response)
	}

	// Disable completions since they are not applicable.
	// Can be overridden by manual implementation in `override.go`.
	cmd.ValidArgsFunction = cobra.NoFileCompletions

	// Apply optional overrides to this command.
	for _, fn := range listExamplesOverrides {
		fn(cmd, &listExamplesReq)
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
	cmd.Short = `*Beta* List Supervisor Agents.`
	cmd.Long = `This command is in Beta and may change without notice.

List Supervisor Agents.

  Lists Supervisor Agents.`

	cmd.Annotations = make(map[string]string)
	cmd.Annotations["launch_stage"] = "PUBLIC_BETA"
	cmd.Annotations["launch_stage_display"] = "Beta"

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
	cmd.Short = `*Beta* List Tools.`
	cmd.Long = `This command is in Beta and may change without notice.

List Tools.

  Lists Tools under a Supervisor Agent.

  Arguments:
    PARENT: Parent resource to list from. Format:
      supervisor-agents/{supervisor_agent_id}`

	cmd.Annotations = make(map[string]string)
	cmd.Annotations["launch_stage"] = "PUBLIC_BETA"
	cmd.Annotations["launch_stage_display"] = "Beta"

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

// start set-permissions command

// Slice with functions to override default command behavior.
// Functions can be added from the `init()` function in manually curated files in this directory.
var setPermissionsOverrides []func(
	*cobra.Command,
	*supervisoragents.SupervisorAgentPermissionsRequest,
)

func newSetPermissions() *cobra.Command {
	cmd := &cobra.Command{}

	var setPermissionsReq supervisoragents.SupervisorAgentPermissionsRequest
	var setPermissionsJson flags.JsonFlag

	cmd.Flags().Var(&setPermissionsJson, "json", `either inline JSON string or @path/to/file.json with request body`)

	// TODO: array: access_control_list

	cmd.Use = "set-permissions SUPERVISOR_AGENT_ID"
	cmd.Short = `*Beta* Set supervisor agent permissions.`
	cmd.Long = `This command is in Beta and may change without notice.

Set supervisor agent permissions.

  Sets permissions on an object, replacing existing permissions if they exist.
  Deletes all direct permissions if none are specified. Objects can inherit
  permissions from their root object.

  Arguments:
    SUPERVISOR_AGENT_ID: The supervisor agent for which to get or manage permissions.`

	cmd.Annotations = make(map[string]string)
	cmd.Annotations["launch_stage"] = "PUBLIC_BETA"
	cmd.Annotations["launch_stage_display"] = "Beta"

	cmd.Args = func(cmd *cobra.Command, args []string) error {
		check := root.ExactArgs(1)
		return check(cmd, args)
	}

	cmd.PreRunE = root.MustWorkspaceClient
	cmd.RunE = func(cmd *cobra.Command, args []string) (err error) {
		ctx := cmd.Context()
		w := cmdctx.WorkspaceClient(ctx)

		if cmd.Flags().Changed("json") {
			diags := setPermissionsJson.Unmarshal(&setPermissionsReq)
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
		setPermissionsReq.SupervisorAgentId = args[0]

		response, err := w.SupervisorAgents.SetPermissions(ctx, setPermissionsReq)
		if err != nil {
			return err
		}

		return cmdio.Render(ctx, response)
	}

	// Disable completions since they are not applicable.
	// Can be overridden by manual implementation in `override.go`.
	cmd.ValidArgsFunction = cobra.NoFileCompletions

	// Apply optional overrides to this command.
	for _, fn := range setPermissionsOverrides {
		fn(cmd, &setPermissionsReq)
	}

	return cmd
}

// start update-example command

// Slice with functions to override default command behavior.
// Functions can be added from the `init()` function in manually curated files in this directory.
var updateExampleOverrides []func(
	*cobra.Command,
	*supervisoragents.UpdateExampleRequest,
)

func newUpdateExample() *cobra.Command {
	cmd := &cobra.Command{}

	var updateExampleReq supervisoragents.UpdateExampleRequest
	updateExampleReq.Example = supervisoragents.Example{}
	var updateExampleJson flags.JsonFlag

	cmd.Flags().Var(&updateExampleJson, "json", `either inline JSON string or @path/to/file.json with request body`)

	cmd.Flags().StringVar(&updateExampleReq.Example.Name, "name", updateExampleReq.Example.Name, `Full resource name: supervisor-agents/{supervisor_agent_id}/examples/{example_id}.`)

	cmd.Use = "update-example NAME UPDATE_MASK QUESTION GUIDELINES"
	cmd.Short = `*Beta* Update an example in a Supervisor Agent.`
	cmd.Long = `This command is in Beta and may change without notice.

Update an example in a Supervisor Agent.

  Updates an example in a Supervisor Agent.

  Arguments:
    NAME: The resource name of the example to update. Format:
      supervisor-agents/{supervisor_agent_id}/examples/{example_id}
    UPDATE_MASK: Comma-delimited list of fields to update on the example. Allowed values:
      question, guidelines. Examples: - question - question,guidelines
    QUESTION: The example question.
    GUIDELINES: Guidelines for answering the question.`

	cmd.Annotations = make(map[string]string)
	cmd.Annotations["launch_stage"] = "PUBLIC_BETA"
	cmd.Annotations["launch_stage_display"] = "Beta"

	cmd.Args = func(cmd *cobra.Command, args []string) error {
		if cmd.Flags().Changed("json") {
			err := root.ExactArgs(2)(cmd, args)
			if err != nil {
				return fmt.Errorf("when --json flag is specified, provide only NAME, UPDATE_MASK as positional arguments. Provide 'question', 'guidelines' in your JSON input")
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
			diags := updateExampleJson.Unmarshal(&updateExampleReq.Example)
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
		updateExampleReq.Name = args[0]
		if args[1] != "" {
			updateMaskArray := strings.Split(args[1], ",")
			updateExampleReq.UpdateMask = *fieldmask.New(updateMaskArray)
		}
		if !cmd.Flags().Changed("json") {
			updateExampleReq.Example.Question = args[2]
		}
		if !cmd.Flags().Changed("json") {
			_, err = fmt.Sscan(args[3], &updateExampleReq.Example.Guidelines)
			if err != nil {
				return fmt.Errorf("invalid GUIDELINES: %s", args[3])
			}

		}

		response, err := w.SupervisorAgents.UpdateExample(ctx, updateExampleReq)
		if err != nil {
			return err
		}

		return cmdio.Render(ctx, response)
	}

	// Disable completions since they are not applicable.
	// Can be overridden by manual implementation in `override.go`.
	cmd.ValidArgsFunction = cobra.NoFileCompletions

	// Apply optional overrides to this command.
	for _, fn := range updateExampleOverrides {
		fn(cmd, &updateExampleReq)
	}

	return cmd
}

// start update-permissions command

// Slice with functions to override default command behavior.
// Functions can be added from the `init()` function in manually curated files in this directory.
var updatePermissionsOverrides []func(
	*cobra.Command,
	*supervisoragents.SupervisorAgentPermissionsRequest,
)

func newUpdatePermissions() *cobra.Command {
	cmd := &cobra.Command{}

	var updatePermissionsReq supervisoragents.SupervisorAgentPermissionsRequest
	var updatePermissionsJson flags.JsonFlag

	cmd.Flags().Var(&updatePermissionsJson, "json", `either inline JSON string or @path/to/file.json with request body`)

	// TODO: array: access_control_list

	cmd.Use = "update-permissions SUPERVISOR_AGENT_ID"
	cmd.Short = `*Beta* Update supervisor agent permissions.`
	cmd.Long = `This command is in Beta and may change without notice.

Update supervisor agent permissions.

  Updates the permissions on a supervisor agent. Supervisor agents can inherit
  permissions from their root object.

  Arguments:
    SUPERVISOR_AGENT_ID: The supervisor agent for which to get or manage permissions.`

	cmd.Annotations = make(map[string]string)
	cmd.Annotations["launch_stage"] = "PUBLIC_BETA"
	cmd.Annotations["launch_stage_display"] = "Beta"

	cmd.Args = func(cmd *cobra.Command, args []string) error {
		check := root.ExactArgs(1)
		return check(cmd, args)
	}

	cmd.PreRunE = root.MustWorkspaceClient
	cmd.RunE = func(cmd *cobra.Command, args []string) (err error) {
		ctx := cmd.Context()
		w := cmdctx.WorkspaceClient(ctx)

		if cmd.Flags().Changed("json") {
			diags := updatePermissionsJson.Unmarshal(&updatePermissionsReq)
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
		updatePermissionsReq.SupervisorAgentId = args[0]

		response, err := w.SupervisorAgents.UpdatePermissions(ctx, updatePermissionsReq)
		if err != nil {
			return err
		}

		return cmdio.Render(ctx, response)
	}

	// Disable completions since they are not applicable.
	// Can be overridden by manual implementation in `override.go`.
	cmd.ValidArgsFunction = cobra.NoFileCompletions

	// Apply optional overrides to this command.
	for _, fn := range updatePermissionsOverrides {
		fn(cmd, &updatePermissionsReq)
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

	cmd.Flags().StringVar(&updateSupervisorAgentReq.SupervisorAgent.Description, "description", updateSupervisorAgentReq.SupervisorAgent.Description, `Description of what this agent can do (user-facing).`)
	cmd.Flags().StringVar(&updateSupervisorAgentReq.SupervisorAgent.Instructions, "instructions", updateSupervisorAgentReq.SupervisorAgent.Instructions, `Optional natural-language instructions for the supervisor agent.`)
	cmd.Flags().StringVar(&updateSupervisorAgentReq.SupervisorAgent.Name, "name", updateSupervisorAgentReq.SupervisorAgent.Name, `The resource name of the SupervisorAgent.`)

	cmd.Use = "update-supervisor-agent NAME UPDATE_MASK DISPLAY_NAME"
	cmd.Short = `*Beta* Update a Supervisor Agent.`
	cmd.Long = `This command is in Beta and may change without notice.

Update a Supervisor Agent.

  Updates a Supervisor Agent. The fields that are required depend on the paths
  specified in update_mask. Only fields included in the mask will be updated.

  Arguments:
    NAME: The resource name of the SupervisorAgent. Format:
      supervisor-agents/{supervisor_agent_id}
    UPDATE_MASK: Field mask for fields to be updated.
    DISPLAY_NAME: The display name of the Supervisor Agent, unique at workspace level.`

	cmd.Annotations = make(map[string]string)
	cmd.Annotations["launch_stage"] = "PUBLIC_BETA"
	cmd.Annotations["launch_stage_display"] = "Beta"

	cmd.Args = func(cmd *cobra.Command, args []string) error {
		if cmd.Flags().Changed("json") {
			err := root.ExactArgs(2)(cmd, args)
			if err != nil {
				return fmt.Errorf("when --json flag is specified, provide only NAME, UPDATE_MASK as positional arguments. Provide 'display_name' in your JSON input")
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
	cmd.Flags().StringVar(&updateToolReq.Tool.Description, "description", updateToolReq.Tool.Description, `Description of what this tool does (user-facing).`)
	// TODO: complex arg: genie_space
	// TODO: complex arg: knowledge_assistant
	cmd.Flags().StringVar(&updateToolReq.Tool.Name, "name", updateToolReq.Tool.Name, `Full resource name: supervisor-agents/{supervisor_agent_id}/tools/{tool_id}.`)
	// TODO: complex arg: uc_connection
	// TODO: complex arg: uc_function
	// TODO: complex arg: volume

	cmd.Use = "update-tool NAME UPDATE_MASK TOOL_TYPE"
	cmd.Short = `*Beta* Update a Tool.`
	cmd.Long = `This command is in Beta and may change without notice.

Update a Tool.

  Updates a Tool. Only the description field can be updated. To change
  immutable fields such as tool type, spec, or tool ID, delete the tool and
  recreate it.

  Arguments:
    NAME: Full resource name:
      supervisor-agents/{supervisor_agent_id}/tools/{tool_id}
    UPDATE_MASK: Field mask for fields to be updated.
    TOOL_TYPE: Tool type. Must be one of: "genie_space", "knowledge_assistant",
      "uc_function", "uc_connection", "app", "volume", "dashboard",
      "serving_endpoint", "table", "vector_search_index", "catalog", "schema",
      "supervisor_agent", "web_search". The legacy values "lakeview_dashboard"
      and "uc_table" are also accepted and remain equivalent to "dashboard" and
      "table" respectively.`

	cmd.Annotations = make(map[string]string)
	cmd.Annotations["launch_stage"] = "PUBLIC_BETA"
	cmd.Annotations["launch_stage_display"] = "Beta"

	cmd.Args = func(cmd *cobra.Command, args []string) error {
		if cmd.Flags().Changed("json") {
			err := root.ExactArgs(2)(cmd, args)
			if err != nil {
				return fmt.Errorf("when --json flag is specified, provide only NAME, UPDATE_MASK as positional arguments. Provide 'tool_type' in your JSON input")
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
