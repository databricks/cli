// Code generated from OpenAPI specs by Databricks SDK Generator. DO NOT EDIT.

package access_control

import (
	"fmt"

	"github.com/databricks/cli/cmd/root"
	"github.com/databricks/cli/libs/cmdctx"
	"github.com/databricks/cli/libs/cmdio"
	"github.com/databricks/cli/libs/flags"
	"github.com/databricks/databricks-sdk-go/service/iam"
	"github.com/spf13/cobra"
)

// Slice with functions to override default command behavior.
// Functions can be added from the `init()` function in manually curated files in this directory.
var cmdOverrides []func(*cobra.Command)

func New() *cobra.Command {
	cmd := &cobra.Command{
		Use:     "access-control",
		Short:   `Rule based Access Control for Databricks Resources.`,
		Long:    `Rule based Access Control for Databricks Resources.`,
		GroupID: "iam",
		Annotations: map[string]string{
			"package": "iam",
		},

		// This service is being previewed; hide from help output.
		Hidden: true,
		RunE:   root.ReportUnknownSubcommand,
	}

	// Add methods
	cmd.AddCommand(newCheckPolicy())

	// Apply optional overrides to this command.
	for _, fn := range cmdOverrides {
		fn(cmd)
	}

	return cmd
}

// start check-policy command

// Slice with functions to override default command behavior.
// Functions can be added from the `init()` function in manually curated files in this directory.
var checkPolicyOverrides []func(
	*cobra.Command,
	*iam.CheckPolicyRequest,
)

func newCheckPolicy() *cobra.Command {
	cmd := &cobra.Command{}

	var checkPolicyReq iam.CheckPolicyRequest
	var checkPolicyJson flags.JsonFlag

	cmd.Flags().Var(&checkPolicyJson, "json", `either inline JSON string or @path/to/file.json with request body`)

	// TODO: complex arg: resource_info

	cmd.Use = "check-policy"
	cmd.Short = `Check access policy to a resource.`
	cmd.Long = `Check access policy to a resource.`

	cmd.Annotations = make(map[string]string)

	cmd.PreRunE = root.MustWorkspaceClient
	cmd.RunE = func(cmd *cobra.Command, args []string) (err error) {
		ctx := cmd.Context()
		w := cmdctx.WorkspaceClient(ctx)

		if cmd.Flags().Changed("json") {
			diags := checkPolicyJson.Unmarshal(&checkPolicyReq)
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

		response, err := w.AccessControl.CheckPolicy(ctx, checkPolicyReq)
		if err != nil {
			return err
		}
		return cmdio.Render(ctx, response)
	}

	// Disable completions since they are not applicable.
	// Can be overridden by manual implementation in `override.go`.
	cmd.ValidArgsFunction = cobra.NoFileCompletions

	// Apply optional overrides to this command.
	for _, fn := range checkPolicyOverrides {
		fn(cmd, &checkPolicyReq)
	}

	return cmd
}

// end service AccessControl
