// Code generated from OpenAPI specs by Databricks SDK Generator. DO NOT EDIT.

package ai_gateway

import (
	"fmt"
	"strings"

	"github.com/databricks/cli/cmd/root"
	"github.com/databricks/cli/libs/cmdctx"
	"github.com/databricks/cli/libs/cmdio"
	"github.com/databricks/cli/libs/flags"
	"github.com/databricks/databricks-sdk-go/common/types/fieldmask"
	"github.com/databricks/databricks-sdk-go/service/catalog"
	"github.com/spf13/cobra"
)

// Slice with functions to override default command behavior.
// Functions can be added from the `init()` function in manually curated files in this directory.
var cmdOverrides []func(*cobra.Command)

func New() *cobra.Command {
	cmd := &cobra.Command{
		Use:   "ai-gateway",
		Short: `*Beta* Govern AI workloads in Unity Catalog.`,
		Long: `This command is in Beta and may change without notice.

Govern AI workloads in Unity Catalog. This API manages the Unity Catalog
  securables that bring centralized access control, lineage, and auditing to
  AI-serving entities: model services (governed access to foundation models and
  external LLMs), model provider services (governed connections to external
  model providers), and MCP services (governed Model Context Protocol servers).`,
		GroupID: "catalog",
		RunE:    root.ReportUnknownSubcommand,
	}

	cmd.Annotations = make(map[string]string)
	cmd.Annotations["launch_stage"] = "PUBLIC_BETA"
	cmd.Annotations["launch_stage_display"] = "Beta"

	// Add methods
	cmd.AddCommand(newCreateMcpService())
	cmd.AddCommand(newCreateModelProviderService())
	cmd.AddCommand(newCreateModelService())
	cmd.AddCommand(newDeleteMcpService())
	cmd.AddCommand(newDeleteModelProviderService())
	cmd.AddCommand(newDeleteModelService())
	cmd.AddCommand(newGetMcpService())
	cmd.AddCommand(newGetModelProviderService())
	cmd.AddCommand(newGetModelService())
	cmd.AddCommand(newListMcpServices())
	cmd.AddCommand(newListModelProviderServices())
	cmd.AddCommand(newListModelServices())
	cmd.AddCommand(newUpdateMcpService())
	cmd.AddCommand(newUpdateModelProviderService())
	cmd.AddCommand(newUpdateModelService())

	// Apply optional overrides to this command.
	for _, fn := range cmdOverrides {
		fn(cmd)
	}

	return cmd
}

// start create-mcp-service command

// Slice with functions to override default command behavior.
// Functions can be added from the `init()` function in manually curated files in this directory.
var createMcpServiceOverrides []func(
	*cobra.Command,
	*catalog.CreateMcpServiceRequest,
)

