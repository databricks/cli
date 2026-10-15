// Code generated from OpenAPI specs by Databricks SDK Generator. DO NOT EDIT.

package apps_settings

import (
	"errors"
	"fmt"

	"github.com/databricks/cli/cmd/root"
	"github.com/databricks/cli/libs/cmdctx"
	"github.com/databricks/cli/libs/cmdgen"
	"github.com/databricks/cli/libs/cmdio"
	"github.com/databricks/cli/libs/flags"
	"github.com/spf13/cobra"

	apps "github.com/databricks/cli/libs/sdktypes/apps/v1"
	sdkapps "github.com/databricks/sdk-go/apps/v1"
)

// Slice with functions to override default command behavior.
// Functions can be added from the `init()` function in manually curated files in this directory.
var cmdOverrides []func(*cobra.Command)

func New() *cobra.Command {
	cmd := &cobra.Command{
		Use:   "apps-settings",
		Short: `Apps Settings manage the settings for the Apps service on a customer's Databricks instance.`,
		Long: `Apps Settings manage the settings for the Apps service on a customer's
  Databricks instance.`,
		GroupID: "apps",

		// This service is being previewed; hide from help output.
		Hidden: true,
		RunE:   root.ReportUnknownSubcommand,
	}

	cmd.Annotations = make(map[string]string)
	cmd.Annotations["launch_stage"] = "PRIVATE_PREVIEW"
	cmd.Annotations["launch_stage_display"] = "Private Preview"

	// Add methods
	cmd.AddCommand(newCreateCustomTemplate())
	cmd.AddCommand(newDeleteCustomTemplate())
	cmd.AddCommand(newGetCustomTemplate())
	cmd.AddCommand(newListCustomTemplates())
	cmd.AddCommand(newUpdateCustomTemplate())

	// Apply optional overrides to this command.
	for _, fn := range cmdOverrides {
		fn(cmd)
	}

	return cmd
}

// start create-custom-template command

// Slice with functions to override default command behavior.
// Functions can be added from the `init()` function in manually curated files in this directory.
var createCustomTemplateOverrides []func(
	*cobra.Command,
	*apps.CreateCustomTemplateRequest,
)

func newCreateCustomTemplate() *cobra.Command {
	cmd := &cobra.Command{}
	var createCustomTemplateReq apps.CreateCustomTemplateRequest
	createCustomTemplateReq.Template = &apps.CustomTemplate{}

	var createCustomTemplateJson flags.JsonFlag

	cmd.Flags().Var(&createCustomTemplateJson, "json", `either inline JSON string or @path/to/file.json with request body`)

	flags.StringPtrVar(cmd.Flags(), &createCustomTemplateReq.Template.Description, "description", `The description of the template.`)

	cmd.Use = "create-custom-template NAME GIT_REPO PATH MANIFEST GIT_PROVIDER"
	cmd.Short = `Create a template.`
	cmd.Long = `Create a template.

  Creates a custom template.

  Arguments:
    NAME: The name of the template. It must contain only alphanumeric characters,
      hyphens, underscores, and whitespaces. It must be unique within the
      workspace.
    GIT_REPO: The Git repository URL that the template resides in.
    PATH: The path to the template within the Git repository.
    MANIFEST: The manifest of the template. It defines fields and default values when
      installing the template.
    GIT_PROVIDER: The Git provider of the template.`

	cmd.Annotations = make(map[string]string)
	cmd.Annotations["launch_stage"] = "PRIVATE_PREVIEW"
	cmd.Annotations["launch_stage_display"] = "Private Preview"

	cmd.Args = func(cmd *cobra.Command, args []string) error {
		if cmd.Flags().Changed("json") {
			err := root.ExactArgs(0)(cmd, args)
			if err != nil {
				return errors.New("when --json flag is specified, no positional arguments are allowed. Provide 'name', 'git_repo', 'path', 'manifest', 'git_provider' in your JSON input")
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
			diags := createCustomTemplateJson.Unmarshal(&createCustomTemplateReq.Template)
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
			createCustomTemplateReq.Template.Name = new(args[0])
		}
		if !cmd.Flags().Changed("json") {
			createCustomTemplateReq.Template.GitRepo = new(args[1])
		}
		if !cmd.Flags().Changed("json") {
			createCustomTemplateReq.Template.Path = new(args[2])
		}
		if !cmd.Flags().Changed("json") {
			_ = args[3] // TODO: complex arg: manifest
		}
		if !cmd.Flags().Changed("json") {
			createCustomTemplateReq.Template.GitProvider = new(args[4])
		}

		sdkReq, err := apps.CreateCustomTemplateRequestToSDK(&createCustomTemplateReq)
		if err != nil {
			return err
		}
		response, err := w.Apps.CreateCustomTemplate(ctx, *sdkReq)
		if err != nil {
			return err
		}

		out, err := apps.CustomTemplateFromSDK(response)
		if err != nil {
			return err
		}
		return cmdio.Render(ctx, out)
	}

	// Disable completions since they are not applicable.
	// Can be overridden by manual implementation in `override.go`.
	cmd.ValidArgsFunction = cobra.NoFileCompletions

	// Apply optional overrides to this command.
	for _, fn := range createCustomTemplateOverrides {
		fn(cmd, &createCustomTemplateReq)
	}

	return cmd
}

// start delete-custom-template command

// Slice with functions to override default command behavior.
// Functions can be added from the `init()` function in manually curated files in this directory.
var deleteCustomTemplateOverrides []func(
	*cobra.Command,
	*sdkapps.DeleteCustomTemplateRequest,
)

func newDeleteCustomTemplate() *cobra.Command {
	cmd := &cobra.Command{}
	var deleteCustomTemplateReq sdkapps.DeleteCustomTemplateRequest

	cmd.Use = "delete-custom-template NAME"
	cmd.Short = `Delete a template.`
	cmd.Long = `Delete a template.

  Deletes the custom template with the specified name.

  Arguments:
    NAME: The name of the custom template.`

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

		deleteCustomTemplateReq.Name = new(args[0])

		sdkReq := &deleteCustomTemplateReq
		response, err := w.Apps.DeleteCustomTemplate(ctx, *sdkReq)
		if err != nil {
			return err
		}

		out, err := apps.CustomTemplateFromSDK(response)
		if err != nil {
			return err
		}
		return cmdio.Render(ctx, out)
	}

	// Disable completions since they are not applicable.
	// Can be overridden by manual implementation in `override.go`.
	cmd.ValidArgsFunction = cobra.NoFileCompletions

	// Apply optional overrides to this command.
	for _, fn := range deleteCustomTemplateOverrides {
		fn(cmd, &deleteCustomTemplateReq)
	}

	return cmd
}

