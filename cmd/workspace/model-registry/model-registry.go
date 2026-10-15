// Code generated from OpenAPI specs by Databricks SDK Generator. DO NOT EDIT.

package model_registry

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
	modelregistry "github.com/databricks/cli/libs/sdktypes/modelregistry/v1"
	sdkaccessmanagement "github.com/databricks/sdk-go/accessmanagement/v1"
)

// Slice with functions to override default command behavior.
// Functions can be added from the `init()` function in manually curated files in this directory.
var cmdOverrides []func(*cobra.Command)

func New() *cobra.Command {
	cmd := &cobra.Command{
		Use:   "model-registry",
		Short: `Note: This API reference documents APIs for the Workspace Model Registry.`,
		Long: `Note: This API reference documents APIs for the Workspace Model Registry.
  Databricks recommends using [Models in Unity
  Catalog](/api/workspace/registeredmodels) instead. Models in Unity Catalog
  provides centralized model governance, cross-workspace access, lineage, and
  deployment. Workspace Model Registry will be deprecated in the future.

  The Workspace Model Registry is a centralized model repository and a UI and
  set of APIs that enable you to manage the full lifecycle of MLflow Models.`,
		GroupID: "ml",
		RunE:    root.ReportUnknownSubcommand,
	}

	cmd.Annotations = make(map[string]string)
	cmd.Annotations["launch_stage"] = "GA"
	cmd.Annotations["launch_stage_display"] = "GA"

	// Add methods
	cmd.AddCommand(newApproveTransitionRequest())
	cmd.AddCommand(newCreateComment())
	cmd.AddCommand(newCreateModel())
	cmd.AddCommand(newCreateModelVersion())
	cmd.AddCommand(newCreateTransitionRequest())
	cmd.AddCommand(newCreateWebhook())
	cmd.AddCommand(newDeleteComment())
	cmd.AddCommand(newDeleteModel())
	cmd.AddCommand(newDeleteModelTag())
	cmd.AddCommand(newDeleteModelVersion())
	cmd.AddCommand(newDeleteModelVersionTag())
	cmd.AddCommand(newDeleteTransitionRequest())
	cmd.AddCommand(newDeleteWebhook())
	cmd.AddCommand(newGetLatestVersions())
	cmd.AddCommand(newGetModel())
	cmd.AddCommand(newGetModelVersion())
	cmd.AddCommand(newGetModelVersionDownloadUri())
	cmd.AddCommand(newGetPermissionLevels())
	cmd.AddCommand(newGetPermissions())
	cmd.AddCommand(newListModels())
	cmd.AddCommand(newListTransitionRequests())
	cmd.AddCommand(newListWebhooks())
	cmd.AddCommand(newRejectTransitionRequest())
	cmd.AddCommand(newRenameModel())
	cmd.AddCommand(newSearchModelVersions())
	cmd.AddCommand(newSearchModels())
	cmd.AddCommand(newSetModelTag())
	cmd.AddCommand(newSetModelVersionTag())
	cmd.AddCommand(newSetPermissions())
	cmd.AddCommand(newTestRegistryWebhook())
	cmd.AddCommand(newTransitionStage())
	cmd.AddCommand(newUpdateComment())
	cmd.AddCommand(newUpdateModel())
	cmd.AddCommand(newUpdateModelVersion())
	cmd.AddCommand(newUpdatePermissions())
	cmd.AddCommand(newUpdateWebhook())

	// Apply optional overrides to this command.
	for _, fn := range cmdOverrides {
		fn(cmd)
	}

	return cmd
}

// start approve-transition-request command

// Slice with functions to override default command behavior.
// Functions can be added from the `init()` function in manually curated files in this directory.
var approveTransitionRequestOverrides []func(
	*cobra.Command,
	*modelregistry.ApproveTransitionRequest,
)

