package experimental

import (
	"context"
	"fmt"
	"strings"

	"github.com/spf13/cobra"

	"github.com/databricks/cli/cmd/root"
	"github.com/databricks/cli/libs/browser"
	"github.com/databricks/cli/libs/cmdctx"
	"github.com/databricks/cli/libs/cmdio"
	"github.com/databricks/cli/libs/log"
	"github.com/databricks/cli/libs/workspaceurls"
)

var currentWorkspaceID = func(ctx context.Context) (int64, error) {
	return cmdctx.WorkspaceClient(ctx).CurrentWorkspaceID(ctx)
}

var openWorkspaceURL = browser.Open

func newWorkspaceOpenCommand() *cobra.Command {
	var printURL bool

	cmd := &cobra.Command{
		Use:   "open [flags] RESOURCE_TYPE ID_OR_PATH",
		Short: "Open a workspace resource or print its URL",
		Long: fmt.Sprintf(`Open a workspace resource in the default browser, or print its URL.

Supported resource types: %s.

For registered_models, use the dot-separated name (catalog.schema.model)
and it will be converted to the correct URL path automatically.

Examples:
  databricks experimental open jobs 123456789
  databricks experimental open notebooks /Users/user@example.com/my-notebook
  databricks experimental open clusters 0123-456789-abcdef
  databricks experimental open registered_models catalog.schema.my_model
  databricks experimental open jobs 123456789 --url`, strings.Join(workspaceurls.ResourceTypes(), ", ")),
		Args:    cobra.ExactArgs(2),
		PreRunE: root.MustWorkspaceClient,
		RunE: func(cmd *cobra.Command, args []string) error {
			ctx := cmd.Context()
			w := cmdctx.WorkspaceClient(ctx)

			resourceType := args[0]
			id := args[1]

			workspaceID, err := currentWorkspaceID(ctx)
			if err != nil {
				log.Warnf(ctx, "Could not determine workspace ID: %v", err)
			}

			resourceURL, err := workspaceurls.BuildResourceURL(w.Config.Host, resourceType, id, workspaceID)
			if err != nil {
				return err
			}

			if printURL {
				_, err := fmt.Fprintln(cmd.OutOrStdout(), resourceURL)
				return err
			}

			if !browser.IsDisabled(ctx) {
				cmdio.LogString(ctx, fmt.Sprintf("Opening %s %s in the browser...", resourceType, id))
			}

			return openWorkspaceURL(ctx, resourceURL)
		},
		ValidArgsFunction: func(cmd *cobra.Command, args []string, toComplete string) ([]string, cobra.ShellCompDirective) {
			if len(args) == 0 {
				return workspaceurls.ResourceTypes(), cobra.ShellCompDirectiveNoFileComp
			}
			return nil, cobra.ShellCompDirectiveNoFileComp
		},
	}

	cmd.Flags().BoolVar(&printURL, "url", false, "Print the workspace URL instead of opening the browser")

	return cmd
}
