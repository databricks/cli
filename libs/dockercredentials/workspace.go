package dockercredentials

import (
	"context"
	"fmt"
	"strings"

	"github.com/databricks/cli/libs/auth"
	"github.com/databricks/cli/libs/databrickscfg/profile"
	"github.com/databricks/databricks-sdk-go"
	"github.com/databricks/databricks-sdk-go/service/catalog"
)

// WorkspaceID uses the selected profile's ID, resolving it only when absent.
// The client is only needed for profiles without an ID.
func WorkspaceID(ctx context.Context, p profile.Profile, w *databricks.WorkspaceClient) (string, error) {
	if p.WorkspaceID != "" && p.WorkspaceID != auth.WorkspaceIDNone {
		return p.WorkspaceID, nil
	}
	// Neither an ambient ID nor the CLI-only "none" sentinel may become a routing header.
	w.Config.WorkspaceID = ""
	id, err := auth.ResolveWorkspaceID(ctx, w)
	if err != nil {
		err = fmt.Errorf("resolve workspace ID for profile %q: %w. Run databricks auth login --host <workspace-url> and retry with that profile", p.Name, err)
		return "", RewriteProfileError(ctx, p, err)
	}
	return id, nil
}

// WorkspaceRegion obtains and validates the region returned by the workspace's metastore.
func WorkspaceRegion(ctx context.Context, p profile.Profile, metastores catalog.MetastoresInterface) (string, error) {
	summary, err := metastores.Summary(ctx)
	if err != nil {
		return "", RewriteProfileError(ctx, p, fmt.Errorf("resolve workspace region for profile %q: %w", p.Name, err))
	}
	region := strings.TrimSpace(summary.Region)
	if region == "" {
		return "", fmt.Errorf("resolve workspace region for profile %q: metastore summary did not include a region", p.Name)
	}
	return region, nil
}
