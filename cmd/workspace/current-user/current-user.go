// Code generated from OpenAPI specs by Databricks SDK Generator. DO NOT EDIT.

package current_user

import (
	"github.com/databricks/cli/cmd/root"
	"github.com/databricks/cli/libs/cmdctx"
	"github.com/databricks/cli/libs/cmdio"
	"github.com/databricks/databricks-sdk-go/service/iam"
	"github.com/spf13/cobra"
)

// Slice with functions to override default command behavior.
// Functions can be added from the `init()` function in manually curated files in this directory.
var cmdOverrides []func(*cobra.Command)

func New() *cobra.Command {
	cmd := &cobra.Command{
		Use:   "current-user",
		Short: `*Public Preview* This API allows retrieving information about currently authenticated user or service principal.`,
		Long: `This command is in Public Preview and may change without notice.

This API allows retrieving information about currently authenticated user or
  service principal.`,
		GroupID: "iam",
		RunE:    root.ReportUnknownSubcommand,
	}

	cmd.Annotations = make(map[string]string)
	cmd.Annotations["launch_stage"] = "PUBLIC_PREVIEW"
	cmd.Annotations["launch_stage_display"] = "Public Preview"

	// Add methods
	cmd.AddCommand(newMe())

	// Apply optional overrides to this command.
	for _, fn := range cmdOverrides {
		fn(cmd)
	}

	return cmd
}

// start me command

// Slice with functions to override default command behavior.
// Functions can be added from the `init()` function in manually curated files in this directory.
var meOverrides []func(
	*cobra.Command,
	*iam.MeRequest,
)

func newMe() *cobra.Command {
	cmd := &cobra.Command{}

	var meReq iam.MeRequest

	cmd.Flags().StringVar(&meReq.Attributes, "attributes", meReq.Attributes, `Comma-separated list of attributes to return in response.`)
	cmd.Flags().StringVar(&meReq.ExcludedAttributes, "excluded-attributes", meReq.ExcludedAttributes, `Comma-separated list of attributes to exclude in response.`)

	cmd.Use = "me"
	cmd.Short = `*Public Preview* Get current user info.`
	cmd.Long = `This command is in Public Preview and may change without notice.

Get current user info.

  Get details about the current method caller's identity.`

	cmd.Annotations = make(map[string]string)
	cmd.Annotations["launch_stage"] = "PUBLIC_PREVIEW"
	cmd.Annotations["launch_stage_display"] = "Public Preview"

	cmd.Args = func(cmd *cobra.Command, args []string) error {
		check := root.ExactArgs(0)
		return check(cmd, args)
	}

	cmd.PreRunE = root.MustWorkspaceClient
	cmd.RunE = func(cmd *cobra.Command, args []string) (err error) {
		ctx := cmd.Context()
		w := cmdctx.WorkspaceClient(ctx)

		response, err := w.CurrentUser.Me(ctx, meReq)
		if err != nil {
			return err
		}

		return cmdio.Render(ctx, response)
	}

	// Disable completions since they are not applicable.
	// Can be overridden by manual implementation in `override.go`.
	cmd.ValidArgsFunction = cobra.NoFileCompletions

	// Apply optional overrides to this command.
	for _, fn := range meOverrides {
		fn(cmd, &meReq)
	}

	return cmd
}

// end service CurrentUser
