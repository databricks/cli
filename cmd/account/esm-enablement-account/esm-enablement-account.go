// Code generated from OpenAPI specs by Databricks SDK Generator. DO NOT EDIT.

package esm_enablement_account

import (
	"fmt"

	"github.com/databricks/cli/cmd/root"
	"github.com/databricks/cli/libs/cmdctx"
	"github.com/databricks/cli/libs/cmdio"
	"github.com/databricks/cli/libs/flags"
	"github.com/databricks/databricks-sdk-go/service/settings"
	"github.com/spf13/cobra"
)

// Slice with functions to override default command behavior.
// Functions can be added from the `init()` function in manually curated files in this directory.
var cmdOverrides []func(*cobra.Command)

func New() *cobra.Command {
	cmd := &cobra.Command{
		Use:   "esm-enablement-account",
		Short: `The enhanced security monitoring setting at the account level controls whether to enable the feature on new workspaces.`,
		Long: `The enhanced security monitoring setting at the account level controls whether
  to enable the feature on new workspaces. By default, this account-level
  setting is disabled for new workspaces. After workspace creation, account
  admins can enable enhanced security monitoring individually for each
  workspace.`,
		RunE: root.ReportUnknownSubcommand,
	}

	// Add methods
	cmd.AddCommand(newGet())
	cmd.AddCommand(newUpdate())

	// Apply optional overrides to this command.
	for _, fn := range cmdOverrides {
		fn(cmd)
	}

	return cmd
}

// start get command

// Slice with functions to override default command behavior.
// Functions can be added from the `init()` function in manually curated files in this directory.
var getOverrides []func(
	*cobra.Command,
	*settings.GetEsmEnablementAccountSettingRequest,
)

func newGet() *cobra.Command {
	cmd := &cobra.Command{}

	var getReq settings.GetEsmEnablementAccountSettingRequest

	cmd.Flags().StringVar(&getReq.Etag, "etag", getReq.Etag, `etag used for versioning.`)

	cmd.Use = "get"
	cmd.Short = `Get the enhanced security monitoring setting for new workspaces.`
	cmd.Long = `Get the enhanced security monitoring setting for new workspaces.
  
  Gets the enhanced security monitoring setting for new workspaces.`

	cmd.Annotations = make(map[string]string)

	cmd.Args = func(cmd *cobra.Command, args []string) error {
		check := root.ExactArgs(0)
		return check(cmd, args)
	}

	cmd.PreRunE = root.MustAccountClient
	cmd.RunE = func(cmd *cobra.Command, args []string) (err error) {
		ctx := cmd.Context()
		a := cmdctx.AccountClient(ctx)

		response, err := a.Settings.EsmEnablementAccount().Get(ctx, getReq)
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

// start update command

// Slice with functions to override default command behavior.
// Functions can be added from the `init()` function in manually curated files in this directory.
var updateOverrides []func(
	*cobra.Command,
	*settings.UpdateEsmEnablementAccountSettingRequest,
)

func newUpdate() *cobra.Command {
	cmd := &cobra.Command{}

	var updateReq settings.UpdateEsmEnablementAccountSettingRequest
	var updateJson flags.JsonFlag

	cmd.Flags().Var(&updateJson, "json", `either inline JSON string or @path/to/file.json with request body`)

	cmd.Use = "update"
	cmd.Short = `Update the enhanced security monitoring setting for new workspaces.`
	cmd.Long = `Update the enhanced security monitoring setting for new workspaces.
  
  Updates the value of the enhanced security monitoring setting for new
  workspaces.`

	cmd.Annotations = make(map[string]string)

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
		} else {
			return fmt.Errorf("please provide command input in JSON format by specifying the --json flag")
		}

		response, err := a.Settings.EsmEnablementAccount().Update(ctx, updateReq)
		if err != nil {
			return err
		}
		return cmdio.Render(ctx, response)
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

// end service EsmEnablementAccount
