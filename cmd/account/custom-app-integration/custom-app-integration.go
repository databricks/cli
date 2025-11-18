// Code generated from OpenAPI specs by Databricks SDK Generator. DO NOT EDIT.

package custom_app_integration

import (
	"github.com/databricks/cli/cmd/root"
	"github.com/databricks/cli/libs/cmdctx"
	"github.com/databricks/cli/libs/cmdio"
	"github.com/databricks/cli/libs/flags"
	"github.com/databricks/databricks-sdk-go/service/oauth2"
	"github.com/spf13/cobra"
)

// Slice with functions to override default command behavior.
// Functions can be added from the `init()` function in manually curated files in this directory.
var cmdOverrides []func(*cobra.Command)

func New() *cobra.Command {
	cmd := &cobra.Command{
		Use:   "custom-app-integration",
		Short: `These APIs enable administrators to manage custom OAuth app integrations, which is required for adding/using Custom OAuth App Integration like Tableau Cloud for Databricks in AWS cloud.`,
		Long: `These APIs enable administrators to manage custom OAuth app integrations,
  which is required for adding/using Custom OAuth App Integration like Tableau
  Cloud for Databricks in AWS cloud.`,
		GroupID: "oauth2",
		Annotations: map[string]string{
			"package": "oauth2",
		},
		RunE: root.ReportUnknownSubcommand,
	}

	// Add methods
	cmd.AddCommand(newCreate())
	cmd.AddCommand(newDelete())
	cmd.AddCommand(newGet())
	cmd.AddCommand(newList())
	cmd.AddCommand(newUpdate())

	// Apply optional overrides to this command.
	for _, fn := range cmdOverrides {
		fn(cmd)
	}

	return cmd
}

// start create command

// Slice with functions to override default command behavior.
// Functions can be added from the `init()` function in manually curated files in this directory.
var createOverrides []func(
	*cobra.Command,
	*oauth2.CreateCustomAppIntegration,
)

