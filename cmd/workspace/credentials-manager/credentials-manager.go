// Code generated from OpenAPI specs by Databricks SDK Generator. DO NOT EDIT.

package credentials_manager

import (
	"errors"

	"github.com/databricks/cli/cmd/root"
	"github.com/databricks/cli/libs/apiclient"
	"github.com/databricks/cli/libs/auth"
	"github.com/databricks/cli/libs/cmdctx"
	"github.com/databricks/cli/libs/cmdio"
	"github.com/databricks/cli/libs/flags"
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

		// This service is being previewed; hide from help output.
		Hidden: true,
		RunE:   root.ReportUnknownSubcommand,
	}

	cmd.Annotations = make(map[string]string)
	cmd.Annotations["launch_stage"] = "PRIVATE_PREVIEW"
	cmd.Annotations["launch_stage_display"] = "Private Preview"

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
	*map[string]any,
)

func newExchangeToken() *cobra.Command {
	cmd := &cobra.Command{}
	var exchangeTokenReq map[string]any
	exchangeTokenReq = map[string]any{}

	var exchangeTokenJson flags.JsonFlag

	cmd.Flags().Var(&exchangeTokenJson, "json", `either inline JSON string or @path/to/file.json with request body`)

	cmd.Use = "exchange-token"
	cmd.Short = `Exchange token.`
	cmd.Long = `Exchange token.

  Exchange tokens with an Identity Provider to get a new access token. It allows
  specifying scopes to determine token permissions.

  POST /exchange-tokens/token is the documented public form, expressed via
  google.api.http below. GET /exchange-tokens/$exchange is a legacy alias used
  by the Spark driver's OAuth refresh path (DBHttpClient#get sends a body via
  HttpGetWithEntity) and stays on the legacy option (rpc).endpoints
  annotation: its path contains a literal $, which google.api.http's LITERAL
  grammar does not allow, and HttpPathParser does not percent-decode template
  segments (so encoding as %24exchange would not match the literal $exchange
  path the Spark driver sends). Per-endpoint visibility: PUBLIC_UNDOCUMENTED
  preserves the DECO-7732 intent of suppressing the GET alias from the public
  API spec.`

	cmd.Annotations = make(map[string]string)
	cmd.Annotations["launch_stage"] = "PRIVATE_PREVIEW"
	cmd.Annotations["launch_stage_display"] = "Private Preview"

	cmd.PreRunE = root.MustWorkspaceClient
	cmd.RunE = func(cmd *cobra.Command, args []string) (err error) {
		ctx := cmd.Context()
		w := cmdctx.WorkspaceClient(ctx)
		_ = w

		if cmd.Flags().Changed("json") {
			diags := exchangeTokenJson.Unmarshal(&exchangeTokenReq)
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

		rawQuery := map[string]any{}
		rawPath := "/api/2.0/credentials-manager/exchange-tokens/token"

		api, err := apiclient.New(w.Config)
		if err != nil {
			return err
		}
		var response any
		err = api.Do(ctx, "POST", rawPath, auth.WorkspaceIDHeaders(w.Config), rawQuery, exchangeTokenReq, &response)
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