func newCreateMcpService() *cobra.Command {
	cmd := &cobra.Command{}

	var createMcpServiceReq catalog.CreateMcpServiceRequest
	createMcpServiceReq.McpService = catalog.McpService{}
	var createMcpServiceJson flags.JsonFlag

	cmd.Flags().Var(&createMcpServiceJson, "json", `either inline JSON string or @path/to/file.json with request body`)

	cmd.Flags().StringVar(&createMcpServiceReq.McpService.Comment, "comment", createMcpServiceReq.McpService.Comment, `User-provided description.`)
	// TODO: complex arg: config
	cmd.Flags().StringVar(&createMcpServiceReq.McpService.Name, "name", createMcpServiceReq.McpService.Name, `Resource name of the MCP service.`)
	cmd.Flags().StringVar(&createMcpServiceReq.McpService.Owner, "owner", createMcpServiceReq.McpService.Owner, `The owner of the MCP service.`)

	cmd.Use = "create-mcp-service PARENT MCP_SERVICE_ID"
	cmd.Short = `*Beta* Create an MCP service.`
	cmd.Long = `This command is in Beta and may change without notice.

Create an MCP service.

  Creates an MCP service in a Unity Catalog schema. An MCP (Model Context
  Protocol) service is a governed securable that registers an MCP server and
  exposes its tools for discovery, access control, and invocation. The caller
  supplies the leaf name in mcp_service_id.

  You must be the owner of the parent schema or have the CREATE_SERVICE and
  USE_SCHEMA privileges on the parent schema and USE_CATALOG on the parent
  catalog. You also need USE_CONNECTION on the connection the MCP service
  references.

  Arguments:
    PARENT: Name of the parent schema. Format: schemas/{catalog}.{schema}. Each
      {...} component is capped at 255 characters individually.
    MCP_SERVICE_ID: Name for the MCP service, e.g. "my_mcp_service".`

	cmd.Annotations = make(map[string]string)
	cmd.Annotations["launch_stage"] = "PUBLIC_BETA"
	cmd.Annotations["launch_stage_display"] = "Beta"

	cmd.Args = func(cmd *cobra.Command, args []string) error {
		check := root.ExactArgs(2)
		return check(cmd, args)
	}

	cmd.PreRunE = root.MustWorkspaceClient
	cmd.RunE = func(cmd *cobra.Command, args []string) (err error) {
		ctx := cmd.Context()
		w := cmdctx.WorkspaceClient(ctx)

		if cmd.Flags().Changed("json") {
			diags := createMcpServiceJson.Unmarshal(&createMcpServiceReq.McpService)
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
		createMcpServiceReq.Parent = args[0]
		createMcpServiceReq.McpServiceId = args[1]

		response, err := w.AiGateway.CreateMcpService(ctx, createMcpServiceReq)
		if err != nil {
			return err
		}

		return cmdio.Render(ctx, response)
	}

	// Disable completions since they are not applicable.
	// Can be overridden by manual implementation in `override.go`.
	cmd.ValidArgsFunction = cobra.NoFileCompletions

	// Apply optional overrides to this command.
	for _, fn := range createMcpServiceOverrides {
		fn(cmd, &createMcpServiceReq)
	}

	return cmd
}

// start create-model-provider-service command

// Slice with functions to override default command behavior.
// Functions can be added from the `init()` function in manually curated files in this directory.
var createModelProviderServiceOverrides []func(
	*cobra.Command,
	*catalog.CreateModelProviderServiceRequest,
)

func newCreateModelProviderService() *cobra.Command {
	cmd := &cobra.Command{}

	var createModelProviderServiceReq catalog.CreateModelProviderServiceRequest
	createModelProviderServiceReq.ModelProviderService = catalog.ModelProviderService{}
	var createModelProviderServiceJson flags.JsonFlag

	cmd.Flags().Var(&createModelProviderServiceJson, "json", `either inline JSON string or @path/to/file.json with request body`)

	cmd.Flags().StringVar(&createModelProviderServiceReq.ModelProviderService.Comment, "comment", createModelProviderServiceReq.ModelProviderService.Comment, `User-provided description.`)
	// TODO: complex arg: config
	cmd.Flags().StringVar(&createModelProviderServiceReq.ModelProviderService.Name, "name", createModelProviderServiceReq.ModelProviderService.Name, `Resource name of the provider service.`)
	cmd.Flags().StringVar(&createModelProviderServiceReq.ModelProviderService.Owner, "owner", createModelProviderServiceReq.ModelProviderService.Owner, `The owner of the model provider service.`)

	cmd.Use = "create-model-provider-service PARENT MODEL_PROVIDER_SERVICE_ID"
	cmd.Short = `*Beta* Create a model provider service.`
	cmd.Long = `This command is in Beta and may change without notice.

Create a model provider service.

  Creates a model provider service in a Unity Catalog schema. A model provider
  service is a governed connection to an external model provider (for example
  OpenAI, Azure OpenAI, or Amazon Bedrock) that model services reference to
  invoke that provider. The caller supplies the leaf name in
  model_provider_service_id.

  You must be the owner of the parent schema or have the CREATE_SERVICE and
  USE_SCHEMA privileges on the parent schema and USE_CATALOG on the parent
  catalog.

  Arguments:
    PARENT: Name of the parent schema. Format: schemas/{catalog}.{schema}. Each
      {...} component is capped at 255 characters individually.
    MODEL_PROVIDER_SERVICE_ID: Name for the model provider service, e.g. "openai_prod".`

	cmd.Annotations = make(map[string]string)
	cmd.Annotations["launch_stage"] = "PUBLIC_BETA"
	cmd.Annotations["launch_stage_display"] = "Beta"

	cmd.Args = func(cmd *cobra.Command, args []string) error {
		check := root.ExactArgs(2)
		return check(cmd, args)
	}

	cmd.PreRunE = root.MustWorkspaceClient
	cmd.RunE = func(cmd *cobra.Command, args []string) (err error) {
		ctx := cmd.Context()
		w := cmdctx.WorkspaceClient(ctx)

		if cmd.Flags().Changed("json") {
			diags := createModelProviderServiceJson.Unmarshal(&createModelProviderServiceReq.ModelProviderService)
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
		createModelProviderServiceReq.Parent = args[0]
		createModelProviderServiceReq.ModelProviderServiceId = args[1]

		response, err := w.AiGateway.CreateModelProviderService(ctx, createModelProviderServiceReq)
		if err != nil {
			return err
		}

		return cmdio.Render(ctx, response)
	}

	// Disable completions since they are not applicable.
	// Can be overridden by manual implementation in `override.go`.
	cmd.ValidArgsFunction = cobra.NoFileCompletions

	// Apply optional overrides to this command.
	for _, fn := range createModelProviderServiceOverrides {
		fn(cmd, &createModelProviderServiceReq)
	}

	return cmd
}

// start create-model-service command

// Slice with functions to override default command behavior.
// Functions can be added from the `init()` function in manually curated files in this directory.
var createModelServiceOverrides []func(
	*cobra.Command,
	*catalog.CreateModelServiceRequest,
)

func newCreateModelService() *cobra.Command {
	cmd := &cobra.Command{}

	var createModelServiceReq catalog.CreateModelServiceRequest
	createModelServiceReq.ModelService = catalog.ModelService{}
	var createModelServiceJson flags.JsonFlag

	cmd.Flags().Var(&createModelServiceJson, "json", `either inline JSON string or @path/to/file.json with request body`)

	cmd.Flags().StringVar(&createModelServiceReq.ModelService.Comment, "comment", createModelServiceReq.ModelService.Comment, `User-provided description.`)
	// TODO: complex arg: config
	cmd.Flags().StringVar(&createModelServiceReq.ModelService.Name, "name", createModelServiceReq.ModelService.Name, `Resource name of the model service.`)
	cmd.Flags().StringVar(&createModelServiceReq.ModelService.Owner, "owner", createModelServiceReq.ModelService.Owner, `The owner of the model service.`)
	// TODO: array: supported_api_types

	cmd.Use = "create-model-service PARENT MODEL_SERVICE_ID"
	cmd.Short = `*Beta* Create a model service.`
	cmd.Long = `This command is in Beta and may change without notice.

Create a model service.

  Creates a model service in a Unity Catalog schema. A model service is a
  governed AI Gateway endpoint that routes inference requests to one or more
  model destinations. The caller supplies the leaf name in model_service_id.

  You must be the owner of the parent schema or have the CREATE_SERVICE and
  USE_SCHEMA privileges on the parent schema and USE_CATALOG on the parent
  catalog.

  Arguments:
    PARENT: Name of the parent schema. Format: schemas/{catalog}.{schema}. Each
      {...} component is capped at 255 characters individually.
    MODEL_SERVICE_ID: Name for the model service, e.g. "my_model_service".`

	cmd.Annotations = make(map[string]string)
	cmd.Annotations["launch_stage"] = "PUBLIC_BETA"
	cmd.Annotations["launch_stage_display"] = "Beta"

	cmd.Args = func(cmd *cobra.Command, args []string) error {
		check := root.ExactArgs(2)
		return check(cmd, args)
	}

	cmd.PreRunE = root.MustWorkspaceClient
	cmd.RunE = func(cmd *cobra.Command, args []string) (err error) {
		ctx := cmd.Context()
		w := cmdctx.WorkspaceClient(ctx)

		if cmd.Flags().Changed("json") {
			diags := createModelServiceJson.Unmarshal(&createModelServiceReq.ModelService)
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
		createModelServiceReq.Parent = args[0]
		createModelServiceReq.ModelServiceId = args[1]

		response, err := w.AiGateway.CreateModelService(ctx, createModelServiceReq)
		if err != nil {
			return err
		}

		return cmdio.Render(ctx, response)
	}

	// Disable completions since they are not applicable.
	// Can be overridden by manual implementation in `override.go`.
	cmd.ValidArgsFunction = cobra.NoFileCompletions

	// Apply optional overrides to this command.
	for _, fn := range createModelServiceOverrides {
		fn(cmd, &createModelServiceReq)
	}

	return cmd
}

// start delete-mcp-service command

// Slice with functions to override default command behavior.
// Functions can be added from the `init()` function in manually curated files in this directory.
var deleteMcpServiceOverrides []func(
	*cobra.Command,
	*catalog.DeleteMcpServiceRequest,
)

func newDeleteMcpService() *cobra.Command {
	cmd := &cobra.Command{}

	var deleteMcpServiceReq catalog.DeleteMcpServiceRequest

	cmd.Flags().StringVar(&deleteMcpServiceReq.Etag, "etag", deleteMcpServiceReq.Etag, `If-match precondition: when set, the delete proceeds only if the current server-side etag matches.`)

	cmd.Use = "delete-mcp-service NAME"
	cmd.Short = `*Beta* Delete an MCP service.`
	cmd.Long = `This command is in Beta and may change without notice.

Delete an MCP service.

  Deletes the MCP service identified by its resource name. Optionally supply an
  etag to make the delete conditional on the MCP service not having changed
  since it was read.

  You must be the owner of the MCP service or have MANAGE on it, plus
  USE_CATALOG on the parent catalog and USE_SCHEMA on the parent schema.

  Arguments:
    NAME: Resource name of the MCP service. Format:
      mcp-services/{catalog}.{schema}.{mcp_service}. Each {...} component is
      capped at 255 characters individually.`

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

		deleteMcpServiceReq.Name = args[0]

		err = w.AiGateway.DeleteMcpService(ctx, deleteMcpServiceReq)
		if err != nil {
			return err
		}
		return nil
	}

	// Disable completions since they are not applicable.
	// Can be overridden by manual implementation in `override.go`.
	cmd.ValidArgsFunction = cobra.NoFileCompletions

	// Apply optional overrides to this command.
	for _, fn := range deleteMcpServiceOverrides {
		fn(cmd, &deleteMcpServiceReq)
	}

	return cmd
}

// start delete-model-provider-service command

// Slice with functions to override default command behavior.
// Functions can be added from the `init()` function in manually curated files in this directory.
var deleteModelProviderServiceOverrides []func(
	*cobra.Command,
	*catalog.DeleteModelProviderServiceRequest,
)

func newDeleteModelProviderService() *cobra.Command {
	cmd := &cobra.Command{}

	var deleteModelProviderServiceReq catalog.DeleteModelProviderServiceRequest

	cmd.Flags().StringVar(&deleteModelProviderServiceReq.Etag, "etag", deleteModelProviderServiceReq.Etag, `If-match precondition: when set, the delete proceeds only if the current server-side etag matches.`)

	cmd.Use = "delete-model-provider-service NAME"
	cmd.Short = `*Beta* Delete a model provider service.`
	cmd.Long = `This command is in Beta and may change without notice.

Delete a model provider service.

  Deletes the model provider service identified by its resource name. Optionally
  supply an etag to make the delete conditional on the model provider service
  not having changed since it was read.

  You must be the owner of the model provider service or have MANAGE on it,
  plus USE_CATALOG on the parent catalog and USE_SCHEMA on the parent
  schema.

  Arguments:
    NAME: Resource name of the model provider service. Format:
      model-provider-services/{catalog}.{schema}.{model_provider_service}.
      Each {...} component is capped at 255 characters individually.`

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

		deleteModelProviderServiceReq.Name = args[0]

		err = w.AiGateway.DeleteModelProviderService(ctx, deleteModelProviderServiceReq)
		if err != nil {
			return err
		}
		return nil
	}

	// Disable completions since they are not applicable.
	// Can be overridden by manual implementation in `override.go`.
	cmd.ValidArgsFunction = cobra.NoFileCompletions

	// Apply optional overrides to this command.
	for _, fn := range deleteModelProviderServiceOverrides {
		fn(cmd, &deleteModelProviderServiceReq)
	}

	return cmd
}

// start delete-model-service command

// Slice with functions to override default command behavior.
// Functions can be added from the `init()` function in manually curated files in this directory.
var deleteModelServiceOverrides []func(
	*cobra.Command,
	*catalog.DeleteModelServiceRequest,
)

func newDeleteModelService() *cobra.Command {
	cmd := &cobra.Command{}

	var deleteModelServiceReq catalog.DeleteModelServiceRequest

	cmd.Flags().StringVar(&deleteModelServiceReq.Etag, "etag", deleteModelServiceReq.Etag, `If-match precondition: when set, the delete proceeds only if the current server-side etag matches.`)

	cmd.Use = "delete-model-service NAME"
	cmd.Short = `*Beta* Delete a model service.`
	cmd.Long = `This command is in Beta and may change without notice.

Delete a model service.

  Deletes the model service identified by its resource name. Optionally supply
  an etag to make the delete conditional on the model service not having
  changed since it was read.

  You must be the owner of the model service or have MANAGE on it, plus
  USE_CATALOG on the parent catalog and USE_SCHEMA on the parent schema.

  Arguments:
    NAME: Resource name of the model service. Format:
      model-services/{catalog}.{schema}.{model_service}. Each {...}
      component is capped at 255 characters individually.`

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

		deleteModelServiceReq.Name = args[0]

		err = w.AiGateway.DeleteModelService(ctx, deleteModelServiceReq)
		if err != nil {
			return err
		}
		return nil
	}

	// Disable completions since they are not applicable.
	// Can be overridden by manual implementation in `override.go`.
	cmd.ValidArgsFunction = cobra.NoFileCompletions

	// Apply optional overrides to this command.
	for _, fn := range deleteModelServiceOverrides {
		fn(cmd, &deleteModelServiceReq)
	}

	return cmd
}