// start get-custom-template command

// Slice with functions to override default command behavior.
// Functions can be added from the `init()` function in manually curated files in this directory.
var getCustomTemplateOverrides []func(
	*cobra.Command,
	*sdkapps.GetCustomTemplateRequest,
)

func newGetCustomTemplate() *cobra.Command {
	cmd := &cobra.Command{}
	var getCustomTemplateReq sdkapps.GetCustomTemplateRequest

	cmd.Use = "get-custom-template NAME"
	cmd.Short = `Get a template.`
	cmd.Long = `Get a template.

  Gets the custom template with the specified name.

  Arguments:
    NAME: The name of the custom template.`

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

		getCustomTemplateReq.Name = new(args[0])

		sdkReq := &getCustomTemplateReq
		response, err := w.Apps.GetCustomTemplate(ctx, *sdkReq)
		if err != nil {
			return err
		}

		out, err := apps.CustomTemplateFromSDK(response)
		if err != nil {
			return err
		}
		return cmdio.Render(ctx, out)
	}

	// Disable completions since they are not applicable.
	// Can be overridden by manual implementation in `override.go`.
	cmd.ValidArgsFunction = cobra.NoFileCompletions

	// Apply optional overrides to this command.
	for _, fn := range getCustomTemplateOverrides {
		fn(cmd, &getCustomTemplateReq)
	}

	return cmd
}

// start list-custom-templates command

// Slice with functions to override default command behavior.
// Functions can be added from the `init()` function in manually curated files in this directory.
var listCustomTemplatesOverrides []func(
	*cobra.Command,
	*apps.ListCustomTemplatesRequest,
)

