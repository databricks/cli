// Code generated from OpenAPI specs by Databricks SDK Generator. DO NOT EDIT.

package access_control

import (
	"context"
	"errors"

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

// Spec declares the command for this service.
var Spec = root.CommandSpec{
	Use:     "access-control",
	Short:   `Rule based Access Control for Databricks Resources.`,
	GroupID: "iam",

	// This service is being previewed; hide from help output.
	Hidden: true,
	Build:  buildCommand,
}

func buildCommand(ctx context.Context, cmd *cobra.Command) {
	cmd.Long = `Rule based Access Control for Databricks Resources.`
	cmd.RunE = root.ReportUnknownSubcommand

	cmd.Annotations = map[string]string{
		"launch_stage":         "PRIVATE_PREVIEW",
		"launch_stage_display": "Private Preview",
	}

	// Add methods
	cmd.AddCommand(newCheckPolicy())

	// Apply optional overrides to this command.
	for _, fn := range cmdOverrides {
		fn(cmd)
	}
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
	cmd.Annotations["launch_stage"] = "PRIVATE_PREVIEW"
	cmd.Annotations["launch_stage_display"] = "Private Preview"

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
				err := cmdio.RenderDiagnostics(ctx, diags)
				if err != nil {
					return err
				}
			}
		} else {
			return errors.New("please provide command input in JSON format by specifying the --json flag")
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