// start get-mcp-service command

// Slice with functions to override default command behavior.
// Functions can be added from the `init()` function in manually curated files in this directory.
var getMcpServiceOverrides []func(
	*cobra.Command,
	*catalog.GetMcpServiceRequest,
)

func newGetMcpService() *cobra.Command {
	cmd := &cobra.Command{}

	var getMcpServiceReq catalog.GetMcpServiceRequest

	cmd.Use = "get-mcp-service NAME"
	cmd.Short = `*Beta* Get an MCP service.`
	cmd.Long = `This command is in Beta and may change without notice.

Get an MCP service.

  Returns the MCP service identified by its resource name.

  You must be the owner of the MCP service or have EXECUTE, READ_METADATA,
  or MANAGE on it, plus USE_CATALOG on the parent catalog and USE_SCHEMA
  on the parent schema.

  Arguments:
    NAME: Resource name of the MCP service. Format:
      mcp-services/{catalog}.{schema}.{mcp_service}. Each {...} component is
      capped at 255 characters individually.`

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

		getMcpServiceReq.Name = args[0]

		response, err := w.AiGateway.GetMcpService(ctx, getMcpServiceReq)
		if err != nil {
			return err
		}

		return cmdio.Render(ctx, response)
	}

	// Disable completions since they are not applicable.
	// Can be overridden by manual implementation in `override.go`.
	cmd.ValidArgsFunction = cobra.NoFileCompletions

	// Apply optional overrides to this command.
	for _, fn := range getMcpServiceOverrides {
		fn(cmd, &getMcpServiceReq)
	}

	return cmd
}