func newListCustomTemplates() *cobra.Command {
	cmd := &cobra.Command{}
	var listCustomTemplatesReq apps.ListCustomTemplatesRequest

	// Registered for all paginated methods. Validated at call time in the
	// method-call template. Paginated list methods never have Wait or LRO
	// branches, so the method-call path is always reached.
	var listCustomTemplatesLimit int

	flags.IntPtrVar(cmd.Flags(), &listCustomTemplatesReq.PageSize, "page-size", `Upper bound for items returned.`)

	// Limit flag for total result capping.
	cmd.Flags().IntVar(&listCustomTemplatesLimit, "limit", 0, `Maximum number of results to return.`)

	// Hidden pagination flags (internal API parameters).
	flags.StringPtrVar(cmd.Flags(), &listCustomTemplatesReq.PageToken, "page-token", `Pagination token.`)
	cmd.Flags().Lookup("page-token").Hidden = true

	cmd.Use = "list-custom-templates"
	cmd.Short = `List templates.`
	cmd.Long = `List templates.

  Lists all custom templates in the workspace.`

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
		_ = w

		sdkReq, err := apps.ListCustomTemplatesRequestToSDK(&listCustomTemplatesReq)
		if err != nil {
			return err
		}
		if listCustomTemplatesLimit < 0 {
			return fmt.Errorf("--limit must be a non-negative integer, got %d", listCustomTemplatesLimit)
		}
		if listCustomTemplatesLimit > 0 {
			ctx = cmdio.WithLimit(ctx, listCustomTemplatesLimit)
		}
		response := w.Apps.ListCustomTemplatesIter(ctx, *sdkReq)
		return cmdio.RenderIterator(ctx, cmdgen.MapSeq(response, apps.CustomTemplateFromSDK))
	}

	// Disable completions since they are not applicable.
	// Can be overridden by manual implementation in `override.go`.
	cmd.ValidArgsFunction = cobra.NoFileCompletions

	// Apply optional overrides to this command.
	for _, fn := range listCustomTemplatesOverrides {
		fn(cmd, &listCustomTemplatesReq)
	}

	return cmd
}

// start update-custom-template command

// Slice with functions to override default command behavior.
// Functions can be added from the `init()` function in manually curated files in this directory.
var updateCustomTemplateOverrides []func(
	*cobra.Command,
	*apps.UpdateCustomTemplateRequest,
)

func newUpdateCustomTemplate() *cobra.Command {
	cmd := &cobra.Command{}
	var updateCustomTemplateReq apps.UpdateCustomTemplateRequest
	updateCustomTemplateReq.Template = &apps.CustomTemplate{}

	var updateCustomTemplateJson flags.JsonFlag

	cmd.Flags().Var(&updateCustomTemplateJson, "json", `either inline JSON string or @path/to/file.json with request body`)

	flags.StringPtrVar(cmd.Flags(), &updateCustomTemplateReq.Template.Description, "description", `The description of the template.`)

	cmd.Use = "update-custom-template NAME GIT_REPO PATH MANIFEST GIT_PROVIDER"
	cmd.Short = `Update a template.`
	cmd.Long = `Update a template.

  Updates the custom template with the specified name. Note that the template
  name cannot be updated.

  Arguments:
    NAME: The name of the template. It must contain only alphanumeric characters,
      hyphens, underscores, and whitespaces. It must be unique within the
      workspace.
    GIT_REPO: The Git repository URL that the template resides in.
    PATH: The path to the template within the Git repository.
    MANIFEST: The manifest of the template. It defines fields and default values when
      installing the template.
    GIT_PROVIDER: The Git provider of the template.`

	cmd.Annotations = make(map[string]string)
	cmd.Annotations["launch_stage"] = "PRIVATE_PREVIEW"
	cmd.Annotations["launch_stage_display"] = "Private Preview"

	cmd.Args = func(cmd *cobra.Command, args []string) error {
		if cmd.Flags().Changed("json") {
			err := root.ExactArgs(1)(cmd, args)
			if err != nil {
				return errors.New("when --json flag is specified, provide only NAME as positional arguments. Provide 'name', 'git_repo', 'path', 'manifest', 'git_provider' in your JSON input")
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
			diags := updateCustomTemplateJson.Unmarshal(&updateCustomTemplateReq.Template)
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
		updateCustomTemplateReq.Template.Name = new(args[0])
		if !cmd.Flags().Changed("json") {
			updateCustomTemplateReq.Template.GitRepo = new(args[1])
		}
		if !cmd.Flags().Changed("json") {
			updateCustomTemplateReq.Template.Path = new(args[2])
		}
		if !cmd.Flags().Changed("json") {
			_ = args[3] // TODO: complex arg: manifest
		}
		if !cmd.Flags().Changed("json") {
			updateCustomTemplateReq.Template.GitProvider = new(args[4])
		}

		sdkReq, err := apps.UpdateCustomTemplateRequestToSDK(&updateCustomTemplateReq)
		if err != nil {
			return err
		}
		response, err := w.Apps.UpdateCustomTemplate(ctx, *sdkReq)
		if err != nil {
			return err
		}

		out, err := apps.CustomTemplateFromSDK(response)
		if err != nil {
			return err
		}
		return cmdio.Render(ctx, out)
	}

	// Disable completions since they are not applicable.
	// Can be overridden by manual implementation in `override.go`.
	cmd.ValidArgsFunction = cobra.NoFileCompletions

	// Apply optional overrides to this command.
	for _, fn := range updateCustomTemplateOverrides {
		fn(cmd, &updateCustomTemplateReq)
	}

	return cmd
}

// end service AppsSettings
