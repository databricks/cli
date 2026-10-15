// Code generated from OpenAPI specs by Databricks SDK Generator. DO NOT EDIT.

package custom_app_integration

import (
	"fmt"

	"github.com/databricks/cli/cmd/root"
	"github.com/databricks/cli/libs/cmdctx"
	"github.com/databricks/cli/libs/cmdgen"
	"github.com/databricks/cli/libs/cmdio"
	"github.com/databricks/cli/libs/flags"
	"github.com/spf13/cobra"

	oauth "github.com/databricks/cli/libs/sdktypes/oauth/v1"
	sdkoauth "github.com/databricks/sdk-go/oauth/v1"
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
		RunE:    root.ReportUnknownSubcommand,
	}

	cmd.Annotations = make(map[string]string)
	cmd.Annotations["launch_stage"] = "GA"
	cmd.Annotations["launch_stage_display"] = "GA"

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
	*oauth.CreateCustomOAuthAppIntegrationRequest,
)

func newCreate() *cobra.Command {
	cmd := &cobra.Command{}
	var createReq oauth.CreateCustomOAuthAppIntegrationRequest

	var createJson flags.JsonFlag

	cmd.Flags().Var(&createJson, "json", `either inline JSON string or @path/to/file.json with request body`)

	flags.BoolPtrVar(cmd.Flags(), &createReq.Confidential, "confidential", `This field indicates whether an OAuth client secret is required to authenticate this client.`)
	flags.StringPtrVar(cmd.Flags(), &createReq.Name, "name", `Name of the custom OAuth app.`)
	// TODO: array: redirect_urls
	// TODO: array: scopes
	// TODO: complex arg: token_access_policy
	// TODO: array: user_authorized_scopes

	cmd.Use = "create"
	cmd.Short = `Create Custom OAuth App Integration.`
	cmd.Long = `Create Custom OAuth App Integration.

  Create Custom OAuth App Integration.

  You can retrieve the custom OAuth app integration via
  [CustomAppIntegration/get].

  [CustomAppIntegration/get]: https://docs.databricks.com/api/account/customappintegration/get`

	cmd.Annotations = make(map[string]string)
	cmd.Annotations["launch_stage"] = "GA"
	cmd.Annotations["launch_stage_display"] = "GA"

	cmd.Args = func(cmd *cobra.Command, args []string) error {
		check := root.ExactArgs(0)
		return check(cmd, args)
	}

	cmd.PreRunE = root.MustAccountClient
	cmd.RunE = func(cmd *cobra.Command, args []string) (err error) {
		ctx := cmd.Context()
		a := cmdctx.AccountClient(ctx)
		_ = a

		if cmd.Flags().Changed("json") {
			diags := createJson.Unmarshal(&createReq)
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

		sdkReq, err := oauth.CreateCustomOAuthAppIntegrationRequestToSDK(&createReq)
		if err != nil {
			return err
		}
		response, err := a.OAuth.CreateCustomOAuthAppIntegration(ctx, *sdkReq)
		if err != nil {
			return err
		}

		out, err := oauth.CustomOAuthAppIntegrationSecretFromSDK(response)
		if err != nil {
			return err
		}
		return cmdio.Render(ctx, out)
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
	*sdkoauth.DeleteCustomOAuthAppIntegrationRequest,
)

func newDelete() *cobra.Command {
	cmd := &cobra.Command{}
	var deleteReq sdkoauth.DeleteCustomOAuthAppIntegrationRequest

	cmd.Use = "delete INTEGRATION_ID"
	cmd.Short = `Delete Custom OAuth App Integration.`
	cmd.Long = `Delete Custom OAuth App Integration.

  Delete an existing Custom OAuth App Integration. You can retrieve the custom
  OAuth app integration via [CustomAppIntegration/get].

  [CustomAppIntegration/get]: https://docs.databricks.com/api/account/customappintegration/get`

	cmd.Annotations = make(map[string]string)
	cmd.Annotations["launch_stage"] = "GA"
	cmd.Annotations["launch_stage_display"] = "GA"

	cmd.Args = func(cmd *cobra.Command, args []string) error {
		check := root.ExactArgs(1)
		return check(cmd, args)
	}

	cmd.PreRunE = root.MustAccountClient
	cmd.RunE = func(cmd *cobra.Command, args []string) (err error) {
		ctx := cmd.Context()
		a := cmdctx.AccountClient(ctx)
		_ = a

		deleteReq.IntegrationId = new(args[0])

		sdkReq := &deleteReq
		response, err := a.OAuth.DeleteCustomOAuthAppIntegration(ctx, *sdkReq)
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
	*sdkoauth.GetCustomOAuthAppIntegrationRequest,
)

func newGet() *cobra.Command {
	cmd := &cobra.Command{}
	var getReq sdkoauth.GetCustomOAuthAppIntegrationRequest

	cmd.Use = "get INTEGRATION_ID"
	cmd.Short = `Get OAuth Custom App Integration.`
	cmd.Long = `Get OAuth Custom App Integration.

  Gets the Custom OAuth App Integration for the given integration id.

  Arguments:
    INTEGRATION_ID: The OAuth app integration ID.`

	cmd.Annotations = make(map[string]string)
	cmd.Annotations["launch_stage"] = "GA"
	cmd.Annotations["launch_stage_display"] = "GA"

	cmd.Args = func(cmd *cobra.Command, args []string) error {
		check := root.ExactArgs(1)
		return check(cmd, args)
	}

	cmd.PreRunE = root.MustAccountClient
	cmd.RunE = func(cmd *cobra.Command, args []string) (err error) {
		ctx := cmd.Context()
		a := cmdctx.AccountClient(ctx)
		_ = a

		getReq.IntegrationId = new(args[0])

		sdkReq := &getReq
		response, err := a.OAuth.GetCustomOAuthAppIntegration(ctx, *sdkReq)
		if err != nil {
			return err
		}

		out, err := oauth.CustomOAuthAppIntegrationFromSDK(response)
		if err != nil {
			return err
		}
		return cmdio.Render(ctx, out)
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
	*oauth.ListCustomOAuthAppIntegrationsRequest,
)

func newList() *cobra.Command {
	cmd := &cobra.Command{}
	var listReq oauth.ListCustomOAuthAppIntegrationsRequest

	// Registered for all paginated methods. Validated at call time in the
	// method-call template. Paginated list methods never have Wait or LRO
	// branches, so the method-call path is always reached.
	var listLimit int

	flags.BoolPtrVar(cmd.Flags(), &listReq.IncludeCreatorUsername, "include-creator-username", ``)
	flags.IntPtrVar(cmd.Flags(), &listReq.PageSize, "page-size", ``)

	// Limit flag for total result capping.
	cmd.Flags().IntVar(&listLimit, "limit", 0, `Maximum number of results to return.`)

	// Hidden pagination flags (internal API parameters).
	flags.StringPtrVar(cmd.Flags(), &listReq.PageToken, "page-token", `Pagination token.`)
	cmd.Flags().Lookup("page-token").Hidden = true

	cmd.Use = "list"
	cmd.Short = `Get custom oauth app integrations.`
	cmd.Long = `Get custom oauth app integrations.

  Get the list of custom OAuth app integrations for the specified Databricks
  account`

	cmd.Annotations = make(map[string]string)
	cmd.Annotations["launch_stage"] = "GA"
	cmd.Annotations["launch_stage_display"] = "GA"

	cmd.Args = func(cmd *cobra.Command, args []string) error {
		check := root.ExactArgs(0)
		return check(cmd, args)
	}

	cmd.PreRunE = root.MustAccountClient
	cmd.RunE = func(cmd *cobra.Command, args []string) (err error) {
		ctx := cmd.Context()
		a := cmdctx.AccountClient(ctx)
		_ = a

		sdkReq, err := oauth.ListCustomOAuthAppIntegrationsRequestToSDK(&listReq)
		if err != nil {
			return err
		}
		if listLimit < 0 {
			return fmt.Errorf("--limit must be a non-negative integer, got %d", listLimit)
		}
		if listLimit > 0 {
			ctx = cmdio.WithLimit(ctx, listLimit)
		}
		response := a.OAuth.ListCustomOAuthAppIntegrationsIter(ctx, *sdkReq)
		return cmdio.RenderIterator(ctx, cmdgen.MapSeq(response, oauth.CustomOAuthAppIntegrationFromSDK))
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
	*oauth.UpdateCustomOAuthAppIntegrationRequest,
)

func newUpdate() *cobra.Command {
	cmd := &cobra.Command{}
	var updateReq oauth.UpdateCustomOAuthAppIntegrationRequest

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
  OAuth app integration via [CustomAppIntegration/get].

  [CustomAppIntegration/get]: https://docs.databricks.com/api/account/customappintegration/get`

	cmd.Annotations = make(map[string]string)
	cmd.Annotations["launch_stage"] = "GA"
	cmd.Annotations["launch_stage_display"] = "GA"

	cmd.Args = func(cmd *cobra.Command, args []string) error {
		check := root.ExactArgs(1)
		return check(cmd, args)
	}

	cmd.PreRunE = root.MustAccountClient
	cmd.RunE = func(cmd *cobra.Command, args []string) (err error) {
		ctx := cmd.Context()
		a := cmdctx.AccountClient(ctx)
		_ = a

		if cmd.Flags().Changed("json") {
			diags := updateJson.Unmarshal(&updateReq)
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
		updateReq.IntegrationId = new(args[0])

		sdkReq, err := oauth.UpdateCustomOAuthAppIntegrationRequestToSDK(&updateReq)
		if err != nil {
			return err
		}
		response, err := a.OAuth.UpdateCustomOAuthAppIntegration(ctx, *sdkReq)
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
	for _, fn := range updateOverrides {
		fn(cmd, &updateReq)
	}

	return cmd
}

// end service CustomAppIntegration