// start get-model-provider-service command

// Slice with functions to override default command behavior.
// Functions can be added from the `init()` function in manually curated files in this directory.
var getModelProviderServiceOverrides []func(
	*cobra.Command,
	*catalog.GetModelProviderServiceRequest,
)

func newGetModelProviderService() *cobra.Command {
	cmd := &cobra.Command{}

	var getModelProviderServiceReq catalog.GetModelProviderServiceRequest

	cmd.Use = "get-model-provider-service NAME"
	cmd.Short = `*Beta* Get a model provider service.`
	cmd.Long = `This command is in Beta and may change without notice.

Get a model provider service.

  Returns the model provider service identified by its resource name.

  You must be the owner of the model provider service or have EXECUTE,
  READ_METADATA, or MANAGE on it, plus USE_CATALOG on the parent catalog
  and USE_SCHEMA on the parent schema.

  Arguments:
    NAME: Resource name of the model provider service. Format:
      model-provider-services/{catalog}.{schema}.{model_provider_service}.
      Each {...} component is capped at 255 characters individually.`

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

		getModelProviderServiceReq.Name = args[0]

		response, err := w.AiGateway.GetModelProviderService(ctx, getModelProviderServiceReq)
		if err != nil {
			return err
		}

		return cmdio.Render(ctx, response)
	}

	// Disable completions since they are not applicable.
	// Can be overridden by manual implementation in `override.go`.
	cmd.ValidArgsFunction = cobra.NoFileCompletions

	// Apply optional overrides to this command.
	for _, fn := range getModelProviderServiceOverrides {
		fn(cmd, &getModelProviderServiceReq)
	}

	return cmd
}

// start get-model-service command

// Slice with functions to override default command behavior.
// Functions can be added from the `init()` function in manually curated files in this directory.
var getModelServiceOverrides []func(
	*cobra.Command,
	*catalog.GetModelServiceRequest,
)

