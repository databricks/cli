// Code generated from OpenAPI specs by Databricks SDK Generator. DO NOT EDIT.

package ai_gateway

import (
	"fmt"

	"github.com/databricks/cli/cmd/root"
	"github.com/databricks/cli/libs/cmdctx"
	"github.com/databricks/cli/libs/cmdgen"
	"github.com/databricks/cli/libs/cmdio"
	"github.com/databricks/cli/libs/flags"
	"github.com/spf13/cobra"

	aigateway "github.com/databricks/cli/libs/sdktypes/aigateway/v1"
	sdkaigateway "github.com/databricks/sdk-go/aigateway/v1"
)

// Slice with functions to override default command behavior.
// Functions can be added from the `init()` function in manually curated files in this directory.
var cmdOverrides []func(*cobra.Command)

func New() *cobra.Command {
	cmd := &cobra.Command{
		Use:   "ai-gateway",
		Short: `Govern AI workloads in Unity Catalog.`,
		Long: `Govern AI workloads in Unity Catalog. This API manages the Unity Catalog
  securables that bring centralized access control, lineage, and auditing to
  AI-serving entities: model services (governed access to foundation models and
  external LLMs), model provider services (governed resources for external model
  providers), and MCP services (governed Model Context Protocol servers).`,
		GroupID: "catalog",
		RunE:    root.ReportUnknownSubcommand,
	}

	cmd.Annotations = make(map[string]string)
	cmd.Annotations["launch_stage"] = "GA"
	cmd.Annotations["launch_stage_display"] = "GA"

	// Add methods
	cmd.AddCommand(newCreateMcpService())
	cmd.AddCommand(newCreateMcpServiceUserMappedCredential())
	cmd.AddCommand(newCreateModelProviderService())
	cmd.AddCommand(newCreateModelService())
	cmd.AddCommand(newCreateSkill())
	cmd.AddCommand(newDeleteMcpService())
	cmd.AddCommand(newDeleteMcpServiceUserMappedCredential())
	cmd.AddCommand(newDeleteModelProviderService())
	cmd.AddCommand(newDeleteModelService())
	cmd.AddCommand(newDeleteSkill())
	cmd.AddCommand(newFinalizeSkill())
	cmd.AddCommand(newGetMcpService())
	cmd.AddCommand(newGetMcpServiceUserMappedCredential())
	cmd.AddCommand(newGetModelProviderService())
	cmd.AddCommand(newGetModelService())
	cmd.AddCommand(newGetSkill())
	cmd.AddCommand(newListMcpServices())
	cmd.AddCommand(newListModelProviderServices())
	cmd.AddCommand(newListModelServices())
	cmd.AddCommand(newListSkills())
	cmd.AddCommand(newUpdateMcpService())
	cmd.AddCommand(newUpdateModelProviderService())
	cmd.AddCommand(newUpdateModelService())
	cmd.AddCommand(newUpdateSkill())

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
	*aigateway.CreateMcpServiceRequest,
)

