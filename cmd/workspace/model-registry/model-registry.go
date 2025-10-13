// Code generated from OpenAPI specs by Databricks SDK Generator. DO NOT EDIT.

package model_registry

import (
	"fmt"

	"github.com/databricks/cli/cmd/root"
	"github.com/databricks/cli/libs/cmdctx"
	"github.com/databricks/cli/libs/cmdio"
	"github.com/databricks/cli/libs/flags"
	"github.com/databricks/databricks-sdk-go/service/ml"
	"github.com/spf13/cobra"
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
		Annotations: map[string]string{
			"package": "ml",
		},
		RunE: root.ReportUnknownSubcommand,
	}

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
	*ml.ApproveTransitionRequest,
)

func newApproveTransitionRequest() *cobra.Command {
	cmd := &cobra.Command{}

	var approveTransitionRequestReq ml.ApproveTransitionRequest
	var approveTransitionRequestJson flags.JsonFlag

	cmd.Flags().Var(&approveTransitionRequestJson, "json", `either inline JSON string or @path/to/file.json with request body`)

	cmd.Flags().StringVar(&approveTransitionRequestReq.Comment, "comment", approveTransitionRequestReq.Comment, `User-provided comment on the action.`)

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

	cmd.Args = func(cmd *cobra.Command, args []string) error {
		if cmd.Flags().Changed("json") {
			err := root.ExactArgs(0)(cmd, args)
			if err != nil {
				return fmt.Errorf("when --json flag is specified, no positional arguments are required. Provide 'name', 'version', 'stage', 'archive_existing_versions' in your JSON input")
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
			diags := approveTransitionRequestJson.Unmarshal(&approveTransitionRequestReq)
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
			approveTransitionRequestReq.Name = args[0]
		}
		if !cmd.Flags().Changed("json") {
			approveTransitionRequestReq.Version = args[1]
		}
		if !cmd.Flags().Changed("json") {
			approveTransitionRequestReq.Stage = args[2]
		}
		if !cmd.Flags().Changed("json") {
			_, err = fmt.Sscan(args[3], &approveTransitionRequestReq.ArchiveExistingVersions)
			if err != nil {
				return fmt.Errorf("invalid ARCHIVE_EXISTING_VERSIONS: %s", args[3])
			}
		}

		response, err := w.ModelRegistry.ApproveTransitionRequest(ctx, approveTransitionRequestReq)
		if err != nil {
			return err
		}
		return cmdio.Render(ctx, response)
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
	*ml.CreateComment,
)

func newCreateComment() *cobra.Command {
	cmd := &cobra.Command{}

	var createCommentReq ml.CreateComment
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

	cmd.Args = func(cmd *cobra.Command, args []string) error {
		if cmd.Flags().Changed("json") {
			err := root.ExactArgs(0)(cmd, args)
			if err != nil {
				return fmt.Errorf("when --json flag is specified, no positional arguments are required. Provide 'name', 'version', 'comment' in your JSON input")
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
			diags := createCommentJson.Unmarshal(&createCommentReq)
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
			createCommentReq.Name = args[0]
		}
		if !cmd.Flags().Changed("json") {
			createCommentReq.Version = args[1]
		}
		if !cmd.Flags().Changed("json") {
			createCommentReq.Comment = args[2]
		}

		response, err := w.ModelRegistry.CreateComment(ctx, createCommentReq)
		if err != nil {
			return err
		}
		return cmdio.Render(ctx, response)
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
	*ml.CreateModelRequest,
)

func newCreateModel() *cobra.Command {
	cmd := &cobra.Command{}

	var createModelReq ml.CreateModelRequest
	var createModelJson flags.JsonFlag

	cmd.Flags().Var(&createModelJson, "json", `either inline JSON string or @path/to/file.json with request body`)

	cmd.Flags().StringVar(&createModelReq.Description, "description", createModelReq.Description, `Optional description for registered model.`)
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

	cmd.Args = func(cmd *cobra.Command, args []string) error {
		if cmd.Flags().Changed("json") {
			err := root.ExactArgs(0)(cmd, args)
			if err != nil {
				return fmt.Errorf("when --json flag is specified, no positional arguments are required. Provide 'name' in your JSON input")
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
			diags := createModelJson.Unmarshal(&createModelReq)
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
			createModelReq.Name = args[0]
		}

		response, err := w.ModelRegistry.CreateModel(ctx, createModelReq)
		if err != nil {
			return err
		}
		return cmdio.Render(ctx, response)
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
	*ml.CreateModelVersionRequest,
)

func newCreateModelVersion() *cobra.Command {
	cmd := &cobra.Command{}

	var createModelVersionReq ml.CreateModelVersionRequest
	var createModelVersionJson flags.JsonFlag

	cmd.Flags().Var(&createModelVersionJson, "json", `either inline JSON string or @path/to/file.json with request body`)

	cmd.Flags().StringVar(&createModelVersionReq.Description, "description", createModelVersionReq.Description, `Optional description for model version.`)
	cmd.Flags().StringVar(&createModelVersionReq.RunId, "run-id", createModelVersionReq.RunId, `MLflow run ID for correlation, if source was generated by an experiment run in MLflow tracking server.`)
	cmd.Flags().StringVar(&createModelVersionReq.RunLink, "run-link", createModelVersionReq.RunLink, `MLflow run link - this is the exact link of the run that generated this model version, potentially hosted at another instance of MLflow.`)
	// TODO: array: tags

	cmd.Use = "create-model-version NAME SOURCE"
	cmd.Short = `Create a model version.`
	cmd.Long = `Create a model version.
  
  Creates a model version.

  Arguments:
    NAME: Register model under this name
    SOURCE: URI indicating the location of the model artifacts.`

	cmd.Annotations = make(map[string]string)

	cmd.Args = func(cmd *cobra.Command, args []string) error {
		if cmd.Flags().Changed("json") {
			err := root.ExactArgs(0)(cmd, args)
			if err != nil {
				return fmt.Errorf("when --json flag is specified, no positional arguments are required. Provide 'name', 'source' in your JSON input")
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
			diags := createModelVersionJson.Unmarshal(&createModelVersionReq)
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
			createModelVersionReq.Name = args[0]
		}
		if !cmd.Flags().Changed("json") {
			createModelVersionReq.Source = args[1]
		}

		response, err := w.ModelRegistry.CreateModelVersion(ctx, createModelVersionReq)
		if err != nil {
			return err
		}
		return cmdio.Render(ctx, response)
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
	*ml.CreateTransitionRequest,
)

func newCreateTransitionRequest() *cobra.Command {
	cmd := &cobra.Command{}

	var createTransitionRequestReq ml.CreateTransitionRequest
	var createTransitionRequestJson flags.JsonFlag

	cmd.Flags().Var(&createTransitionRequestJson, "json", `either inline JSON string or @path/to/file.json with request body`)

	cmd.Flags().StringVar(&createTransitionRequestReq.Comment, "comment", createTransitionRequestReq.Comment, `User-provided comment on the action.`)

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

	cmd.Args = func(cmd *cobra.Command, args []string) error {
		if cmd.Flags().Changed("json") {
			err := root.ExactArgs(0)(cmd, args)
			if err != nil {
				return fmt.Errorf("when --json flag is specified, no positional arguments are required. Provide 'name', 'version', 'stage' in your JSON input")
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
			diags := createTransitionRequestJson.Unmarshal(&createTransitionRequestReq)
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
			createTransitionRequestReq.Name = args[0]
		}
		if !cmd.Flags().Changed("json") {
			createTransitionRequestReq.Version = args[1]
		}
		if !cmd.Flags().Changed("json") {
			createTransitionRequestReq.Stage = args[2]
		}

		response, err := w.ModelRegistry.CreateTransitionRequest(ctx, createTransitionRequestReq)
		if err != nil {
			return err
		}
		return cmdio.Render(ctx, response)
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
	*ml.CreateRegistryWebhook,
)

func newCreateWebhook() *cobra.Command {
	cmd := &cobra.Command{}

	var createWebhookReq ml.CreateRegistryWebhook
	var createWebhookJson flags.JsonFlag

	cmd.Flags().Var(&createWebhookJson, "json", `either inline JSON string or @path/to/file.json with request body`)

	cmd.Flags().StringVar(&createWebhookReq.Description, "description", createWebhookReq.Description, `User-specified description for the webhook.`)
	// TODO: complex arg: http_url_spec
	// TODO: complex arg: job_spec
	cmd.Flags().StringVar(&createWebhookReq.ModelName, "model-name", createWebhookReq.ModelName, `If model name is not specified, a registry-wide webhook is created that listens for the specified events across all versions of all registered models.`)
	cmd.Flags().Var(&createWebhookReq.Status, "status", `Enable or disable triggering the webhook, or put the webhook into test mode. Supported values: [ACTIVE, DISABLED, TEST_MODE]`)

	cmd.Use = "create-webhook"
	cmd.Short = `Create a webhook.`
	cmd.Long = `Create a webhook.
  
  **NOTE:** This endpoint is in Public Preview. Creates a registry webhook.`

	cmd.Annotations = make(map[string]string)

	cmd.PreRunE = root.MustWorkspaceClient
	cmd.RunE = func(cmd *cobra.Command, args []string) (err error) {
		ctx := cmd.Context()
		w := cmdctx.WorkspaceClient(ctx)

		if cmd.Flags().Changed("json") {
			diags := createWebhookJson.Unmarshal(&createWebhookReq)
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

		response, err := w.ModelRegistry.CreateWebhook(ctx, createWebhookReq)
		if err != nil {
			return err
		}
		return cmdio.Render(ctx, response)
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
	*ml.DeleteCommentRequest,
)

func newDeleteComment() *cobra.Command {
	cmd := &cobra.Command{}

	var deleteCommentReq ml.DeleteCommentRequest

	cmd.Use = "delete-comment ID"
	cmd.Short = `Delete a comment.`
	cmd.Long = `Delete a comment.
  
  Deletes a comment on a model version.

  Arguments:
    ID: Unique identifier of an activity`

	cmd.Annotations = make(map[string]string)

	cmd.Args = func(cmd *cobra.Command, args []string) error {
		check := root.ExactArgs(1)
		return check(cmd, args)
	}

	cmd.PreRunE = root.MustWorkspaceClient
	cmd.RunE = func(cmd *cobra.Command, args []string) (err error) {
		ctx := cmd.Context()
		w := cmdctx.WorkspaceClient(ctx)

		deleteCommentReq.Id = args[0]

		err = w.ModelRegistry.DeleteComment(ctx, deleteCommentReq)
		if err != nil {
			return err
		}
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
	*ml.DeleteModelRequest,
)

func newDeleteModel() *cobra.Command {
	cmd := &cobra.Command{}

	var deleteModelReq ml.DeleteModelRequest

	cmd.Use = "delete-model NAME"
	cmd.Short = `Delete a model.`
	cmd.Long = `Delete a model.
  
  Deletes a registered model.

  Arguments:
    NAME: Registered model unique name identifier.`

	cmd.Annotations = make(map[string]string)

	cmd.Args = func(cmd *cobra.Command, args []string) error {
		check := root.ExactArgs(1)
		return check(cmd, args)
	}

	cmd.PreRunE = root.MustWorkspaceClient
	cmd.RunE = func(cmd *cobra.Command, args []string) (err error) {
		ctx := cmd.Context()
		w := cmdctx.WorkspaceClient(ctx)

		deleteModelReq.Name = args[0]

		err = w.ModelRegistry.DeleteModel(ctx, deleteModelReq)
		if err != nil {
			return err
		}
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
	*ml.DeleteModelTagRequest,
)

func newDeleteModelTag() *cobra.Command {
	cmd := &cobra.Command{}

	var deleteModelTagReq ml.DeleteModelTagRequest

	cmd.Use = "delete-model-tag NAME KEY"
	cmd.Short = `Delete a model tag.`
	cmd.Long = `Delete a model tag.
  
  Deletes the tag for a registered model.

  Arguments:
    NAME: Name of the registered model that the tag was logged under.
    KEY: Name of the tag. The name must be an exact match; wild-card deletion is
      not supported. Maximum size is 250 bytes.`

	cmd.Annotations = make(map[string]string)

	cmd.Args = func(cmd *cobra.Command, args []string) error {
		check := root.ExactArgs(2)
		return check(cmd, args)
	}

	cmd.PreRunE = root.MustWorkspaceClient
	cmd.RunE = func(cmd *cobra.Command, args []string) (err error) {
		ctx := cmd.Context()
		w := cmdctx.WorkspaceClient(ctx)

		deleteModelTagReq.Name = args[0]
		deleteModelTagReq.Key = args[1]

		err = w.ModelRegistry.DeleteModelTag(ctx, deleteModelTagReq)
		if err != nil {
			return err
		}
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
	*ml.DeleteModelVersionRequest,
)

func newDeleteModelVersion() *cobra.Command {
	cmd := &cobra.Command{}

	var deleteModelVersionReq ml.DeleteModelVersionRequest

	cmd.Use = "delete-model-version NAME VERSION"
	cmd.Short = `Delete a model version.`
	cmd.Long = `Delete a model version.
  
  Deletes a model version.

  Arguments:
    NAME: Name of the registered model
    VERSION: Model version number`

	cmd.Annotations = make(map[string]string)

	cmd.Args = func(cmd *cobra.Command, args []string) error {
		check := root.ExactArgs(2)
		return check(cmd, args)
	}

	cmd.PreRunE = root.MustWorkspaceClient
	cmd.RunE = func(cmd *cobra.Command, args []string) (err error) {
		ctx := cmd.Context()
		w := cmdctx.WorkspaceClient(ctx)

		deleteModelVersionReq.Name = args[0]
		deleteModelVersionReq.Version = args[1]

		err = w.ModelRegistry.DeleteModelVersion(ctx, deleteModelVersionReq)
		if err != nil {
			return err
		}
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
	*ml.DeleteModelVersionTagRequest,
)

func newDeleteModelVersionTag() *cobra.Command {
	cmd := &cobra.Command{}

	var deleteModelVersionTagReq ml.DeleteModelVersionTagRequest

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

	cmd.Args = func(cmd *cobra.Command, args []string) error {
		check := root.ExactArgs(3)
		return check(cmd, args)
	}

	cmd.PreRunE = root.MustWorkspaceClient
	cmd.RunE = func(cmd *cobra.Command, args []string) (err error) {
		ctx := cmd.Context()
		w := cmdctx.WorkspaceClient(ctx)

		deleteModelVersionTagReq.Name = args[0]
		deleteModelVersionTagReq.Version = args[1]
		deleteModelVersionTagReq.Key = args[2]

		err = w.ModelRegistry.DeleteModelVersionTag(ctx, deleteModelVersionTagReq)
		if err != nil {
			return err
		}
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
	*ml.DeleteTransitionRequestRequest,
)

func newDeleteTransitionRequest() *cobra.Command {
	cmd := &cobra.Command{}

	var deleteTransitionRequestReq ml.DeleteTransitionRequestRequest

	cmd.Flags().StringVar(&deleteTransitionRequestReq.Comment, "comment", deleteTransitionRequestReq.Comment, `User-provided comment on the action.`)

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

	cmd.Args = func(cmd *cobra.Command, args []string) error {
		check := root.ExactArgs(4)
		return check(cmd, args)
	}

	cmd.PreRunE = root.MustWorkspaceClient
	cmd.RunE = func(cmd *cobra.Command, args []string) (err error) {
		ctx := cmd.Context()
		w := cmdctx.WorkspaceClient(ctx)

		deleteTransitionRequestReq.Name = args[0]
		deleteTransitionRequestReq.Version = args[1]
		deleteTransitionRequestReq.Stage = args[2]
		deleteTransitionRequestReq.Creator = args[3]

		response, err := w.ModelRegistry.DeleteTransitionRequest(ctx, deleteTransitionRequestReq)
		if err != nil {
			return err
		}
		return cmdio.Render(ctx, response)
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
	*ml.DeleteWebhookRequest,
)

func newDeleteWebhook() *cobra.Command {
	cmd := &cobra.Command{}

	var deleteWebhookReq ml.DeleteWebhookRequest

	cmd.Use = "delete-webhook ID"
	cmd.Short = `Delete a webhook.`
	cmd.Long = `Delete a webhook.
  
  **NOTE:** This endpoint is in Public Preview. Deletes a registry webhook.

  Arguments:
    ID: Webhook ID required to delete a registry webhook.`

	cmd.Annotations = make(map[string]string)

	cmd.Args = func(cmd *cobra.Command, args []string) error {
		check := root.ExactArgs(1)
		return check(cmd, args)
	}

	cmd.PreRunE = root.MustWorkspaceClient
	cmd.RunE = func(cmd *cobra.Command, args []string) (err error) {
		ctx := cmd.Context()
		w := cmdctx.WorkspaceClient(ctx)

		deleteWebhookReq.Id = args[0]

		err = w.ModelRegistry.DeleteWebhook(ctx, deleteWebhookReq)
		if err != nil {
			return err
		}
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
	*ml.GetLatestVersionsRequest,
)

func newGetLatestVersions() *cobra.Command {
	cmd := &cobra.Command{}

	var getLatestVersionsReq ml.GetLatestVersionsRequest
	var getLatestVersionsJson flags.JsonFlag

	cmd.Flags().Var(&getLatestVersionsJson, "json", `either inline JSON string or @path/to/file.json with request body`)

	// TODO: array: stages

	cmd.Use = "get-latest-versions NAME"
	cmd.Short = `Get the latest version.`
	cmd.Long = `Get the latest version.
  
  Gets the latest version of a registered model.

  Arguments:
    NAME: Registered model unique name identifier.`

	cmd.Annotations = make(map[string]string)

	cmd.Args = func(cmd *cobra.Command, args []string) error {
		if cmd.Flags().Changed("json") {
			err := root.ExactArgs(0)(cmd, args)
			if err != nil {
				return fmt.Errorf("when --json flag is specified, no positional arguments are required. Provide 'name' in your JSON input")
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
			diags := getLatestVersionsJson.Unmarshal(&getLatestVersionsReq)
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
			getLatestVersionsReq.Name = args[0]
		}

		response := w.ModelRegistry.GetLatestVersions(ctx, getLatestVersionsReq)
		return cmdio.RenderIterator(ctx, response)
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
	*ml.GetModelRequest,
)

func newGetModel() *cobra.Command {
	cmd := &cobra.Command{}

	var getModelReq ml.GetModelRequest

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

	cmd.Args = func(cmd *cobra.Command, args []string) error {
		check := root.ExactArgs(1)
		return check(cmd, args)
	}

	cmd.PreRunE = root.MustWorkspaceClient
	cmd.RunE = func(cmd *cobra.Command, args []string) (err error) {
		ctx := cmd.Context()
		w := cmdctx.WorkspaceClient(ctx)

		getModelReq.Name = args[0]

		response, err := w.ModelRegistry.GetModel(ctx, getModelReq)
		if err != nil {
			return err
		}
		return cmdio.Render(ctx, response)
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
	*ml.GetModelVersionRequest,
)

func newGetModelVersion() *cobra.Command {
	cmd := &cobra.Command{}

	var getModelVersionReq ml.GetModelVersionRequest

	cmd.Use = "get-model-version NAME VERSION"
	cmd.Short = `Get a model version.`
	cmd.Long = `Get a model version.

  Arguments:
    NAME: Name of the registered model
    VERSION: Model version number`

	cmd.Annotations = make(map[string]string)

	cmd.Args = func(cmd *cobra.Command, args []string) error {
		check := root.ExactArgs(2)
		return check(cmd, args)
	}

	cmd.PreRunE = root.MustWorkspaceClient
	cmd.RunE = func(cmd *cobra.Command, args []string) (err error) {
		ctx := cmd.Context()
		w := cmdctx.WorkspaceClient(ctx)

		getModelVersionReq.Name = args[0]
		getModelVersionReq.Version = args[1]

		response, err := w.ModelRegistry.GetModelVersion(ctx, getModelVersionReq)
		if err != nil {
			return err
		}
		return cmdio.Render(ctx, response)
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
	*ml.GetModelVersionDownloadUriRequest,
)

func newGetModelVersionDownloadUri() *cobra.Command {
	cmd := &cobra.Command{}

	var getModelVersionDownloadUriReq ml.GetModelVersionDownloadUriRequest

	cmd.Use = "get-model-version-download-uri NAME VERSION"
	cmd.Short = `Get a model version URI.`
	cmd.Long = `Get a model version URI.
  
  Gets a URI to download the model version.

  Arguments:
    NAME: Name of the registered model
    VERSION: Model version number`

	cmd.Annotations = make(map[string]string)

	cmd.Args = func(cmd *cobra.Command, args []string) error {
		check := root.ExactArgs(2)
		return check(cmd, args)
	}

	cmd.PreRunE = root.MustWorkspaceClient
	cmd.RunE = func(cmd *cobra.Command, args []string) (err error) {
		ctx := cmd.Context()
		w := cmdctx.WorkspaceClient(ctx)

		getModelVersionDownloadUriReq.Name = args[0]
		getModelVersionDownloadUriReq.Version = args[1]

		response, err := w.ModelRegistry.GetModelVersionDownloadUri(ctx, getModelVersionDownloadUriReq)
		if err != nil {
			return err
		}
		return cmdio.Render(ctx, response)
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
	*ml.GetRegisteredModelPermissionLevelsRequest,
)

func newGetPermissionLevels() *cobra.Command {
	cmd := &cobra.Command{}

	var getPermissionLevelsReq ml.GetRegisteredModelPermissionLevelsRequest

	cmd.Use = "get-permission-levels REGISTERED_MODEL_ID"
	cmd.Short = `Get registered model permission levels.`
	cmd.Long = `Get registered model permission levels.
  
  Gets the permission levels that a user can have on an object.

  Arguments:
    REGISTERED_MODEL_ID: The registered model for which to get or manage permissions.`

	cmd.Annotations = make(map[string]string)

	cmd.Args = func(cmd *cobra.Command, args []string) error {
		check := root.ExactArgs(1)
		return check(cmd, args)
	}

	cmd.PreRunE = root.MustWorkspaceClient
	cmd.RunE = func(cmd *cobra.Command, args []string) (err error) {
		ctx := cmd.Context()
		w := cmdctx.WorkspaceClient(ctx)

		getPermissionLevelsReq.RegisteredModelId = args[0]

		response, err := w.ModelRegistry.GetPermissionLevels(ctx, getPermissionLevelsReq)
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
	*ml.GetRegisteredModelPermissionsRequest,
)

func newGetPermissions() *cobra.Command {
	cmd := &cobra.Command{}

	var getPermissionsReq ml.GetRegisteredModelPermissionsRequest

	cmd.Use = "get-permissions REGISTERED_MODEL_ID"
	cmd.Short = `Get registered model permissions.`
	cmd.Long = `Get registered model permissions.
  
  Gets the permissions of a registered model. Registered models can inherit
  permissions from their root object.

  Arguments:
    REGISTERED_MODEL_ID: The registered model for which to get or manage permissions.`

	cmd.Annotations = make(map[string]string)

	cmd.Args = func(cmd *cobra.Command, args []string) error {
		check := root.ExactArgs(1)
		return check(cmd, args)
	}

	cmd.PreRunE = root.MustWorkspaceClient
	cmd.RunE = func(cmd *cobra.Command, args []string) (err error) {
		ctx := cmd.Context()
		w := cmdctx.WorkspaceClient(ctx)

		getPermissionsReq.RegisteredModelId = args[0]

		response, err := w.ModelRegistry.GetPermissions(ctx, getPermissionsReq)
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

// start list-models command

// Slice with functions to override default command behavior.
// Functions can be added from the `init()` function in manually curated files in this directory.
var listModelsOverrides []func(
	*cobra.Command,
	*ml.ListModelsRequest,
)

func newListModels() *cobra.Command {
	cmd := &cobra.Command{}

	var listModelsReq ml.ListModelsRequest

	cmd.Flags().Int64Var(&listModelsReq.MaxResults, "max-results", listModelsReq.MaxResults, `Maximum number of registered models desired.`)
	cmd.Flags().StringVar(&listModelsReq.PageToken, "page-token", listModelsReq.PageToken, `Pagination token to go to the next page based on a previous query.`)

	cmd.Use = "list-models"
	cmd.Short = `List models.`
	cmd.Long = `List models.
  
  Lists all available registered models, up to the limit specified in
  __max_results__.`

	cmd.Annotations = make(map[string]string)

	cmd.Args = func(cmd *cobra.Command, args []string) error {
		check := root.ExactArgs(0)
		return check(cmd, args)
	}

	cmd.PreRunE = root.MustWorkspaceClient
	cmd.RunE = func(cmd *cobra.Command, args []string) (err error) {
		ctx := cmd.Context()
		w := cmdctx.WorkspaceClient(ctx)

		response := w.ModelRegistry.ListModels(ctx, listModelsReq)
		return cmdio.RenderIterator(ctx, response)
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
	*ml.ListTransitionRequestsRequest,
)

func newListTransitionRequests() *cobra.Command {
	cmd := &cobra.Command{}

	var listTransitionRequestsReq ml.ListTransitionRequestsRequest

	cmd.Use = "list-transition-requests NAME VERSION"
	cmd.Short = `List transition requests.`
	cmd.Long = `List transition requests.
  
  Gets a list of all open stage transition requests for the model version.

  Arguments:
    NAME: Name of the registered model.
    VERSION: Version of the model.`

	cmd.Annotations = make(map[string]string)

	cmd.Args = func(cmd *cobra.Command, args []string) error {
		check := root.ExactArgs(2)
		return check(cmd, args)
	}

	cmd.PreRunE = root.MustWorkspaceClient
	cmd.RunE = func(cmd *cobra.Command, args []string) (err error) {
		ctx := cmd.Context()
		w := cmdctx.WorkspaceClient(ctx)

		listTransitionRequestsReq.Name = args[0]
		listTransitionRequestsReq.Version = args[1]

		response := w.ModelRegistry.ListTransitionRequests(ctx, listTransitionRequestsReq)
		return cmdio.RenderIterator(ctx, response)
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
	*ml.ListWebhooksRequest,
)

func newListWebhooks() *cobra.Command {
	cmd := &cobra.Command{}

	var listWebhooksReq ml.ListWebhooksRequest

	// TODO: array: events
	cmd.Flags().Int64Var(&listWebhooksReq.MaxResults, "max-results", listWebhooksReq.MaxResults, ``)
	cmd.Flags().StringVar(&listWebhooksReq.ModelName, "model-name", listWebhooksReq.ModelName, `Registered model name If not specified, all webhooks associated with the specified events are listed, regardless of their associated model.`)
	cmd.Flags().StringVar(&listWebhooksReq.PageToken, "page-token", listWebhooksReq.PageToken, `Token indicating the page of artifact results to fetch.`)

	cmd.Use = "list-webhooks"
	cmd.Short = `List registry webhooks.`
	cmd.Long = `List registry webhooks.
  
  **NOTE:** This endpoint is in Public Preview. Lists all registry webhooks.`

	cmd.Annotations = make(map[string]string)

	cmd.Args = func(cmd *cobra.Command, args []string) error {
		check := root.ExactArgs(0)
		return check(cmd, args)
	}

	cmd.PreRunE = root.MustWorkspaceClient
	cmd.RunE = func(cmd *cobra.Command, args []string) (err error) {
		ctx := cmd.Context()
		w := cmdctx.WorkspaceClient(ctx)

		response := w.ModelRegistry.ListWebhooks(ctx, listWebhooksReq)
		return cmdio.RenderIterator(ctx, response)
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
	*ml.RejectTransitionRequest,
)

func newRejectTransitionRequest() *cobra.Command {
	cmd := &cobra.Command{}

	var rejectTransitionRequestReq ml.RejectTransitionRequest
	var rejectTransitionRequestJson flags.JsonFlag

	cmd.Flags().Var(&rejectTransitionRequestJson, "json", `either inline JSON string or @path/to/file.json with request body`)

	cmd.Flags().StringVar(&rejectTransitionRequestReq.Comment, "comment", rejectTransitionRequestReq.Comment, `User-provided comment on the action.`)

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

	cmd.Args = func(cmd *cobra.Command, args []string) error {
		if cmd.Flags().Changed("json") {
			err := root.ExactArgs(0)(cmd, args)
			if err != nil {
				return fmt.Errorf("when --json flag is specified, no positional arguments are required. Provide 'name', 'version', 'stage' in your JSON input")
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
			diags := rejectTransitionRequestJson.Unmarshal(&rejectTransitionRequestReq)
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
			rejectTransitionRequestReq.Name = args[0]
		}
		if !cmd.Flags().Changed("json") {
			rejectTransitionRequestReq.Version = args[1]
		}
		if !cmd.Flags().Changed("json") {
			rejectTransitionRequestReq.Stage = args[2]
		}

		response, err := w.ModelRegistry.RejectTransitionRequest(ctx, rejectTransitionRequestReq)
		if err != nil {
			return err
		}
		return cmdio.Render(ctx, response)
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
	*ml.RenameModelRequest,
)

func newRenameModel() *cobra.Command {
	cmd := &cobra.Command{}

	var renameModelReq ml.RenameModelRequest
	var renameModelJson flags.JsonFlag

	cmd.Flags().Var(&renameModelJson, "json", `either inline JSON string or @path/to/file.json with request body`)

	cmd.Flags().StringVar(&renameModelReq.NewName, "new-name", renameModelReq.NewName, `If provided, updates the name for this registered_model.`)

	cmd.Use = "rename-model NAME"
	cmd.Short = `Rename a model.`
	cmd.Long = `Rename a model.
  
  Renames a registered model.

  Arguments:
    NAME: Registered model unique name identifier.`

	cmd.Annotations = make(map[string]string)

	cmd.Args = func(cmd *cobra.Command, args []string) error {
		if cmd.Flags().Changed("json") {
			err := root.ExactArgs(0)(cmd, args)
			if err != nil {
				return fmt.Errorf("when --json flag is specified, no positional arguments are required. Provide 'name' in your JSON input")
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
			diags := renameModelJson.Unmarshal(&renameModelReq)
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
			renameModelReq.Name = args[0]
		}

		response, err := w.ModelRegistry.RenameModel(ctx, renameModelReq)
		if err != nil {
			return err
		}
		return cmdio.Render(ctx, response)
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
	*ml.SearchModelVersionsRequest,
)

func newSearchModelVersions() *cobra.Command {
	cmd := &cobra.Command{}

	var searchModelVersionsReq ml.SearchModelVersionsRequest

	cmd.Flags().StringVar(&searchModelVersionsReq.Filter, "filter", searchModelVersionsReq.Filter, `String filter condition, like "name='my-model-name'".`)
	cmd.Flags().Int64Var(&searchModelVersionsReq.MaxResults, "max-results", searchModelVersionsReq.MaxResults, `Maximum number of models desired.`)
	// TODO: array: order_by
	cmd.Flags().StringVar(&searchModelVersionsReq.PageToken, "page-token", searchModelVersionsReq.PageToken, `Pagination token to go to next page based on previous search query.`)

	cmd.Use = "search-model-versions"
	cmd.Short = `Search model versions.`
	cmd.Long = `Search model versions.
  
  Searches for specific model versions based on the supplied __filter__.`

	cmd.Annotations = make(map[string]string)

	cmd.Args = func(cmd *cobra.Command, args []string) error {
		check := root.ExactArgs(0)
		return check(cmd, args)
	}

	cmd.PreRunE = root.MustWorkspaceClient
	cmd.RunE = func(cmd *cobra.Command, args []string) (err error) {
		ctx := cmd.Context()
		w := cmdctx.WorkspaceClient(ctx)

		response := w.ModelRegistry.SearchModelVersions(ctx, searchModelVersionsReq)
		return cmdio.RenderIterator(ctx, response)
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
	*ml.SearchModelsRequest,
)

func newSearchModels() *cobra.Command {
	cmd := &cobra.Command{}

	var searchModelsReq ml.SearchModelsRequest

	cmd.Flags().StringVar(&searchModelsReq.Filter, "filter", searchModelsReq.Filter, `String filter condition, like "name LIKE 'my-model-name'".`)
	cmd.Flags().Int64Var(&searchModelsReq.MaxResults, "max-results", searchModelsReq.MaxResults, `Maximum number of models desired.`)
	// TODO: array: order_by
	cmd.Flags().StringVar(&searchModelsReq.PageToken, "page-token", searchModelsReq.PageToken, `Pagination token to go to the next page based on a previous search query.`)

	cmd.Use = "search-models"
	cmd.Short = `Search models.`
	cmd.Long = `Search models.
  
  Search for registered models based on the specified __filter__.`

	cmd.Annotations = make(map[string]string)

	cmd.Args = func(cmd *cobra.Command, args []string) error {
		check := root.ExactArgs(0)
		return check(cmd, args)
	}

	cmd.PreRunE = root.MustWorkspaceClient
	cmd.RunE = func(cmd *cobra.Command, args []string) (err error) {
		ctx := cmd.Context()
		w := cmdctx.WorkspaceClient(ctx)

		response := w.ModelRegistry.SearchModels(ctx, searchModelsReq)
		return cmdio.RenderIterator(ctx, response)
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
	*ml.SetModelTagRequest,
)

func newSetModelTag() *cobra.Command {
	cmd := &cobra.Command{}

	var setModelTagReq ml.SetModelTagRequest
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

	cmd.Args = func(cmd *cobra.Command, args []string) error {
		if cmd.Flags().Changed("json") {
			err := root.ExactArgs(0)(cmd, args)
			if err != nil {
				return fmt.Errorf("when --json flag is specified, no positional arguments are required. Provide 'name', 'key', 'value' in your JSON input")
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
			diags := setModelTagJson.Unmarshal(&setModelTagReq)
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
			setModelTagReq.Name = args[0]
		}
		if !cmd.Flags().Changed("json") {
			setModelTagReq.Key = args[1]
		}
		if !cmd.Flags().Changed("json") {
			setModelTagReq.Value = args[2]
		}

		err = w.ModelRegistry.SetModelTag(ctx, setModelTagReq)
		if err != nil {
			return err
		}
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
	*ml.SetModelVersionTagRequest,
)

func newSetModelVersionTag() *cobra.Command {
	cmd := &cobra.Command{}

	var setModelVersionTagReq ml.SetModelVersionTagRequest
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

	cmd.Args = func(cmd *cobra.Command, args []string) error {
		if cmd.Flags().Changed("json") {
			err := root.ExactArgs(0)(cmd, args)
			if err != nil {
				return fmt.Errorf("when --json flag is specified, no positional arguments are required. Provide 'name', 'version', 'key', 'value' in your JSON input")
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
			diags := setModelVersionTagJson.Unmarshal(&setModelVersionTagReq)
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
			setModelVersionTagReq.Name = args[0]
		}
		if !cmd.Flags().Changed("json") {
			setModelVersionTagReq.Version = args[1]
		}
		if !cmd.Flags().Changed("json") {
			setModelVersionTagReq.Key = args[2]
		}
		if !cmd.Flags().Changed("json") {
			setModelVersionTagReq.Value = args[3]
		}

		err = w.ModelRegistry.SetModelVersionTag(ctx, setModelVersionTagReq)
		if err != nil {
			return err
		}
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
	*ml.RegisteredModelPermissionsRequest,
)

func newSetPermissions() *cobra.Command {
	cmd := &cobra.Command{}

	var setPermissionsReq ml.RegisteredModelPermissionsRequest
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
				err := cmdio.RenderDiagnosticsToErrorOut(ctx, diags)
				if err != nil {
					return err
				}
			}
		}
		setPermissionsReq.RegisteredModelId = args[0]

		response, err := w.ModelRegistry.SetPermissions(ctx, setPermissionsReq)
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

// start test-registry-webhook command

// Slice with functions to override default command behavior.
// Functions can be added from the `init()` function in manually curated files in this directory.
var testRegistryWebhookOverrides []func(
	*cobra.Command,
	*ml.TestRegistryWebhookRequest,
)

func newTestRegistryWebhook() *cobra.Command {
	cmd := &cobra.Command{}

	var testRegistryWebhookReq ml.TestRegistryWebhookRequest
	var testRegistryWebhookJson flags.JsonFlag

	cmd.Flags().Var(&testRegistryWebhookJson, "json", `either inline JSON string or @path/to/file.json with request body`)

	cmd.Flags().Var(&testRegistryWebhookReq.Event, "event", `If event is specified, the test trigger uses the specified event. Supported values: [
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
]`)

	cmd.Use = "test-registry-webhook ID"
	cmd.Short = `Test a webhook.`
	cmd.Long = `Test a webhook.
  
  **NOTE:** This endpoint is in Public Preview. Tests a registry webhook.

  Arguments:
    ID: Webhook ID`

	cmd.Annotations = make(map[string]string)

	cmd.Args = func(cmd *cobra.Command, args []string) error {
		if cmd.Flags().Changed("json") {
			err := root.ExactArgs(0)(cmd, args)
			if err != nil {
				return fmt.Errorf("when --json flag is specified, no positional arguments are required. Provide 'id' in your JSON input")
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
			diags := testRegistryWebhookJson.Unmarshal(&testRegistryWebhookReq)
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
			testRegistryWebhookReq.Id = args[0]
		}

		response, err := w.ModelRegistry.TestRegistryWebhook(ctx, testRegistryWebhookReq)
		if err != nil {
			return err
		}
		return cmdio.Render(ctx, response)
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
	*ml.TransitionModelVersionStageDatabricks,
)

func newTransitionStage() *cobra.Command {
	cmd := &cobra.Command{}

	var transitionStageReq ml.TransitionModelVersionStageDatabricks
	var transitionStageJson flags.JsonFlag

	cmd.Flags().Var(&transitionStageJson, "json", `either inline JSON string or @path/to/file.json with request body`)

	cmd.Flags().StringVar(&transitionStageReq.Comment, "comment", transitionStageReq.Comment, `User-provided comment on the action.`)

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

	cmd.Args = func(cmd *cobra.Command, args []string) error {
		if cmd.Flags().Changed("json") {
			err := root.ExactArgs(0)(cmd, args)
			if err != nil {
				return fmt.Errorf("when --json flag is specified, no positional arguments are required. Provide 'name', 'version', 'stage', 'archive_existing_versions' in your JSON input")
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
			diags := transitionStageJson.Unmarshal(&transitionStageReq)
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
			transitionStageReq.Name = args[0]
		}
		if !cmd.Flags().Changed("json") {
			transitionStageReq.Version = args[1]
		}
		if !cmd.Flags().Changed("json") {
			transitionStageReq.Stage = args[2]
		}
		if !cmd.Flags().Changed("json") {
			_, err = fmt.Sscan(args[3], &transitionStageReq.ArchiveExistingVersions)
			if err != nil {
				return fmt.Errorf("invalid ARCHIVE_EXISTING_VERSIONS: %s", args[3])
			}
		}

		response, err := w.ModelRegistry.TransitionStage(ctx, transitionStageReq)
		if err != nil {
			return err
		}
		return cmdio.Render(ctx, response)
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
	*ml.UpdateComment,
)

func newUpdateComment() *cobra.Command {
	cmd := &cobra.Command{}

	var updateCommentReq ml.UpdateComment
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

	cmd.Args = func(cmd *cobra.Command, args []string) error {
		if cmd.Flags().Changed("json") {
			err := root.ExactArgs(0)(cmd, args)
			if err != nil {
				return fmt.Errorf("when --json flag is specified, no positional arguments are required. Provide 'id', 'comment' in your JSON input")
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
			diags := updateCommentJson.Unmarshal(&updateCommentReq)
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
			updateCommentReq.Id = args[0]
		}
		if !cmd.Flags().Changed("json") {
			updateCommentReq.Comment = args[1]
		}

		response, err := w.ModelRegistry.UpdateComment(ctx, updateCommentReq)
		if err != nil {
			return err
		}
		return cmdio.Render(ctx, response)
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
	*ml.UpdateModelRequest,
)

func newUpdateModel() *cobra.Command {
	cmd := &cobra.Command{}

	var updateModelReq ml.UpdateModelRequest
	var updateModelJson flags.JsonFlag

	cmd.Flags().Var(&updateModelJson, "json", `either inline JSON string or @path/to/file.json with request body`)

	cmd.Flags().StringVar(&updateModelReq.Description, "description", updateModelReq.Description, `If provided, updates the description for this registered_model.`)

	cmd.Use = "update-model NAME"
	cmd.Short = `Update model.`
	cmd.Long = `Update model.
  
  Updates a registered model.

  Arguments:
    NAME: Registered model unique name identifier.`

	cmd.Annotations = make(map[string]string)

	cmd.Args = func(cmd *cobra.Command, args []string) error {
		if cmd.Flags().Changed("json") {
			err := root.ExactArgs(0)(cmd, args)
			if err != nil {
				return fmt.Errorf("when --json flag is specified, no positional arguments are required. Provide 'name' in your JSON input")
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
			diags := updateModelJson.Unmarshal(&updateModelReq)
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
			updateModelReq.Name = args[0]
		}

		response, err := w.ModelRegistry.UpdateModel(ctx, updateModelReq)
		if err != nil {
			return err
		}
		return cmdio.Render(ctx, response)
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
	*ml.UpdateModelVersionRequest,
)

func newUpdateModelVersion() *cobra.Command {
	cmd := &cobra.Command{}

	var updateModelVersionReq ml.UpdateModelVersionRequest
	var updateModelVersionJson flags.JsonFlag

	cmd.Flags().Var(&updateModelVersionJson, "json", `either inline JSON string or @path/to/file.json with request body`)

	cmd.Flags().StringVar(&updateModelVersionReq.Description, "description", updateModelVersionReq.Description, `If provided, updates the description for this registered_model.`)

	cmd.Use = "update-model-version NAME VERSION"
	cmd.Short = `Update model version.`
	cmd.Long = `Update model version.
  
  Updates the model version.

  Arguments:
    NAME: Name of the registered model
    VERSION: Model version number`

	cmd.Annotations = make(map[string]string)

	cmd.Args = func(cmd *cobra.Command, args []string) error {
		if cmd.Flags().Changed("json") {
			err := root.ExactArgs(0)(cmd, args)
			if err != nil {
				return fmt.Errorf("when --json flag is specified, no positional arguments are required. Provide 'name', 'version' in your JSON input")
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
			diags := updateModelVersionJson.Unmarshal(&updateModelVersionReq)
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
			updateModelVersionReq.Name = args[0]
		}
		if !cmd.Flags().Changed("json") {
			updateModelVersionReq.Version = args[1]
		}

		response, err := w.ModelRegistry.UpdateModelVersion(ctx, updateModelVersionReq)
		if err != nil {
			return err
		}
		return cmdio.Render(ctx, response)
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
	*ml.RegisteredModelPermissionsRequest,
)

func newUpdatePermissions() *cobra.Command {
	cmd := &cobra.Command{}

	var updatePermissionsReq ml.RegisteredModelPermissionsRequest
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
				err := cmdio.RenderDiagnosticsToErrorOut(ctx, diags)
				if err != nil {
					return err
				}
			}
		}
		updatePermissionsReq.RegisteredModelId = args[0]

		response, err := w.ModelRegistry.UpdatePermissions(ctx, updatePermissionsReq)
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

// start update-webhook command

// Slice with functions to override default command behavior.
// Functions can be added from the `init()` function in manually curated files in this directory.
var updateWebhookOverrides []func(
	*cobra.Command,
	*ml.UpdateRegistryWebhook,
)

func newUpdateWebhook() *cobra.Command {
	cmd := &cobra.Command{}

	var updateWebhookReq ml.UpdateRegistryWebhook
	var updateWebhookJson flags.JsonFlag

	cmd.Flags().Var(&updateWebhookJson, "json", `either inline JSON string or @path/to/file.json with request body`)

	cmd.Flags().StringVar(&updateWebhookReq.Description, "description", updateWebhookReq.Description, `User-specified description for the webhook.`)
	// TODO: array: events
	// TODO: complex arg: http_url_spec
	// TODO: complex arg: job_spec
	cmd.Flags().Var(&updateWebhookReq.Status, "status", `Supported values: [ACTIVE, DISABLED, TEST_MODE]`)

	cmd.Use = "update-webhook ID"
	cmd.Short = `Update a webhook.`
	cmd.Long = `Update a webhook.
  
  **NOTE:** This endpoint is in Public Preview. Updates a registry webhook.

  Arguments:
    ID: Webhook ID`

	cmd.Annotations = make(map[string]string)

	cmd.Args = func(cmd *cobra.Command, args []string) error {
		if cmd.Flags().Changed("json") {
			err := root.ExactArgs(0)(cmd, args)
			if err != nil {
				return fmt.Errorf("when --json flag is specified, no positional arguments are required. Provide 'id' in your JSON input")
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
			diags := updateWebhookJson.Unmarshal(&updateWebhookReq)
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
			updateWebhookReq.Id = args[0]
		}

		response, err := w.ModelRegistry.UpdateWebhook(ctx, updateWebhookReq)
		if err != nil {
			return err
		}
		return cmdio.Render(ctx, response)
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