func newGetModelService() *cobra.Command {
	cmd := &cobra.Command{}

	var getModelServiceReq catalog.GetModelServiceRequest

	cmd.Use = "get-model-service NAME"
	cmd.Short = `*Beta* Get a model service.`
	cmd.Long = `This command is in Beta and may change without notice.

Get a model service.

  Returns the model service identified by its resource name.

  You must be the owner of the model service or have EXECUTE, READ_METADATA,
  or MANAGE on it, plus USE_CATALOG on the parent catalog and USE_SCHEMA
  on the parent schema.

  Arguments:
    NAME: Resource name of the model service. Format:
      model-services/{catalog}.{schema}.{model_service}. Each {...}
      component is capped at 255 characters individually.`

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

		getModelServiceReq.Name = args[0]

		response, err := w.AiGateway.GetModelService(ctx, getModelServiceReq)
		if err != nil {
			return err
		}

		return cmdio.Render(ctx, response)
	}

	// Disable completions since they are not applicable.
	// Can be overridden by manual implementation in `override.go`.
	cmd.ValidArgsFunction = cobra.NoFileCompletions

	// Apply optional overrides to this command.
	for _, fn := range getModelServiceOverrides {
		fn(cmd, &getModelServiceReq)
	}

	return cmd
}

// start list-mcp-services command

// Slice with functions to override default command behavior.
// Functions can be added from the `init()` function in manually curated files in this directory.
var listMcpServicesOverrides []func(
	*cobra.Command,
	*catalog.ListMcpServicesRequest,
)

func newListMcpServices() *cobra.Command {
	cmd := &cobra.Command{}

	var listMcpServicesReq catalog.ListMcpServicesRequest
	// Registered for all paginated methods. Validated at call time in the
	// method-call template. Paginated list methods never have Wait or LRO
	// branches, so the method-call path is always reached.
	var listMcpServicesLimit int

	cmd.Flags().IntVar(&listMcpServicesReq.PageSize, "page-size", listMcpServicesReq.PageSize, `Maximum number of MCP services to return.`)
	cmd.Flags().StringVar(&listMcpServicesReq.Parent, "parent", listMcpServicesReq.Parent, `Name of the parent schema to list within, as schemas/{catalog}.{schema}.`)
	cmd.Flags().Var(&listMcpServicesReq.View, "view", `View selector controlling which fields are populated per row. Supported values: [BASIC, FULL]`)

	// Limit flag for total result capping.
	cmd.Flags().IntVar(&listMcpServicesLimit, "limit", 0, `Maximum number of results to return.`)

	// Hidden pagination flags (internal API parameters).
	cmd.Flags().StringVar(&listMcpServicesReq.PageToken, "page-token", listMcpServicesReq.PageToken, `Pagination token.`)
	cmd.Flags().Lookup("page-token").Hidden = true

	cmd.Use = "list-mcp-services"
	cmd.Short = `*Beta* List MCP services.`
	cmd.Long = `This command is in Beta and may change without notice.

List MCP services.

  Lists the MCP services in a Unity Catalog schema. Provide parent as
  schemas/{catalog}.{schema}. Results are paginated; pass the returned
  next_page_token to fetch subsequent pages.

  Requires USE_CATALOG on the parent catalog and USE_SCHEMA on the parent
  schema. Only MCP services the caller can access (as owner or through
  EXECUTE, READ_METADATA, or MANAGE) are returned.`

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

		response := w.AiGateway.ListMcpServices(ctx, listMcpServicesReq)
		if listMcpServicesLimit < 0 {
			return fmt.Errorf("--limit must be a non-negative integer, got %d", listMcpServicesLimit)
		}
		if listMcpServicesLimit > 0 {
			ctx = cmdio.WithLimit(ctx, listMcpServicesLimit)
		}

		return cmdio.RenderIterator(ctx, response)
	}

	// Disable completions since they are not applicable.
	// Can be overridden by manual implementation in `override.go`.
	cmd.ValidArgsFunction = cobra.NoFileCompletions

	// Apply optional overrides to this command.
	for _, fn := range listMcpServicesOverrides {
		fn(cmd, &listMcpServicesReq)
	}

	return cmd
}

// start list-model-provider-services command

// Slice with functions to override default command behavior.
// Functions can be added from the `init()` function in manually curated files in this directory.
var listModelProviderServicesOverrides []func(
	*cobra.Command,
	*catalog.ListModelProviderServicesRequest,
)

