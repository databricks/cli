// Code generated from OpenAPI specs by Databricks SDK Generator. DO NOT EDIT.

package credentials_manager

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
		Use:   "credentials-manager",
		Short: `Credentials manager interacts with with Identity Providers to to perform token exchanges using stored credentials and refresh tokens.`,
		Long: `Credentials manager interacts with with Identity Providers to to perform token
  exchanges using stored credentials and refresh tokens.`,
		GroupID: "settings",
		Annotations: map[string]string{
			"package": "settings",
		},

		// This service is being previewed; hide from help output.
		Hidden: true,
		RunE:   root.ReportUnknownSubcommand,
	}

	// Add methods
	cmd.AddCommand(newExchangeToken())

	// Apply optional overrides to this command.
	for _, fn := range cmdOverrides {
		fn(cmd)
	}

	return cmd
}

// start exchange-token command

// Slice with functions to override default command behavior.
// Functions can be added from the `init()` function in manually curated files in this directory.
var exchangeTokenOverrides []func(
	*cobra.Command,
	*settings.ExchangeTokenRequest,
)

func newExchangeToken() *cobra.Command {
	cmd := &cobra.Command{}

	var exchangeTokenReq settings.ExchangeTokenRequest
	var exchangeTokenJson flags.JsonFlag

	cmd.Flags().Var(&exchangeTokenJson, "json", `either inline JSON string or @path/to/file.json with request body`)

	cmd.Use = "exchange-token"
	cmd.Short = `Exchange token.`
	cmd.Long = `Exchange token.
  
  Exchange tokens with an Identity Provider to get a new access token. It allows
  specifying scopes to determine token permissions.`

	cmd.Annotations = make(map[string]string)

	cmd.PreRunE = root.MustWorkspaceClient
	cmd.RunE = func(cmd *cobra.Command, args []string) (err error) {
		ctx := cmd.Context()
		w := cmdctx.WorkspaceClient(ctx)

		if cmd.Flags().Changed("json") {
			diags := exchangeTokenJson.Unmarshal(&exchangeTokenReq)
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

		response, err := w.CredentialsManager.ExchangeToken(ctx, exchangeTokenReq)
		if err != nil {
			return err
		}
		return cmdio.Render(ctx, response)
	}

	// Disable completions since they are not applicable.
	// Can be overridden by manual implementation in `override.go`.
	cmd.ValidArgsFunction = cobra.NoFileCompletions

	// Apply optional overrides to this command.
	for _, fn := range exchangeTokenOverrides {
		fn(cmd, &exchangeTokenReq)
	}

	return cmd
}

// end service CredentialsManager
