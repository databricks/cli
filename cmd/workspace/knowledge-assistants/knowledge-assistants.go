// Code generated from OpenAPI specs by Databricks SDK Generator. DO NOT EDIT.

package knowledge_assistants

import (
	"errors"
	"fmt"

	"github.com/databricks/cli/cmd/root"
	"github.com/databricks/cli/libs/cmdctx"
	"github.com/databricks/cli/libs/cmdgen"
	"github.com/databricks/cli/libs/cmdio"
	"github.com/databricks/cli/libs/flags"
	"github.com/spf13/cobra"

	accessmanagement "github.com/databricks/cli/libs/sdktypes/accessmanagement/v1"
	knowledgeassistants "github.com/databricks/cli/libs/sdktypes/knowledgeassistants/v1"
	sdkaccessmanagement "github.com/databricks/sdk-go/accessmanagement/v1"
	sdkknowledgeassistants "github.com/databricks/sdk-go/knowledgeassistants/v1"
)

// Slice with functions to override default command behavior.
// Functions can be added from the `init()` function in manually curated files in this directory.
var cmdOverrides []func(*cobra.Command)

func New() *cobra.Command {
	cmd := &cobra.Command{
		Use:   "knowledge-assistants",
		Short: `*Beta* Manage Knowledge Assistants and related resources.`,
		Long: `This command is in Beta and may change without notice.

Manage Knowledge Assistants and related resources.`,
		GroupID: "agentbricks",
		RunE:    root.ReportUnknownSubcommand,
	}

	cmd.Annotations = make(map[string]string)
	cmd.Annotations["launch_stage"] = "PUBLIC_BETA"
	cmd.Annotations["launch_stage_display"] = "Beta"

	// Add methods
	cmd.AddCommand(newCreateExample())
	cmd.AddCommand(newCreateKnowledgeAssistant())
	cmd.AddCommand(newCreateKnowledgeSource())
	cmd.AddCommand(newDeleteExample())
	cmd.AddCommand(newDeleteKnowledgeAssistant())
	cmd.AddCommand(newDeleteKnowledgeSource())
	cmd.AddCommand(newGetExample())
	cmd.AddCommand(newGetKnowledgeAssistant())
	cmd.AddCommand(newGetKnowledgeSource())
	cmd.AddCommand(newGetPermissionLevels())
	cmd.AddCommand(newGetPermissions())
	cmd.AddCommand(newListExamples())
	cmd.AddCommand(newListKnowledgeAssistants())
	cmd.AddCommand(newListKnowledgeSources())
	cmd.AddCommand(newSetPermissions())
	cmd.AddCommand(newSyncKnowledgeSources())
	cmd.AddCommand(newUpdateExample())
	cmd.AddCommand(newUpdateKnowledgeAssistant())
	cmd.AddCommand(newUpdateKnowledgeSource())
	cmd.AddCommand(newUpdatePermissions())

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
	*knowledgeassistants.CreateExampleRequest,
)