func newListModelProviderServices() *cobra.Command {
	cmd := &cobra.Command{}

	var listModelProviderServicesReq catalog.ListModelProviderServicesRequest
	// Registered for all paginated methods. Validated at call time in the
	// method-call template. Paginated list methods never have Wait or LRO
	// branches, so the method-call path is always reached.
	var listModelProviderServicesLimit int

	cmd.Flags().IntVar(&listModelProviderServicesReq.PageSize, "page-size", listModelProviderServicesReq.PageSize, `Maximum number of provider services to return.`)
	cmd.Flags().StringVar(&listModelProviderServicesReq.Parent, "parent", listModelProviderServicesReq.Parent, `Name of the parent schema to list within, as schemas/{catalog}.{schema}.`)
	cmd.Flags().Var(&listModelProviderServicesReq.View, "view", `View selector controlling which fields are populated per row. Supported values: [BASIC, FULL]`)

	// Limit flag for total result capping.
	cmd.Flags().IntVar(&listModelProviderServicesLimit, "limit", 0, `Maximum number of results to return.`)

	// Hidden pagination flags (internal API parameters).
	cmd.Flags().StringVar(&listModelProviderServicesReq.PageToken, "page-token", listModelProviderServicesReq.PageToken, `Pagination token.`)
	cmd.Flags().Lookup("page-token").Hidden = true

	cmd.Use = "list-model-provider-services"
	cmd.Short = `*Beta* List model provider services.`
	cmd.Long = `This command is in Beta and may change without notice.

List model provider services.

  Lists the model provider services in a Unity Catalog schema. Provide parent
  as schemas/{catalog}.{schema}. Results are paginated; pass the returned
  next_page_token to fetch subsequent pages.

  Requires USE_CATALOG on the parent catalog and USE_SCHEMA on the parent
  schema. Only model provider services the caller can access (as owner or
  through EXECUTE, READ_METADATA, or MANAGE) are returned.`

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

		response := w.AiGateway.ListModelProviderServices(ctx, listModelProviderServicesReq)
		if listModelProviderServicesLimit < 0 {
			return fmt.Errorf("--limit must be a non-negative integer, got %d", listModelProviderServicesLimit)
		}
		if listModelProviderServicesLimit > 0 {
			ctx = cmdio.WithLimit(ctx, listModelProviderServicesLimit)
		}

		return cmdio.RenderIterator(ctx, response)
	}

	// Disable completions since they are not applicable.
	// Can be overridden by manual implementation in `override.go`.
	cmd.ValidArgsFunction = cobra.NoFileCompletions

	// Apply optional overrides to this command.
	for _, fn := range listModelProviderServicesOverrides {
		fn(cmd, &listModelProviderServicesReq)
	}

	return cmd
}

// start list-model-services command

// Slice with functions to override default command behavior.
// Functions can be added from the `init()` function in manually curated files in this directory.
var listModelServicesOverrides []func(
	*cobra.Command,
	*catalog.ListModelServicesRequest,
)

func newListModelServices() *cobra.Command {
	cmd := &cobra.Command{}

	var listModelServicesReq catalog.ListModelServicesRequest
	// Registered for all paginated methods. Validated at call time in the
	// method-call template. Paginated list methods never have Wait or LRO
	// branches, so the method-call path is always reached.
	var listModelServicesLimit int

	cmd.Flags().IntVar(&listModelServicesReq.PageSize, "page-size", listModelServicesReq.PageSize, `Maximum number of model services to return.`)
	cmd.Flags().StringVar(&listModelServicesReq.Parent, "parent", listModelServicesReq.Parent, `Name of the parent schema to list within, as schemas/{catalog}.{schema}.`)
	cmd.Flags().Var(&listModelServicesReq.View, "view", `View selector controlling which fields are populated per row. Supported values: [BASIC, FULL]`)

	// Limit flag for total result capping.
	cmd.Flags().IntVar(&listModelServicesLimit, "limit", 0, `Maximum number of results to return.`)

	// Hidden pagination flags (internal API parameters).
	cmd.Flags().StringVar(&listModelServicesReq.PageToken, "page-token", listModelServicesReq.PageToken, `Pagination token.`)
	cmd.Flags().Lookup("page-token").Hidden = true

	cmd.Use = "list-model-services"
	cmd.Short = `*Beta* List model services.`
	cmd.Long = `This command is in Beta and may change without notice.

List model services.

  Lists the model services in a Unity Catalog schema. Provide parent as
  schemas/{catalog}.{schema}. Results are paginated; pass the returned
  next_page_token to fetch subsequent pages.

  Requires USE_CATALOG on the parent catalog and USE_SCHEMA on the parent
  schema. Only model services the caller can access (as owner or through
  EXECUTE, READ_METADATA, or MANAGE) are returned.`

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

		response := w.AiGateway.ListModelServices(ctx, listModelServicesReq)
		if listModelServicesLimit < 0 {
			return fmt.Errorf("--limit must be a non-negative integer, got %d", listModelServicesLimit)
		}
		if listModelServicesLimit > 0 {
			ctx = cmdio.WithLimit(ctx, listModelServicesLimit)
		}

		return cmdio.RenderIterator(ctx, response)
	}

	// Disable completions since they are not applicable.
	// Can be overridden by manual implementation in `override.go`.
	cmd.ValidArgsFunction = cobra.NoFileCompletions

	// Apply optional overrides to this command.
	for _, fn := range listModelServicesOverrides {
		fn(cmd, &listModelServicesReq)
	}

	return cmd
}

// start update-mcp-service command

// Slice with functions to override default command behavior.
// Functions can be added from the `init()` function in manually curated files in this directory.
var updateMcpServiceOverrides []func(
	*cobra.Command,
	*catalog.UpdateMcpServiceRequest,
)

