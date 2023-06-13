// Code generated from OpenAPI specs by Databricks SDK Generator. DO NOT EDIT.

package current_user

import (
	"github.com/databricks/cli/cmd/root"
	"github.com/databricks/cli/libs/cmdio"
	"github.com/spf13/cobra"
)

var Cmd = &cobra.Command{
	Use:   "current-user",
	Short: `This API allows retrieving information about currently authenticated user or service principal.`,
	Long: `This API allows retrieving information about currently authenticated user or
  service principal.`,
}

// start me command

func init() {
	Cmd.AddCommand(meCmd)

}

var meCmd = &cobra.Command{
	Use:   "me",
	Short: `Get current user info.`,
	Long: `Get current user info.
  
  Get details about the current method caller's identity.`,

	Annotations: map[string]string{
		"package": "iam",
	},
	PreRunE: root.MustWorkspaceClient,
	RunE: func(cmd *cobra.Command, args []string) (err error) {
		ctx := cmd.Context()
		w := root.WorkspaceClient(ctx)
		response, err := w.CurrentUser.Me(ctx)
		if err != nil {
			return err
		}
		return cmdio.Render(ctx, response)
	},
}

// end service CurrentUser