func newCreateMcpService() *cobra.Command {
	cmd := &cobra.Command{}
	var createMcpServiceReq aigateway.CreateMcpServiceRequest
	createMcpServiceReq.McpService = &aigateway.McpService{}

	var createMcpServiceJson flags.JsonFlag

	cmd.Flags().Var(&createMcpServiceJson, "json", `either inline JSON string or @path/to/file.json with request body`)

	flags.StringPtrVar(cmd.Flags(), &createMcpServiceReq.McpService.Comment, "comment", `User-provided description.`)
	// TODO: complex arg: config
	flags.StringPtrVar(cmd.Flags(), &createMcpServiceReq.McpService.Name, "name", `Resource name of the MCP service.`)

	cmd.Use = "create-mcp-service PARENT MCP_SERVICE_ID"
	cmd.Short = `Create an MCP service.`
	cmd.Long = `Create an MCP service.

  Creates an MCP service in a Unity Catalog schema. An MCP (Model Context
  Protocol) service is a governed securable that registers an MCP server and
  exposes its tools for discovery, access control, and invocation. Specify its
  name in mcp_service_id.

  You must be the owner of the parent schema or have the CREATE_SERVICE and
  USE_SCHEMA privileges on the parent schema and USE_CATALOG on the parent
  catalog. You also need USE_CONNECTION on the connection the MCP service
  references.

  Arguments:
    PARENT: Name of the parent schema. Format: schemas/{catalog}.{schema}. Each
      {...} component is capped at 255 characters individually.
    MCP_SERVICE_ID: Name for the MCP service, e.g. "my_mcp_service".`

	cmd.Annotations = make(map[string]string)
	cmd.Annotations["launch_stage"] = "GA"
	cmd.Annotations["launch_stage_display"] = "GA"

	cmd.Args = func(cmd *cobra.Command, args []string) error {
		check := root.ExactArgs(2)
		return check(cmd, args)
	}

	cmd.PreRunE = root.MustWorkspaceClient
	cmd.RunE = func(cmd *cobra.Command, args []string) (err error) {
		ctx := cmd.Context()
		w := cmdctx.WorkspaceClient(ctx)
		_ = w

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
		createMcpServiceReq.Parent = new(args[0])
		createMcpServiceReq.McpServiceId = new(args[1])

		sdkReq, err := aigateway.CreateMcpServiceRequestToSDK(&createMcpServiceReq)
		if err != nil {
			return err
		}
		response, err := w.AiGateway.CreateMcpService(ctx, *sdkReq)
		if err != nil {
			return err
		}

		out, err := aigateway.McpServiceFromSDK(response)
		if err != nil {
			return err
		}
		return cmdio.Render(ctx, out)
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

// start create-mcp-service-user-mapped-credential command

// Slice with functions to override default command behavior.
// Functions can be added from the `init()` function in manually curated files in this directory.
var createMcpServiceUserMappedCredentialOverrides []func(
	*cobra.Command,
	*aigateway.CreateMcpServiceUserMappedCredentialRequest,
)

func newCreateMcpServiceUserMappedCredential() *cobra.Command {
	cmd := &cobra.Command{}
	var createMcpServiceUserMappedCredentialReq aigateway.CreateMcpServiceUserMappedCredentialRequest
	createMcpServiceUserMappedCredentialReq.Login = &aigateway.McpServiceUserMappedCredentialLogin{}

	var createMcpServiceUserMappedCredentialJson flags.JsonFlag

	cmd.Flags().Var(&createMcpServiceUserMappedCredentialJson, "json", `either inline JSON string or @path/to/file.json with request body`)

	// TODO: map via StringToStringVar: options

	cmd.Use = "create-mcp-service-user-mapped-credential NAME"
	cmd.Short = `*Beta* Create an MCP service user credential.`
	cmd.Long = `This command is in Beta and may change without notice.

Create an MCP service user credential.

  Logs the caller in to an MCP service: creates their per-user OAuth credential,
  or re-authenticates it if one already exists. The request body carries the
  OAuth exchange fields.

  You must be the owner of the MCP service or have EXECUTE on it, plus
  USE_CATALOG on the parent catalog and USE_SCHEMA on the parent schema.

  Arguments:
    NAME: Resource name of the MCP service. Format:
      mcp-services/{catalog}.{schema}.{mcp_service}.`

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
			diags := createMcpServiceUserMappedCredentialJson.Unmarshal(&createMcpServiceUserMappedCredentialReq.Login)
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
		createMcpServiceUserMappedCredentialReq.Name = new(args[0])

		sdkReq, err := aigateway.CreateMcpServiceUserMappedCredentialRequestToSDK(&createMcpServiceUserMappedCredentialReq)
		if err != nil {
			return err
		}
		response, err := w.AiGateway.CreateMcpServiceUserMappedCredential(ctx, *sdkReq)
		if err != nil {
			return err
		}

		out, err := aigateway.McpServiceUserMappedCredentialFromSDK(response)
		if err != nil {
			return err
		}
		return cmdio.Render(ctx, out)
	}

	// Disable completions since they are not applicable.
	// Can be overridden by manual implementation in `override.go`.
	cmd.ValidArgsFunction = cobra.NoFileCompletions

	// Apply optional overrides to this command.
	for _, fn := range createMcpServiceUserMappedCredentialOverrides {
		fn(cmd, &createMcpServiceUserMappedCredentialReq)
	}

	return cmd
}

// start create-model-provider-service command

// Slice with functions to override default command behavior.
// Functions can be added from the `init()` function in manually curated files in this directory.
var createModelProviderServiceOverrides []func(
	*cobra.Command,
	*aigateway.CreateModelProviderServiceRequest,
)

func newCreateModelProviderService() *cobra.Command {
	cmd := &cobra.Command{}
	var createModelProviderServiceReq aigateway.CreateModelProviderServiceRequest
	createModelProviderServiceReq.ModelProviderService = &aigateway.ModelProviderService{}

	var createModelProviderServiceJson flags.JsonFlag

	cmd.Flags().Var(&createModelProviderServiceJson, "json", `either inline JSON string or @path/to/file.json with request body`)

	flags.StringPtrVar(cmd.Flags(), &createModelProviderServiceReq.ModelProviderService.Comment, "comment", `User-provided description.`)
	// TODO: complex arg: config
	flags.StringPtrVar(cmd.Flags(), &createModelProviderServiceReq.ModelProviderService.Name, "name", `Resource name of the provider service.`)

	cmd.Use = "create-model-provider-service PARENT MODEL_PROVIDER_SERVICE_ID"
	cmd.Short = `Create a model provider service.`
	cmd.Long = `Create a model provider service.

  Creates a model provider service in a Unity Catalog schema. A model provider
  service stores authentication and request configuration for an external model
  provider, such as OpenAI, Azure OpenAI, or Amazon Bedrock. Model services
  reference it to invoke the provider. Specify its name in
  model_provider_service_id.

  You must be the owner of the parent schema or have the CREATE_SERVICE and
  USE_SCHEMA privileges on the parent schema and USE_CATALOG on the parent
  catalog. Inline credentials additionally require CREATE_CONNECTION on the
  parent schema. When using a Unity Catalog service credential, you must have
  ACCESS on that credential.

  Arguments:
    PARENT: Name of the parent schema. Format: schemas/{catalog}.{schema}. Each
      {...} component is capped at 255 characters individually.
    MODEL_PROVIDER_SERVICE_ID: Name for the model provider service, e.g. "openai_prod".`

	cmd.Annotations = make(map[string]string)
	cmd.Annotations["launch_stage"] = "GA"
	cmd.Annotations["launch_stage_display"] = "GA"

	cmd.Args = func(cmd *cobra.Command, args []string) error {
		check := root.ExactArgs(2)
		return check(cmd, args)
	}

	cmd.PreRunE = root.MustWorkspaceClient
	cmd.RunE = func(cmd *cobra.Command, args []string) (err error) {
		ctx := cmd.Context()
		w := cmdctx.WorkspaceClient(ctx)
		_ = w

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
		createModelProviderServiceReq.Parent = new(args[0])
		createModelProviderServiceReq.ModelProviderServiceId = new(args[1])

		sdkReq, err := aigateway.CreateModelProviderServiceRequestToSDK(&createModelProviderServiceReq)
		if err != nil {
			return err
		}
		response, err := w.AiGateway.CreateModelProviderService(ctx, *sdkReq)
		if err != nil {
			return err
		}

		out, err := aigateway.ModelProviderServiceFromSDK(response)
		if err != nil {
			return err
		}
		return cmdio.Render(ctx, out)
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
	*aigateway.CreateModelServiceRequest,
)

func newCreateModelService() *cobra.Command {
	cmd := &cobra.Command{}
	var createModelServiceReq aigateway.CreateModelServiceRequest
	createModelServiceReq.ModelService = &aigateway.ModelService{}

	var createModelServiceJson flags.JsonFlag

	cmd.Flags().Var(&createModelServiceJson, "json", `either inline JSON string or @path/to/file.json with request body`)

	flags.StringPtrVar(cmd.Flags(), &createModelServiceReq.ModelService.Comment, "comment", `User-provided description.`)
	// TODO: complex arg: config
	flags.StringPtrVar(cmd.Flags(), &createModelServiceReq.ModelService.Name, "name", `Resource name of the model service.`)
	// TODO: array: supported_api_types

	cmd.Use = "create-model-service PARENT MODEL_SERVICE_ID"
	cmd.Short = `Create a model service.`
	cmd.Long = `Create a model service.

  Creates a model service in a Unity Catalog schema. A model service is a
  governed AI Gateway endpoint that routes inference requests to one or more
  model destinations. Specify its name in model_service_id.

  You must be the owner of the parent schema or have the CREATE_SERVICE and
  USE_SCHEMA privileges on the parent schema and USE_CATALOG on the parent
  catalog. For every destination, you also need USE_CATALOG and USE_SCHEMA
  on its parent and EXECUTE on the referenced Unity Catalog model or model
  provider service. A provisioned-throughput destination additionally requires
  CAN_MANAGE on its Model Serving endpoint. Configuring an inference table
  additionally requires CREATE_TABLE.

  Arguments:
    PARENT: Name of the parent schema. Format: schemas/{catalog}.{schema}. Each
      {...} component is capped at 255 characters individually.
    MODEL_SERVICE_ID: Name for the model service, e.g. "my_model_service".`

	cmd.Annotations = make(map[string]string)
	cmd.Annotations["launch_stage"] = "GA"
	cmd.Annotations["launch_stage_display"] = "GA"

	cmd.Args = func(cmd *cobra.Command, args []string) error {
		check := root.ExactArgs(2)
		return check(cmd, args)
	}

	cmd.PreRunE = root.MustWorkspaceClient
	cmd.RunE = func(cmd *cobra.Command, args []string) (err error) {
		ctx := cmd.Context()
		w := cmdctx.WorkspaceClient(ctx)
		_ = w

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
		createModelServiceReq.Parent = new(args[0])
		createModelServiceReq.ModelServiceId = new(args[1])

		sdkReq, err := aigateway.CreateModelServiceRequestToSDK(&createModelServiceReq)
		if err != nil {
			return err
		}
		response, err := w.AiGateway.CreateModelService(ctx, *sdkReq)
		if err != nil {
			return err
		}

		out, err := aigateway.ModelServiceFromSDK(response)
		if err != nil {
			return err
		}
		return cmdio.Render(ctx, out)
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

// start create-skill command

// Slice with functions to override default command behavior.
// Functions can be added from the `init()` function in manually curated files in this directory.
var createSkillOverrides []func(
	*cobra.Command,
	*aigateway.CreateSkillRequest,
)

func newCreateSkill() *cobra.Command {
	cmd := &cobra.Command{}
	var createSkillReq aigateway.CreateSkillRequest
	createSkillReq.Skill = &aigateway.Skill{}

	var createSkillJson flags.JsonFlag

	cmd.Flags().Var(&createSkillJson, "json", `either inline JSON string or @path/to/file.json with request body`)

	flags.StringPtrVar(cmd.Flags(), &createSkillReq.Skill.Comment, "comment", `User-provided comment for the skill.`)
	flags.StringPtrVar(cmd.Flags(), &createSkillReq.Skill.Name, "name", `Resource name of the skill.`)

	cmd.Use = "create-skill PARENT SKILL_ID"
	cmd.Short = `*Beta* Create a skill.`
	cmd.Long = `This command is in Beta and may change without notice.

Create a skill.

  Creates a skill in a Unity Catalog schema and provisions its managed bundle
  storage. Specify its name in skill_id. The request contains an optional
  comment but no bundle bytes. Upload bundle files through the Files API, then
  call FinalizeSkill.

  You must be the owner of the parent schema or have CREATE_VOLUME and
  USE_SCHEMA on it, plus USE_CATALOG on the parent catalog.

  Arguments:
    PARENT: Name of the parent schema. Format: schemas/{catalog}.{schema}. Each
      {...} component is capped at 255 characters individually.
    SKILL_ID: Name for the skill, e.g. "basic-math". The server normalizes this
      identifier to lowercase. It is independent of the bundle name read from
      SKILL.md.`

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
		_ = w

		if cmd.Flags().Changed("json") {
			diags := createSkillJson.Unmarshal(&createSkillReq.Skill)
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
		createSkillReq.Parent = new(args[0])
		createSkillReq.SkillId = new(args[1])

		sdkReq, err := aigateway.CreateSkillRequestToSDK(&createSkillReq)
		if err != nil {
			return err
		}
		response, err := w.AiGateway.CreateSkill(ctx, *sdkReq)
		if err != nil {
			return err
		}

		out, err := aigateway.SkillFromSDK(response)
		if err != nil {
			return err
		}
		return cmdio.Render(ctx, out)
	}

	// Disable completions since they are not applicable.
	// Can be overridden by manual implementation in `override.go`.
	cmd.ValidArgsFunction = cobra.NoFileCompletions

	// Apply optional overrides to this command.
	for _, fn := range createSkillOverrides {
		fn(cmd, &createSkillReq)
	}

	return cmd
}

// start delete-mcp-service command

// Slice with functions to override default command behavior.
// Functions can be added from the `init()` function in manually curated files in this directory.
var deleteMcpServiceOverrides []func(
	*cobra.Command,
	*aigateway.DeleteMcpServiceRequest,
)

func newDeleteMcpService() *cobra.Command {
	cmd := &cobra.Command{}
	var deleteMcpServiceReq aigateway.DeleteMcpServiceRequest

	flags.Base64Var(cmd.Flags(), &deleteMcpServiceReq.Etag, "etag", `Optimistic concurrency token from the most recent read.`)

	cmd.Use = "delete-mcp-service NAME"
	cmd.Short = `Delete an MCP service.`
	cmd.Long = `Delete an MCP service.

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
	cmd.Annotations["launch_stage"] = "GA"
	cmd.Annotations["launch_stage_display"] = "GA"

	cmd.Args = func(cmd *cobra.Command, args []string) error {
		check := root.ExactArgs(1)
		return check(cmd, args)
	}

	cmd.PreRunE = root.MustWorkspaceClient
	cmd.RunE = func(cmd *cobra.Command, args []string) (err error) {
		ctx := cmd.Context()
		w := cmdctx.WorkspaceClient(ctx)
		_ = w

		deleteMcpServiceReq.Name = new(args[0])

		sdkReq, err := aigateway.DeleteMcpServiceRequestToSDK(&deleteMcpServiceReq)
		if err != nil {
			return err
		}
		err = w.AiGateway.DeleteMcpService(ctx, *sdkReq)
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

// start delete-mcp-service-user-mapped-credential command

// Slice with functions to override default command behavior.
// Functions can be added from the `init()` function in manually curated files in this directory.
var deleteMcpServiceUserMappedCredentialOverrides []func(
	*cobra.Command,
	*sdkaigateway.DeleteMcpServiceUserMappedCredentialRequest,
)

func newDeleteMcpServiceUserMappedCredential() *cobra.Command {
	cmd := &cobra.Command{}
	var deleteMcpServiceUserMappedCredentialReq sdkaigateway.DeleteMcpServiceUserMappedCredentialRequest

	cmd.Use = "delete-mcp-service-user-mapped-credential NAME"
	cmd.Short = `*Beta* Delete an MCP service user credential.`
	cmd.Long = `This command is in Beta and may change without notice.

Delete an MCP service user credential.

  Revokes (deletes) the caller's per-user OAuth credential for an MCP service
  (logout).

  You must be the owner of the MCP service or have EXECUTE on it, plus
  USE_CATALOG on the parent catalog and USE_SCHEMA on the parent schema.

  Arguments:
    NAME: Resource name of the MCP service. Format:
      mcp-services/{catalog}.{schema}.{mcp_service}.`

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

		deleteMcpServiceUserMappedCredentialReq.Name = new(args[0])

		sdkReq := &deleteMcpServiceUserMappedCredentialReq
		response, err := w.AiGateway.DeleteMcpServiceUserMappedCredential(ctx, *sdkReq)
		if err != nil {
			return err
		}

		_ = response
		return nil
	}

	// Disable completions since they are not applicable.
	// Can be overridden by manual implementation in `override.go`.
	cmd.ValidArgsFunction = cobra.NoFileCompletions

	// Apply optional overrides to this command.
	for _, fn := range deleteMcpServiceUserMappedCredentialOverrides {
		fn(cmd, &deleteMcpServiceUserMappedCredentialReq)
	}

	return cmd
}

// start delete-model-provider-service command

// Slice with functions to override default command behavior.
// Functions can be added from the `init()` function in manually curated files in this directory.
var deleteModelProviderServiceOverrides []func(
	*cobra.Command,
	*aigateway.DeleteModelProviderServiceRequest,
)

func newDeleteModelProviderService() *cobra.Command {
	cmd := &cobra.Command{}
	var deleteModelProviderServiceReq aigateway.DeleteModelProviderServiceRequest

	flags.Base64Var(cmd.Flags(), &deleteModelProviderServiceReq.Etag, "etag", `Optimistic concurrency token from the most recent read.`)

	cmd.Use = "delete-model-provider-service NAME"
	cmd.Short = `Delete a model provider service.`
	cmd.Long = `Delete a model provider service.

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
	cmd.Annotations["launch_stage"] = "GA"
	cmd.Annotations["launch_stage_display"] = "GA"

	cmd.Args = func(cmd *cobra.Command, args []string) error {
		check := root.ExactArgs(1)
		return check(cmd, args)
	}

	cmd.PreRunE = root.MustWorkspaceClient
	cmd.RunE = func(cmd *cobra.Command, args []string) (err error) {
		ctx := cmd.Context()
		w := cmdctx.WorkspaceClient(ctx)
		_ = w

		deleteModelProviderServiceReq.Name = new(args[0])

		sdkReq, err := aigateway.DeleteModelProviderServiceRequestToSDK(&deleteModelProviderServiceReq)
		if err != nil {
			return err
		}
		err = w.AiGateway.DeleteModelProviderService(ctx, *sdkReq)
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
	*aigateway.DeleteModelServiceRequest,
)

func newDeleteModelService() *cobra.Command {
	cmd := &cobra.Command{}
	var deleteModelServiceReq aigateway.DeleteModelServiceRequest

	flags.Base64Var(cmd.Flags(), &deleteModelServiceReq.Etag, "etag", `Optimistic concurrency token from the most recent read.`)

	cmd.Use = "delete-model-service NAME"
	cmd.Short = `Delete a model service.`
	cmd.Long = `Delete a model service.

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
	cmd.Annotations["launch_stage"] = "GA"
	cmd.Annotations["launch_stage_display"] = "GA"

	cmd.Args = func(cmd *cobra.Command, args []string) error {
		check := root.ExactArgs(1)
		return check(cmd, args)
	}

	cmd.PreRunE = root.MustWorkspaceClient
	cmd.RunE = func(cmd *cobra.Command, args []string) (err error) {
		ctx := cmd.Context()
		w := cmdctx.WorkspaceClient(ctx)
		_ = w

		deleteModelServiceReq.Name = new(args[0])

		sdkReq, err := aigateway.DeleteModelServiceRequestToSDK(&deleteModelServiceReq)
		if err != nil {
			return err
		}
		err = w.AiGateway.DeleteModelService(ctx, *sdkReq)
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

// start delete-skill command

// Slice with functions to override default command behavior.
// Functions can be added from the `init()` function in manually curated files in this directory.
var deleteSkillOverrides []func(
	*cobra.Command,
	*aigateway.DeleteSkillRequest,
)

func newDeleteSkill() *cobra.Command {
	cmd := &cobra.Command{}
	var deleteSkillReq aigateway.DeleteSkillRequest

	flags.Base64Var(cmd.Flags(), &deleteSkillReq.Etag, "etag", `Optimistic concurrency token from the most recent read.`)

	cmd.Use = "delete-skill NAME"
	cmd.Short = `*Beta* Delete a skill.`
	cmd.Long = `This command is in Beta and may change without notice.

Delete a skill.

  Deletes the skill identified by its resource name and makes its managed bundle
  path unavailable. Managed bundle data is deleted asynchronously. Optionally
  supply an etag to make the delete conditional on the skill not having
  changed since it was read.

  You must be the owner of the skill or have MANAGE on it, plus USE_CATALOG
  on the parent catalog and USE_SCHEMA on the parent schema.

  Arguments:
    NAME: Full resource name of the skill. Format:
      skills/{catalog}.{schema}.{skill}. Each {...} component is capped at
      255 characters individually.`

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

		deleteSkillReq.Name = new(args[0])

		sdkReq, err := aigateway.DeleteSkillRequestToSDK(&deleteSkillReq)
		if err != nil {
			return err
		}
		err = w.AiGateway.DeleteSkill(ctx, *sdkReq)
		if err != nil {
			return err
		}
		return nil
	}

	// Disable completions since they are not applicable.
	// Can be overridden by manual implementation in `override.go`.
	cmd.ValidArgsFunction = cobra.NoFileCompletions

	// Apply optional overrides to this command.
	for _, fn := range deleteSkillOverrides {
		fn(cmd, &deleteSkillReq)
	}

	return cmd
}

// start finalize-skill command

// Slice with functions to override default command behavior.
// Functions can be added from the `init()` function in manually curated files in this directory.
var finalizeSkillOverrides []func(
	*cobra.Command,
	*sdkaigateway.FinalizeSkillRequest,
)

func newFinalizeSkill() *cobra.Command {
	cmd := &cobra.Command{}
	var finalizeSkillReq sdkaigateway.FinalizeSkillRequest

	cmd.Use = "finalize-skill NAME"
	cmd.Short = `*Beta* Finalize a skill.`
	cmd.Long = `This command is in Beta and may change without notice.

Finalize a skill.

  Finalizes a skill after its bundle is uploaded. This method reads SKILL.md
  through the Files API using the caller's authorization. Its YAML frontmatter
  must contain an agentskills.io-compliant name and a nonblank description
  within the configured UTF-8 byte limit. On success, it replaces bundle_name
  and description; refreshes finalize_time, update_time, and updated_by;
  and returns the updated skill. comment is preserved. Re-finalization uses
  the latest SKILL.md and is last-write-wins without an etag precondition.
  Validation failures do not change metadata.

  You must be the owner of the skill or have READ_VOLUME on it, plus
  USE_CATALOG on the parent catalog and USE_SCHEMA on the parent schema.

  Arguments:
    NAME: Full resource name of the skill. Format:
      skills/{catalog}.{schema}.{skill}. Each {...} component is capped at
      255 characters individually.`

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

		finalizeSkillReq.Name = new(args[0])

		sdkReq := &finalizeSkillReq
		response, err := w.AiGateway.FinalizeSkill(ctx, *sdkReq)
		if err != nil {
			return err
		}

		out, err := aigateway.SkillFromSDK(response)
		if err != nil {
			return err
		}
		return cmdio.Render(ctx, out)
	}

	// Disable completions since they are not applicable.
	// Can be overridden by manual implementation in `override.go`.
	cmd.ValidArgsFunction = cobra.NoFileCompletions

	// Apply optional overrides to this command.
	for _, fn := range finalizeSkillOverrides {
		fn(cmd, &finalizeSkillReq)
	}

	return cmd
}

// start get-mcp-service command

// Slice with functions to override default command behavior.
// Functions can be added from the `init()` function in manually curated files in this directory.
var getMcpServiceOverrides []func(
	*cobra.Command,
	*sdkaigateway.GetMcpServiceRequest,
)

func newGetMcpService() *cobra.Command {
	cmd := &cobra.Command{}
	var getMcpServiceReq sdkaigateway.GetMcpServiceRequest

	cmd.Use = "get-mcp-service NAME"
	cmd.Short = `Get an MCP service.`
	cmd.Long = `Get an MCP service.

  Returns the MCP service identified by its resource name.

  You must be the owner of the MCP service or have EXECUTE, READ_METADATA,
  or MANAGE on it, plus USE_CATALOG on the parent catalog and USE_SCHEMA
  on the parent schema.

  Arguments:
    NAME: Resource name of the MCP service. Format:
      mcp-services/{catalog}.{schema}.{mcp_service}. Each {...} component is
      capped at 255 characters individually.`

	cmd.Annotations = make(map[string]string)
	cmd.Annotations["launch_stage"] = "GA"
	cmd.Annotations["launch_stage_display"] = "GA"

	cmd.Args = func(cmd *cobra.Command, args []string) error {
		check := root.ExactArgs(1)
		return check(cmd, args)
	}

	cmd.PreRunE = root.MustWorkspaceClient
	cmd.RunE = func(cmd *cobra.Command, args []string) (err error) {
		ctx := cmd.Context()
		w := cmdctx.WorkspaceClient(ctx)
		_ = w

		getMcpServiceReq.Name = new(args[0])

		sdkReq := &getMcpServiceReq
		response, err := w.AiGateway.GetMcpService(ctx, *sdkReq)
		if err != nil {
			return err
		}

		out, err := aigateway.McpServiceFromSDK(response)
		if err != nil {
			return err
		}
		return cmdio.Render(ctx, out)
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

// start get-mcp-service-user-mapped-credential command

// Slice with functions to override default command behavior.
// Functions can be added from the `init()` function in manually curated files in this directory.
var getMcpServiceUserMappedCredentialOverrides []func(
	*cobra.Command,
	*sdkaigateway.GetMcpServiceUserMappedCredentialRequest,
)

func newGetMcpServiceUserMappedCredential() *cobra.Command {
	cmd := &cobra.Command{}
	var getMcpServiceUserMappedCredentialReq sdkaigateway.GetMcpServiceUserMappedCredentialRequest

	cmd.Use = "get-mcp-service-user-mapped-credential NAME"
	cmd.Short = `*Beta* Get MCP service user credential status.`
	cmd.Long = `This command is in Beta and may change without notice.

Get MCP service user credential status.

  Returns the caller's per-user OAuth login state for an MCP service. Read
  provisioning_info.state: ACTIVE means the caller is logged in and the
  credential is usable; any other state (for example a failed or
  still-provisioning login) means the login has not completed and the caller
  should log in again. If the caller has no credential yet, the RPC returns
  NOT_FOUND.

  You must be the owner of the MCP service or have EXECUTE on it, plus
  USE_CATALOG on the parent catalog and USE_SCHEMA on the parent schema.

  Arguments:
    NAME: Resource name of the MCP service. Format:
      mcp-services/{catalog}.{schema}.{mcp_service}.`

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

		getMcpServiceUserMappedCredentialReq.Name = new(args[0])

		sdkReq := &getMcpServiceUserMappedCredentialReq
		response, err := w.AiGateway.GetMcpServiceUserMappedCredential(ctx, *sdkReq)
		if err != nil {
			return err
		}

		out, err := aigateway.McpServiceUserMappedCredentialFromSDK(response)
		if err != nil {
			return err
		}
		return cmdio.Render(ctx, out)
	}

	// Disable completions since they are not applicable.
	// Can be overridden by manual implementation in `override.go`.
	cmd.ValidArgsFunction = cobra.NoFileCompletions

	// Apply optional overrides to this command.
	for _, fn := range getMcpServiceUserMappedCredentialOverrides {
		fn(cmd, &getMcpServiceUserMappedCredentialReq)
	}

	return cmd
}

// start get-model-provider-service command

// Slice with functions to override default command behavior.
// Functions can be added from the `init()` function in manually curated files in this directory.
var getModelProviderServiceOverrides []func(
	*cobra.Command,
	*sdkaigateway.GetModelProviderServiceRequest,
)

func newGetModelProviderService() *cobra.Command {
	cmd := &cobra.Command{}
	var getModelProviderServiceReq sdkaigateway.GetModelProviderServiceRequest

	cmd.Use = "get-model-provider-service NAME"
	cmd.Short = `Get a model provider service.`
	cmd.Long = `Get a model provider service.

  Returns the model provider service identified by its resource name.

  You must be the owner of the model provider service or have EXECUTE,
  READ_METADATA, or MANAGE on it, plus USE_CATALOG on the parent catalog
  and USE_SCHEMA on the parent schema.

  Arguments:
    NAME: Resource name of the model provider service. Format:
      model-provider-services/{catalog}.{schema}.{model_provider_service}.
      Each {...} component is capped at 255 characters individually.`

	cmd.Annotations = make(map[string]string)
	cmd.Annotations["launch_stage"] = "GA"
	cmd.Annotations["launch_stage_display"] = "GA"

	cmd.Args = func(cmd *cobra.Command, args []string) error {
		check := root.ExactArgs(1)
		return check(cmd, args)
	}

	cmd.PreRunE = root.MustWorkspaceClient
	cmd.RunE = func(cmd *cobra.Command, args []string) (err error) {
		ctx := cmd.Context()
		w := cmdctx.WorkspaceClient(ctx)
		_ = w

		getModelProviderServiceReq.Name = new(args[0])

		sdkReq := &getModelProviderServiceReq
		response, err := w.AiGateway.GetModelProviderService(ctx, *sdkReq)
		if err != nil {
			return err
		}

		out, err := aigateway.ModelProviderServiceFromSDK(response)
		if err != nil {
			return err
		}
		return cmdio.Render(ctx, out)
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
	*sdkaigateway.GetModelServiceRequest,
)

func newGetModelService() *cobra.Command {
	cmd := &cobra.Command{}
	var getModelServiceReq sdkaigateway.GetModelServiceRequest

	cmd.Use = "get-model-service NAME"
	cmd.Short = `Get a model service.`
	cmd.Long = `Get a model service.

  Returns the model service identified by its resource name.

  You must be the owner of the model service or have EXECUTE, READ_METADATA,
  or MANAGE on it, plus USE_CATALOG on the parent catalog and USE_SCHEMA
  on the parent schema.

  Arguments:
    NAME: Resource name of the model service. Format:
      model-services/{catalog}.{schema}.{model_service}. Each {...}
      component is capped at 255 characters individually.`

	cmd.Annotations = make(map[string]string)
	cmd.Annotations["launch_stage"] = "GA"
	cmd.Annotations["launch_stage_display"] = "GA"

	cmd.Args = func(cmd *cobra.Command, args []string) error {
		check := root.ExactArgs(1)
		return check(cmd, args)
	}

	cmd.PreRunE = root.MustWorkspaceClient
	cmd.RunE = func(cmd *cobra.Command, args []string) (err error) {
		ctx := cmd.Context()
		w := cmdctx.WorkspaceClient(ctx)
		_ = w

		getModelServiceReq.Name = new(args[0])

		sdkReq := &getModelServiceReq
		response, err := w.AiGateway.GetModelService(ctx, *sdkReq)
		if err != nil {
			return err
		}

		out, err := aigateway.ModelServiceFromSDK(response)
		if err != nil {
			return err
		}
		return cmdio.Render(ctx, out)
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

// start get-skill command

// Slice with functions to override default command behavior.
// Functions can be added from the `init()` function in manually curated files in this directory.
var getSkillOverrides []func(
	*cobra.Command,
	*sdkaigateway.GetSkillRequest,
)

func newGetSkill() *cobra.Command {
	cmd := &cobra.Command{}
	var getSkillReq sdkaigateway.GetSkillRequest

	cmd.Use = "get-skill NAME"
	cmd.Short = `*Beta* Get a skill.`
	cmd.Long = `This command is in Beta and may change without notice.

Get a skill.

  Returns the skill identified by its resource name.

  You must be the owner of the skill or have READ_VOLUME, READ_METADATA, or
  MANAGE on it, plus USE_CATALOG on the parent catalog and USE_SCHEMA on
  the parent schema.

  Arguments:
    NAME: Full resource name of the skill. Format:
      skills/{catalog}.{schema}.{skill}. Each {...} component is capped at
      255 characters individually.`

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

		getSkillReq.Name = new(args[0])

		sdkReq := &getSkillReq
		response, err := w.AiGateway.GetSkill(ctx, *sdkReq)
		if err != nil {
			return err
		}

		out, err := aigateway.SkillFromSDK(response)
		if err != nil {
			return err
		}
		return cmdio.Render(ctx, out)
	}

	// Disable completions since they are not applicable.
	// Can be overridden by manual implementation in `override.go`.
	cmd.ValidArgsFunction = cobra.NoFileCompletions

	// Apply optional overrides to this command.
	for _, fn := range getSkillOverrides {
		fn(cmd, &getSkillReq)
	}

	return cmd
}

// start list-mcp-services command

// Slice with functions to override default command behavior.
// Functions can be added from the `init()` function in manually curated files in this directory.
var listMcpServicesOverrides []func(
	*cobra.Command,
	*aigateway.ListMcpServicesRequest,
)

func newListMcpServices() *cobra.Command {
	cmd := &cobra.Command{}
	var listMcpServicesReq aigateway.ListMcpServicesRequest

	// Registered for all paginated methods. Validated at call time in the
	// method-call template. Paginated list methods never have Wait or LRO
	// branches, so the method-call path is always reached.
	var listMcpServicesLimit int

	flags.IntPtrVar(cmd.Flags(), &listMcpServicesReq.PageSize, "page-size", `Maximum number of MCP services to return.`)
	flags.StringPtrVar(cmd.Flags(), &listMcpServicesReq.Parent, "parent", `Parent schema to list within, in the form schemas/{catalog}.{schema}.`)
	flags.EnumVar(cmd.Flags(), &listMcpServicesReq.View, "view", `Fields to return for each service. Supported values: [BASIC, FULL]`, "ListMcpServicesRequestView", "BASIC", "FULL")

	// Limit flag for total result capping.
	cmd.Flags().IntVar(&listMcpServicesLimit, "limit", 0, `Maximum number of results to return.`)

	// Hidden pagination flags (internal API parameters).
	flags.StringPtrVar(cmd.Flags(), &listMcpServicesReq.PageToken, "page-token", `Pagination token.`)
	cmd.Flags().Lookup("page-token").Hidden = true

	cmd.Use = "list-mcp-services"
	cmd.Short = `List MCP services.`
	cmd.Long = `List MCP services.

  Lists the MCP services in a Unity Catalog schema. Provide parent as
  schemas/{catalog}.{schema}. Results are paginated; pass the returned
  next_page_token to fetch subsequent pages.

  Requires USE_CATALOG on the parent catalog and USE_SCHEMA on the parent
  schema. Only MCP services the caller can access (as owner or through
  EXECUTE, READ_METADATA, or MANAGE) are returned.`

	cmd.Annotations = make(map[string]string)
	cmd.Annotations["launch_stage"] = "GA"
	cmd.Annotations["launch_stage_display"] = "GA"

	cmd.Args = func(cmd *cobra.Command, args []string) error {
		check := root.ExactArgs(0)
		return check(cmd, args)
	}

	cmd.PreRunE = root.MustWorkspaceClient
	cmd.RunE = func(cmd *cobra.Command, args []string) (err error) {
		ctx := cmd.Context()
		w := cmdctx.WorkspaceClient(ctx)
		_ = w

		sdkReq, err := aigateway.ListMcpServicesRequestToSDK(&listMcpServicesReq)
		if err != nil {
			return err
		}
		if listMcpServicesLimit < 0 {
			return fmt.Errorf("--limit must be a non-negative integer, got %d", listMcpServicesLimit)
		}
		if listMcpServicesLimit > 0 {
			ctx = cmdio.WithLimit(ctx, listMcpServicesLimit)
		}
		response := w.AiGateway.ListMcpServicesIter(ctx, *sdkReq)
		return cmdio.RenderIterator(ctx, cmdgen.MapSeq(response, aigateway.McpServiceFromSDK))
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
	*aigateway.ListModelProviderServicesRequest,
)

func newListModelProviderServices() *cobra.Command {
	cmd := &cobra.Command{}
	var listModelProviderServicesReq aigateway.ListModelProviderServicesRequest

	// Registered for all paginated methods. Validated at call time in the
	// method-call template. Paginated list methods never have Wait or LRO
	// branches, so the method-call path is always reached.
	var listModelProviderServicesLimit int

	flags.IntPtrVar(cmd.Flags(), &listModelProviderServicesReq.PageSize, "page-size", `Maximum number of provider services to return.`)
	flags.StringPtrVar(cmd.Flags(), &listModelProviderServicesReq.Parent, "parent", `Parent schema to list within, in the form schemas/{catalog}.{schema}.`)
	flags.EnumVar(cmd.Flags(), &listModelProviderServicesReq.View, "view", `Fields to return for each service. Supported values: [BASIC, FULL]`, "ListModelProviderServicesRequestView", "BASIC", "FULL")

	// Limit flag for total result capping.
	cmd.Flags().IntVar(&listModelProviderServicesLimit, "limit", 0, `Maximum number of results to return.`)

	// Hidden pagination flags (internal API parameters).
	flags.StringPtrVar(cmd.Flags(), &listModelProviderServicesReq.PageToken, "page-token", `Pagination token.`)
	cmd.Flags().Lookup("page-token").Hidden = true

	cmd.Use = "list-model-provider-services"
	cmd.Short = `List model provider services.`
	cmd.Long = `List model provider services.

  Lists the model provider services in a Unity Catalog schema. Provide parent
  as schemas/{catalog}.{schema}. Results are paginated; pass the returned
  next_page_token to fetch subsequent pages.

  Requires USE_CATALOG on the parent catalog and USE_SCHEMA on the parent
  schema. Only model provider services the caller can access (as owner or
  through EXECUTE, READ_METADATA, or MANAGE) are returned.`

	cmd.Annotations = make(map[string]string)
	cmd.Annotations["launch_stage"] = "GA"
	cmd.Annotations["launch_stage_display"] = "GA"

	cmd.Args = func(cmd *cobra.Command, args []string) error {
		check := root.ExactArgs(0)
		return check(cmd, args)
	}

	cmd.PreRunE = root.MustWorkspaceClient
	cmd.RunE = func(cmd *cobra.Command, args []string) (err error) {
		ctx := cmd.Context()
		w := cmdctx.WorkspaceClient(ctx)
		_ = w

		sdkReq, err := aigateway.ListModelProviderServicesRequestToSDK(&listModelProviderServicesReq)
		if err != nil {
			return err
		}
		if listModelProviderServicesLimit < 0 {
			return fmt.Errorf("--limit must be a non-negative integer, got %d", listModelProviderServicesLimit)
		}
		if listModelProviderServicesLimit > 0 {
			ctx = cmdio.WithLimit(ctx, listModelProviderServicesLimit)
		}
		response := w.AiGateway.ListModelProviderServicesIter(ctx, *sdkReq)
		return cmdio.RenderIterator(ctx, cmdgen.MapSeq(response, aigateway.ModelProviderServiceFromSDK))
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
	*aigateway.ListModelServicesRequest,
)

func newListModelServices() *cobra.Command {
	cmd := &cobra.Command{}
	var listModelServicesReq aigateway.ListModelServicesRequest

	// Registered for all paginated methods. Validated at call time in the
	// method-call template. Paginated list methods never have Wait or LRO
	// branches, so the method-call path is always reached.
	var listModelServicesLimit int

	flags.IntPtrVar(cmd.Flags(), &listModelServicesReq.PageSize, "page-size", `Maximum number of model services to return.`)
	flags.StringPtrVar(cmd.Flags(), &listModelServicesReq.Parent, "parent", `Parent schema to list within, in the form schemas/{catalog}.{schema}.`)
	flags.EnumVar(cmd.Flags(), &listModelServicesReq.View, "view", `Fields to return for each service. Supported values: [BASIC, FULL]`, "ListModelServicesRequestView", "BASIC", "FULL")

	// Limit flag for total result capping.
	cmd.Flags().IntVar(&listModelServicesLimit, "limit", 0, `Maximum number of results to return.`)

	// Hidden pagination flags (internal API parameters).
	flags.StringPtrVar(cmd.Flags(), &listModelServicesReq.PageToken, "page-token", `Pagination token.`)
	cmd.Flags().Lookup("page-token").Hidden = true

	cmd.Use = "list-model-services"
	cmd.Short = `List model services.`
	cmd.Long = `List model services.

  Lists the model services in a Unity Catalog schema. Provide parent as
  schemas/{catalog}.{schema}. Results are paginated; pass the returned
  next_page_token to fetch subsequent pages.

  Requires USE_CATALOG on the parent catalog and USE_SCHEMA on the parent
  schema. Only model services the caller can access (as owner or through
  EXECUTE, READ_METADATA, or MANAGE) are returned.`

	cmd.Annotations = make(map[string]string)
	cmd.Annotations["launch_stage"] = "GA"
	cmd.Annotations["launch_stage_display"] = "GA"

	cmd.Args = func(cmd *cobra.Command, args []string) error {
		check := root.ExactArgs(0)
		return check(cmd, args)
	}

	cmd.PreRunE = root.MustWorkspaceClient
	cmd.RunE = func(cmd *cobra.Command, args []string) (err error) {
		ctx := cmd.Context()
		w := cmdctx.WorkspaceClient(ctx)
		_ = w

		sdkReq, err := aigateway.ListModelServicesRequestToSDK(&listModelServicesReq)
		if err != nil {
			return err
		}
		if listModelServicesLimit < 0 {
			return fmt.Errorf("--limit must be a non-negative integer, got %d", listModelServicesLimit)
		}
		if listModelServicesLimit > 0 {
			ctx = cmdio.WithLimit(ctx, listModelServicesLimit)
		}
		response := w.AiGateway.ListModelServicesIter(ctx, *sdkReq)
		return cmdio.RenderIterator(ctx, cmdgen.MapSeq(response, aigateway.ModelServiceFromSDK))
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

// start list-skills command

// Slice with functions to override default command behavior.
// Functions can be added from the `init()` function in manually curated files in this directory.
var listSkillsOverrides []func(
	*cobra.Command,
	*aigateway.ListSkillsRequest,
)

func newListSkills() *cobra.Command {
	cmd := &cobra.Command{}
	var listSkillsReq aigateway.ListSkillsRequest

	// Registered for all paginated methods. Validated at call time in the
	// method-call template. Paginated list methods never have Wait or LRO
	// branches, so the method-call path is always reached.
	var listSkillsLimit int

	flags.IntPtrVar(cmd.Flags(), &listSkillsReq.PageSize, "page-size", `Maximum number of skills to return.`)

	// Limit flag for total result capping.
	cmd.Flags().IntVar(&listSkillsLimit, "limit", 0, `Maximum number of results to return.`)

	// Hidden pagination flags (internal API parameters).
	flags.StringPtrVar(cmd.Flags(), &listSkillsReq.PageToken, "page-token", `Pagination token.`)
	cmd.Flags().Lookup("page-token").Hidden = true

	cmd.Use = "list-skills PARENT"
	cmd.Short = `*Beta* List skills.`
	cmd.Long = `This command is in Beta and may change without notice.

List skills.

  Lists skills in a Unity Catalog schema. Provide parent as
  schemas/{catalog}.{schema}. Results are paginated; pass the returned
  next_page_token to fetch subsequent pages.

  Requires USE_CATALOG on the parent catalog and USE_SCHEMA on the parent
  schema. Only skills the caller can access as owner or through READ_VOLUME,
  READ_METADATA, or MANAGE are returned.

  Arguments:
    PARENT: Name of the parent schema. Format: schemas/{catalog}.{schema}. Each
      {...} component is capped at 255 characters individually.

      Required: skill listing is schema-scoped, so parent must be set; an
      unset or empty parent is rejected with INVALID_PARAMETER_VALUE.`

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

		listSkillsReq.Parent = new(args[0])

		sdkReq, err := aigateway.ListSkillsRequestToSDK(&listSkillsReq)
		if err != nil {
			return err
		}
		if listSkillsLimit < 0 {
			return fmt.Errorf("--limit must be a non-negative integer, got %d", listSkillsLimit)
		}
		if listSkillsLimit > 0 {
			ctx = cmdio.WithLimit(ctx, listSkillsLimit)
		}
		response := w.AiGateway.ListSkillsIter(ctx, *sdkReq)
		return cmdio.RenderIterator(ctx, cmdgen.MapSeq(response, aigateway.SkillFromSDK))
	}

	// Disable completions since they are not applicable.
	// Can be overridden by manual implementation in `override.go`.
	cmd.ValidArgsFunction = cobra.NoFileCompletions

	// Apply optional overrides to this command.
	for _, fn := range listSkillsOverrides {
		fn(cmd, &listSkillsReq)
	}

	return cmd
}

// start update-mcp-service command

// Slice with functions to override default command behavior.
// Functions can be added from the `init()` function in manually curated files in this directory.
var updateMcpServiceOverrides []func(
	*cobra.Command,
	*aigateway.UpdateMcpServiceRequest,
)

func newUpdateMcpService() *cobra.Command {
	cmd := &cobra.Command{}
	var updateMcpServiceReq aigateway.UpdateMcpServiceRequest
	updateMcpServiceReq.McpService = &aigateway.McpService{}

	var updateMcpServiceJson flags.JsonFlag

	cmd.Flags().Var(&updateMcpServiceJson, "json", `either inline JSON string or @path/to/file.json with request body`)

	flags.Base64Var(cmd.Flags(), &updateMcpServiceReq.Etag, "etag", `Optimistic concurrency token from the most recent read.`)
	flags.StringPtrVar(cmd.Flags(), &updateMcpServiceReq.McpService.Comment, "comment", `User-provided description.`)
	// TODO: complex arg: config
	flags.StringPtrVar(cmd.Flags(), &updateMcpServiceReq.McpService.Name, "name", `Resource name of the MCP service.`)

	cmd.Use = "update-mcp-service NAME UPDATE_MASK"
	cmd.Short = `Update an MCP service.`
	cmd.Long = `Update an MCP service.

  Updates an MCP service. Only the fields named in update_mask are changed;
  the resource name is immutable. Optionally supply an etag to make the update
  conditional on the MCP service not having changed since it was read.

  You must be the owner of the MCP service or have MANAGE on it, plus
  USE_CATALOG on the parent catalog and USE_SCHEMA on the parent schema.
  When changing config.source_connection.name, the MCP service owner must also
  have USE_CONNECTION on the new connection.

  Arguments:
    NAME: Resource name of the MCP service. Format:
      mcp-services/{catalog}.{schema}.{mcp_service}. Each {...} component is
      capped at 255 characters individually. Server-derived on Create from
      parent + mcp_service_id; required and immutable on Update/Get/Delete.
    UPDATE_MASK: Fields to update. Use config to replace the entire configuration. The
      replacement must include every required field; any optional field you omit
      is cleared. To preserve sibling fields, use one or more granular paths:
      comment, config.source_connection.name,
      config.include_tool_selectors, or config.rate_limits.`

	cmd.Annotations = make(map[string]string)
	cmd.Annotations["launch_stage"] = "GA"
	cmd.Annotations["launch_stage_display"] = "GA"

	cmd.Args = func(cmd *cobra.Command, args []string) error {
		check := root.ExactArgs(2)
		return check(cmd, args)
	}

	cmd.PreRunE = root.MustWorkspaceClient
	cmd.RunE = func(cmd *cobra.Command, args []string) (err error) {
		ctx := cmd.Context()
		w := cmdctx.WorkspaceClient(ctx)
		_ = w

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
		updateMcpServiceReq.McpService.Name = new(args[0])
		if args[1] != "" {
			updateMcpServiceReq.UpdateMask = new(args[1])
		}

		sdkReq, err := aigateway.UpdateMcpServiceRequestToSDK(&updateMcpServiceReq)
		if err != nil {
			return err
		}
		response, err := w.AiGateway.UpdateMcpService(ctx, *sdkReq)
		if err != nil {
			return err
		}

		out, err := aigateway.McpServiceFromSDK(response)
		if err != nil {
			return err
		}
		return cmdio.Render(ctx, out)
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
	*aigateway.UpdateModelProviderServiceRequest,
)

func newUpdateModelProviderService() *cobra.Command {
	cmd := &cobra.Command{}
	var updateModelProviderServiceReq aigateway.UpdateModelProviderServiceRequest
	updateModelProviderServiceReq.ModelProviderService = &aigateway.ModelProviderService{}

	var updateModelProviderServiceJson flags.JsonFlag

	cmd.Flags().Var(&updateModelProviderServiceJson, "json", `either inline JSON string or @path/to/file.json with request body`)

	flags.Base64Var(cmd.Flags(), &updateModelProviderServiceReq.Etag, "etag", `Optimistic concurrency token from the most recent read.`)
	flags.StringPtrVar(cmd.Flags(), &updateModelProviderServiceReq.ModelProviderService.Comment, "comment", `User-provided description.`)
	// TODO: complex arg: config
	flags.StringPtrVar(cmd.Flags(), &updateModelProviderServiceReq.ModelProviderService.Name, "name", `Resource name of the provider service.`)

	cmd.Use = "update-model-provider-service NAME UPDATE_MASK"
	cmd.Short = `Update a model provider service.`
	cmd.Long = `Update a model provider service.

  Updates a model provider service. Only the fields named in update_mask are
  changed; the resource name and provider type are immutable. Optionally supply
  an etag to make the update conditional on the model provider service not
  having changed since it was read.

  You must be the owner of the model provider service or have MANAGE on it,
  plus USE_CATALOG on the parent catalog and USE_SCHEMA on the parent
  schema.

  Updating config.provider cannot change the provider type. Authentication
  mode changes require feature availability and support for both modes on the
  selected provider.

  Arguments:
    NAME: Resource name of the provider service. Format:
      model-provider-services/{catalog}.{schema}.{model_provider_service}.
      Each {...} component is capped at 255 characters individually.
      Server-derived on Create from parent + model_provider_service_id;
      required and immutable on Update/Get/Delete.
    UPDATE_MASK: Fields to update. Use config to replace the entire configuration. The
      replacement must include every required field; any optional field you omit
      is cleared. To preserve sibling fields, use one or more granular paths:
      comment; config.provider to replace the active provider-specific value
      (for example, config.openai; the mask path remains config.provider);
      config.allow_all_targets, config.targets, config.forward_headers,
      config.forward_query_parameters, config.forward_unmanaged_paths,
      config.rate_limits, config.pricing.default_discount_basis_points, or
      config.inference_table. For a pricing update, an explicitly empty
      pricing object clears the discount but retains the object; omitting
      pricing clears it. config.pricing is not a supported mask path. The
      provider type is immutable. A config or config.provider replacement
      that carries no authentication material preserves the existing
      authentication binding; input-only plaintext does not need to be read back
      and re-sent.`

	cmd.Annotations = make(map[string]string)
	cmd.Annotations["launch_stage"] = "GA"
	cmd.Annotations["launch_stage_display"] = "GA"

	cmd.Args = func(cmd *cobra.Command, args []string) error {
		check := root.ExactArgs(2)
		return check(cmd, args)
	}

	cmd.PreRunE = root.MustWorkspaceClient
	cmd.RunE = func(cmd *cobra.Command, args []string) (err error) {
		ctx := cmd.Context()
		w := cmdctx.WorkspaceClient(ctx)
		_ = w

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
		updateModelProviderServiceReq.ModelProviderService.Name = new(args[0])
		if args[1] != "" {
			updateModelProviderServiceReq.UpdateMask = new(args[1])
		}

		sdkReq, err := aigateway.UpdateModelProviderServiceRequestToSDK(&updateModelProviderServiceReq)
		if err != nil {
			return err
		}
		response, err := w.AiGateway.UpdateModelProviderService(ctx, *sdkReq)
		if err != nil {
			return err
		}

		out, err := aigateway.ModelProviderServiceFromSDK(response)
		if err != nil {
			return err
		}
		return cmdio.Render(ctx, out)
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
	*aigateway.UpdateModelServiceRequest,
)

func newUpdateModelService() *cobra.Command {
	cmd := &cobra.Command{}
	var updateModelServiceReq aigateway.UpdateModelServiceRequest
	updateModelServiceReq.ModelService = &aigateway.ModelService{}

	var updateModelServiceJson flags.JsonFlag

	cmd.Flags().Var(&updateModelServiceJson, "json", `either inline JSON string or @path/to/file.json with request body`)

	flags.Base64Var(cmd.Flags(), &updateModelServiceReq.Etag, "etag", `Optimistic concurrency token from the most recent read.`)
	flags.StringPtrVar(cmd.Flags(), &updateModelServiceReq.ModelService.Comment, "comment", `User-provided description.`)
	// TODO: complex arg: config
	flags.StringPtrVar(cmd.Flags(), &updateModelServiceReq.ModelService.Name, "name", `Resource name of the model service.`)
	// TODO: array: supported_api_types

	cmd.Use = "update-model-service NAME UPDATE_MASK"
	cmd.Short = `Update a model service.`
	cmd.Long = `Update a model service.

  Updates a model service. Only the fields named in update_mask are changed;
  the resource name is immutable. Optionally supply an etag to make the update
  conditional on the model service not having changed since it was read.

  You must be the owner of the model service or have MANAGE on it, plus
  USE_CATALOG on the parent catalog and USE_SCHEMA on the parent schema.
  When changing destinations, both you and the model service owner need
  USE_CATALOG and USE_SCHEMA on each destination's parent and EXECUTE on
  the referenced Unity Catalog model or model provider service. A
  provisioned-throughput destination additionally requires CAN_MANAGE for you
  and CAN_QUERY for the model service owner. Adding an inference table
  additionally requires CREATE_TABLE.

  Arguments:
    NAME: Resource name of the model service. Format:
      model-services/{catalog}.{schema}.{model_service}. Each {...}
      component is capped at 255 characters individually. Server-derived on
      Create from parent + model_service_id; required and immutable on
      Update/Get/Delete.
    UPDATE_MASK: Fields to update. Use config to replace the entire configuration. The
      replacement must include every required field; any optional field you omit
      is cleared. To preserve sibling fields, use one or more granular paths:
      comment, config.routing.destinations,
      config.routing.fallback.destinations, config.rate_limits, or
      config.inference_table. Intermediate paths such as config.routing and
      config.routing.fallback are not supported.`

	cmd.Annotations = make(map[string]string)
	cmd.Annotations["launch_stage"] = "GA"
	cmd.Annotations["launch_stage_display"] = "GA"

	cmd.Args = func(cmd *cobra.Command, args []string) error {
		check := root.ExactArgs(2)
		return check(cmd, args)
	}

	cmd.PreRunE = root.MustWorkspaceClient
	cmd.RunE = func(cmd *cobra.Command, args []string) (err error) {
		ctx := cmd.Context()
		w := cmdctx.WorkspaceClient(ctx)
		_ = w

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
		updateModelServiceReq.ModelService.Name = new(args[0])
		if args[1] != "" {
			updateModelServiceReq.UpdateMask = new(args[1])
		}

		sdkReq, err := aigateway.UpdateModelServiceRequestToSDK(&updateModelServiceReq)
		if err != nil {
			return err
		}
		response, err := w.AiGateway.UpdateModelService(ctx, *sdkReq)
		if err != nil {
			return err
		}

		out, err := aigateway.ModelServiceFromSDK(response)
		if err != nil {
			return err
		}
		return cmdio.Render(ctx, out)
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

// start update-skill command

// Slice with functions to override default command behavior.
// Functions can be added from the `init()` function in manually curated files in this directory.
var updateSkillOverrides []func(
	*cobra.Command,
	*aigateway.UpdateSkillRequest,
)

func newUpdateSkill() *cobra.Command {
	cmd := &cobra.Command{}
	var updateSkillReq aigateway.UpdateSkillRequest
	updateSkillReq.Skill = &aigateway.Skill{}

	var updateSkillJson flags.JsonFlag

	cmd.Flags().Var(&updateSkillJson, "json", `either inline JSON string or @path/to/file.json with request body`)

	flags.Base64Var(cmd.Flags(), &updateSkillReq.Etag, "etag", `Optimistic concurrency token from the most recent read.`)
	flags.StringPtrVar(cmd.Flags(), &updateSkillReq.Skill.Comment, "comment", `User-provided comment for the skill.`)
	flags.StringPtrVar(cmd.Flags(), &updateSkillReq.Skill.Name, "name", `Resource name of the skill.`)

	cmd.Use = "update-skill NAME UPDATE_MASK"
	cmd.Short = `*Beta* Update a skill.`
	cmd.Long = `This command is in Beta and may change without notice.

Update a skill.

  Updates a skill. Only fields named in update_mask are changed; currently
  only comment is supported. The resource name is immutable. Optionally supply
  an etag to make the update conditional on the skill not having changed since
  it was read. Bundle files, grants, tags, and ownership are unchanged.

  You must be the owner of the skill or have MANAGE on it, plus USE_CATALOG
  on the parent catalog and USE_SCHEMA on the parent schema.

  Arguments:
    NAME: Resource name of the skill. Format: skills/{catalog}.{schema}.{skill}.
      Each {...} component is capped at 255 characters individually.
      Server-derived on Create from parent + skill_id; required and
      immutable on Update/Get/Delete.
    UPDATE_MASK: Fields to update; validated against skill. REQUIRED, matching the
      sibling Update RPCs. comment is the only mutable field.`

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
		_ = w

		if cmd.Flags().Changed("json") {
			diags := updateSkillJson.Unmarshal(&updateSkillReq.Skill)
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
		updateSkillReq.Skill.Name = new(args[0])
		if args[1] != "" {
			updateSkillReq.UpdateMask = new(args[1])
		}

		sdkReq, err := aigateway.UpdateSkillRequestToSDK(&updateSkillReq)
		if err != nil {
			return err
		}
		response, err := w.AiGateway.UpdateSkill(ctx, *sdkReq)
		if err != nil {
			return err
		}

		out, err := aigateway.SkillFromSDK(response)
		if err != nil {
			return err
		}
		return cmdio.Render(ctx, out)
	}

	// Disable completions since they are not applicable.
	// Can be overridden by manual implementation in `override.go`.
	cmd.ValidArgsFunction = cobra.NoFileCompletions

	// Apply optional overrides to this command.
	for _, fn := range updateSkillOverrides {
		fn(cmd, &updateSkillReq)
	}

	return cmd
}

// end service AiGateway
