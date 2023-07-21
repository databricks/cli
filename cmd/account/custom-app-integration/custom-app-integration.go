// Code generated from OpenAPI specs by Databricks SDK Generator. DO NOT EDIT.

package custom_app_integration

import (
	"fmt"

	"github.com/databricks/cli/cmd/root"
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
		Short: `These APIs enable administrators to manage custom oauth app integrations, which is required for adding/using Custom OAuth App Integration like Tableau Cloud for Databricks in AWS cloud.`,
		Long: `These APIs enable administrators to manage custom oauth app integrations,
  which is required for adding/using Custom OAuth App Integration like Tableau
  Cloud for Databricks in AWS cloud.
  
  **Note:** You can only add/use the OAuth custom application integrations when
  OAuth enrollment status is enabled. For more details see
  :method:OAuthEnrollment/create`,
		GroupID: "oauth2",
		Annotations: map[string]string{
			"package": "oauth2",
		},
	}

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

	// TODO: short flags
	cmd.Flags().Var(&createJson, "json", `either inline JSON string or @path/to/file.json with request body`)

	cmd.Flags().BoolVar(&createReq.Confidential, "confidential", createReq.Confidential, `indicates if an oauth client-secret should be generated.`)
	// TODO: complex arg: token_access_policy

	cmd.Use = "create"
	cmd.Short = `Create Custom OAuth App Integration.`
	cmd.Long = `Create Custom OAuth App Integration.
  
  Create Custom OAuth App Integration.
  
  You can retrieve the custom oauth app integration via
  :method:CustomAppIntegration/get.`

	cmd.Annotations = make(map[string]string)

	cmd.PreRunE = root.MustAccountClient
	cmd.RunE = func(cmd *cobra.Command, args []string) (err error) {
		ctx := cmd.Context()
		a := root.AccountClient(ctx)

		if cmd.Flags().Changed("json") {
			err = createJson.Unmarshal(&createReq)
			if err != nil {
				return err
			}
		} else {
			return fmt.Errorf("please provide command input in JSON format by specifying the --json flag")
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

	// TODO: short flags

	cmd.Use = "delete INTEGRATION_ID"
	cmd.Short = `Delete Custom OAuth App Integration.`
	cmd.Long = `Delete Custom OAuth App Integration.
  
  Delete an existing Custom OAuth App Integration. You can retrieve the custom
  oauth app integration via :method:CustomAppIntegration/get.`

	cmd.Annotations = make(map[string]string)

	cmd.Args = func(cmd *cobra.Command, args []string) error {
		check := cobra.ExactArgs(1)
		return check(cmd, args)
	}

	cmd.PreRunE = root.MustAccountClient
	cmd.RunE = func(cmd *cobra.Command, args []string) (err error) {
		ctx := cmd.Context()
		a := root.AccountClient(ctx)

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

	// TODO: short flags

	cmd.Use = "get INTEGRATION_ID"
	cmd.Short = `Get OAuth Custom App Integration.`
	cmd.Long = `Get OAuth Custom App Integration.
  
  Gets the Custom OAuth App Integration for the given integration id.`

	cmd.Annotations = make(map[string]string)

	cmd.Args = func(cmd *cobra.Command, args []string) error {
		check := cobra.ExactArgs(1)
		return check(cmd, args)
	}

	cmd.PreRunE = root.MustAccountClient
	cmd.RunE = func(cmd *cobra.Command, args []string) (err error) {
		ctx := cmd.Context()
		a := root.AccountClient(ctx)

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
)

func newList() *cobra.Command {
	cmd := &cobra.Command{}

	cmd.Use = "list"
	cmd.Short = `Get custom oauth app integrations.`
	cmd.Long = `Get custom oauth app integrations.
  
  Get the list of custom oauth app integrations for the specified Databricks
  account`

	cmd.Annotations = make(map[string]string)

	cmd.PreRunE = root.MustAccountClient
	cmd.RunE = func(cmd *cobra.Command, args []string) (err error) {
		ctx := cmd.Context()
		a := root.AccountClient(ctx)
		response, err := a.CustomAppIntegration.ListAll(ctx)
		if err != nil {
			return err
		}
		return cmdio.Render(ctx, response)
	}

	// Disable completions since they are not applicable.
	// Can be overridden by manual implementation in `override.go`.
	cmd.ValidArgsFunction = cobra.NoFileCompletions

	// Apply optional overrides to this command.
	for _, fn := range listOverrides {
		fn(cmd)
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

	// TODO: short flags
	cmd.Flags().Var(&updateJson, "json", `either inline JSON string or @path/to/file.json with request body`)

	// TODO: array: redirect_urls
	// TODO: complex arg: token_access_policy

	cmd.Use = "update INTEGRATION_ID"
	cmd.Short = `Updates Custom OAuth App Integration.`
	cmd.Long = `Updates Custom OAuth App Integration.
  
  Updates an existing custom OAuth App Integration. You can retrieve the custom
  oauth app integration via :method:CustomAppIntegration/get.`

	cmd.Annotations = make(map[string]string)

	cmd.Args = func(cmd *cobra.Command, args []string) error {
		check := cobra.ExactArgs(1)
		return check(cmd, args)
	}

	cmd.PreRunE = root.MustAccountClient
	cmd.RunE = func(cmd *cobra.Command, args []string) (err error) {
		ctx := cmd.Context()
		a := root.AccountClient(ctx)

		if cmd.Flags().Changed("json") {
			err = updateJson.Unmarshal(&updateReq)
			if err != nil {
				return err
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