func newCreateExample() *cobra.Command {
	cmd := &cobra.Command{}
	var createExampleReq knowledgeassistants.CreateExampleRequest
	createExampleReq.Example = &knowledgeassistants.Example{}

	var createExampleJson flags.JsonFlag

	cmd.Flags().Var(&createExampleJson, "json", `either inline JSON string or @path/to/file.json with request body`)

	// TODO: array: guidelines
	flags.StringPtrVar(cmd.Flags(), &createExampleReq.Example.Name, "name", `Full resource name: knowledge-assistants/{knowledge_assistant_id}/examples/{example_id}.`)

	cmd.Use = "create-example PARENT QUESTION"
	cmd.Short = `*Beta* Create an example for a Knowledge Assistant.`
	cmd.Long = `This command is in Beta and may change without notice.

Create an example for a Knowledge Assistant.

  Creates an example for a Knowledge Assistant.

  Arguments:
    PARENT: Parent resource where this example will be created. Format:
      knowledge-assistants/{knowledge_assistant_id}
    QUESTION: The example question.`

	cmd.Annotations = make(map[string]string)
	cmd.Annotations["launch_stage"] = "PUBLIC_BETA"
	cmd.Annotations["launch_stage_display"] = "Beta"

	cmd.Args = func(cmd *cobra.Command, args []string) error {
		if cmd.Flags().Changed("json") {
			err := root.ExactArgs(1)(cmd, args)
			if err != nil {
				return errors.New("when --json flag is specified, provide only PARENT as positional arguments. Provide 'question' in your JSON input")
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
		createExampleReq.Parent = new(args[0])
		if !cmd.Flags().Changed("json") {
			createExampleReq.Example.Question = new(args[1])
		}

		sdkReq, err := knowledgeassistants.CreateExampleRequestToSDK(&createExampleReq)
		if err != nil {
			return err
		}
		response, err := w.KnowledgeAssistants.CreateExample(ctx, *sdkReq)
		if err != nil {
			return err
		}

		out, err := knowledgeassistants.ExampleFromSDK(response)
		if err != nil {
			return err
		}
		return cmdio.Render(ctx, out)
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

// start create-knowledge-assistant command

// Slice with functions to override default command behavior.
// Functions can be added from the `init()` function in manually curated files in this directory.
var createKnowledgeAssistantOverrides []func(
	*cobra.Command,
	*knowledgeassistants.CreateKnowledgeAssistantRequest,
)

func newCreateKnowledgeAssistant() *cobra.Command {
	cmd := &cobra.Command{}
	var createKnowledgeAssistantReq knowledgeassistants.CreateKnowledgeAssistantRequest
	createKnowledgeAssistantReq.KnowledgeAssistant = &knowledgeassistants.KnowledgeAssistant{}

	var createKnowledgeAssistantJson flags.JsonFlag

	cmd.Flags().Var(&createKnowledgeAssistantJson, "json", `either inline JSON string or @path/to/file.json with request body`)

	flags.StringPtrVar(cmd.Flags(), &createKnowledgeAssistantReq.KnowledgeAssistant.Instructions, "instructions", `Additional global instructions on how the agent should generate answers.`)
	flags.StringPtrVar(cmd.Flags(), &createKnowledgeAssistantReq.KnowledgeAssistant.Name, "name", `The resource name of the Knowledge Assistant.`)

	cmd.Use = "create-knowledge-assistant DISPLAY_NAME DESCRIPTION"
	cmd.Short = `*Beta* Create a Knowledge Assistant.`
	cmd.Long = `This command is in Beta and may change without notice.

Create a Knowledge Assistant.

  Creates a Knowledge Assistant.

  Arguments:
    DISPLAY_NAME: The display name of the Knowledge Assistant, unique at workspace level.
      Required when creating a Knowledge Assistant. When updating a Knowledge
      Assistant, optional unless included in update_mask.
    DESCRIPTION: Description of what this agent can do (user-facing). Required when
      creating a Knowledge Assistant. When updating a Knowledge Assistant,
      optional unless included in update_mask.`

	cmd.Annotations = make(map[string]string)
	cmd.Annotations["launch_stage"] = "PUBLIC_BETA"
	cmd.Annotations["launch_stage_display"] = "Beta"

	cmd.Args = func(cmd *cobra.Command, args []string) error {
		if cmd.Flags().Changed("json") {
			err := root.ExactArgs(0)(cmd, args)
			if err != nil {
				return errors.New("when --json flag is specified, no positional arguments are allowed. Provide 'display_name', 'description' in your JSON input")
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
			diags := createKnowledgeAssistantJson.Unmarshal(&createKnowledgeAssistantReq.KnowledgeAssistant)
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
			createKnowledgeAssistantReq.KnowledgeAssistant.DisplayName = new(args[0])
		}
		if !cmd.Flags().Changed("json") {
			createKnowledgeAssistantReq.KnowledgeAssistant.Description = new(args[1])
		}

		sdkReq, err := knowledgeassistants.CreateKnowledgeAssistantRequestToSDK(&createKnowledgeAssistantReq)
		if err != nil {
			return err
		}
		response, err := w.KnowledgeAssistants.CreateKnowledgeAssistant(ctx, *sdkReq)
		if err != nil {
			return err
		}

		out, err := knowledgeassistants.KnowledgeAssistantFromSDK(response)
		if err != nil {
			return err
		}
		return cmdio.Render(ctx, out)
	}

	// Disable completions since they are not applicable.
	// Can be overridden by manual implementation in `override.go`.
	cmd.ValidArgsFunction = cobra.NoFileCompletions

	// Apply optional overrides to this command.
	for _, fn := range createKnowledgeAssistantOverrides {
		fn(cmd, &createKnowledgeAssistantReq)
	}

	return cmd
}

// start create-knowledge-source command

// Slice with functions to override default command behavior.
// Functions can be added from the `init()` function in manually curated files in this directory.
var createKnowledgeSourceOverrides []func(
	*cobra.Command,
	*knowledgeassistants.CreateKnowledgeSourceRequest,
)

func newCreateKnowledgeSource() *cobra.Command {
	cmd := &cobra.Command{}
	var createKnowledgeSourceReq knowledgeassistants.CreateKnowledgeSourceRequest
	createKnowledgeSourceReq.KnowledgeSource = &knowledgeassistants.KnowledgeSource{}

	var createKnowledgeSourceJson flags.JsonFlag

	cmd.Flags().Var(&createKnowledgeSourceJson, "json", `either inline JSON string or @path/to/file.json with request body`)

	// TODO: complex arg: file_table
	// TODO: complex arg: files
	// TODO: complex arg: index
	flags.StringPtrVar(cmd.Flags(), &createKnowledgeSourceReq.KnowledgeSource.Name, "name", `Full resource name: knowledge-assistants/{knowledge_assistant_id}/knowledge-sources/{knowledge_source_id}.`)

	cmd.Use = "create-knowledge-source PARENT DISPLAY_NAME DESCRIPTION SOURCE_TYPE"
	cmd.Short = `*Beta* Create a Knowledge Source.`
	cmd.Long = `This command is in Beta and may change without notice.

Create a Knowledge Source.

  Creates a Knowledge Source under a Knowledge Assistant.

  Arguments:
    PARENT: Parent resource where this source will be created. Format:
      knowledge-assistants/{knowledge_assistant_id}
    DISPLAY_NAME: Human-readable display name of the knowledge source. Required when
      creating a Knowledge Source. When updating a Knowledge Source, optional
      unless included in update_mask.
    DESCRIPTION: Description of the knowledge source. Required when creating a Knowledge
      Source. When updating a Knowledge Source, optional unless included in
      update_mask.
    SOURCE_TYPE: The type of the source: "index", "files", or "file_table". Required when
      creating a Knowledge Source. When updating a Knowledge Source, this field
      is ignored.`

	cmd.Annotations = make(map[string]string)
	cmd.Annotations["launch_stage"] = "PUBLIC_BETA"
	cmd.Annotations["launch_stage_display"] = "Beta"

	cmd.Args = func(cmd *cobra.Command, args []string) error {
		if cmd.Flags().Changed("json") {
			err := root.ExactArgs(1)(cmd, args)
			if err != nil {
				return errors.New("when --json flag is specified, provide only PARENT as positional arguments. Provide 'display_name', 'description', 'source_type' in your JSON input")
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
		_ = w

		if cmd.Flags().Changed("json") {
			diags := createKnowledgeSourceJson.Unmarshal(&createKnowledgeSourceReq.KnowledgeSource)
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
		createKnowledgeSourceReq.Parent = new(args[0])
		if !cmd.Flags().Changed("json") {
			createKnowledgeSourceReq.KnowledgeSource.DisplayName = new(args[1])
		}
		if !cmd.Flags().Changed("json") {
			createKnowledgeSourceReq.KnowledgeSource.Description = new(args[2])
		}
		if !cmd.Flags().Changed("json") {
			createKnowledgeSourceReq.KnowledgeSource.SourceType = new(args[3])
		}

		sdkReq, err := knowledgeassistants.CreateKnowledgeSourceRequestToSDK(&createKnowledgeSourceReq)
		if err != nil {
			return err
		}
		response, err := w.KnowledgeAssistants.CreateKnowledgeSource(ctx, *sdkReq)
		if err != nil {
			return err
		}

		out, err := knowledgeassistants.KnowledgeSourceFromSDK(response)
		if err != nil {
			return err
		}
		return cmdio.Render(ctx, out)
	}

	// Disable completions since they are not applicable.
	// Can be overridden by manual implementation in `override.go`.
	cmd.ValidArgsFunction = cobra.NoFileCompletions

	// Apply optional overrides to this command.
	for _, fn := range createKnowledgeSourceOverrides {
		fn(cmd, &createKnowledgeSourceReq)
	}

	return cmd
}

// start delete-example command

// Slice with functions to override default command behavior.
// Functions can be added from the `init()` function in manually curated files in this directory.
var deleteExampleOverrides []func(
	*cobra.Command,
	*sdkknowledgeassistants.DeleteExampleRequest,
)

func newDeleteExample() *cobra.Command {
	cmd := &cobra.Command{}
	var deleteExampleReq sdkknowledgeassistants.DeleteExampleRequest

	cmd.Use = "delete-example NAME"
	cmd.Short = `*Beta* Delete an example from a Knowledge Assistant.`
	cmd.Long = `This command is in Beta and may change without notice.

Delete an example from a Knowledge Assistant.

  Deletes an example from a Knowledge Assistant.

  Arguments:
    NAME: The resource name of the example to delete. Format:
      knowledge-assistants/{knowledge_assistant_id}/examples/{example_id}`

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
		_ = w

		deleteExampleReq.Name = new(args[0])

		sdkReq := &deleteExampleReq
		err = w.KnowledgeAssistants.DeleteExample(ctx, *sdkReq)
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

// start delete-knowledge-assistant command

// Slice with functions to override default command behavior.
// Functions can be added from the `init()` function in manually curated files in this directory.
var deleteKnowledgeAssistantOverrides []func(
	*cobra.Command,
	*sdkknowledgeassistants.DeleteKnowledgeAssistantRequest,
)

func newDeleteKnowledgeAssistant() *cobra.Command {
	cmd := &cobra.Command{}
	var deleteKnowledgeAssistantReq sdkknowledgeassistants.DeleteKnowledgeAssistantRequest

	cmd.Use = "delete-knowledge-assistant NAME"
	cmd.Short = `*Beta* Delete a Knowledge Assistant.`
	cmd.Long = `This command is in Beta and may change without notice.

Delete a Knowledge Assistant.

  Deletes a Knowledge Assistant.

  Arguments:
    NAME: The resource name of the knowledge assistant to be deleted. Format:
      knowledge-assistants/{knowledge_assistant_id}`

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
		_ = w

		deleteKnowledgeAssistantReq.Name = new(args[0])

		sdkReq := &deleteKnowledgeAssistantReq
		err = w.KnowledgeAssistants.DeleteKnowledgeAssistant(ctx, *sdkReq)
		if err != nil {
			return err
		}
		return nil
	}

	// Disable completions since they are not applicable.
	// Can be overridden by manual implementation in `override.go`.
	cmd.ValidArgsFunction = cobra.NoFileCompletions

	// Apply optional overrides to this command.
	for _, fn := range deleteKnowledgeAssistantOverrides {
		fn(cmd, &deleteKnowledgeAssistantReq)
	}

	return cmd
}

// start delete-knowledge-source command

// Slice with functions to override default command behavior.
// Functions can be added from the `init()` function in manually curated files in this directory.
var deleteKnowledgeSourceOverrides []func(
	*cobra.Command,
	*sdkknowledgeassistants.DeleteKnowledgeSourceRequest,
)

func newDeleteKnowledgeSource() *cobra.Command {
	cmd := &cobra.Command{}
	var deleteKnowledgeSourceReq sdkknowledgeassistants.DeleteKnowledgeSourceRequest

	cmd.Use = "delete-knowledge-source NAME"
	cmd.Short = `*Beta* Delete a Knowledge Source.`
	cmd.Long = `This command is in Beta and may change without notice.

Delete a Knowledge Source.

  Deletes a Knowledge Source.

  Arguments:
    NAME: The resource name of the Knowledge Source to delete. Format:
      knowledge-assistants/{knowledge_assistant_id}/knowledge-sources/{knowledge_source_id}`

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
		_ = w

		deleteKnowledgeSourceReq.Name = new(args[0])

		sdkReq := &deleteKnowledgeSourceReq
		err = w.KnowledgeAssistants.DeleteKnowledgeSource(ctx, *sdkReq)
		if err != nil {
			return err
		}
		return nil
	}

	// Disable completions since they are not applicable.
	// Can be overridden by manual implementation in `override.go`.
	cmd.ValidArgsFunction = cobra.NoFileCompletions

	// Apply optional overrides to this command.
	for _, fn := range deleteKnowledgeSourceOverrides {
		fn(cmd, &deleteKnowledgeSourceReq)
	}

	return cmd
}

// start get-example command

// Slice with functions to override default command behavior.
// Functions can be added from the `init()` function in manually curated files in this directory.
var getExampleOverrides []func(
	*cobra.Command,
	*sdkknowledgeassistants.GetExampleRequest,
)

func newGetExample() *cobra.Command {
	cmd := &cobra.Command{}
	var getExampleReq sdkknowledgeassistants.GetExampleRequest

	cmd.Use = "get-example NAME"
	cmd.Short = `*Beta* Get an example from a Knowledge Assistant.`
	cmd.Long = `This command is in Beta and may change without notice.

Get an example from a Knowledge Assistant.

  Gets an example from a Knowledge Assistant.

  Arguments:
    NAME: The resource name of the example. Format:
      knowledge-assistants/{knowledge_assistant_id}/examples/{example_id}`

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
		_ = w

		getExampleReq.Name = new(args[0])

		sdkReq := &getExampleReq
		response, err := w.KnowledgeAssistants.GetExample(ctx, *sdkReq)
		if err != nil {
			return err
		}

		out, err := knowledgeassistants.ExampleFromSDK(response)
		if err != nil {
			return err
		}
		return cmdio.Render(ctx, out)
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

// start get-knowledge-assistant command

// Slice with functions to override default command behavior.
// Functions can be added from the `init()` function in manually curated files in this directory.
var getKnowledgeAssistantOverrides []func(
	*cobra.Command,
	*sdkknowledgeassistants.GetKnowledgeAssistantRequest,
)

func newGetKnowledgeAssistant() *cobra.Command {
	cmd := &cobra.Command{}
	var getKnowledgeAssistantReq sdkknowledgeassistants.GetKnowledgeAssistantRequest

	cmd.Use = "get-knowledge-assistant NAME"
	cmd.Short = `*Beta* Get a Knowledge Assistant.`
	cmd.Long = `This command is in Beta and may change without notice.

Get a Knowledge Assistant.

  Gets a Knowledge Assistant.

  Arguments:
    NAME: The resource name of the knowledge assistant. Format:
      knowledge-assistants/{knowledge_assistant_id}`

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
		_ = w

		getKnowledgeAssistantReq.Name = new(args[0])

		sdkReq := &getKnowledgeAssistantReq
		response, err := w.KnowledgeAssistants.GetKnowledgeAssistant(ctx, *sdkReq)
		if err != nil {
			return err
		}

		out, err := knowledgeassistants.KnowledgeAssistantFromSDK(response)
		if err != nil {
			return err
		}
		return cmdio.Render(ctx, out)
	}

	// Disable completions since they are not applicable.
	// Can be overridden by manual implementation in `override.go`.
	cmd.ValidArgsFunction = cobra.NoFileCompletions

	// Apply optional overrides to this command.
	for _, fn := range getKnowledgeAssistantOverrides {
		fn(cmd, &getKnowledgeAssistantReq)
	}

	return cmd
}

// start get-knowledge-source command

// Slice with functions to override default command behavior.
// Functions can be added from the `init()` function in manually curated files in this directory.
var getKnowledgeSourceOverrides []func(
	*cobra.Command,
	*sdkknowledgeassistants.GetKnowledgeSourceRequest,
)

func newGetKnowledgeSource() *cobra.Command {
	cmd := &cobra.Command{}
	var getKnowledgeSourceReq sdkknowledgeassistants.GetKnowledgeSourceRequest

	cmd.Use = "get-knowledge-source NAME"
	cmd.Short = `*Beta* Get a Knowledge Source.`
	cmd.Long = `This command is in Beta and may change without notice.

Get a Knowledge Source.

  Gets a Knowledge Source.

  Arguments:
    NAME: The resource name of the Knowledge Source. Format:
      knowledge-assistants/{knowledge_assistant_id}/knowledge-sources/{knowledge_source_id}`

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
		_ = w

		getKnowledgeSourceReq.Name = new(args[0])

		sdkReq := &getKnowledgeSourceReq
		response, err := w.KnowledgeAssistants.GetKnowledgeSource(ctx, *sdkReq)
		if err != nil {
			return err
		}

		out, err := knowledgeassistants.KnowledgeSourceFromSDK(response)
		if err != nil {
			return err
		}
		return cmdio.Render(ctx, out)
	}

	// Disable completions since they are not applicable.
	// Can be overridden by manual implementation in `override.go`.
	cmd.ValidArgsFunction = cobra.NoFileCompletions

	// Apply optional overrides to this command.
	for _, fn := range getKnowledgeSourceOverrides {
		fn(cmd, &getKnowledgeSourceReq)
	}

	return cmd
}

// start get-permission-levels command

// Slice with functions to override default command behavior.
// Functions can be added from the `init()` function in manually curated files in this directory.
var getPermissionLevelsOverrides []func(
	*cobra.Command,
	*sdkaccessmanagement.ListPermissionLevelsRequest,
)

func newGetPermissionLevels() *cobra.Command {
	cmd := &cobra.Command{}
	var getPermissionLevelsReq sdkaccessmanagement.ListPermissionLevelsRequest

	cmd.Use = "get-permission-levels KNOWLEDGE_ASSISTANT_ID"
	cmd.Short = `*Beta* Get knowledge assistant permission levels.`
	cmd.Long = `This command is in Beta and may change without notice.

Get knowledge assistant permission levels.

  Gets the permission levels that a user can have on an object.

  Arguments:
    KNOWLEDGE_ASSISTANT_ID: The knowledge assistant for which to get or manage permissions.`

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
		_ = w

		_ = args[0] // TODO: field not in sdk-go request: knowledge_assistant_id

		getPermissionLevelsReq.RequestObjectType = new("knowledge-assistants")

		sdkReq := &getPermissionLevelsReq
		response, err := w.AccessManagement.ListPermissionLevels(ctx, *sdkReq)
		if err != nil {
			return err
		}

		out, err := accessmanagement.ListPermissionLevelsResponseFromSDK(response)
		if err != nil {
			return err
		}
		return cmdio.Render(ctx, out)
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
	*sdkaccessmanagement.GetObjectPermissionsRequest,
)

func newGetPermissions() *cobra.Command {
	cmd := &cobra.Command{}
	var getPermissionsReq sdkaccessmanagement.GetObjectPermissionsRequest

	cmd.Use = "get-permissions KNOWLEDGE_ASSISTANT_ID"
	cmd.Short = `*Beta* Get knowledge assistant permissions.`
	cmd.Long = `This command is in Beta and may change without notice.

Get knowledge assistant permissions.

  Gets the permissions of a knowledge assistant. Knowledge assistants can
  inherit permissions from their root object.

  Arguments:
    KNOWLEDGE_ASSISTANT_ID: The knowledge assistant for which to get or manage permissions.`

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
		_ = w

		_ = args[0] // TODO: field not in sdk-go request: knowledge_assistant_id

		getPermissionsReq.RequestObjectType = new("knowledge-assistants")

		sdkReq := &getPermissionsReq
		response, err := w.AccessManagement.GetObjectPermissions(ctx, *sdkReq)
		if err != nil {
			return err
		}

		out, err := accessmanagement.PermissionsResponseFromSDK(response)
		if err != nil {
			return err
		}
		return cmdio.Render(ctx, out)
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

// start list-examples command

// Slice with functions to override default command behavior.
// Functions can be added from the `init()` function in manually curated files in this directory.
var listExamplesOverrides []func(
	*cobra.Command,
	*knowledgeassistants.ListExamplesRequest,
)

func newListExamples() *cobra.Command {
	cmd := &cobra.Command{}
	var listExamplesReq knowledgeassistants.ListExamplesRequest

	// Registered for all paginated methods. Validated at call time in the
	// method-call template. Paginated list methods never have Wait or LRO
	// branches, so the method-call path is always reached.
	var listExamplesLimit int

	flags.IntPtrVar(cmd.Flags(), &listExamplesReq.PageSize, "page-size", `The maximum number of examples to return.`)

	// Limit flag for total result capping.
	cmd.Flags().IntVar(&listExamplesLimit, "limit", 0, `Maximum number of results to return.`)

	// Hidden pagination flags (internal API parameters).
	flags.StringPtrVar(cmd.Flags(), &listExamplesReq.PageToken, "page-token", `Pagination token.`)
	cmd.Flags().Lookup("page-token").Hidden = true

	cmd.Use = "list-examples PARENT"
	cmd.Short = `*Beta* List examples for a Knowledge Assistant.`
	cmd.Long = `This command is in Beta and may change without notice.

List examples for a Knowledge Assistant.

  Lists examples under a Knowledge Assistant.

  Arguments:
    PARENT: Parent resource to list from. Format:
      knowledge-assistants/{knowledge_assistant_id}`

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
		_ = w

		listExamplesReq.Parent = new(args[0])

		sdkReq, err := knowledgeassistants.ListExamplesRequestToSDK(&listExamplesReq)
		if err != nil {
			return err
		}
		if listExamplesLimit < 0 {
			return fmt.Errorf("--limit must be a non-negative integer, got %d", listExamplesLimit)
		}
		if listExamplesLimit > 0 {
			ctx = cmdio.WithLimit(ctx, listExamplesLimit)
		}
		response := w.KnowledgeAssistants.ListExamplesIter(ctx, *sdkReq)
		return cmdio.RenderIterator(ctx, cmdgen.MapSeq(response, knowledgeassistants.ExampleFromSDK))
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

// start list-knowledge-assistants command

// Slice with functions to override default command behavior.
// Functions can be added from the `init()` function in manually curated files in this directory.
var listKnowledgeAssistantsOverrides []func(
	*cobra.Command,
	*knowledgeassistants.ListKnowledgeAssistantsRequest,
)

func newListKnowledgeAssistants() *cobra.Command {
	cmd := &cobra.Command{}
	var listKnowledgeAssistantsReq knowledgeassistants.ListKnowledgeAssistantsRequest

	// Registered for all paginated methods. Validated at call time in the
	// method-call template. Paginated list methods never have Wait or LRO
	// branches, so the method-call path is always reached.
	var listKnowledgeAssistantsLimit int

	flags.IntPtrVar(cmd.Flags(), &listKnowledgeAssistantsReq.PageSize, "page-size", `The maximum number of knowledge assistants to return.`)

	// Limit flag for total result capping.
	cmd.Flags().IntVar(&listKnowledgeAssistantsLimit, "limit", 0, `Maximum number of results to return.`)

	// Hidden pagination flags (internal API parameters).
	flags.StringPtrVar(cmd.Flags(), &listKnowledgeAssistantsReq.PageToken, "page-token", `Pagination token.`)
	cmd.Flags().Lookup("page-token").Hidden = true

	cmd.Use = "list-knowledge-assistants"
	cmd.Short = `*Beta* List Knowledge Assistants.`
	cmd.Long = `This command is in Beta and may change without notice.

List Knowledge Assistants.

  List Knowledge Assistants`

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
		_ = w

		sdkReq, err := knowledgeassistants.ListKnowledgeAssistantsRequestToSDK(&listKnowledgeAssistantsReq)
		if err != nil {
			return err
		}
		if listKnowledgeAssistantsLimit < 0 {
			return fmt.Errorf("--limit must be a non-negative integer, got %d", listKnowledgeAssistantsLimit)
		}
		if listKnowledgeAssistantsLimit > 0 {
			ctx = cmdio.WithLimit(ctx, listKnowledgeAssistantsLimit)
		}
		response := w.KnowledgeAssistants.ListKnowledgeAssistantsIter(ctx, *sdkReq)
		return cmdio.RenderIterator(ctx, cmdgen.MapSeq(response, knowledgeassistants.KnowledgeAssistantFromSDK))
	}

	// Disable completions since they are not applicable.
	// Can be overridden by manual implementation in `override.go`.
	cmd.ValidArgsFunction = cobra.NoFileCompletions

	// Apply optional overrides to this command.
	for _, fn := range listKnowledgeAssistantsOverrides {
		fn(cmd, &listKnowledgeAssistantsReq)
	}

	return cmd
}

// start list-knowledge-sources command

// Slice with functions to override default command behavior.
// Functions can be added from the `init()` function in manually curated files in this directory.
var listKnowledgeSourcesOverrides []func(
	*cobra.Command,
	*knowledgeassistants.ListKnowledgeSourcesRequest,
)

func newListKnowledgeSources() *cobra.Command {
	cmd := &cobra.Command{}
	var listKnowledgeSourcesReq knowledgeassistants.ListKnowledgeSourcesRequest

	// Registered for all paginated methods. Validated at call time in the
	// method-call template. Paginated list methods never have Wait or LRO
	// branches, so the method-call path is always reached.
	var listKnowledgeSourcesLimit int

	flags.IntPtrVar(cmd.Flags(), &listKnowledgeSourcesReq.PageSize, "page-size", ``)

	// Limit flag for total result capping.
	cmd.Flags().IntVar(&listKnowledgeSourcesLimit, "limit", 0, `Maximum number of results to return.`)

	// Hidden pagination flags (internal API parameters).
	flags.StringPtrVar(cmd.Flags(), &listKnowledgeSourcesReq.PageToken, "page-token", `Pagination token.`)
	cmd.Flags().Lookup("page-token").Hidden = true

	cmd.Use = "list-knowledge-sources PARENT"
	cmd.Short = `*Beta* List Knowledge Sources.`
	cmd.Long = `This command is in Beta and may change without notice.

List Knowledge Sources.

  Lists Knowledge Sources under a Knowledge Assistant.

  Arguments:
    PARENT: Parent resource to list from. Format:
      knowledge-assistants/{knowledge_assistant_id}`

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
		_ = w

		listKnowledgeSourcesReq.Parent = new(args[0])

		sdkReq, err := knowledgeassistants.ListKnowledgeSourcesRequestToSDK(&listKnowledgeSourcesReq)
		if err != nil {
			return err
		}
		if listKnowledgeSourcesLimit < 0 {
			return fmt.Errorf("--limit must be a non-negative integer, got %d", listKnowledgeSourcesLimit)
		}
		if listKnowledgeSourcesLimit > 0 {
			ctx = cmdio.WithLimit(ctx, listKnowledgeSourcesLimit)
		}
		response := w.KnowledgeAssistants.ListKnowledgeSourcesIter(ctx, *sdkReq)
		return cmdio.RenderIterator(ctx, cmdgen.MapSeq(response, knowledgeassistants.KnowledgeSourceFromSDK))
	}

	// Disable completions since they are not applicable.
	// Can be overridden by manual implementation in `override.go`.
	cmd.ValidArgsFunction = cobra.NoFileCompletions

	// Apply optional overrides to this command.
	for _, fn := range listKnowledgeSourcesOverrides {
		fn(cmd, &listKnowledgeSourcesReq)
	}

	return cmd
}

// start set-permissions command

// Slice with functions to override default command behavior.
// Functions can be added from the `init()` function in manually curated files in this directory.
var setPermissionsOverrides []func(
	*cobra.Command,
	*accessmanagement.SetObjectPermissionsRequest,
)

func newSetPermissions() *cobra.Command {
	cmd := &cobra.Command{}
	var setPermissionsReq accessmanagement.SetObjectPermissionsRequest

	var setPermissionsJson flags.JsonFlag

	cmd.Flags().Var(&setPermissionsJson, "json", `either inline JSON string or @path/to/file.json with request body`)

	// TODO: array: access_control_list

	cmd.Use = "set-permissions KNOWLEDGE_ASSISTANT_ID"
	cmd.Short = `*Beta* Set knowledge assistant permissions.`
	cmd.Long = `This command is in Beta and may change without notice.

Set knowledge assistant permissions.

  Sets permissions on an object, replacing existing permissions if they exist.
  Deletes all direct permissions if none are specified. Objects can inherit
  permissions from their root object.

  Arguments:
    KNOWLEDGE_ASSISTANT_ID: The knowledge assistant for which to get or manage permissions.`

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
		_ = w

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
		_ = args[0] // TODO: field not in sdk-go request: knowledge_assistant_id

		setPermissionsReq.RequestObjectType = new("knowledge-assistants")

		sdkReq, err := accessmanagement.SetObjectPermissionsRequestToSDK(&setPermissionsReq)
		if err != nil {
			return err
		}
		response, err := w.AccessManagement.SetObjectPermissions(ctx, *sdkReq)
		if err != nil {
			return err
		}

		out, err := accessmanagement.PermissionsResponseFromSDK(response)
		if err != nil {
			return err
		}
		return cmdio.Render(ctx, out)
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

// start sync-knowledge-sources command

// Slice with functions to override default command behavior.
// Functions can be added from the `init()` function in manually curated files in this directory.
var syncKnowledgeSourcesOverrides []func(
	*cobra.Command,
	*knowledgeassistants.SyncKnowledgeSourcesRequest,
)

func newSyncKnowledgeSources() *cobra.Command {
	cmd := &cobra.Command{}
	var syncKnowledgeSourcesReq knowledgeassistants.SyncKnowledgeSourcesRequest

	cmd.Use = "sync-knowledge-sources NAME"
	cmd.Short = `*Beta* Syncs all Knowledge Sources for a Knowledge Assistant.`
	cmd.Long = `This command is in Beta and may change without notice.

Syncs all Knowledge Sources for a Knowledge Assistant.

  Sync all non-index Knowledge Sources for a Knowledge Assistant (index sources
  do not require sync)

  Arguments:
    NAME: The resource name of the Knowledge Assistant. Format:
      knowledge-assistants/{knowledge_assistant_id}`

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
		_ = w

		syncKnowledgeSourcesReq.Name = new(args[0])

		sdkReq, err := knowledgeassistants.SyncKnowledgeSourcesRequestToSDK(&syncKnowledgeSourcesReq)
		if err != nil {
			return err
		}
		err = w.KnowledgeAssistants.SyncKnowledgeSources(ctx, *sdkReq)
		if err != nil {
			return err
		}
		return nil
	}

	// Disable completions since they are not applicable.
	// Can be overridden by manual implementation in `override.go`.
	cmd.ValidArgsFunction = cobra.NoFileCompletions

	// Apply optional overrides to this command.
	for _, fn := range syncKnowledgeSourcesOverrides {
		fn(cmd, &syncKnowledgeSourcesReq)
	}

	return cmd
}

// start update-example command

// Slice with functions to override default command behavior.
// Functions can be added from the `init()` function in manually curated files in this directory.
var updateExampleOverrides []func(
	*cobra.Command,
	*knowledgeassistants.UpdateExampleRequest,
)

func newUpdateExample() *cobra.Command {
	cmd := &cobra.Command{}
	var updateExampleReq knowledgeassistants.UpdateExampleRequest
	updateExampleReq.Example = &knowledgeassistants.Example{}

	var updateExampleJson flags.JsonFlag

	cmd.Flags().Var(&updateExampleJson, "json", `either inline JSON string or @path/to/file.json with request body`)

	// TODO: array: guidelines
	flags.StringPtrVar(cmd.Flags(), &updateExampleReq.Name, "name", `Full resource name: knowledge-assistants/{knowledge_assistant_id}/examples/{example_id}.`)

	cmd.Use = "update-example NAME UPDATE_MASK QUESTION"
	cmd.Short = `*Beta* Update an example in a Knowledge Assistant.`
	cmd.Long = `This command is in Beta and may change without notice.

Update an example in a Knowledge Assistant.

  Updates an example in a Knowledge Assistant.

  Arguments:
    NAME: The resource name of the example to update. Format:
      knowledge-assistants/{knowledge_assistant_id}/examples/{example_id}
    UPDATE_MASK: Comma-delimited list of fields to update on the example. Allowed values:
      question, guidelines. Examples: - question - question,guidelines
    QUESTION: The example question.`

	cmd.Annotations = make(map[string]string)
	cmd.Annotations["launch_stage"] = "PUBLIC_BETA"
	cmd.Annotations["launch_stage_display"] = "Beta"

	cmd.Args = func(cmd *cobra.Command, args []string) error {
		if cmd.Flags().Changed("json") {
			err := root.ExactArgs(2)(cmd, args)
			if err != nil {
				return errors.New("when --json flag is specified, provide only NAME, UPDATE_MASK as positional arguments. Provide 'question' in your JSON input")
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
		_ = w

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
		updateExampleReq.Name = new(args[0])
		if args[1] != "" {
			updateExampleReq.UpdateMask = new(args[1])
		}
		if !cmd.Flags().Changed("json") {
			updateExampleReq.Example.Question = new(args[2])
		}

		sdkReq, err := knowledgeassistants.UpdateExampleRequestToSDK(&updateExampleReq)
		if err != nil {
			return err
		}
		response, err := w.KnowledgeAssistants.UpdateExample(ctx, *sdkReq)
		if err != nil {
			return err
		}

		out, err := knowledgeassistants.ExampleFromSDK(response)
		if err != nil {
			return err
		}
		return cmdio.Render(ctx, out)
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

// start update-knowledge-assistant command

// Slice with functions to override default command behavior.
// Functions can be added from the `init()` function in manually curated files in this directory.
var updateKnowledgeAssistantOverrides []func(
	*cobra.Command,
	*knowledgeassistants.UpdateKnowledgeAssistantRequest,
)

func newUpdateKnowledgeAssistant() *cobra.Command {
	cmd := &cobra.Command{}
	var updateKnowledgeAssistantReq knowledgeassistants.UpdateKnowledgeAssistantRequest
	updateKnowledgeAssistantReq.KnowledgeAssistant = &knowledgeassistants.KnowledgeAssistant{}

	var updateKnowledgeAssistantJson flags.JsonFlag

	cmd.Flags().Var(&updateKnowledgeAssistantJson, "json", `either inline JSON string or @path/to/file.json with request body`)

	flags.StringPtrVar(cmd.Flags(), &updateKnowledgeAssistantReq.KnowledgeAssistant.Instructions, "instructions", `Additional global instructions on how the agent should generate answers.`)
	flags.StringPtrVar(cmd.Flags(), &updateKnowledgeAssistantReq.KnowledgeAssistant.Name, "name", `The resource name of the Knowledge Assistant.`)

	cmd.Use = "update-knowledge-assistant NAME UPDATE_MASK DISPLAY_NAME DESCRIPTION"
	cmd.Short = `*Beta* Update a Knowledge Assistant.`
	cmd.Long = `This command is in Beta and may change without notice.

Update a Knowledge Assistant.

  Updates a Knowledge Assistant.

  Arguments:
    NAME: The resource name of the Knowledge Assistant. Format:
      knowledge-assistants/{knowledge_assistant_id}
    UPDATE_MASK: Comma-delimited list of fields to update on the Knowledge Assistant.
      Allowed values: display_name, description, instructions. Examples: -
      display_name - description,instructions
    DISPLAY_NAME: The display name of the Knowledge Assistant, unique at workspace level.
      Required when creating a Knowledge Assistant. When updating a Knowledge
      Assistant, optional unless included in update_mask.
    DESCRIPTION: Description of what this agent can do (user-facing). Required when
      creating a Knowledge Assistant. When updating a Knowledge Assistant,
      optional unless included in update_mask.`

	cmd.Annotations = make(map[string]string)
	cmd.Annotations["launch_stage"] = "PUBLIC_BETA"
	cmd.Annotations["launch_stage_display"] = "Beta"

	cmd.Args = func(cmd *cobra.Command, args []string) error {
		if cmd.Flags().Changed("json") {
			err := root.ExactArgs(2)(cmd, args)
			if err != nil {
				return errors.New("when --json flag is specified, provide only NAME, UPDATE_MASK as positional arguments. Provide 'display_name', 'description' in your JSON input")
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
		_ = w

		if cmd.Flags().Changed("json") {
			diags := updateKnowledgeAssistantJson.Unmarshal(&updateKnowledgeAssistantReq.KnowledgeAssistant)
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
		updateKnowledgeAssistantReq.KnowledgeAssistant.Name = new(args[0])
		if args[1] != "" {
			updateKnowledgeAssistantReq.UpdateMask = new(args[1])
		}
		if !cmd.Flags().Changed("json") {
			updateKnowledgeAssistantReq.KnowledgeAssistant.DisplayName = new(args[2])
		}
		if !cmd.Flags().Changed("json") {
			updateKnowledgeAssistantReq.KnowledgeAssistant.Description = new(args[3])
		}

		sdkReq, err := knowledgeassistants.UpdateKnowledgeAssistantRequestToSDK(&updateKnowledgeAssistantReq)
		if err != nil {
			return err
		}
		response, err := w.KnowledgeAssistants.UpdateKnowledgeAssistant(ctx, *sdkReq)
		if err != nil {
			return err
		}

		out, err := knowledgeassistants.KnowledgeAssistantFromSDK(response)
		if err != nil {
			return err
		}
		return cmdio.Render(ctx, out)
	}

	// Disable completions since they are not applicable.
	// Can be overridden by manual implementation in `override.go`.
	cmd.ValidArgsFunction = cobra.NoFileCompletions

	// Apply optional overrides to this command.
	for _, fn := range updateKnowledgeAssistantOverrides {
		fn(cmd, &updateKnowledgeAssistantReq)
	}

	return cmd
}

// start update-knowledge-source command

// Slice with functions to override default command behavior.
// Functions can be added from the `init()` function in manually curated files in this directory.
var updateKnowledgeSourceOverrides []func(
	*cobra.Command,
	*knowledgeassistants.UpdateKnowledgeSourceRequest,
)

func newUpdateKnowledgeSource() *cobra.Command {
	cmd := &cobra.Command{}
	var updateKnowledgeSourceReq knowledgeassistants.UpdateKnowledgeSourceRequest
	updateKnowledgeSourceReq.KnowledgeSource = &knowledgeassistants.KnowledgeSource{}

	var updateKnowledgeSourceJson flags.JsonFlag

	cmd.Flags().Var(&updateKnowledgeSourceJson, "json", `either inline JSON string or @path/to/file.json with request body`)

	// TODO: complex arg: file_table
	// TODO: complex arg: files
	// TODO: complex arg: index
	flags.StringPtrVar(cmd.Flags(), &updateKnowledgeSourceReq.Name, "name", `Full resource name: knowledge-assistants/{knowledge_assistant_id}/knowledge-sources/{knowledge_source_id}.`)

	cmd.Use = "update-knowledge-source NAME UPDATE_MASK DISPLAY_NAME DESCRIPTION SOURCE_TYPE"
	cmd.Short = `*Beta* Update a Knowledge Source.`
	cmd.Long = `This command is in Beta and may change without notice.

Update a Knowledge Source.

  Updates a Knowledge Source.

  Arguments:
    NAME: The resource name of the Knowledge Source to update. Format:
      knowledge-assistants/{knowledge_assistant_id}/knowledge-sources/{knowledge_source_id}
    UPDATE_MASK: Comma-delimited list of fields to update on the Knowledge Source. Allowed
      values: display_name, description. Examples: - display_name -
      display_name,description
    DISPLAY_NAME: Human-readable display name of the knowledge source. Required when
      creating a Knowledge Source. When updating a Knowledge Source, optional
      unless included in update_mask.
    DESCRIPTION: Description of the knowledge source. Required when creating a Knowledge
      Source. When updating a Knowledge Source, optional unless included in
      update_mask.
    SOURCE_TYPE: The type of the source: "index", "files", or "file_table". Required when
      creating a Knowledge Source. When updating a Knowledge Source, this field
      is ignored.`

	cmd.Annotations = make(map[string]string)
	cmd.Annotations["launch_stage"] = "PUBLIC_BETA"
	cmd.Annotations["launch_stage_display"] = "Beta"

	cmd.Args = func(cmd *cobra.Command, args []string) error {
		if cmd.Flags().Changed("json") {
			err := root.ExactArgs(2)(cmd, args)
			if err != nil {
				return errors.New("when --json flag is specified, provide only NAME, UPDATE_MASK as positional arguments. Provide 'display_name', 'description', 'source_type' in your JSON input")
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
		_ = w

		if cmd.Flags().Changed("json") {
			diags := updateKnowledgeSourceJson.Unmarshal(&updateKnowledgeSourceReq.KnowledgeSource)
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
		updateKnowledgeSourceReq.Name = new(args[0])
		if args[1] != "" {
			updateKnowledgeSourceReq.UpdateMask = new(args[1])
		}
		if !cmd.Flags().Changed("json") {
			updateKnowledgeSourceReq.KnowledgeSource.DisplayName = new(args[2])
		}
		if !cmd.Flags().Changed("json") {
			updateKnowledgeSourceReq.KnowledgeSource.Description = new(args[3])
		}
		if !cmd.Flags().Changed("json") {
			updateKnowledgeSourceReq.KnowledgeSource.SourceType = new(args[4])
		}

		sdkReq, err := knowledgeassistants.UpdateKnowledgeSourceRequestToSDK(&updateKnowledgeSourceReq)
		if err != nil {
			return err
		}
		response, err := w.KnowledgeAssistants.UpdateKnowledgeSource(ctx, *sdkReq)
		if err != nil {
			return err
		}

		out, err := knowledgeassistants.KnowledgeSourceFromSDK(response)
		if err != nil {
			return err
		}
		return cmdio.Render(ctx, out)
	}

	// Disable completions since they are not applicable.
	// Can be overridden by manual implementation in `override.go`.
	cmd.ValidArgsFunction = cobra.NoFileCompletions

	// Apply optional overrides to this command.
	for _, fn := range updateKnowledgeSourceOverrides {
		fn(cmd, &updateKnowledgeSourceReq)
	}

	return cmd
}

// start update-permissions command

// Slice with functions to override default command behavior.
// Functions can be added from the `init()` function in manually curated files in this directory.
var updatePermissionsOverrides []func(
	*cobra.Command,
	*accessmanagement.UpdateObjectPermissionsRequest,
)

func newUpdatePermissions() *cobra.Command {
	cmd := &cobra.Command{}
	var updatePermissionsReq accessmanagement.UpdateObjectPermissionsRequest

	var updatePermissionsJson flags.JsonFlag

	cmd.Flags().Var(&updatePermissionsJson, "json", `either inline JSON string or @path/to/file.json with request body`)

	// TODO: array: access_control_list

	cmd.Use = "update-permissions KNOWLEDGE_ASSISTANT_ID"
	cmd.Short = `*Beta* Update knowledge assistant permissions.`
	cmd.Long = `This command is in Beta and may change without notice.

Update knowledge assistant permissions.

  Updates the permissions on a knowledge assistant. Knowledge assistants can
  inherit permissions from their root object.

  Arguments:
    KNOWLEDGE_ASSISTANT_ID: The knowledge assistant for which to get or manage permissions.`

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
		_ = w

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
		_ = args[0] // TODO: field not in sdk-go request: knowledge_assistant_id

		updatePermissionsReq.RequestObjectType = new("knowledge-assistants")

		sdkReq, err := accessmanagement.UpdateObjectPermissionsRequestToSDK(&updatePermissionsReq)
		if err != nil {
			return err
		}
		response, err := w.AccessManagement.UpdateObjectPermissions(ctx, *sdkReq)
		if err != nil {
			return err
		}

		out, err := accessmanagement.PermissionsResponseFromSDK(response)
		if err != nil {
			return err
		}
		return cmdio.Render(ctx, out)
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

// end service KnowledgeAssistants
