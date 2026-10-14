package dockercredentials

import (
	"context"
	"fmt"
	"strings"

	"github.com/databricks/cli/libs/auth"
	"github.com/databricks/cli/libs/databrickscfg/profile"
	"github.com/databricks/cli/libs/sdkclients"
	"github.com/databricks/cli/libs/utils"
	"github.com/databricks/sdk-go/options/call"
	"github.com/databricks/sdk-go/uc/metastores/v1"
)

// WorkspaceID uses the selected profile's ID, resolving it only when absent.
// The client is only needed for profiles without an ID.
func WorkspaceID(ctx context.Context, p profile.Profile, w *sdkclients.Clients) (string, error) {
	if p.WorkspaceID != "" && p.WorkspaceID != profile.WorkspaceIDNone {
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
// MetastoreSummarizer is the part of the metastores client WorkspaceRegion
// needs; *metastores.Client implements it.
type MetastoreSummarizer interface {
	GetMetastoreSummary(ctx context.Context, req metastores.GetMetastoreSummaryRequest, opts ...call.Option) (*metastores.GetMetastoreSummaryResponse, error)
}

func WorkspaceRegion(ctx context.Context, p profile.Profile, api MetastoreSummarizer) (string, error) {
	summary, err := api.GetMetastoreSummary(ctx, metastores.GetMetastoreSummaryRequest{})
	if err != nil {
		return "", RewriteProfileError(ctx, p, fmt.Errorf("resolve workspace region for profile %q: %w", p.Name, err))
	}
	region := strings.TrimSpace(utils.Deref(summary.Region))
	if region == "" {
		return "", fmt.Errorf("resolve workspace region for profile %q: metastore summary did not include a region", p.Name)
	}
	return region, nil
}
