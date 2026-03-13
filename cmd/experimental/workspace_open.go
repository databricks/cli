package experimental

import (
	"fmt"
	"net/url"
	"sort"
	"strconv"
	"strings"

	"github.com/spf13/cobra"

	"github.com/databricks/cli/cmd/root"
	"github.com/databricks/cli/libs/browser"
	"github.com/databricks/cli/libs/cmdctx"
	"github.com/databricks/cli/libs/cmdio"
)

// resourceURLPatterns maps resource type names to their URL path patterns.
// Patterns starting with "#" use URL fragments instead of path segments.
var resourceURLPatterns = map[string]string{
	"job":       "/jobs/%s",
	"notebook":  "#notebook/%s",
	"cluster":   "/compute/clusters/%s",
	"pipeline":  "/pipelines/%s",
	"dashboard": "/dashboardsv3/%s/published",
	"warehouse": "/sql/warehouses/%s",
	"query":     "/sql/editor/%s",
	"app":       "/apps/%s",
}

func resourceTypeNames() []string {
	names := make([]string, 0, len(resourceURLPatterns))
	for name := range resourceURLPatterns {
		names = append(names, name)
	}
	sort.Strings(names)
	return names
}

// buildWorkspaceURL constructs a full workspace URL for a given resource type and ID.
func buildWorkspaceURL(host, resourceType, id string, workspaceID int64) (string, error) {
	pattern, ok := resourceURLPatterns[resourceType]
	if !ok {
		return "", fmt.Errorf("unknown resource type %q, must be one of: %v", resourceType, resourceTypeNames())
	}

	baseURL, err := url.Parse(host)
	if err != nil {
		return "", fmt.Errorf("invalid workspace host %q: %w", host, err)
	}

	// Append ?o=<workspace-id> when the workspace ID is not already in the
	// hostname (e.g. vanity URLs or legacy workspace URLs).
	// See https://docs.databricks.com/en/workspace/workspace-details.html
	if workspaceID != 0 {
		orgID := strconv.FormatInt(workspaceID, 10)
		if !hasWorkspaceIDInHostname(baseURL.Hostname(), orgID) {
			values := baseURL.Query()
			values.Add("o", orgID)
			baseURL.RawQuery = values.Encode()
		}
	}

	resourcePath := fmt.Sprintf(pattern, id)

	// Fragment-based URLs (starting with #) need special handling.
	if len(resourcePath) > 0 && resourcePath[0] == '#' {
		baseURL.Path = "/"
		baseURL.Fragment = resourcePath[1:]
	} else {
		baseURL.Path = resourcePath
	}

	return baseURL.String(), nil
}

func hasWorkspaceIDInHostname(hostname, workspaceID string) bool {
	remainder, ok := strings.CutPrefix(strings.ToLower(hostname), "adb-"+workspaceID)
	return ok && (remainder == "" || strings.HasPrefix(remainder, "."))
}

func newWorkspaceOpenCommand() *cobra.Command {
	cmd := &cobra.Command{
		Use:   "open RESOURCE_TYPE ID_OR_PATH",
		Short: "Open a workspace resource in the browser",
		Long: `Open a workspace resource in the default browser.

Supported resource types: job, notebook, cluster, pipeline, dashboard, warehouse, query, app.

Examples:
  databricks experimental open job 123456789
  databricks experimental open notebook /Users/user@example.com/my-notebook
  databricks experimental open cluster 0123-456789-abcdef`,
		Args:    cobra.ExactArgs(2),
		PreRunE: root.MustWorkspaceClient,
		RunE: func(cmd *cobra.Command, args []string) error {
			ctx := cmd.Context()
			w := cmdctx.WorkspaceClient(ctx)

			resourceType := args[0]
			id := args[1]

			workspaceID, err := w.CurrentWorkspaceID(ctx)
			if err != nil {
				workspaceID = 0
			}

			resourceURL, err := buildWorkspaceURL(w.Config.Host, resourceType, id, workspaceID)
			if err != nil {
				return err
			}

			if !browser.IsDisabled(ctx) {
				cmdio.LogString(ctx, fmt.Sprintf("Opening %s %s in the browser...", resourceType, id))
			}

			return browser.OpenURL(ctx, ".", resourceURL)
		},
		ValidArgsFunction: func(cmd *cobra.Command, args []string, toComplete string) ([]string, cobra.ShellCompDirective) {
			if len(args) == 0 {
				return resourceTypeNames(), cobra.ShellCompDirectiveNoFileComp
			}
			return nil, cobra.ShellCompDirectiveNoFileComp
		},
	}

	return cmd
}