func newApproveTransitionRequest() *cobra.Command {
	cmd := &cobra.Command{}
	var approveTransitionRequestReq modelregistry.ApproveTransitionRequest

	var approveTransitionRequestJson flags.JsonFlag

	cmd.Flags().Var(&approveTransitionRequestJson, "json", `either inline JSON string or @path/to/file.json with request body`)

	flags.StringPtrVar(cmd.Flags(), &approveTransitionRequestReq.Comment, "comment", `User-provided comment on the action.`)

	cmd.Use = "approve-transition-request NAME VERSION STAGE ARCHIVE_EXISTING_VERSIONS"
	cmd.Short = `Approve transition request.`
	cmd.Long = `Approve transition request.

  Approves a model version stage transition request.

  Arguments:
    NAME: Name of the model.
    VERSION: Version of the model.
    STAGE: Target stage of the transition. Valid values are:

      * None: The initial stage of a model version.

      * Staging: Staging or pre-production stage.

      * Production: Production stage.

      * Archived: Archived stage.
    ARCHIVE_EXISTING_VERSIONS: Specifies whether to archive all current model versions in the target
      stage.`

	cmd.Annotations = make(map[string]string)
	cmd.Annotations["launch_stage"] = "GA"
	cmd.Annotations["launch_stage_display"] = "GA"

	cmd.Args = func(cmd *cobra.Command, args []string) error {
		if cmd.Flags().Changed("json") {
			err := root.ExactArgs(0)(cmd, args)
			if err != nil {
				return errors.New("when --json flag is specified, no positional arguments are allowed. Provide 'name', 'version', 'stage', 'archive_existing_versions' in your JSON input")
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
			diags := approveTransitionRequestJson.Unmarshal(&approveTransitionRequestReq)
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
			approveTransitionRequestReq.Name = new(args[0])
		}
		if !cmd.Flags().Changed("json") {
			approveTransitionRequestReq.Version = new(args[1])
		}
		if !cmd.Flags().Changed("json") {
			approveTransitionRequestReq.Stage = new(args[2])
		}
		if !cmd.Flags().Changed("json") {
			var approveTransitionRequestArchiveExistingVersionsArg bool
			_, err = fmt.Sscan(args[3], &approveTransitionRequestArchiveExistingVersionsArg)
			if err != nil {
				return fmt.Errorf("invalid ARCHIVE_EXISTING_VERSIONS: %s", args[3])
			}
			approveTransitionRequestReq.ArchiveExistingVersions = &approveTransitionRequestArchiveExistingVersionsArg
		}

		sdkReq, err := modelregistry.ApproveTransitionRequestToSDK(&approveTransitionRequestReq)
		if err != nil {
			return err
		}
		response, err := w.ModelRegistry.ApproveTransitionRequest(ctx, *sdkReq)
		if err != nil {
			return err
		}

		out, err := modelregistry.ApproveTransitionResponseFromSDK(response)
		if err != nil {
			return err
		}
		return cmdio.Render(ctx, out)
	}

	// Disable completions since they are not applicable.
	// Can be overridden by manual implementation in `override.go`.
	cmd.ValidArgsFunction = cobra.NoFileCompletions

	// Apply optional overrides to this command.
	for _, fn := range approveTransitionRequestOverrides {
		fn(cmd, &approveTransitionRequestReq)
	}

	return cmd
}

// start create-comment command

// Slice with functions to override default command behavior.
// Functions can be added from the `init()` function in manually curated files in this directory.
var createCommentOverrides []func(
	*cobra.Command,
	*modelregistry.CreateCommentRequest,
)

func newCreateComment() *cobra.Command {
	cmd := &cobra.Command{}
	var createCommentReq modelregistry.CreateCommentRequest

	var createCommentJson flags.JsonFlag

	cmd.Flags().Var(&createCommentJson, "json", `either inline JSON string or @path/to/file.json with request body`)

	cmd.Use = "create-comment NAME VERSION COMMENT"
	cmd.Short = `Post a comment.`
	cmd.Long = `Post a comment.

  Posts a comment on a model version. A comment can be submitted either by a
  user or programmatically to display relevant information about the model. For
  example, test results or deployment errors.

  Arguments:
    NAME: Name of the model.
    VERSION: Version of the model.
    COMMENT: User-provided comment on the action.`

	cmd.Annotations = make(map[string]string)
	cmd.Annotations["launch_stage"] = "GA"
	cmd.Annotations["launch_stage_display"] = "GA"

	cmd.Args = func(cmd *cobra.Command, args []string) error {
		if cmd.Flags().Changed("json") {
			err := root.ExactArgs(0)(cmd, args)
			if err != nil {
				return errors.New("when --json flag is specified, no positional arguments are allowed. Provide 'name', 'version', 'comment' in your JSON input")
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
			diags := createCommentJson.Unmarshal(&createCommentReq)
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
			createCommentReq.Name = new(args[0])
		}
		if !cmd.Flags().Changed("json") {
			createCommentReq.Version = new(args[1])
		}
		if !cmd.Flags().Changed("json") {
			createCommentReq.Comment = new(args[2])
		}

		sdkReq, err := modelregistry.CreateCommentRequestToSDK(&createCommentReq)
		if err != nil {
			return err
		}
		response, err := w.ModelRegistry.CreateComment(ctx, *sdkReq)
		if err != nil {
			return err
		}

		out, err := modelregistry.CreateCommentResponseFromSDK(response)
		if err != nil {
			return err
		}
		return cmdio.Render(ctx, out)
	}

	// Disable completions since they are not applicable.
	// Can be overridden by manual implementation in `override.go`.
	cmd.ValidArgsFunction = cobra.NoFileCompletions

	// Apply optional overrides to this command.
	for _, fn := range createCommentOverrides {
		fn(cmd, &createCommentReq)
	}

	return cmd
}

// start create-model command

// Slice with functions to override default command behavior.
// Functions can be added from the `init()` function in manually curated files in this directory.
var createModelOverrides []func(
	*cobra.Command,
	*modelregistry.CreateRegisteredModelRequest,
)

func newCreateModel() *cobra.Command {
	cmd := &cobra.Command{}
	var createModelReq modelregistry.CreateRegisteredModelRequest

	var createModelJson flags.JsonFlag

	cmd.Flags().Var(&createModelJson, "json", `either inline JSON string or @path/to/file.json with request body`)

	flags.StringPtrVar(cmd.Flags(), &createModelReq.Description, "description", `Optional description for registered model.`)
	// TODO: array: tags

	cmd.Use = "create-model NAME"
	cmd.Short = `Create a model.`
	cmd.Long = `Create a model.

  Creates a new registered model with the name specified in the request body.
  Throws RESOURCE_ALREADY_EXISTS if a registered model with the given name
  exists.

  Arguments:
    NAME: Register models under this name`

	cmd.Annotations = make(map[string]string)
	cmd.Annotations["launch_stage"] = "GA"
	cmd.Annotations["launch_stage_display"] = "GA"

	cmd.Args = func(cmd *cobra.Command, args []string) error {
		if cmd.Flags().Changed("json") {
			err := root.ExactArgs(0)(cmd, args)
			if err != nil {
				return errors.New("when --json flag is specified, no positional arguments are allowed. Provide 'name' in your JSON input")
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
		_ = w

		if cmd.Flags().Changed("json") {
			diags := createModelJson.Unmarshal(&createModelReq)
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
			createModelReq.Name = new(args[0])
		}

		sdkReq, err := modelregistry.CreateRegisteredModelRequestToSDK(&createModelReq)
		if err != nil {
			return err
		}
		response, err := w.ModelRegistry.CreateRegisteredModel(ctx, *sdkReq)
		if err != nil {
			return err
		}

		out, err := modelregistry.CreateRegisteredModelResponseFromSDK(response)
		if err != nil {
			return err
		}
		return cmdio.Render(ctx, out)
	}

	// Disable completions since they are not applicable.
	// Can be overridden by manual implementation in `override.go`.
	cmd.ValidArgsFunction = cobra.NoFileCompletions

	// Apply optional overrides to this command.
	for _, fn := range createModelOverrides {
		fn(cmd, &createModelReq)
	}

	return cmd
}

// start create-model-version command

// Slice with functions to override default command behavior.
// Functions can be added from the `init()` function in manually curated files in this directory.
var createModelVersionOverrides []func(
	*cobra.Command,
	*modelregistry.CreateModelVersionRequest,
)

func newCreateModelVersion() *cobra.Command {
	cmd := &cobra.Command{}
	var createModelVersionReq modelregistry.CreateModelVersionRequest

	var createModelVersionJson flags.JsonFlag

	cmd.Flags().Var(&createModelVersionJson, "json", `either inline JSON string or @path/to/file.json with request body`)

	flags.StringPtrVar(cmd.Flags(), &createModelVersionReq.Description, "description", `Optional description for model version.`)
	flags.StringPtrVar(cmd.Flags(), &createModelVersionReq.RunId, "run-id", `MLflow run ID for correlation, if source was generated by an experiment run in MLflow tracking server.`)
	flags.StringPtrVar(cmd.Flags(), &createModelVersionReq.RunLink, "run-link", `MLflow run link - this is the exact link of the run that generated this model version, potentially hosted at another instance of MLflow.`)
	// TODO: array: tags

	cmd.Use = "create-model-version NAME SOURCE"
	cmd.Short = `Create a model version.`
	cmd.Long = `Create a model version.

  Creates a model version.

  Arguments:
    NAME: Register model under this name
    SOURCE: URI indicating the location of the model artifacts.`

	cmd.Annotations = make(map[string]string)
	cmd.Annotations["launch_stage"] = "GA"
	cmd.Annotations["launch_stage_display"] = "GA"

	cmd.Args = func(cmd *cobra.Command, args []string) error {
		if cmd.Flags().Changed("json") {
			err := root.ExactArgs(0)(cmd, args)
			if err != nil {
				return errors.New("when --json flag is specified, no positional arguments are allowed. Provide 'name', 'source' in your JSON input")
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
			diags := createModelVersionJson.Unmarshal(&createModelVersionReq)
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
			createModelVersionReq.Name = new(args[0])
		}
		if !cmd.Flags().Changed("json") {
			createModelVersionReq.Source = new(args[1])
		}

		sdkReq, err := modelregistry.CreateModelVersionRequestToSDK(&createModelVersionReq)
		if err != nil {
			return err
		}
		response, err := w.ModelRegistry.CreateModelVersion(ctx, *sdkReq)
		if err != nil {
			return err
		}

		out, err := modelregistry.CreateModelVersionResponseFromSDK(response)
		if err != nil {
			return err
		}
		return cmdio.Render(ctx, out)
	}

	// Disable completions since they are not applicable.
	// Can be overridden by manual implementation in `override.go`.
	cmd.ValidArgsFunction = cobra.NoFileCompletions

	// Apply optional overrides to this command.
	for _, fn := range createModelVersionOverrides {
		fn(cmd, &createModelVersionReq)
	}

	return cmd
}

// start create-transition-request command

// Slice with functions to override default command behavior.
// Functions can be added from the `init()` function in manually curated files in this directory.
var createTransitionRequestOverrides []func(
	*cobra.Command,
	*modelregistry.CreateTransitionRequest,
)

func newCreateTransitionRequest() *cobra.Command {
	cmd := &cobra.Command{}
	var createTransitionRequestReq modelregistry.CreateTransitionRequest

	var createTransitionRequestJson flags.JsonFlag

	cmd.Flags().Var(&createTransitionRequestJson, "json", `either inline JSON string or @path/to/file.json with request body`)

	flags.StringPtrVar(cmd.Flags(), &createTransitionRequestReq.Comment, "comment", `User-provided comment on the action.`)

	cmd.Use = "create-transition-request NAME VERSION STAGE"
	cmd.Short = `Make a transition request.`
	cmd.Long = `Make a transition request.

  Creates a model version stage transition request.

  Arguments:
    NAME: Name of the model.
    VERSION: Version of the model.
    STAGE: Target stage of the transition. Valid values are:

      * None: The initial stage of a model version.

      * Staging: Staging or pre-production stage.

      * Production: Production stage.

      * Archived: Archived stage.`

	cmd.Annotations = make(map[string]string)
	cmd.Annotations["launch_stage"] = "GA"
	cmd.Annotations["launch_stage_display"] = "GA"

	cmd.Args = func(cmd *cobra.Command, args []string) error {
		if cmd.Flags().Changed("json") {
			err := root.ExactArgs(0)(cmd, args)
			if err != nil {
				return errors.New("when --json flag is specified, no positional arguments are allowed. Provide 'name', 'version', 'stage' in your JSON input")
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
			diags := createTransitionRequestJson.Unmarshal(&createTransitionRequestReq)
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
			createTransitionRequestReq.Name = new(args[0])
		}
		if !cmd.Flags().Changed("json") {
			createTransitionRequestReq.Version = new(args[1])
		}
		if !cmd.Flags().Changed("json") {
			createTransitionRequestReq.Stage = new(args[2])
		}

		sdkReq, err := modelregistry.CreateTransitionRequestToSDK(&createTransitionRequestReq)
		if err != nil {
			return err
		}
		response, err := w.ModelRegistry.CreateTransitionRequest(ctx, *sdkReq)
		if err != nil {
			return err
		}

		out, err := modelregistry.CreateTransitionResponseFromSDK(response)
		if err != nil {
			return err
		}
		return cmdio.Render(ctx, out)
	}

	// Disable completions since they are not applicable.
	// Can be overridden by manual implementation in `override.go`.
	cmd.ValidArgsFunction = cobra.NoFileCompletions

	// Apply optional overrides to this command.
	for _, fn := range createTransitionRequestOverrides {
		fn(cmd, &createTransitionRequestReq)
	}

	return cmd
}

// start create-webhook command

// Slice with functions to override default command behavior.
// Functions can be added from the `init()` function in manually curated files in this directory.
var createWebhookOverrides []func(
	*cobra.Command,
	*modelregistry.CreateRegistryWebhookRequest,
)

func newCreateWebhook() *cobra.Command {
	cmd := &cobra.Command{}
	var createWebhookReq modelregistry.CreateRegistryWebhookRequest

	var createWebhookJson flags.JsonFlag

	cmd.Flags().Var(&createWebhookJson, "json", `either inline JSON string or @path/to/file.json with request body`)

	flags.StringPtrVar(cmd.Flags(), &createWebhookReq.Description, "description", `User-specified description for the webhook.`)
	// TODO: complex arg: http_url_spec
	// TODO: complex arg: job_spec
	flags.StringPtrVar(cmd.Flags(), &createWebhookReq.ModelName, "model-name", `If model name is not specified, a registry-wide webhook is created that listens for the specified events across all versions of all registered models.`)
	flags.EnumVar(cmd.Flags(), &createWebhookReq.Status, "status", `Enable or disable triggering the webhook, or put the webhook into test mode. Supported values: [ACTIVE, DISABLED, TEST_MODE]`, "RegistryWebhookStatus", "ACTIVE", "DISABLED", "TEST_MODE")

	cmd.Use = "create-webhook"
	cmd.Short = `Create a webhook.`
	cmd.Long = `Create a webhook.

  **NOTE:** This endpoint is in Public Preview. Creates a registry webhook.`

	cmd.Annotations = make(map[string]string)
	cmd.Annotations["launch_stage"] = "GA"
	cmd.Annotations["launch_stage_display"] = "GA"

	cmd.PreRunE = root.MustWorkspaceClient
	cmd.RunE = func(cmd *cobra.Command, args []string) (err error) {
		ctx := cmd.Context()
		w := cmdctx.WorkspaceClient(ctx)
		_ = w

		if cmd.Flags().Changed("json") {
			diags := createWebhookJson.Unmarshal(&createWebhookReq)
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

		sdkReq, err := modelregistry.CreateRegistryWebhookRequestToSDK(&createWebhookReq)
		if err != nil {
			return err
		}
		response, err := w.ModelRegistry.CreateRegistryWebhook(ctx, *sdkReq)
		if err != nil {
			return err
		}

		out, err := modelregistry.CreateRegistryWebhookResponseFromSDK(response)
		if err != nil {
			return err
		}
		return cmdio.Render(ctx, out)
	}

	// Disable completions since they are not applicable.
	// Can be overridden by manual implementation in `override.go`.
	cmd.ValidArgsFunction = cobra.NoFileCompletions

	// Apply optional overrides to this command.
	for _, fn := range createWebhookOverrides {
		fn(cmd, &createWebhookReq)
	}

	return cmd
}

// start delete-comment command

// Slice with functions to override default command behavior.
// Functions can be added from the `init()` function in manually curated files in this directory.
var deleteCommentOverrides []func(
	*cobra.Command,
	*modelregistry.DeleteCommentRequest,
)

func newDeleteComment() *cobra.Command {
	cmd := &cobra.Command{}
	var deleteCommentReq modelregistry.DeleteCommentRequest

	cmd.Use = "delete-comment ID"
	cmd.Short = `Delete a comment.`
	cmd.Long = `Delete a comment.

  Deletes a comment on a model version.

  Arguments:
    ID: Unique identifier of an activity`

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

		deleteCommentReq.Id = new(args[0])

		sdkReq, err := modelregistry.DeleteCommentRequestToSDK(&deleteCommentReq)
		if err != nil {
			return err
		}
		response, err := w.ModelRegistry.DeleteComment(ctx, *sdkReq)
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
	for _, fn := range deleteCommentOverrides {
		fn(cmd, &deleteCommentReq)
	}

	return cmd
}

// start delete-model command

// Slice with functions to override default command behavior.
// Functions can be added from the `init()` function in manually curated files in this directory.
var deleteModelOverrides []func(
	*cobra.Command,
	*modelregistry.DeleteRegisteredModelRequest,
)

func newDeleteModel() *cobra.Command {
	cmd := &cobra.Command{}
	var deleteModelReq modelregistry.DeleteRegisteredModelRequest

	cmd.Use = "delete-model NAME"
	cmd.Short = `Delete a model.`
	cmd.Long = `Delete a model.

  Deletes a registered model.

  Arguments:
    NAME: Registered model unique name identifier.`

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

		deleteModelReq.Name = new(args[0])

		sdkReq, err := modelregistry.DeleteRegisteredModelRequestToSDK(&deleteModelReq)
		if err != nil {
			return err
		}
		response, err := w.ModelRegistry.DeleteRegisteredModel(ctx, *sdkReq)
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
	for _, fn := range deleteModelOverrides {
		fn(cmd, &deleteModelReq)
	}

	return cmd
}

// start delete-model-tag command

// Slice with functions to override default command behavior.
// Functions can be added from the `init()` function in manually curated files in this directory.
var deleteModelTagOverrides []func(
	*cobra.Command,
	*modelregistry.DeleteRegisteredModelTagRequest,
)

func newDeleteModelTag() *cobra.Command {
	cmd := &cobra.Command{}
	var deleteModelTagReq modelregistry.DeleteRegisteredModelTagRequest

	cmd.Use = "delete-model-tag NAME KEY"
	cmd.Short = `Delete a model tag.`
	cmd.Long = `Delete a model tag.

  Deletes the tag for a registered model.

  Arguments:
    NAME: Name of the registered model that the tag was logged under.
    KEY: Name of the tag. The name must be an exact match; wild-card deletion is
      not supported. Maximum size is 250 bytes.`

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

		deleteModelTagReq.Name = new(args[0])
		deleteModelTagReq.Key = new(args[1])

		sdkReq, err := modelregistry.DeleteRegisteredModelTagRequestToSDK(&deleteModelTagReq)
		if err != nil {
			return err
		}
		response, err := w.ModelRegistry.DeleteRegisteredModelTag(ctx, *sdkReq)
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
	for _, fn := range deleteModelTagOverrides {
		fn(cmd, &deleteModelTagReq)
	}

	return cmd
}

// start delete-model-version command

// Slice with functions to override default command behavior.
// Functions can be added from the `init()` function in manually curated files in this directory.
var deleteModelVersionOverrides []func(
	*cobra.Command,
	*modelregistry.DeleteModelVersionRequest,
)

func newDeleteModelVersion() *cobra.Command {
	cmd := &cobra.Command{}
	var deleteModelVersionReq modelregistry.DeleteModelVersionRequest

	cmd.Use = "delete-model-version NAME VERSION"
	cmd.Short = `Delete a model version.`
	cmd.Long = `Delete a model version.

  Deletes a model version.

  Arguments:
    NAME: Name of the registered model
    VERSION: Model version number`

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

		deleteModelVersionReq.Name = new(args[0])
		deleteModelVersionReq.Version = new(args[1])

		sdkReq, err := modelregistry.DeleteModelVersionRequestToSDK(&deleteModelVersionReq)
		if err != nil {
			return err
		}
		response, err := w.ModelRegistry.DeleteModelVersion(ctx, *sdkReq)
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
	for _, fn := range deleteModelVersionOverrides {
		fn(cmd, &deleteModelVersionReq)
	}

	return cmd
}

// start delete-model-version-tag command

// Slice with functions to override default command behavior.
// Functions can be added from the `init()` function in manually curated files in this directory.
var deleteModelVersionTagOverrides []func(
	*cobra.Command,
	*modelregistry.DeleteModelVersionTagRequest,
)

func newDeleteModelVersionTag() *cobra.Command {
	cmd := &cobra.Command{}
	var deleteModelVersionTagReq modelregistry.DeleteModelVersionTagRequest

	cmd.Use = "delete-model-version-tag NAME VERSION KEY"
	cmd.Short = `Delete a model version tag.`
	cmd.Long = `Delete a model version tag.

  Deletes a model version tag.

  Arguments:
    NAME: Name of the registered model that the tag was logged under.
    VERSION: Model version number that the tag was logged under.
    KEY: Name of the tag. The name must be an exact match; wild-card deletion is
      not supported. Maximum size is 250 bytes.`

	cmd.Annotations = make(map[string]string)
	cmd.Annotations["launch_stage"] = "GA"
	cmd.Annotations["launch_stage_display"] = "GA"

	cmd.Args = func(cmd *cobra.Command, args []string) error {
		check := root.ExactArgs(3)
		return check(cmd, args)
	}

	cmd.PreRunE = root.MustWorkspaceClient
	cmd.RunE = func(cmd *cobra.Command, args []string) (err error) {
		ctx := cmd.Context()
		w := cmdctx.WorkspaceClient(ctx)
		_ = w

		deleteModelVersionTagReq.Name = new(args[0])
		deleteModelVersionTagReq.Version = new(args[1])
		deleteModelVersionTagReq.Key = new(args[2])

		sdkReq, err := modelregistry.DeleteModelVersionTagRequestToSDK(&deleteModelVersionTagReq)
		if err != nil {
			return err
		}
		response, err := w.ModelRegistry.DeleteModelVersionTag(ctx, *sdkReq)
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
	for _, fn := range deleteModelVersionTagOverrides {
		fn(cmd, &deleteModelVersionTagReq)
	}

	return cmd
}

// start delete-transition-request command

// Slice with functions to override default command behavior.
// Functions can be added from the `init()` function in manually curated files in this directory.
var deleteTransitionRequestOverrides []func(
	*cobra.Command,
	*modelregistry.DeleteTransitionRequest,
)

func newDeleteTransitionRequest() *cobra.Command {
	cmd := &cobra.Command{}
	var deleteTransitionRequestReq modelregistry.DeleteTransitionRequest

	flags.StringPtrVar(cmd.Flags(), &deleteTransitionRequestReq.Comment, "comment", `User-provided comment on the action.`)

	cmd.Use = "delete-transition-request NAME VERSION STAGE CREATOR"
	cmd.Short = `Delete a transition request.`
	cmd.Long = `Delete a transition request.

  Cancels a model version stage transition request.

  Arguments:
    NAME: Name of the model.
    VERSION: Version of the model.
    STAGE: Target stage of the transition request. Valid values are:

      * None: The initial stage of a model version.

      * Staging: Staging or pre-production stage.

      * Production: Production stage.

      * Archived: Archived stage.
    CREATOR: Username of the user who created this request. Of the transition requests
      matching the specified details, only the one transition created by this
      user will be deleted.`

	cmd.Annotations = make(map[string]string)
	cmd.Annotations["launch_stage"] = "GA"
	cmd.Annotations["launch_stage_display"] = "GA"

	cmd.Args = func(cmd *cobra.Command, args []string) error {
		check := root.ExactArgs(4)
		return check(cmd, args)
	}

	cmd.PreRunE = root.MustWorkspaceClient
	cmd.RunE = func(cmd *cobra.Command, args []string) (err error) {
		ctx := cmd.Context()
		w := cmdctx.WorkspaceClient(ctx)
		_ = w

		deleteTransitionRequestReq.Name = new(args[0])
		deleteTransitionRequestReq.Version = new(args[1])
		deleteTransitionRequestReq.Stage = new(args[2])
		deleteTransitionRequestReq.Creator = new(args[3])

		sdkReq, err := modelregistry.DeleteTransitionRequestToSDK(&deleteTransitionRequestReq)
		if err != nil {
			return err
		}
		response, err := w.ModelRegistry.DeleteTransitionRequest(ctx, *sdkReq)
		if err != nil {
			return err
		}

		out, err := modelregistry.DeleteTransitionResponseFromSDK(response)
		if err != nil {
			return err
		}
		return cmdio.Render(ctx, out)
	}

	// Disable completions since they are not applicable.
	// Can be overridden by manual implementation in `override.go`.
	cmd.ValidArgsFunction = cobra.NoFileCompletions

	// Apply optional overrides to this command.
	for _, fn := range deleteTransitionRequestOverrides {
		fn(cmd, &deleteTransitionRequestReq)
	}

	return cmd
}

// start delete-webhook command

// Slice with functions to override default command behavior.
// Functions can be added from the `init()` function in manually curated files in this directory.
var deleteWebhookOverrides []func(
	*cobra.Command,
	*modelregistry.DeleteRegistryWebhookRequest,
)

func newDeleteWebhook() *cobra.Command {
	cmd := &cobra.Command{}
	var deleteWebhookReq modelregistry.DeleteRegistryWebhookRequest

	cmd.Use = "delete-webhook ID"
	cmd.Short = `Delete a webhook.`
	cmd.Long = `Delete a webhook.

  **NOTE:** This endpoint is in Public Preview. Deletes a registry webhook.

  Arguments:
    ID: Webhook ID required to delete a registry webhook.`

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

		deleteWebhookReq.Id = new(args[0])

		sdkReq, err := modelregistry.DeleteRegistryWebhookRequestToSDK(&deleteWebhookReq)
		if err != nil {
			return err
		}
		response, err := w.ModelRegistry.DeleteRegistryWebhook(ctx, *sdkReq)
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
	for _, fn := range deleteWebhookOverrides {
		fn(cmd, &deleteWebhookReq)
	}

	return cmd
}

// start get-latest-versions command

// Slice with functions to override default command behavior.
// Functions can be added from the `init()` function in manually curated files in this directory.
var getLatestVersionsOverrides []func(
	*cobra.Command,
	*modelregistry.ListLatestVersionsRequest,
)

func newGetLatestVersions() *cobra.Command {
	cmd := &cobra.Command{}
	var getLatestVersionsReq modelregistry.ListLatestVersionsRequest

	var getLatestVersionsJson flags.JsonFlag
	// Registered for all paginated methods. Validated at call time in the
	// method-call template. Paginated list methods never have Wait or LRO
	// branches, so the method-call path is always reached.
	var getLatestVersionsLimit int

	cmd.Flags().Var(&getLatestVersionsJson, "json", `either inline JSON string or @path/to/file.json with request body`)

	// TODO: array: stages

	// Limit flag for total result capping.
	cmd.Flags().IntVar(&getLatestVersionsLimit, "limit", 0, `Maximum number of results to return.`)

	// Hidden pagination flags (internal API parameters).

	cmd.Use = "get-latest-versions NAME"
	cmd.Short = `Get the latest version.`
	cmd.Long = `Get the latest version.

  Gets the latest version of a registered model.

  Arguments:
    NAME: Registered model unique name identifier.`

	cmd.Annotations = make(map[string]string)
	cmd.Annotations["launch_stage"] = "GA"
	cmd.Annotations["launch_stage_display"] = "GA"

	cmd.Args = func(cmd *cobra.Command, args []string) error {
		if cmd.Flags().Changed("json") {
			err := root.ExactArgs(0)(cmd, args)
			if err != nil {
				return errors.New("when --json flag is specified, no positional arguments are allowed. Provide 'name' in your JSON input")
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
		_ = w

		if cmd.Flags().Changed("json") {
			diags := getLatestVersionsJson.Unmarshal(&getLatestVersionsReq)
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
			getLatestVersionsReq.Name = new(args[0])
		}

		sdkReq, err := modelregistry.ListLatestVersionsRequestToSDK(&getLatestVersionsReq)
		if err != nil {
			return err
		}
		response, err := w.ModelRegistry.ListLatestVersions(ctx, *sdkReq)
		if err != nil {
			return err
		}
		if getLatestVersionsLimit < 0 {
			return fmt.Errorf("--limit must be a non-negative integer, got %d", getLatestVersionsLimit)
		}
		if getLatestVersionsLimit > 0 {
			ctx = cmdio.WithLimit(ctx, getLatestVersionsLimit)
		}

		out, err := modelregistry.GetLatestVersionsResponseFromSDK(response)
		if err != nil {
			return err
		}
		return cmdio.Render(ctx, out)
	}

	// Disable completions since they are not applicable.
	// Can be overridden by manual implementation in `override.go`.
	cmd.ValidArgsFunction = cobra.NoFileCompletions

	// Apply optional overrides to this command.
	for _, fn := range getLatestVersionsOverrides {
		fn(cmd, &getLatestVersionsReq)
	}

	return cmd
}

// start get-model command

// Slice with functions to override default command behavior.
// Functions can be added from the `init()` function in manually curated files in this directory.
var getModelOverrides []func(
	*cobra.Command,
	*modelregistry.GetRegisteredModelDatabricksRequest,
)

func newGetModel() *cobra.Command {
	cmd := &cobra.Command{}
	var getModelReq modelregistry.GetRegisteredModelDatabricksRequest

	cmd.Use = "get-model NAME"
	cmd.Short = `Get model.`
	cmd.Long = `Get model.

  Get the details of a model. This is a Databricks workspace version of the
  [MLflow endpoint] that also returns the model's Databricks workspace ID and
  the permission level of the requesting user on the model.

  [MLflow endpoint]: https://www.mlflow.org/docs/latest/rest-api.html#get-registeredmodel

  Arguments:
    NAME: Registered model unique name identifier.`

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

		getModelReq.Name = new(args[0])

		sdkReq, err := modelregistry.GetRegisteredModelDatabricksRequestToSDK(&getModelReq)
		if err != nil {
			return err
		}
		response, err := w.ModelRegistry.GetRegisteredModelDatabricks(ctx, *sdkReq)
		if err != nil {
			return err
		}

		out, err := modelregistry.GetRegisteredModelDatabricksResponseFromSDK(response)
		if err != nil {
			return err
		}
		return cmdio.Render(ctx, out)
	}

	// Disable completions since they are not applicable.
	// Can be overridden by manual implementation in `override.go`.
	cmd.ValidArgsFunction = cobra.NoFileCompletions

	// Apply optional overrides to this command.
	for _, fn := range getModelOverrides {
		fn(cmd, &getModelReq)
	}

	return cmd
}

// start get-model-version command

// Slice with functions to override default command behavior.
// Functions can be added from the `init()` function in manually curated files in this directory.
var getModelVersionOverrides []func(
	*cobra.Command,
	*modelregistry.GetModelVersionRequest,
)

func newGetModelVersion() *cobra.Command {
	cmd := &cobra.Command{}
	var getModelVersionReq modelregistry.GetModelVersionRequest

	cmd.Use = "get-model-version NAME VERSION"
	cmd.Short = `Get a model version.`
	cmd.Long = `Get a model version.

  Arguments:
    NAME: Name of the registered model
    VERSION: Model version number`

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

		getModelVersionReq.Name = new(args[0])
		getModelVersionReq.Version = new(args[1])

		sdkReq, err := modelregistry.GetModelVersionRequestToSDK(&getModelVersionReq)
		if err != nil {
			return err
		}
		response, err := w.ModelRegistry.GetModelVersion(ctx, *sdkReq)
		if err != nil {
			return err
		}

		out, err := modelregistry.GetModelVersionResponseFromSDK(response)
		if err != nil {
			return err
		}
		return cmdio.Render(ctx, out)
	}

	// Disable completions since they are not applicable.
	// Can be overridden by manual implementation in `override.go`.
	cmd.ValidArgsFunction = cobra.NoFileCompletions

	// Apply optional overrides to this command.
	for _, fn := range getModelVersionOverrides {
		fn(cmd, &getModelVersionReq)
	}

	return cmd
}

// start get-model-version-download-uri command

// Slice with functions to override default command behavior.
// Functions can be added from the `init()` function in manually curated files in this directory.
var getModelVersionDownloadUriOverrides []func(
	*cobra.Command,
	*modelregistry.GetModelVersionDownloadUriRequest,
)

func newGetModelVersionDownloadUri() *cobra.Command {
	cmd := &cobra.Command{}
	var getModelVersionDownloadUriReq modelregistry.GetModelVersionDownloadUriRequest

	cmd.Use = "get-model-version-download-uri NAME VERSION"
	cmd.Short = `Get a model version URI.`
	cmd.Long = `Get a model version URI.

  Gets a URI to download the model version.

  Arguments:
    NAME: Name of the registered model
    VERSION: Model version number`

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

		getModelVersionDownloadUriReq.Name = new(args[0])
		getModelVersionDownloadUriReq.Version = new(args[1])

		sdkReq, err := modelregistry.GetModelVersionDownloadUriRequestToSDK(&getModelVersionDownloadUriReq)
		if err != nil {
			return err
		}
		response, err := w.ModelRegistry.GetModelVersionDownloadUri(ctx, *sdkReq)
		if err != nil {
			return err
		}

		out, err := modelregistry.GetModelVersionDownloadUriResponseFromSDK(response)
		if err != nil {
			return err
		}
		return cmdio.Render(ctx, out)
	}

	// Disable completions since they are not applicable.
	// Can be overridden by manual implementation in `override.go`.
	cmd.ValidArgsFunction = cobra.NoFileCompletions

	// Apply optional overrides to this command.
	for _, fn := range getModelVersionDownloadUriOverrides {
		fn(cmd, &getModelVersionDownloadUriReq)
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

	cmd.Use = "get-permission-levels REGISTERED_MODEL_ID"
	cmd.Short = `Get registered model permission levels.`
	cmd.Long = `Get registered model permission levels.

  Gets the permission levels that a user can have on an object.

  Arguments:
    REGISTERED_MODEL_ID: The registered model for which to get or manage permissions.`

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

		_ = args[0] // TODO: field not in sdk-go request: registered_model_id

		getPermissionLevelsReq.RequestObjectType = new("registered-models")

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

	cmd.Use = "get-permissions REGISTERED_MODEL_ID"
	cmd.Short = `Get registered model permissions.`
	cmd.Long = `Get registered model permissions.

  Gets the permissions of a registered model. Registered models can inherit
  permissions from their root object.

  Arguments:
    REGISTERED_MODEL_ID: The registered model for which to get or manage permissions.`

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

		_ = args[0] // TODO: field not in sdk-go request: registered_model_id

		getPermissionsReq.RequestObjectType = new("registered-models")

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

// start list-models command

// Slice with functions to override default command behavior.
// Functions can be added from the `init()` function in manually curated files in this directory.
var listModelsOverrides []func(
	*cobra.Command,
	*modelregistry.ListRegisteredModelsRequest,
)

func newListModels() *cobra.Command {
	cmd := &cobra.Command{}
	var listModelsReq modelregistry.ListRegisteredModelsRequest

	// Registered for all paginated methods. Validated at call time in the
	// method-call template. Paginated list methods never have Wait or LRO
	// branches, so the method-call path is always reached.
	var listModelsLimit int

	flags.Int64PtrVar(cmd.Flags(), &listModelsReq.MaxResults, "max-results", `Maximum number of registered models desired.`)

	// Limit flag for total result capping.
	cmd.Flags().IntVar(&listModelsLimit, "limit", 0, `Maximum number of results to return.`)

	// Hidden pagination flags (internal API parameters).
	flags.StringPtrVar(cmd.Flags(), &listModelsReq.PageToken, "page-token", `Pagination token.`)
	cmd.Flags().Lookup("page-token").Hidden = true

	cmd.Use = "list-models"
	cmd.Short = `List models.`
	cmd.Long = `List models.

  Lists all available registered models, up to the limit specified in
  __max_results__.`

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

		sdkReq, err := modelregistry.ListRegisteredModelsRequestToSDK(&listModelsReq)
		if err != nil {
			return err
		}
		if listModelsLimit < 0 {
			return fmt.Errorf("--limit must be a non-negative integer, got %d", listModelsLimit)
		}
		if listModelsLimit > 0 {
			ctx = cmdio.WithLimit(ctx, listModelsLimit)
		}
		response := w.ModelRegistry.ListRegisteredModelsIter(ctx, *sdkReq)
		return cmdio.RenderIterator(ctx, cmdgen.MapSeq(response, modelregistry.RegisteredModelFromSDK))
	}

	// Disable completions since they are not applicable.
	// Can be overridden by manual implementation in `override.go`.
	cmd.ValidArgsFunction = cobra.NoFileCompletions

	// Apply optional overrides to this command.
	for _, fn := range listModelsOverrides {
		fn(cmd, &listModelsReq)
	}

	return cmd
}

// start list-transition-requests command

// Slice with functions to override default command behavior.
// Functions can be added from the `init()` function in manually curated files in this directory.
var listTransitionRequestsOverrides []func(
	*cobra.Command,
	*modelregistry.ListTransitionRequest,
)

func newListTransitionRequests() *cobra.Command {
	cmd := &cobra.Command{}
	var listTransitionRequestsReq modelregistry.ListTransitionRequest

	// Registered for all paginated methods. Validated at call time in the
	// method-call template. Paginated list methods never have Wait or LRO
	// branches, so the method-call path is always reached.
	var listTransitionRequestsLimit int

	// Limit flag for total result capping.
	cmd.Flags().IntVar(&listTransitionRequestsLimit, "limit", 0, `Maximum number of results to return.`)

	// Hidden pagination flags (internal API parameters).

	cmd.Use = "list-transition-requests NAME VERSION"
	cmd.Short = `List transition requests.`
	cmd.Long = `List transition requests.

  Gets a list of all open stage transition requests for the model version.

  Arguments:
    NAME: Name of the registered model.
    VERSION: Version of the model.`

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

		listTransitionRequestsReq.Name = new(args[0])
		listTransitionRequestsReq.Version = new(args[1])

		sdkReq, err := modelregistry.ListTransitionRequestToSDK(&listTransitionRequestsReq)
		if err != nil {
			return err
		}
		response, err := w.ModelRegistry.ListTransitionRequests(ctx, *sdkReq)
		if err != nil {
			return err
		}
		if listTransitionRequestsLimit < 0 {
			return fmt.Errorf("--limit must be a non-negative integer, got %d", listTransitionRequestsLimit)
		}
		if listTransitionRequestsLimit > 0 {
			ctx = cmdio.WithLimit(ctx, listTransitionRequestsLimit)
		}

		out, err := modelregistry.ListTransitionResponseFromSDK(response)
		if err != nil {
			return err
		}
		return cmdio.Render(ctx, out)
	}

	// Disable completions since they are not applicable.
	// Can be overridden by manual implementation in `override.go`.
	cmd.ValidArgsFunction = cobra.NoFileCompletions

	// Apply optional overrides to this command.
	for _, fn := range listTransitionRequestsOverrides {
		fn(cmd, &listTransitionRequestsReq)
	}

	return cmd
}

// start list-webhooks command

// Slice with functions to override default command behavior.
// Functions can be added from the `init()` function in manually curated files in this directory.
var listWebhooksOverrides []func(
	*cobra.Command,
	*modelregistry.ListRegistryWebhooksRequest,
)

func newListWebhooks() *cobra.Command {
	cmd := &cobra.Command{}
	var listWebhooksReq modelregistry.ListRegistryWebhooksRequest

	// Registered for all paginated methods. Validated at call time in the
	// method-call template. Paginated list methods never have Wait or LRO
	// branches, so the method-call path is always reached.
	var listWebhooksLimit int

	// TODO: array: events
	flags.Int64PtrVar(cmd.Flags(), &listWebhooksReq.MaxResults, "max-results", ``)
	flags.StringPtrVar(cmd.Flags(), &listWebhooksReq.ModelName, "model-name", `Registered model name If not specified, all webhooks associated with the specified events are listed, regardless of their associated model.`)

	// Limit flag for total result capping.
	cmd.Flags().IntVar(&listWebhooksLimit, "limit", 0, `Maximum number of results to return.`)

	// Hidden pagination flags (internal API parameters).
	flags.StringPtrVar(cmd.Flags(), &listWebhooksReq.PageToken, "page-token", `Pagination token.`)
	cmd.Flags().Lookup("page-token").Hidden = true

	cmd.Use = "list-webhooks"
	cmd.Short = `List registry webhooks.`
	cmd.Long = `List registry webhooks.

  **NOTE:** This endpoint is in Public Preview. Lists all registry webhooks.`

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

		sdkReq, err := modelregistry.ListRegistryWebhooksRequestToSDK(&listWebhooksReq)
		if err != nil {
			return err
		}
		if listWebhooksLimit < 0 {
			return fmt.Errorf("--limit must be a non-negative integer, got %d", listWebhooksLimit)
		}
		if listWebhooksLimit > 0 {
			ctx = cmdio.WithLimit(ctx, listWebhooksLimit)
		}
		response := w.ModelRegistry.ListRegistryWebhooksIter(ctx, *sdkReq)
		return cmdio.RenderIterator(ctx, cmdgen.MapSeq(response, modelregistry.RegistryWebhookFromSDK))
	}

	// Disable completions since they are not applicable.
	// Can be overridden by manual implementation in `override.go`.
	cmd.ValidArgsFunction = cobra.NoFileCompletions

	// Apply optional overrides to this command.
	for _, fn := range listWebhooksOverrides {
		fn(cmd, &listWebhooksReq)
	}

	return cmd
}

// start reject-transition-request command

// Slice with functions to override default command behavior.
// Functions can be added from the `init()` function in manually curated files in this directory.
var rejectTransitionRequestOverrides []func(
	*cobra.Command,
	*modelregistry.RejectTransitionRequest,
)

func newRejectTransitionRequest() *cobra.Command {
	cmd := &cobra.Command{}
	var rejectTransitionRequestReq modelregistry.RejectTransitionRequest

	var rejectTransitionRequestJson flags.JsonFlag

	cmd.Flags().Var(&rejectTransitionRequestJson, "json", `either inline JSON string or @path/to/file.json with request body`)

	flags.StringPtrVar(cmd.Flags(), &rejectTransitionRequestReq.Comment, "comment", `User-provided comment on the action.`)

	cmd.Use = "reject-transition-request NAME VERSION STAGE"
	cmd.Short = `Reject a transition request.`
	cmd.Long = `Reject a transition request.

  Rejects a model version stage transition request.

  Arguments:
    NAME: Name of the model.
    VERSION: Version of the model.
    STAGE: Target stage of the transition. Valid values are:

      * None: The initial stage of a model version.

      * Staging: Staging or pre-production stage.

      * Production: Production stage.

      * Archived: Archived stage.`

	cmd.Annotations = make(map[string]string)
	cmd.Annotations["launch_stage"] = "GA"
	cmd.Annotations["launch_stage_display"] = "GA"

	cmd.Args = func(cmd *cobra.Command, args []string) error {
		if cmd.Flags().Changed("json") {
			err := root.ExactArgs(0)(cmd, args)
			if err != nil {
				return errors.New("when --json flag is specified, no positional arguments are allowed. Provide 'name', 'version', 'stage' in your JSON input")
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
			diags := rejectTransitionRequestJson.Unmarshal(&rejectTransitionRequestReq)
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
			rejectTransitionRequestReq.Name = new(args[0])
		}
		if !cmd.Flags().Changed("json") {
			rejectTransitionRequestReq.Version = new(args[1])
		}
		if !cmd.Flags().Changed("json") {
			rejectTransitionRequestReq.Stage = new(args[2])
		}

		sdkReq, err := modelregistry.RejectTransitionRequestToSDK(&rejectTransitionRequestReq)
		if err != nil {
			return err
		}
		response, err := w.ModelRegistry.RejectTransitionRequest(ctx, *sdkReq)
		if err != nil {
			return err
		}

		out, err := modelregistry.RejectTransitionResponseFromSDK(response)
		if err != nil {
			return err
		}
		return cmdio.Render(ctx, out)
	}

	// Disable completions since they are not applicable.
	// Can be overridden by manual implementation in `override.go`.
	cmd.ValidArgsFunction = cobra.NoFileCompletions

	// Apply optional overrides to this command.
	for _, fn := range rejectTransitionRequestOverrides {
		fn(cmd, &rejectTransitionRequestReq)
	}

	return cmd
}

// start rename-model command

// Slice with functions to override default command behavior.
// Functions can be added from the `init()` function in manually curated files in this directory.
var renameModelOverrides []func(
	*cobra.Command,
	*modelregistry.RenameRegisteredModelRequest,
)

func newRenameModel() *cobra.Command {
	cmd := &cobra.Command{}
	var renameModelReq modelregistry.RenameRegisteredModelRequest

	var renameModelJson flags.JsonFlag

	cmd.Flags().Var(&renameModelJson, "json", `either inline JSON string or @path/to/file.json with request body`)

	flags.StringPtrVar(cmd.Flags(), &renameModelReq.NewName, "new-name", `If provided, updates the name for this registered_model.`)

	cmd.Use = "rename-model NAME"
	cmd.Short = `Rename a model.`
	cmd.Long = `Rename a model.

  Renames a registered model.

  Arguments:
    NAME: Registered model unique name identifier.`

	cmd.Annotations = make(map[string]string)
	cmd.Annotations["launch_stage"] = "GA"
	cmd.Annotations["launch_stage_display"] = "GA"

	cmd.Args = func(cmd *cobra.Command, args []string) error {
		if cmd.Flags().Changed("json") {
			err := root.ExactArgs(0)(cmd, args)
			if err != nil {
				return errors.New("when --json flag is specified, no positional arguments are allowed. Provide 'name' in your JSON input")
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
		_ = w

		if cmd.Flags().Changed("json") {
			diags := renameModelJson.Unmarshal(&renameModelReq)
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
			renameModelReq.Name = new(args[0])
		}

		sdkReq, err := modelregistry.RenameRegisteredModelRequestToSDK(&renameModelReq)
		if err != nil {
			return err
		}
		response, err := w.ModelRegistry.RenameRegisteredModel(ctx, *sdkReq)
		if err != nil {
			return err
		}

		out, err := modelregistry.RenameRegisteredModelResponseFromSDK(response)
		if err != nil {
			return err
		}
		return cmdio.Render(ctx, out)
	}

	// Disable completions since they are not applicable.
	// Can be overridden by manual implementation in `override.go`.
	cmd.ValidArgsFunction = cobra.NoFileCompletions

	// Apply optional overrides to this command.
	for _, fn := range renameModelOverrides {
		fn(cmd, &renameModelReq)
	}

	return cmd
}

// start search-model-versions command

// Slice with functions to override default command behavior.
// Functions can be added from the `init()` function in manually curated files in this directory.
var searchModelVersionsOverrides []func(
	*cobra.Command,
	*modelregistry.SearchModelVersionsRequest,
)

func newSearchModelVersions() *cobra.Command {
	cmd := &cobra.Command{}
	var searchModelVersionsReq modelregistry.SearchModelVersionsRequest

	// Registered for all paginated methods. Validated at call time in the
	// method-call template. Paginated list methods never have Wait or LRO
	// branches, so the method-call path is always reached.
	var searchModelVersionsLimit int

	flags.StringPtrVar(cmd.Flags(), &searchModelVersionsReq.Filter, "filter", `String filter condition, like "name='my-model-name'".`)
	flags.Int64PtrVar(cmd.Flags(), &searchModelVersionsReq.MaxResults, "max-results", `Maximum number of models desired.`)
	// TODO: array: order_by

	// Limit flag for total result capping.
	cmd.Flags().IntVar(&searchModelVersionsLimit, "limit", 0, `Maximum number of results to return.`)

	// Hidden pagination flags (internal API parameters).
	flags.StringPtrVar(cmd.Flags(), &searchModelVersionsReq.PageToken, "page-token", `Pagination token.`)
	cmd.Flags().Lookup("page-token").Hidden = true

	cmd.Use = "search-model-versions"
	cmd.Short = `Search model versions.`
	cmd.Long = `Search model versions.

  Searches for specific model versions based on the supplied __filter__.`

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

		sdkReq, err := modelregistry.SearchModelVersionsRequestToSDK(&searchModelVersionsReq)
		if err != nil {
			return err
		}
		if searchModelVersionsLimit < 0 {
			return fmt.Errorf("--limit must be a non-negative integer, got %d", searchModelVersionsLimit)
		}
		if searchModelVersionsLimit > 0 {
			ctx = cmdio.WithLimit(ctx, searchModelVersionsLimit)
		}
		response := w.ModelRegistry.SearchModelVersionsIter(ctx, *sdkReq)
		return cmdio.RenderIterator(ctx, cmdgen.MapSeq(response, modelregistry.ModelVersionFromSDK))
	}

	// Disable completions since they are not applicable.
	// Can be overridden by manual implementation in `override.go`.
	cmd.ValidArgsFunction = cobra.NoFileCompletions

	// Apply optional overrides to this command.
	for _, fn := range searchModelVersionsOverrides {
		fn(cmd, &searchModelVersionsReq)
	}

	return cmd
}

// start search-models command

// Slice with functions to override default command behavior.
// Functions can be added from the `init()` function in manually curated files in this directory.
var searchModelsOverrides []func(
	*cobra.Command,
	*modelregistry.SearchRegisteredModelsRequest,
)

func newSearchModels() *cobra.Command {
	cmd := &cobra.Command{}
	var searchModelsReq modelregistry.SearchRegisteredModelsRequest

	// Registered for all paginated methods. Validated at call time in the
	// method-call template. Paginated list methods never have Wait or LRO
	// branches, so the method-call path is always reached.
	var searchModelsLimit int

	flags.StringPtrVar(cmd.Flags(), &searchModelsReq.Filter, "filter", `String filter condition, like "name LIKE 'my-model-name'".`)
	flags.Int64PtrVar(cmd.Flags(), &searchModelsReq.MaxResults, "max-results", `Maximum number of models desired.`)
	// TODO: array: order_by

	// Limit flag for total result capping.
	cmd.Flags().IntVar(&searchModelsLimit, "limit", 0, `Maximum number of results to return.`)

	// Hidden pagination flags (internal API parameters).
	flags.StringPtrVar(cmd.Flags(), &searchModelsReq.PageToken, "page-token", `Pagination token.`)
	cmd.Flags().Lookup("page-token").Hidden = true

	cmd.Use = "search-models"
	cmd.Short = `Search models.`
	cmd.Long = `Search models.

  Search for registered models based on the specified __filter__.`

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

		sdkReq, err := modelregistry.SearchRegisteredModelsRequestToSDK(&searchModelsReq)
		if err != nil {
			return err
		}
		if searchModelsLimit < 0 {
			return fmt.Errorf("--limit must be a non-negative integer, got %d", searchModelsLimit)
		}
		if searchModelsLimit > 0 {
			ctx = cmdio.WithLimit(ctx, searchModelsLimit)
		}
		response := w.ModelRegistry.SearchRegisteredModelsIter(ctx, *sdkReq)
		return cmdio.RenderIterator(ctx, cmdgen.MapSeq(response, modelregistry.RegisteredModelFromSDK))
	}

	// Disable completions since they are not applicable.
	// Can be overridden by manual implementation in `override.go`.
	cmd.ValidArgsFunction = cobra.NoFileCompletions

	// Apply optional overrides to this command.
	for _, fn := range searchModelsOverrides {
		fn(cmd, &searchModelsReq)
	}

	return cmd
}

// start set-model-tag command

// Slice with functions to override default command behavior.
// Functions can be added from the `init()` function in manually curated files in this directory.
var setModelTagOverrides []func(
	*cobra.Command,
	*modelregistry.SetRegisteredModelTagRequest,
)

func newSetModelTag() *cobra.Command {
	cmd := &cobra.Command{}
	var setModelTagReq modelregistry.SetRegisteredModelTagRequest

	var setModelTagJson flags.JsonFlag

	cmd.Flags().Var(&setModelTagJson, "json", `either inline JSON string or @path/to/file.json with request body`)

	cmd.Use = "set-model-tag NAME KEY VALUE"
	cmd.Short = `Set a tag.`
	cmd.Long = `Set a tag.

  Sets a tag on a registered model.

  Arguments:
    NAME: Unique name of the model.
    KEY: Name of the tag. Maximum size depends on storage backend. If a tag with
      this name already exists, its preexisting value will be replaced by the
      specified value. All storage backends are guaranteed to support key
      values up to 250 bytes in size.
    VALUE: String value of the tag being logged. Maximum size depends on storage
      backend. All storage backends are guaranteed to support key values up to
      5000 bytes in size.`

	cmd.Annotations = make(map[string]string)
	cmd.Annotations["launch_stage"] = "GA"
	cmd.Annotations["launch_stage_display"] = "GA"

	cmd.Args = func(cmd *cobra.Command, args []string) error {
		if cmd.Flags().Changed("json") {
			err := root.ExactArgs(0)(cmd, args)
			if err != nil {
				return errors.New("when --json flag is specified, no positional arguments are allowed. Provide 'name', 'key', 'value' in your JSON input")
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
			diags := setModelTagJson.Unmarshal(&setModelTagReq)
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
			setModelTagReq.Name = new(args[0])
		}
		if !cmd.Flags().Changed("json") {
			setModelTagReq.Key = new(args[1])
		}
		if !cmd.Flags().Changed("json") {
			setModelTagReq.Value = new(args[2])
		}

		sdkReq, err := modelregistry.SetRegisteredModelTagRequestToSDK(&setModelTagReq)
		if err != nil {
			return err
		}
		response, err := w.ModelRegistry.SetRegisteredModelTag(ctx, *sdkReq)
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
	for _, fn := range setModelTagOverrides {
		fn(cmd, &setModelTagReq)
	}

	return cmd
}

// start set-model-version-tag command

// Slice with functions to override default command behavior.
// Functions can be added from the `init()` function in manually curated files in this directory.
var setModelVersionTagOverrides []func(
	*cobra.Command,
	*modelregistry.SetModelVersionTagRequest,
)

func newSetModelVersionTag() *cobra.Command {
	cmd := &cobra.Command{}
	var setModelVersionTagReq modelregistry.SetModelVersionTagRequest

	var setModelVersionTagJson flags.JsonFlag

	cmd.Flags().Var(&setModelVersionTagJson, "json", `either inline JSON string or @path/to/file.json with request body`)

	cmd.Use = "set-model-version-tag NAME VERSION KEY VALUE"
	cmd.Short = `Set a version tag.`
	cmd.Long = `Set a version tag.

  Sets a model version tag.

  Arguments:
    NAME: Unique name of the model.
    VERSION: Model version number.
    KEY: Name of the tag. Maximum size depends on storage backend. If a tag with
      this name already exists, its preexisting value will be replaced by the
      specified value. All storage backends are guaranteed to support key
      values up to 250 bytes in size.
    VALUE: String value of the tag being logged. Maximum size depends on storage
      backend. All storage backends are guaranteed to support key values up to
      5000 bytes in size.`

	cmd.Annotations = make(map[string]string)
	cmd.Annotations["launch_stage"] = "GA"
	cmd.Annotations["launch_stage_display"] = "GA"

	cmd.Args = func(cmd *cobra.Command, args []string) error {
		if cmd.Flags().Changed("json") {
			err := root.ExactArgs(0)(cmd, args)
			if err != nil {
				return errors.New("when --json flag is specified, no positional arguments are allowed. Provide 'name', 'version', 'key', 'value' in your JSON input")
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
			diags := setModelVersionTagJson.Unmarshal(&setModelVersionTagReq)
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
			setModelVersionTagReq.Name = new(args[0])
		}
		if !cmd.Flags().Changed("json") {
			setModelVersionTagReq.Version = new(args[1])
		}
		if !cmd.Flags().Changed("json") {
			setModelVersionTagReq.Key = new(args[2])
		}
		if !cmd.Flags().Changed("json") {
			setModelVersionTagReq.Value = new(args[3])
		}

		sdkReq, err := modelregistry.SetModelVersionTagRequestToSDK(&setModelVersionTagReq)
		if err != nil {
			return err
		}
		response, err := w.ModelRegistry.SetModelVersionTag(ctx, *sdkReq)
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
	for _, fn := range setModelVersionTagOverrides {
		fn(cmd, &setModelVersionTagReq)
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

	cmd.Use = "set-permissions REGISTERED_MODEL_ID"
	cmd.Short = `Set registered model permissions.`
	cmd.Long = `Set registered model permissions.

  Sets permissions on an object, replacing existing permissions if they exist.
  Deletes all direct permissions if none are specified. Objects can inherit
  permissions from their root object.

  Arguments:
    REGISTERED_MODEL_ID: The registered model for which to get or manage permissions.`

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
		_ = args[0] // TODO: field not in sdk-go request: registered_model_id

		setPermissionsReq.RequestObjectType = new("registered-models")

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

// start test-registry-webhook command

// Slice with functions to override default command behavior.
// Functions can be added from the `init()` function in manually curated files in this directory.
var testRegistryWebhookOverrides []func(
	*cobra.Command,
	*modelregistry.TestRegistryWebhookRequest,
)

func newTestRegistryWebhook() *cobra.Command {
	cmd := &cobra.Command{}
	var testRegistryWebhookReq modelregistry.TestRegistryWebhookRequest

	var testRegistryWebhookJson flags.JsonFlag

	cmd.Flags().Var(&testRegistryWebhookJson, "json", `either inline JSON string or @path/to/file.json with request body`)

	flags.EnumVar(cmd.Flags(), &testRegistryWebhookReq.Event, "event", `If event is specified, the test trigger uses the specified event. Supported values: [
  COMMENT_CREATED,
  MODEL_VERSION_CREATED,
  MODEL_VERSION_TAG_SET,
  MODEL_VERSION_TRANSITIONED_STAGE,
  MODEL_VERSION_TRANSITIONED_TO_ARCHIVED,
  MODEL_VERSION_TRANSITIONED_TO_PRODUCTION,
  MODEL_VERSION_TRANSITIONED_TO_STAGING,
  REGISTERED_MODEL_CREATED,
  TRANSITION_REQUEST_CREATED,
  TRANSITION_REQUEST_TO_ARCHIVED_CREATED,
  TRANSITION_REQUEST_TO_PRODUCTION_CREATED,
  TRANSITION_REQUEST_TO_STAGING_CREATED,
]`, "RegistryWebhookEvent", "COMMENT_CREATED", "MODEL_VERSION_CREATED", "MODEL_VERSION_TAG_SET", "MODEL_VERSION_TRANSITIONED_STAGE", "MODEL_VERSION_TRANSITIONED_TO_ARCHIVED", "MODEL_VERSION_TRANSITIONED_TO_PRODUCTION", "MODEL_VERSION_TRANSITIONED_TO_STAGING", "REGISTERED_MODEL_CREATED", "TRANSITION_REQUEST_CREATED", "TRANSITION_REQUEST_TO_ARCHIVED_CREATED", "TRANSITION_REQUEST_TO_PRODUCTION_CREATED", "TRANSITION_REQUEST_TO_STAGING_CREATED")

	cmd.Use = "test-registry-webhook ID"
	cmd.Short = `Test a webhook.`
	cmd.Long = `Test a webhook.

  **NOTE:** This endpoint is in Public Preview. Tests a registry webhook.

  Arguments:
    ID: Webhook ID`

	cmd.Annotations = make(map[string]string)
	cmd.Annotations["launch_stage"] = "GA"
	cmd.Annotations["launch_stage_display"] = "GA"

	cmd.Args = func(cmd *cobra.Command, args []string) error {
		if cmd.Flags().Changed("json") {
			err := root.ExactArgs(0)(cmd, args)
			if err != nil {
				return errors.New("when --json flag is specified, no positional arguments are allowed. Provide 'id' in your JSON input")
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
		_ = w

		if cmd.Flags().Changed("json") {
			diags := testRegistryWebhookJson.Unmarshal(&testRegistryWebhookReq)
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
			testRegistryWebhookReq.Id = new(args[0])
		}

		sdkReq, err := modelregistry.TestRegistryWebhookRequestToSDK(&testRegistryWebhookReq)
		if err != nil {
			return err
		}
		response, err := w.ModelRegistry.TestRegistryWebhook(ctx, *sdkReq)
		if err != nil {
			return err
		}

		out, err := modelregistry.TestRegistryWebhookResponseFromSDK(response)
		if err != nil {
			return err
		}
		return cmdio.Render(ctx, out)
	}

	// Disable completions since they are not applicable.
	// Can be overridden by manual implementation in `override.go`.
	cmd.ValidArgsFunction = cobra.NoFileCompletions

	// Apply optional overrides to this command.
	for _, fn := range testRegistryWebhookOverrides {
		fn(cmd, &testRegistryWebhookReq)
	}

	return cmd
}

// start transition-stage command

// Slice with functions to override default command behavior.
// Functions can be added from the `init()` function in manually curated files in this directory.
var transitionStageOverrides []func(
	*cobra.Command,
	*modelregistry.TransitionModelVersionStageDatabricksRequest,
)

func newTransitionStage() *cobra.Command {
	cmd := &cobra.Command{}
	var transitionStageReq modelregistry.TransitionModelVersionStageDatabricksRequest

	var transitionStageJson flags.JsonFlag

	cmd.Flags().Var(&transitionStageJson, "json", `either inline JSON string or @path/to/file.json with request body`)

	flags.StringPtrVar(cmd.Flags(), &transitionStageReq.Comment, "comment", `User-provided comment on the action.`)

	cmd.Use = "transition-stage NAME VERSION STAGE ARCHIVE_EXISTING_VERSIONS"
	cmd.Short = `Transition a stage.`
	cmd.Long = `Transition a stage.

  Transition a model version's stage. This is a Databricks workspace version of
  the [MLflow endpoint] that also accepts a comment associated with the
  transition to be recorded.

  [MLflow endpoint]: https://www.mlflow.org/docs/latest/rest-api.html#transition-modelversion-stage

  Arguments:
    NAME: Name of the model.
    VERSION: Version of the model.
    STAGE: Target stage of the transition. Valid values are:

      * None: The initial stage of a model version.

      * Staging: Staging or pre-production stage.

      * Production: Production stage.

      * Archived: Archived stage.
    ARCHIVE_EXISTING_VERSIONS: Specifies whether to archive all current model versions in the target
      stage.`

	cmd.Annotations = make(map[string]string)
	cmd.Annotations["launch_stage"] = "GA"
	cmd.Annotations["launch_stage_display"] = "GA"

	cmd.Args = func(cmd *cobra.Command, args []string) error {
		if cmd.Flags().Changed("json") {
			err := root.ExactArgs(0)(cmd, args)
			if err != nil {
				return errors.New("when --json flag is specified, no positional arguments are allowed. Provide 'name', 'version', 'stage', 'archive_existing_versions' in your JSON input")
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
			diags := transitionStageJson.Unmarshal(&transitionStageReq)
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
			transitionStageReq.Name = new(args[0])
		}
		if !cmd.Flags().Changed("json") {
			transitionStageReq.Version = new(args[1])
		}
		if !cmd.Flags().Changed("json") {
			transitionStageReq.Stage = new(args[2])
		}
		if !cmd.Flags().Changed("json") {
			var transitionStageArchiveExistingVersionsArg bool
			_, err = fmt.Sscan(args[3], &transitionStageArchiveExistingVersionsArg)
			if err != nil {
				return fmt.Errorf("invalid ARCHIVE_EXISTING_VERSIONS: %s", args[3])
			}
			transitionStageReq.ArchiveExistingVersions = &transitionStageArchiveExistingVersionsArg
		}

		sdkReq, err := modelregistry.TransitionModelVersionStageDatabricksRequestToSDK(&transitionStageReq)
		if err != nil {
			return err
		}
		response, err := w.ModelRegistry.TransitionModelVersionStageDatabricks(ctx, *sdkReq)
		if err != nil {
			return err
		}

		out, err := modelregistry.TransitionModelVersionStageDatabricksResponseFromSDK(response)
		if err != nil {
			return err
		}
		return cmdio.Render(ctx, out)
	}

	// Disable completions since they are not applicable.
	// Can be overridden by manual implementation in `override.go`.
	cmd.ValidArgsFunction = cobra.NoFileCompletions

	// Apply optional overrides to this command.
	for _, fn := range transitionStageOverrides {
		fn(cmd, &transitionStageReq)
	}

	return cmd
}

// start update-comment command

// Slice with functions to override default command behavior.
// Functions can be added from the `init()` function in manually curated files in this directory.
var updateCommentOverrides []func(
	*cobra.Command,
	*modelregistry.UpdateCommentRequest,
)

func newUpdateComment() *cobra.Command {
	cmd := &cobra.Command{}
	var updateCommentReq modelregistry.UpdateCommentRequest

	var updateCommentJson flags.JsonFlag

	cmd.Flags().Var(&updateCommentJson, "json", `either inline JSON string or @path/to/file.json with request body`)

	cmd.Use = "update-comment ID COMMENT"
	cmd.Short = `Update a comment.`
	cmd.Long = `Update a comment.

  Post an edit to a comment on a model version.

  Arguments:
    ID: Unique identifier of an activity
    COMMENT: User-provided comment on the action.`

	cmd.Annotations = make(map[string]string)
	cmd.Annotations["launch_stage"] = "GA"
	cmd.Annotations["launch_stage_display"] = "GA"

	cmd.Args = func(cmd *cobra.Command, args []string) error {
		if cmd.Flags().Changed("json") {
			err := root.ExactArgs(0)(cmd, args)
			if err != nil {
				return errors.New("when --json flag is specified, no positional arguments are allowed. Provide 'id', 'comment' in your JSON input")
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
			diags := updateCommentJson.Unmarshal(&updateCommentReq)
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
			updateCommentReq.Id = new(args[0])
		}
		if !cmd.Flags().Changed("json") {
			updateCommentReq.Comment = new(args[1])
		}

		sdkReq, err := modelregistry.UpdateCommentRequestToSDK(&updateCommentReq)
		if err != nil {
			return err
		}
		response, err := w.ModelRegistry.UpdateComment(ctx, *sdkReq)
		if err != nil {
			return err
		}

		out, err := modelregistry.UpdateCommentResponseFromSDK(response)
		if err != nil {
			return err
		}
		return cmdio.Render(ctx, out)
	}

	// Disable completions since they are not applicable.
	// Can be overridden by manual implementation in `override.go`.
	cmd.ValidArgsFunction = cobra.NoFileCompletions

	// Apply optional overrides to this command.
	for _, fn := range updateCommentOverrides {
		fn(cmd, &updateCommentReq)
	}

	return cmd
}

// start update-model command

// Slice with functions to override default command behavior.
// Functions can be added from the `init()` function in manually curated files in this directory.
var updateModelOverrides []func(
	*cobra.Command,
	*modelregistry.UpdateRegisteredModelRequest,
)

func newUpdateModel() *cobra.Command {
	cmd := &cobra.Command{}
	var updateModelReq modelregistry.UpdateRegisteredModelRequest

	var updateModelJson flags.JsonFlag

	cmd.Flags().Var(&updateModelJson, "json", `either inline JSON string or @path/to/file.json with request body`)

	flags.StringPtrVar(cmd.Flags(), &updateModelReq.Description, "description", `If provided, updates the description for this registered_model.`)

	cmd.Use = "update-model NAME"
	cmd.Short = `Update model.`
	cmd.Long = `Update model.

  Updates a registered model.

  Arguments:
    NAME: Registered model unique name identifier.`

	cmd.Annotations = make(map[string]string)
	cmd.Annotations["launch_stage"] = "GA"
	cmd.Annotations["launch_stage_display"] = "GA"

	cmd.Args = func(cmd *cobra.Command, args []string) error {
		if cmd.Flags().Changed("json") {
			err := root.ExactArgs(0)(cmd, args)
			if err != nil {
				return errors.New("when --json flag is specified, no positional arguments are allowed. Provide 'name' in your JSON input")
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
		_ = w

		if cmd.Flags().Changed("json") {
			diags := updateModelJson.Unmarshal(&updateModelReq)
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
			updateModelReq.Name = new(args[0])
		}

		sdkReq, err := modelregistry.UpdateRegisteredModelRequestToSDK(&updateModelReq)
		if err != nil {
			return err
		}
		response, err := w.ModelRegistry.UpdateRegisteredModel(ctx, *sdkReq)
		if err != nil {
			return err
		}

		out, err := modelregistry.UpdateRegisteredModelResponseFromSDK(response)
		if err != nil {
			return err
		}
		return cmdio.Render(ctx, out)
	}

	// Disable completions since they are not applicable.
	// Can be overridden by manual implementation in `override.go`.
	cmd.ValidArgsFunction = cobra.NoFileCompletions

	// Apply optional overrides to this command.
	for _, fn := range updateModelOverrides {
		fn(cmd, &updateModelReq)
	}

	return cmd
}

// start update-model-version command

// Slice with functions to override default command behavior.
// Functions can be added from the `init()` function in manually curated files in this directory.
var updateModelVersionOverrides []func(
	*cobra.Command,
	*modelregistry.UpdateModelVersionRequest,
)

func newUpdateModelVersion() *cobra.Command {
	cmd := &cobra.Command{}
	var updateModelVersionReq modelregistry.UpdateModelVersionRequest

	var updateModelVersionJson flags.JsonFlag

	cmd.Flags().Var(&updateModelVersionJson, "json", `either inline JSON string or @path/to/file.json with request body`)

	flags.StringPtrVar(cmd.Flags(), &updateModelVersionReq.Description, "description", `If provided, updates the description for this registered_model.`)

	cmd.Use = "update-model-version NAME VERSION"
	cmd.Short = `Update model version.`
	cmd.Long = `Update model version.

  Updates the model version.

  Arguments:
    NAME: Name of the registered model
    VERSION: Model version number`

	cmd.Annotations = make(map[string]string)
	cmd.Annotations["launch_stage"] = "GA"
	cmd.Annotations["launch_stage_display"] = "GA"

	cmd.Args = func(cmd *cobra.Command, args []string) error {
		if cmd.Flags().Changed("json") {
			err := root.ExactArgs(0)(cmd, args)
			if err != nil {
				return errors.New("when --json flag is specified, no positional arguments are allowed. Provide 'name', 'version' in your JSON input")
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
			diags := updateModelVersionJson.Unmarshal(&updateModelVersionReq)
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
			updateModelVersionReq.Name = new(args[0])
		}
		if !cmd.Flags().Changed("json") {
			updateModelVersionReq.Version = new(args[1])
		}

		sdkReq, err := modelregistry.UpdateModelVersionRequestToSDK(&updateModelVersionReq)
		if err != nil {
			return err
		}
		response, err := w.ModelRegistry.UpdateModelVersion(ctx, *sdkReq)
		if err != nil {
			return err
		}

		out, err := modelregistry.UpdateModelVersionResponseFromSDK(response)
		if err != nil {
			return err
		}
		return cmdio.Render(ctx, out)
	}

	// Disable completions since they are not applicable.
	// Can be overridden by manual implementation in `override.go`.
	cmd.ValidArgsFunction = cobra.NoFileCompletions

	// Apply optional overrides to this command.
	for _, fn := range updateModelVersionOverrides {
		fn(cmd, &updateModelVersionReq)
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

	cmd.Use = "update-permissions REGISTERED_MODEL_ID"
	cmd.Short = `Update registered model permissions.`
	cmd.Long = `Update registered model permissions.

  Updates the permissions on a registered model. Registered models can inherit
  permissions from their root object.

  Arguments:
    REGISTERED_MODEL_ID: The registered model for which to get or manage permissions.`

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
		_ = args[0] // TODO: field not in sdk-go request: registered_model_id

		updatePermissionsReq.RequestObjectType = new("registered-models")

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

// start update-webhook command

// Slice with functions to override default command behavior.
// Functions can be added from the `init()` function in manually curated files in this directory.
var updateWebhookOverrides []func(
	*cobra.Command,
	*modelregistry.UpdateRegistryWebhookRequest,
)

func newUpdateWebhook() *cobra.Command {
	cmd := &cobra.Command{}
	var updateWebhookReq modelregistry.UpdateRegistryWebhookRequest

	var updateWebhookJson flags.JsonFlag

	cmd.Flags().Var(&updateWebhookJson, "json", `either inline JSON string or @path/to/file.json with request body`)

	flags.StringPtrVar(cmd.Flags(), &updateWebhookReq.Description, "description", `User-specified description for the webhook.`)
	// TODO: array: events
	// TODO: complex arg: http_url_spec
	// TODO: complex arg: job_spec
	flags.EnumVar(cmd.Flags(), &updateWebhookReq.Status, "status", `Supported values: [ACTIVE, DISABLED, TEST_MODE]`, "RegistryWebhookStatus", "ACTIVE", "DISABLED", "TEST_MODE")

	cmd.Use = "update-webhook ID"
	cmd.Short = `Update a webhook.`
	cmd.Long = `Update a webhook.

  **NOTE:** This endpoint is in Public Preview. Updates a registry webhook.

  Arguments:
    ID: Webhook ID`

	cmd.Annotations = make(map[string]string)
	cmd.Annotations["launch_stage"] = "GA"
	cmd.Annotations["launch_stage_display"] = "GA"

	cmd.Args = func(cmd *cobra.Command, args []string) error {
		if cmd.Flags().Changed("json") {
			err := root.ExactArgs(0)(cmd, args)
			if err != nil {
				return errors.New("when --json flag is specified, no positional arguments are allowed. Provide 'id' in your JSON input")
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
		_ = w

		if cmd.Flags().Changed("json") {
			diags := updateWebhookJson.Unmarshal(&updateWebhookReq)
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
			updateWebhookReq.Id = new(args[0])
		}

		sdkReq, err := modelregistry.UpdateRegistryWebhookRequestToSDK(&updateWebhookReq)
		if err != nil {
			return err
		}
		response, err := w.ModelRegistry.UpdateRegistryWebhook(ctx, *sdkReq)
		if err != nil {
			return err
		}

		out, err := modelregistry.UpdateRegistryWebhookResponseFromSDK(response)
		if err != nil {
			return err
		}
		return cmdio.Render(ctx, out)
	}

	// Disable completions since they are not applicable.
	// Can be overridden by manual implementation in `override.go`.
	cmd.ValidArgsFunction = cobra.NoFileCompletions

	// Apply optional overrides to this command.
	for _, fn := range updateWebhookOverrides {
		fn(cmd, &updateWebhookReq)
	}

	return cmd
}

// end service ModelRegistry