func newCreate() *cobra.Command {
	cmd := &cobra.Command{}

	var createReq oauth2.CreateCustomAppIntegration
	var createJson flags.JsonFlag

	cmd.Flags().Var(&createJson, "json", `either inline JSON string or @path/to/file.json with request body`)

	cmd.Flags().BoolVar(&createReq.Confidential, "confidential", createReq.Confidential, `This field indicates whether an OAuth client secret is required to authenticate this client.`)
	cmd.Flags().StringVar(&createReq.Name, "name", createReq.Name, `Name of the custom OAuth app.`)
	// TODO: array: redirect_urls
	// TODO: array: scopes
	// TODO: complex arg: token_access_policy
	// TODO: array: user_authorized_scopes

	cmd.Use = "create"
	cmd.Short = `Create Custom OAuth App Integration.`
	cmd.Long = `Create Custom OAuth App Integration.
  
  Create Custom OAuth App Integration.
  
  You can retrieve the custom OAuth app integration via
  :method:CustomAppIntegration/get.`

	cmd.Annotations = make(map[string]string)

	cmd.Args = func(cmd *cobra.Command, args []string) error {
		check := root.ExactArgs(0)
		return check(cmd, args)
	}

	cmd.PreRunE = root.MustAccountClient
	cmd.RunE = func(cmd *cobra.Command, args []string) (err error) {
		ctx := cmd.Context()
		a := cmdctx.AccountClient(ctx)

		if cmd.Flags().Changed("json") {
			diags := createJson.Unmarshal(&createReq)
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

		response, err := a.CustomAppIntegration.Create(ctx, createReq)
		if err != nil {
			return err
		}
		return cmdio.Render(ctx, response)
	}

	// Disable completions since they are not applicable.
	// Can be overridden by manual implementation in `override.go`.
	cmd.ValidArgsFunction = cobra.NoFileCompletions

	// Apply optional overrides to this command.
	for _, fn := range createOverrides {
		fn(cmd, &createReq)
	}

	return cmd
}

// start delete command

// Slice with functions to override default command behavior.
// Functions can be added from the `init()` function in manually curated files in this directory.
var deleteOverrides []func(
	*cobra.Command,
	*oauth2.DeleteCustomAppIntegrationRequest,
)

func newDelete() *cobra.Command {
	cmd := &cobra.Command{}

	var deleteReq oauth2.DeleteCustomAppIntegrationRequest

	cmd.Use = "delete INTEGRATION_ID"
	cmd.Short = `Delete Custom OAuth App Integration.`
	cmd.Long = `Delete Custom OAuth App Integration.
  
  Delete an existing Custom OAuth App Integration. You can retrieve the custom
  OAuth app integration via :method:CustomAppIntegration/get.`

	cmd.Annotations = make(map[string]string)

	cmd.Args = func(cmd *cobra.Command, args []string) error {
		check := root.ExactArgs(1)
		return check(cmd, args)
	}

	cmd.PreRunE = root.MustAccountClient
	cmd.RunE = func(cmd *cobra.Command, args []string) (err error) {
		ctx := cmd.Context()
		a := cmdctx.AccountClient(ctx)

		deleteReq.IntegrationId = args[0]

		err = a.CustomAppIntegration.Delete(ctx, deleteReq)
		if err != nil {
			return err
		}
		return nil
	}

	// Disable completions since they are not applicable.
	// Can be overridden by manual implementation in `override.go`.
	cmd.ValidArgsFunction = cobra.NoFileCompletions

	// Apply optional overrides to this command.
	for _, fn := range deleteOverrides {
		fn(cmd, &deleteReq)
	}

	return cmd
}

// start get command

// Slice with functions to override default command behavior.
// Functions can be added from the `init()` function in manually curated files in this directory.
var getOverrides []func(
	*cobra.Command,
	*oauth2.GetCustomAppIntegrationRequest,
)

func newGet() *cobra.Command {
	cmd := &cobra.Command{}

	var getReq oauth2.GetCustomAppIntegrationRequest

	cmd.Use = "get INTEGRATION_ID"
	cmd.Short = `Get OAuth Custom App Integration.`
	cmd.Long = `Get OAuth Custom App Integration.
  
  Gets the Custom OAuth App Integration for the given integration id.

  Arguments:
    INTEGRATION_ID: The OAuth app integration ID.`

	cmd.Annotations = make(map[string]string)

	cmd.Args = func(cmd *cobra.Command, args []string) error {
		check := root.ExactArgs(1)
		return check(cmd, args)
	}

	cmd.PreRunE = root.MustAccountClient
	cmd.RunE = func(cmd *cobra.Command, args []string) (err error) {
		ctx := cmd.Context()
		a := cmdctx.AccountClient(ctx)

		getReq.IntegrationId = args[0]

		response, err := a.CustomAppIntegration.Get(ctx, getReq)
		if err != nil {
			return err
		}
		return cmdio.Render(ctx, response)
	}

	// Disable completions since they are not applicable.
	// Can be overridden by manual implementation in `override.go`.
	cmd.ValidArgsFunction = cobra.NoFileCompletions

	// Apply optional overrides to this command.
	for _, fn := range getOverrides {
		fn(cmd, &getReq)
	}

	return cmd
}

// start list command

// Slice with functions to override default command behavior.
// Functions can be added from the `init()` function in manually curated files in this directory.
var listOverrides []func(
	*cobra.Command,
	*oauth2.ListCustomAppIntegrationsRequest,
)

func newList() *cobra.Command {
	cmd := &cobra.Command{}

	var listReq oauth2.ListCustomAppIntegrationsRequest

	cmd.Flags().BoolVar(&listReq.IncludeCreatorUsername, "include-creator-username", listReq.IncludeCreatorUsername, ``)
	cmd.Flags().IntVar(&listReq.PageSize, "page-size", listReq.PageSize, ``)
	cmd.Flags().StringVar(&listReq.PageToken, "page-token", listReq.PageToken, ``)

	cmd.Use = "list"
	cmd.Short = `Get custom oauth app integrations.`
	cmd.Long = `Get custom oauth app integrations.
  
  Get the list of custom OAuth app integrations for the specified Databricks
  account`

	cmd.Annotations = make(map[string]string)

	cmd.Args = func(cmd *cobra.Command, args []string) error {
		check := root.ExactArgs(0)
		return check(cmd, args)
	}

	cmd.PreRunE = root.MustAccountClient
	cmd.RunE = func(cmd *cobra.Command, args []string) (err error) {
		ctx := cmd.Context()
		a := cmdctx.AccountClient(ctx)

		response := a.CustomAppIntegration.List(ctx, listReq)
		return cmdio.RenderIterator(ctx, response)
	}

	// Disable completions since they are not applicable.
	// Can be overridden by manual implementation in `override.go`.
	cmd.ValidArgsFunction = cobra.NoFileCompletions

	// Apply optional overrides to this command.
	for _, fn := range listOverrides {
		fn(cmd, &listReq)
	}

	return cmd
}

// start update command

// Slice with functions to override default command behavior.
// Functions can be added from the `init()` function in manually curated files in this directory.
var updateOverrides []func(
	*cobra.Command,
	*oauth2.UpdateCustomAppIntegration,
)

func newUpdate() *cobra.Command {
	cmd := &cobra.Command{}

	var updateReq oauth2.UpdateCustomAppIntegration
	var updateJson flags.JsonFlag

	cmd.Flags().Var(&updateJson, "json", `either inline JSON string or @path/to/file.json with request body`)

	// TODO: array: redirect_urls
	// TODO: array: scopes
	// TODO: complex arg: token_access_policy
	// TODO: array: user_authorized_scopes

	cmd.Use = "update INTEGRATION_ID"
	cmd.Short = `Updates Custom OAuth App Integration.`
	cmd.Long = `Updates Custom OAuth App Integration.
  
  Updates an existing custom OAuth App Integration. You can retrieve the custom
  OAuth app integration via :method:CustomAppIntegration/get.`

	cmd.Annotations = make(map[string]string)

	cmd.Args = func(cmd *cobra.Command, args []string) error {
		check := root.ExactArgs(1)
		return check(cmd, args)
	}

	cmd.PreRunE = root.MustAccountClient
	cmd.RunE = func(cmd *cobra.Command, args []string) (err error) {
		ctx := cmd.Context()
		a := cmdctx.AccountClient(ctx)

		if cmd.Flags().Changed("json") {
			diags := updateJson.Unmarshal(&updateReq)
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
		updateReq.IntegrationId = args[0]

		err = a.CustomAppIntegration.Update(ctx, updateReq)
		if err != nil {
			return err
		}
		return nil
	}

	// Disable completions since they are not applicable.
	// Can be overridden by manual implementation in `override.go`.
	cmd.ValidArgsFunction = cobra.NoFileCompletions

	// Apply optional overrides to this command.
	for _, fn := range updateOverrides {
		fn(cmd, &updateReq)
	}

	return cmd
}

// end service CustomAppIntegration