func newUpdateMcpService() *cobra.Command {
	cmd := &cobra.Command{}

	var updateMcpServiceReq catalog.UpdateMcpServiceRequest
	updateMcpServiceReq.McpService = catalog.McpService{}
	var updateMcpServiceJson flags.JsonFlag

	cmd.Flags().Var(&updateMcpServiceJson, "json", `either inline JSON string or @path/to/file.json with request body`)

	cmd.Flags().StringVar(&updateMcpServiceReq.Etag, "etag", updateMcpServiceReq.Etag, `If-match precondition: when set, the update proceeds only if the current server-side etag matches.`)
	cmd.Flags().StringVar(&updateMcpServiceReq.McpService.Comment, "comment", updateMcpServiceReq.McpService.Comment, `User-provided description.`)
	// TODO: complex arg: config
	cmd.Flags().StringVar(&updateMcpServiceReq.McpService.Name, "name", updateMcpServiceReq.McpService.Name, `Resource name of the MCP service.`)
	cmd.Flags().StringVar(&updateMcpServiceReq.McpService.Owner, "owner", updateMcpServiceReq.McpService.Owner, `The owner of the MCP service.`)

	cmd.Use = "update-mcp-service NAME UPDATE_MASK"
	cmd.Short = `*Beta* Update an MCP service.`
	cmd.Long = `This command is in Beta and may change without notice.

Update an MCP service.

  Updates an MCP service. Only the fields named in update_mask are changed;
  the resource name is immutable. Optionally supply an etag to make the update
  conditional on the MCP service not having changed since it was read.

  You must be the owner of the MCP service or have MANAGE on it, plus
  USE_CATALOG on the parent catalog and USE_SCHEMA on the parent schema.

  Arguments:
    NAME: Resource name of the MCP service. Format:
      mcp-services/{catalog}.{schema}.{mcp_service}. Each {...} component is
      capped at 255 characters individually. Server-derived on Create from
      parent + mcp_service_id; required and immutable on Update/Get/Delete.
    UPDATE_MASK: The list of fields to update. The framework validates each path against
      the mcp_service field above. Wildcard paths (paths: ["*"]) are not
      supported; list each field path explicitly.`

	cmd.Annotations = make(map[string]string)
	cmd.Annotations["launch_stage"] = "PUBLIC_BETA"
	cmd.Annotations["launch_stage_display"] = "Beta"

	cmd.Args = func(cmd *cobra.Command, args []string) error {
		check := root.ExactArgs(2)
		return check(cmd, args)
	}

	cmd.PreRunE = root.MustWorkspaceClient
	cmd.RunE = func(cmd *cobra.Command, args []string) (err error) {
		ctx := cmd.Context()
		w := cmdctx.WorkspaceClient(ctx)

		if cmd.Flags().Changed("json") {
			diags := updateMcpServiceJson.Unmarshal(&updateMcpServiceReq.McpService)
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
		updateMcpServiceReq.Name = args[0]
		if args[1] != "" {
			updateMaskArray := strings.Split(args[1], ",")
			updateMcpServiceReq.UpdateMask = *fieldmask.New(updateMaskArray)
		}

		response, err := w.AiGateway.UpdateMcpService(ctx, updateMcpServiceReq)
		if err != nil {
			return err
		}

		return cmdio.Render(ctx, response)
	}

	// Disable completions since they are not applicable.
	// Can be overridden by manual implementation in `override.go`.
	cmd.ValidArgsFunction = cobra.NoFileCompletions

	// Apply optional overrides to this command.
	for _, fn := range updateMcpServiceOverrides {
		fn(cmd, &updateMcpServiceReq)
	}

	return cmd
}

// start update-model-provider-service command

// Slice with functions to override default command behavior.
// Functions can be added from the `init()` function in manually curated files in this directory.
var updateModelProviderServiceOverrides []func(
	*cobra.Command,
	*catalog.UpdateModelProviderServiceRequest,
)

func newUpdateModelProviderService() *cobra.Command {
	cmd := &cobra.Command{}

	var updateModelProviderServiceReq catalog.UpdateModelProviderServiceRequest
	updateModelProviderServiceReq.ModelProviderService = catalog.ModelProviderService{}
	var updateModelProviderServiceJson flags.JsonFlag

	cmd.Flags().Var(&updateModelProviderServiceJson, "json", `either inline JSON string or @path/to/file.json with request body`)

	cmd.Flags().StringVar(&updateModelProviderServiceReq.Etag, "etag", updateModelProviderServiceReq.Etag, `If-match precondition: when set, the update proceeds only if the current server-side etag matches.`)
	cmd.Flags().StringVar(&updateModelProviderServiceReq.ModelProviderService.Comment, "comment", updateModelProviderServiceReq.ModelProviderService.Comment, `User-provided description.`)
	// TODO: complex arg: config
	cmd.Flags().StringVar(&updateModelProviderServiceReq.ModelProviderService.Name, "name", updateModelProviderServiceReq.ModelProviderService.Name, `Resource name of the provider service.`)
	cmd.Flags().StringVar(&updateModelProviderServiceReq.ModelProviderService.Owner, "owner", updateModelProviderServiceReq.ModelProviderService.Owner, `The owner of the model provider service.`)

	cmd.Use = "update-model-provider-service NAME UPDATE_MASK"
	cmd.Short = `*Beta* Update a model provider service.`
	cmd.Long = `This command is in Beta and may change without notice.

Update a model provider service.

  Updates a model provider service. Only the fields named in update_mask are
  changed; the resource name and provider type are immutable. Optionally supply
  an etag to make the update conditional on the model provider service not
  having changed since it was read.

  You must be the owner of the model provider service or have MANAGE on it,
  plus USE_CATALOG on the parent catalog and USE_SCHEMA on the parent
  schema.

  Arguments:
    NAME: Resource name of the provider service. Format:
      model-provider-services/{catalog}.{schema}.{model_provider_service}.
      Each {...} component is capped at 255 characters individually.
      Server-derived on Create from parent + model_provider_service_id;
      required and immutable on Update/Get/Delete.
    UPDATE_MASK: The list of fields to update. The framework validates each path against
      the model_provider_service field above. Wildcard paths (paths: ["*"])
      are not supported; list each field path explicitly.`

	cmd.Annotations = make(map[string]string)
	cmd.Annotations["launch_stage"] = "PUBLIC_BETA"
	cmd.Annotations["launch_stage_display"] = "Beta"

	cmd.Args = func(cmd *cobra.Command, args []string) error {
		check := root.ExactArgs(2)
		return check(cmd, args)
	}

	cmd.PreRunE = root.MustWorkspaceClient
	cmd.RunE = func(cmd *cobra.Command, args []string) (err error) {
		ctx := cmd.Context()
		w := cmdctx.WorkspaceClient(ctx)

		if cmd.Flags().Changed("json") {
			diags := updateModelProviderServiceJson.Unmarshal(&updateModelProviderServiceReq.ModelProviderService)
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
		updateModelProviderServiceReq.Name = args[0]
		if args[1] != "" {
			updateMaskArray := strings.Split(args[1], ",")
			updateModelProviderServiceReq.UpdateMask = *fieldmask.New(updateMaskArray)
		}

		response, err := w.AiGateway.UpdateModelProviderService(ctx, updateModelProviderServiceReq)
		if err != nil {
			return err
		}

		return cmdio.Render(ctx, response)
	}

	// Disable completions since they are not applicable.
	// Can be overridden by manual implementation in `override.go`.
	cmd.ValidArgsFunction = cobra.NoFileCompletions

	// Apply optional overrides to this command.
	for _, fn := range updateModelProviderServiceOverrides {
		fn(cmd, &updateModelProviderServiceReq)
	}

	return cmd
}

// start update-model-service command

// Slice with functions to override default command behavior.
// Functions can be added from the `init()` function in manually curated files in this directory.
var updateModelServiceOverrides []func(
	*cobra.Command,
	*catalog.UpdateModelServiceRequest,
)

func newUpdateModelService() *cobra.Command {
	cmd := &cobra.Command{}

	var updateModelServiceReq catalog.UpdateModelServiceRequest
	updateModelServiceReq.ModelService = catalog.ModelService{}
	var updateModelServiceJson flags.JsonFlag

	cmd.Flags().Var(&updateModelServiceJson, "json", `either inline JSON string or @path/to/file.json with request body`)

	cmd.Flags().StringVar(&updateModelServiceReq.Etag, "etag", updateModelServiceReq.Etag, `If-match precondition: when set, the update proceeds only if the current server-side etag matches.`)
	cmd.Flags().StringVar(&updateModelServiceReq.ModelService.Comment, "comment", updateModelServiceReq.ModelService.Comment, `User-provided description.`)
	// TODO: complex arg: config
	cmd.Flags().StringVar(&updateModelServiceReq.ModelService.Name, "name", updateModelServiceReq.ModelService.Name, `Resource name of the model service.`)
	cmd.Flags().StringVar(&updateModelServiceReq.ModelService.Owner, "owner", updateModelServiceReq.ModelService.Owner, `The owner of the model service.`)
	// TODO: array: supported_api_types

	cmd.Use = "update-model-service NAME UPDATE_MASK"
	cmd.Short = `*Beta* Update a model service.`
	cmd.Long = `This command is in Beta and may change without notice.

Update a model service.

  Updates a model service. Only the fields named in update_mask are changed;
  the resource name is immutable. Optionally supply an etag to make the update
  conditional on the model service not having changed since it was read.

  You must be the owner of the model service or have MANAGE on it, plus
  USE_CATALOG on the parent catalog and USE_SCHEMA on the parent schema.

  Arguments:
    NAME: Resource name of the model service. Format:
      model-services/{catalog}.{schema}.{model_service}. Each {...}
      component is capped at 255 characters individually. Server-derived on
      Create from parent + model_service_id; required and immutable on
      Update/Get/Delete.
    UPDATE_MASK: The list of fields to update. The framework validates each path against
      the model_service field above. Wildcard paths (paths: ["*"]) are not
      supported; list each field path explicitly.`

	cmd.Annotations = make(map[string]string)
	cmd.Annotations["launch_stage"] = "PUBLIC_BETA"
	cmd.Annotations["launch_stage_display"] = "Beta"

	cmd.Args = func(cmd *cobra.Command, args []string) error {
		check := root.ExactArgs(2)
		return check(cmd, args)
	}

	cmd.PreRunE = root.MustWorkspaceClient
	cmd.RunE = func(cmd *cobra.Command, args []string) (err error) {
		ctx := cmd.Context()
		w := cmdctx.WorkspaceClient(ctx)

		if cmd.Flags().Changed("json") {
			diags := updateModelServiceJson.Unmarshal(&updateModelServiceReq.ModelService)
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
		updateModelServiceReq.Name = args[0]
		if args[1] != "" {
			updateMaskArray := strings.Split(args[1], ",")
			updateModelServiceReq.UpdateMask = *fieldmask.New(updateMaskArray)
		}

		response, err := w.AiGateway.UpdateModelService(ctx, updateModelServiceReq)
		if err != nil {
			return err
		}

		return cmdio.Render(ctx, response)
	}

	// Disable completions since they are not applicable.
	// Can be overridden by manual implementation in `override.go`.
	cmd.ValidArgsFunction = cobra.NoFileCompletions

	// Apply optional overrides to this command.
	for _, fn := range updateModelServiceOverrides {
		fn(cmd, &updateModelServiceReq)
	}

	return cmd
}

// end service AiGateway
