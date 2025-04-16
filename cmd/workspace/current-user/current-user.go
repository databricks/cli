// Code generated from OpenAPI specs by Databricks SDK Generator. DO NOT EDIT.

package current_user

import (
	"github.com/databricks/cli/cmd/root"
	"github.com/databricks/cli/libs/cmdctx"
	"github.com/databricks/cli/libs/cmdio"
	"github.com/spf13/cobra"
)

// Slice with functions to override default command behavior.
// Functions can be added from the `init()` function in manually curated files in this directory.
var cmdOverrides []func(*cobra.Command)

func New() *cobra.Command {
	cmd := &cobra.Command{
		Use:   "current-user",
		Short: `This API allows retrieving information about currently authenticated user or service principal.`,
		Long: `This API allows retrieving information about currently authenticated user or
  service principal.`,
		GroupID: "iam",
		Annotations: map[string]string{
			"package": "iam",
		},
		RunE: root.ReportUnknownSubcommand,
	}

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
)

func newMe() *cobra.Command {
	cmd := &cobra.Command{}

	cmd.Use = "me"
	cmd.Short = `Get current user info.`
	cmd.Long = `Get current user info.
  
  Get details about the current method caller's identity.`

	cmd.Annotations = make(map[string]string)

	cmd.PreRunE = root.MustWorkspaceClient
	cmd.RunE = func(cmd *cobra.Command, args []string) (err error) {
		ctx := cmd.Context()
		w := cmdctx.WorkspaceClient(ctx)
		response, err := w.CurrentUser.Me(ctx)
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
		fn(cmd)
	}

	return cmd
}

// end service CurrentUser
