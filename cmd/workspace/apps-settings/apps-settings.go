// Code generated from OpenAPI specs by Databricks SDK Generator. DO NOT EDIT.

package apps_settings

import (
	"fmt"

	"github.com/databricks/cli/cmd/root"
	"github.com/databricks/cli/libs/cmdctx"
	"github.com/databricks/cli/libs/cmdio"
	"github.com/databricks/cli/libs/flags"
	"github.com/databricks/databricks-sdk-go/service/apps"
	"github.com/spf13/cobra"
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
		Annotations: map[string]string{
			"package": "apps",
		},

		// This service is being previewed; hide from help output.
		Hidden: true,
		RunE:   root.ReportUnknownSubcommand,
	}

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
	createCustomTemplateReq.Template = apps.CustomTemplate{}
	var createCustomTemplateJson flags.JsonFlag

	cmd.Flags().Var(&createCustomTemplateJson, "json", `either inline JSON string or @path/to/file.json with request body`)

	cmd.Flags().StringVar(&createCustomTemplateReq.Template.Description, "description", createCustomTemplateReq.Template.Description, `The description of the template.`)

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

	cmd.Args = func(cmd *cobra.Command, args []string) error {
		if cmd.Flags().Changed("json") {
			err := root.ExactArgs(0)(cmd, args)
			if err != nil {
				return fmt.Errorf("when --json flag is specified, no positional arguments are required. Provide 'name', 'git_repo', 'path', 'manifest', 'git_provider' in your JSON input")
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
			diags := createCustomTemplateJson.Unmarshal(&createCustomTemplateReq.Template)
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
			createCustomTemplateReq.Template.Name = args[0]
		}
		if !cmd.Flags().Changed("json") {
			createCustomTemplateReq.Template.GitRepo = args[1]
		}
		if !cmd.Flags().Changed("json") {
			createCustomTemplateReq.Template.Path = args[2]
		}
		if !cmd.Flags().Changed("json") {
			_, err = fmt.Sscan(args[3], &createCustomTemplateReq.Template.Manifest)
			if err != nil {
				return fmt.Errorf("invalid MANIFEST: %s", args[3])
			}
		}
		if !cmd.Flags().Changed("json") {
			createCustomTemplateReq.Template.GitProvider = args[4]
		}

		response, err := w.AppsSettings.CreateCustomTemplate(ctx, createCustomTemplateReq)
		if err != nil {
			return err
		}
		return cmdio.Render(ctx, response)
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
	*apps.DeleteCustomTemplateRequest,
)

func newDeleteCustomTemplate() *cobra.Command {
	cmd := &cobra.Command{}

	var deleteCustomTemplateReq apps.DeleteCustomTemplateRequest

	cmd.Use = "delete-custom-template NAME"
	cmd.Short = `Delete a template.`
	cmd.Long = `Delete a template.
  
  Deletes the custom template with the specified name.

  Arguments:
    NAME: The name of the custom template.`

	cmd.Annotations = make(map[string]string)

	cmd.Args = func(cmd *cobra.Command, args []string) error {
		check := root.ExactArgs(1)
		return check(cmd, args)
	}

	cmd.PreRunE = root.MustWorkspaceClient
	cmd.RunE = func(cmd *cobra.Command, args []string) (err error) {
		ctx := cmd.Context()
		w := cmdctx.WorkspaceClient(ctx)

		deleteCustomTemplateReq.Name = args[0]

		response, err := w.AppsSettings.DeleteCustomTemplate(ctx, deleteCustomTemplateReq)
		if err != nil {
			return err
		}
		return cmdio.Render(ctx, response)
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
	*apps.GetCustomTemplateRequest,
)

func newGetCustomTemplate() *cobra.Command {
	cmd := &cobra.Command{}

	var getCustomTemplateReq apps.GetCustomTemplateRequest

	cmd.Use = "get-custom-template NAME"
	cmd.Short = `Get a template.`
	cmd.Long = `Get a template.
  
  Gets the custom template with the specified name.

  Arguments:
    NAME: The name of the custom template.`

	cmd.Annotations = make(map[string]string)

	cmd.Args = func(cmd *cobra.Command, args []string) error {
		check := root.ExactArgs(1)
		return check(cmd, args)
	}

	cmd.PreRunE = root.MustWorkspaceClient
	cmd.RunE = func(cmd *cobra.Command, args []string) (err error) {
		ctx := cmd.Context()
		w := cmdctx.WorkspaceClient(ctx)

		getCustomTemplateReq.Name = args[0]

		response, err := w.AppsSettings.GetCustomTemplate(ctx, getCustomTemplateReq)
		if err != nil {
			return err
		}
		return cmdio.Render(ctx, response)
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

	cmd.Flags().IntVar(&listCustomTemplatesReq.PageSize, "page-size", listCustomTemplatesReq.PageSize, `Upper bound for items returned.`)
	cmd.Flags().StringVar(&listCustomTemplatesReq.PageToken, "page-token", listCustomTemplatesReq.PageToken, `Pagination token to go to the next page of custom templates.`)

	cmd.Use = "list-custom-templates"
	cmd.Short = `List templates.`
	cmd.Long = `List templates.
  
  Lists all custom templates in the workspace.`

	cmd.Annotations = make(map[string]string)

	cmd.Args = func(cmd *cobra.Command, args []string) error {
		check := root.ExactArgs(0)
		return check(cmd, args)
	}

	cmd.PreRunE = root.MustWorkspaceClient
	cmd.RunE = func(cmd *cobra.Command, args []string) (err error) {
		ctx := cmd.Context()
		w := cmdctx.WorkspaceClient(ctx)

		response := w.AppsSettings.ListCustomTemplates(ctx, listCustomTemplatesReq)
		return cmdio.RenderIterator(ctx, response)
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
	updateCustomTemplateReq.Template = apps.CustomTemplate{}
	var updateCustomTemplateJson flags.JsonFlag

	cmd.Flags().Var(&updateCustomTemplateJson, "json", `either inline JSON string or @path/to/file.json with request body`)

	cmd.Flags().StringVar(&updateCustomTemplateReq.Template.Description, "description", updateCustomTemplateReq.Template.Description, `The description of the template.`)

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

	cmd.Args = func(cmd *cobra.Command, args []string) error {
		if cmd.Flags().Changed("json") {
			err := root.ExactArgs(1)(cmd, args)
			if err != nil {
				return fmt.Errorf("when --json flag is specified, provide only NAME as positional arguments. Provide 'name', 'git_repo', 'path', 'manifest', 'git_provider' in your JSON input")
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
			diags := updateCustomTemplateJson.Unmarshal(&updateCustomTemplateReq.Template)
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
		updateCustomTemplateReq.Name = args[0]
		if !cmd.Flags().Changed("json") {
			updateCustomTemplateReq.Template.GitRepo = args[1]
		}
		if !cmd.Flags().Changed("json") {
			updateCustomTemplateReq.Template.Path = args[2]
		}
		if !cmd.Flags().Changed("json") {
			_, err = fmt.Sscan(args[3], &updateCustomTemplateReq.Template.Manifest)
			if err != nil {
				return fmt.Errorf("invalid MANIFEST: %s", args[3])
			}
		}
		if !cmd.Flags().Changed("json") {
			updateCustomTemplateReq.Template.GitProvider = args[4]
		}

		response, err := w.AppsSettings.UpdateCustomTemplate(ctx, updateCustomTemplateReq)
		if err != nil {
			return err
		}
		return cmdio.Render(ctx, response)
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
