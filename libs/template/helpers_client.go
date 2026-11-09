package template

import (
	"context"
	"fmt"

	"github.com/databricks/cli/libs/cmdctx"
	"github.com/databricks/cli/libs/databrickscfg"
	"github.com/databricks/cli/libs/databrickscfg/profile"
	"github.com/databricks/cli/libs/env"
	"github.com/databricks/cli/libs/liteswap"
	"github.com/databricks/databricks-sdk-go"
	sdkconfig "github.com/databricks/databricks-sdk-go/config"
)

// helpersProfileEnvVar names a profile for the template helpers that read the workspace
// configuration or call the workspace, such as workspace_host and user_name. The selected
// profile still provides .workspace and is used for all other requests, including writing the
// generated files.
//
// DABs in the Workspace selects the profile for the host of the page, which can be a host shared
// by several workspaces, and sets this to a profile for the workspace's own host. Templates that
// use workspace_host keep generating the workspace's own host, as before .workspace existed.
const helpersProfileEnvVar = "DATABRICKS_BUNDLE_INIT_HELPERS_PROFILE"

type helpersClientKey struct{}

// ConfigureHelpersClient returns a context in which the template helpers use the profile named by
// DATABRICKS_BUNDLE_INIT_HELPERS_PROFILE. It returns ctx unchanged if the variable is not set. It
// fails if that profile has a different workspace ID than the selected profile, so that one
// template doesn't mix two workspaces.
func ConfigureHelpersClient(ctx context.Context) (context.Context, error) {
	name := env.Get(ctx, helpersProfileEnvVar)
	if name == "" {
		return ctx, nil
	}

	// Load the profile the same way as --profile, so that an auth env var such as
	// DATABRICKS_TOKEN fills in only what the profile doesn't set.
	cfg := &sdkconfig.Config{Profile: name, Loaders: databrickscfg.ProfileAuthLoaders}
	liteswap.Apply(ctx, cfg)
	w, err := databricks.NewWorkspaceClient((*databricks.Config)(cfg))
	if err != nil {
		return nil, fmt.Errorf("failed to load profile %q from %s: %w", name, helpersProfileEnvVar, err)
	}

	selectedID := knownWorkspaceID(cmdctx.WorkspaceClient(ctx).Config)
	helpersID := knownWorkspaceID(w.Config)
	if selectedID != "" && helpersID != "" && selectedID != helpersID {
		return nil, fmt.Errorf("profile %q from %s has workspace ID %s, but the selected profile has workspace ID %s", name, helpersProfileEnvVar, helpersID, selectedID)
	}
	return context.WithValue(ctx, helpersClientKey{}, w), nil
}

func knownWorkspaceID(cfg *sdkconfig.Config) string {
	if cfg.WorkspaceID == profile.WorkspaceIDNone {
		return ""
	}
	return cfg.WorkspaceID
}

// helpersClient returns the client for the template helpers, see ConfigureHelpersClient.
func helpersClient(ctx context.Context) *databricks.WorkspaceClient {
	if w, ok := ctx.Value(helpersClientKey{}).(*databricks.WorkspaceClient); ok {
		return w
	}
	return cmdctx.WorkspaceClient(ctx)
}
