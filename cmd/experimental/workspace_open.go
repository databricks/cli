package experimental

import (
	"context"
	"fmt"
	"slices"
	"strings"

	"github.com/spf13/cobra"

	"github.com/databricks/cli/cmd/root"
	"github.com/databricks/cli/libs/browser"
	"github.com/databricks/cli/libs/cmdctx"
	"github.com/databricks/cli/libs/cmdio"
	"github.com/databricks/cli/libs/workspaceurls"
)

var supportedOpenResourceTypes = []string{
	workspaceurls.ResourceApps,
	workspaceurls.ResourceClusters,
	workspaceurls.ResourceDashboards,
	workspaceurls.ResourceExperiments,
	workspaceurls.ResourceJobs,
	workspaceurls.ResourceNotebooks,
	workspaceurls.ResourcePipelines,
	workspaceurls.ResourceQueries,
	workspaceurls.ResourceWarehouses,
}

var currentWorkspaceID = func(ctx context.Context) (int64, error) {
	return cmdctx.WorkspaceClient(ctx).CurrentWorkspaceID(ctx)
}

var openWorkspaceURL = func(ctx context.Context, targetURL string) error {
	return browser.OpenURL(ctx, ".", targetURL)
}

func resourceTypeNames() []string {
	return workspaceurls.SortResourceTypes(supportedOpenResourceTypes)
}

func supportedResourceTypes() string {
	return strings.Join(resourceTypeNames(), ", ")
}

// buildWorkspaceURL constructs a full workspace URL for a given resource type and ID.
func buildWorkspaceURL(host, resourceType, id string, workspaceID int64) (string, error) {
	if !slices.Contains(supportedOpenResourceTypes, resourceType) {
		return "", fmt.Errorf("unknown resource type %q, must be one of: %s", resourceType, supportedResourceTypes())
	}

	pattern, ok := workspaceurls.LookupPattern(resourceType)
	if !ok {
		return "", fmt.Errorf("unknown resource type %q, must be one of: %s", resourceType, supportedResourceTypes())
	}

	return workspaceurls.BuildResourceURL(host, pattern, id, workspaceID)
}

func newWorkspaceOpenCommand() *cobra.Command {
	var printURL bool

	cmd := &cobra.Command{
		Use:   "open [flags] RESOURCE_TYPE ID_OR_PATH",
		Short: "Open a workspace resource or print its URL",
		Long: fmt.Sprintf(`Open a workspace resource in the default browser, or print its URL.

Supported resource types: %s.

Examples:
  databricks experimental open jobs 123456789
  databricks experimental open notebooks /Users/user@example.com/my-notebook
  databricks experimental open clusters 0123-456789-abcdef
  databricks experimental open jobs 123456789 --url`, supportedResourceTypes()),
		Args:    cobra.ExactArgs(2),
		PreRunE: root.MustWorkspaceClient,
		RunE: func(cmd *cobra.Command, args []string) error {
			ctx := cmd.Context()
			w := cmdctx.WorkspaceClient(ctx)

			resourceType := args[0]
			id := args[1]

			workspaceID, err := currentWorkspaceID(ctx)
			if err != nil {
				workspaceID = 0
			}

			resourceURL, err := buildWorkspaceURL(w.Config.Host, resourceType, id, workspaceID)
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
				return resourceTypeNames(), cobra.ShellCompDirectiveNoFileComp
			}
			return nil, cobra.ShellCompDirectiveNoFileComp
		},
	}

	cmd.Flags().BoolVar(&printURL, "url", false, "Print the workspace URL instead of opening the browser")

	return cmd
}
