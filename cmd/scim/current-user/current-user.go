package current_user

import (
	"github.com/databricks/bricks/lib/ui"
	"github.com/databricks/bricks/project"
	"github.com/spf13/cobra"
)

var Cmd = &cobra.Command{
	Use:   "current-user",
	Short: `This API allows retrieving information about currently authenticated user or service principal.`,
}

func init() {
	Cmd.AddCommand(meCmd)

}

var meCmd = &cobra.Command{
	Use:   "me",
	Short: `Get current user info.`,

	PreRunE: project.Configure, // TODO: improve logic for bundle/non-bundle invocations
	RunE: func(cmd *cobra.Command, args []string) error {
		ctx := cmd.Context()
		w := project.Get(ctx).WorkspacesClient()
		response, err := w.CurrentUser.Me(ctx)
		if err != nil {
			return err
		}

		pretty, err := ui.MarshalJSON(response)
		if err != nil {
			return err
		}
		cmd.OutOrStdout().Write(pretty)

		return nil
	},
}
