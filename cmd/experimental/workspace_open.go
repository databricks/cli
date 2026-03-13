package experimental

import (
	"context"
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

// resourceURLPatterns is a hardcoded list of known resource types and their URL path patterns.
// Patterns starting with "#" use URL fragments instead of path segments.
// bundle open uses a similar hardcoded approach via InitializeURL methods on each resource type.
var resourceURLPatterns = map[string]string{
	"apps":        "/apps/%s",
	"clusters":    "/compute/clusters/%s",
	"dashboards":  "/dashboardsv3/%s/published",
	"experiments": "ml/experiments/%s",
	"jobs":        "/jobs/%s",
	"notebooks":   "#notebook/%s",
	"pipelines":   "/pipelines/%s",
	"queries":     "/sql/editor/%s",
	"warehouses":  "/sql/warehouses/%s",
}

var currentWorkspaceID = func(ctx context.Context) (int64, error) {
	return cmdctx.WorkspaceClient(ctx).CurrentWorkspaceID(ctx)
}

var openWorkspaceURL = func(ctx context.Context, targetURL string) error {
	return browser.OpenURL(ctx, ".", targetURL)
}

func resourceTypeNames() []string {
	names := make([]string, 0, len(resourceURLPatterns))
	for name := range resourceURLPatterns {
		names = append(names, name)
	}
	sort.Strings(names)
	return names
}

func supportedResourceTypes() string {
	return strings.Join(resourceTypeNames(), ", ")
}

// buildWorkspaceURL constructs a full workspace URL for a given resource type and ID.
func buildWorkspaceURL(host, resourceType, id string, workspaceID int64) (string, error) {
	pattern, ok := resourceURLPatterns[resourceType]
	if !ok {
		return "", fmt.Errorf("unknown resource type %q, must be one of: %s", resourceType, supportedResourceTypes())
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
