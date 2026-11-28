package aircmd

import (
	"context"
	"errors"
	"fmt"
	"net/http"
	"path"
	"strings"
	"time"
	"uuid"

	"github.com/databricks/cli/libs/cache"
	"github.com/databricks/cli/libs/env"
	"github.com/databricks/databricks-sdk-go"
	"github.com/databricks/databricks-sdk-go/apierr"
	"github.com/databricks/databricks-sdk-go/service/iam"
	"github.com/databricks/databricks-sdk-go/service/workspace"
)

// userWorkspaceDirEnv overrides the per-user workspace directory; mirrors the
// Python CLI's DATABRICKS_INTERNAL_USER_WORKSPACE_DIR escape hatch.
const userWorkspaceDirEnv = "DATABRICKS_INTERNAL_USER_WORKSPACE_DIR"

// currentUserEmail returns the authenticated user's email (works for any domain).
func currentUserEmail(ctx context.Context, w *databricks.WorkspaceClient) (string, error) {
	// Only userName is needed; the full response also resolves groups and entitlements.
	me, err := w.CurrentUser.Me(ctx, iam.MeRequest{Attributes: "userName"})
	if err != nil {
		return "", fmt.Errorf("failed to resolve current user: %w", err)
	}
	return me.UserName, nil
}

// userNameCacheComponent segments the cached user name from the bundle's "user"
// component, which stores the full user. Sharing that entry would hand a bundle a
// userName-only record.
const userNameCacheComponent = "air-user-name"

// userNameCacheTTL matches the bundle's user cache.
const userNameCacheTTL = 30 * time.Minute

// userNameCacheKey is keyed like bundle.UserFingerprint: the host and the
// Authorization header the credential produces, so an entry only serves the
// credential that stored it. The cache stores a hash of the key, never the header.
type userNameCacheKey struct {
	Host       string `json:"host"`
	AuthHeader string `json:"auth_header"`
}

// cachedCurrentUserEmail is currentUserEmail for launch paths. A hit skips the
// SCIM round trip that the launch path otherwise waits on before any upload or submit.
func cachedCurrentUserEmail(ctx context.Context, w *databricks.WorkspaceClient) (string, error) {
	req := &http.Request{Header: http.Header{}}
	if err := w.Config.Authenticate(req); err != nil || req.Header.Get("Authorization") == "" {
		return currentUserEmail(ctx, w)
	}
	key := userNameCacheKey{Host: w.Config.Host, AuthHeader: req.Header.Get("Authorization")}
	c := cache.NewCache(ctx, userNameCacheComponent, userNameCacheTTL, nil)
	return cache.GetOrCompute(ctx, c, key, func(ctx context.Context) (string, error) {
		return currentUserEmail(ctx, w)
	})
}

// userWorkspaceDir returns the user's workspace home, honoring the env override.
func userWorkspaceDir(ctx context.Context, w *databricks.WorkspaceClient) (string, error) {
	if override := env.Get(ctx, userWorkspaceDirEnv); override != "" {
		return override, nil
	}
	override, err := profileValue(w.Config, "databricks_internal_user_workspace_dir")
	if err != nil {
		return "", fmt.Errorf("failed to read selected profile: %w", err)
	}
	if override != "" {
		return override, nil
	}
	email, err := cachedCurrentUserEmail(ctx, w)
	if err != nil {
		return "", err
	}
	return "/Workspace/Users/" + email, nil
}

// cliLaunchDir returns a unique workspace directory for a run's launch artifacts:
// <base>/.air/cli_launch/<experiment>/<run>_<uuid>. run defaults to experiment.
func cliLaunchDir(base, experiment, run string) string {
	if run == "" {
		run = experiment
	}
	unique := strings.ReplaceAll(uuid.New().String(), "-", "")[:16]
	return path.Join(base, ".air", "cli_launch", experiment, run+"_"+unique)
}

// prospectiveLaunchPaths returns the workspace paths a submission would use
// without creating directories or uploading artifacts.
func prospectiveLaunchPaths(ctx context.Context, w *databricks.WorkspaceClient, cfg *runConfig) (string, string, string, error) {
	base, err := userWorkspaceDir(ctx, w)
	if err != nil {
		return "", "", "", err
	}
	runName := ""
	if cfg.MLflowRunName != nil {
		runName = *cfg.MLflowRunName
	}
	launchDir := cliLaunchDir(base, cfg.ExperimentName, runName)
	return base, launchDir, path.Join(launchDir, commandScriptName), nil
}

// ensureExperimentDirectory creates experimentDir if it is missing, matching the
// CLI's convention for its other artifact directories. Without this, a missing
// parent surfaces only as a server-side INTERNAL_ERROR after the run is wasted.
// An empty dir means the default (/Users/<user>/...), which always exists.
func ensureExperimentDirectory(ctx context.Context, w *databricks.WorkspaceClient, experimentDir string) error {
	if experimentDir == "" {
		return nil
	}

	info, err := w.Workspace.GetStatusByPath(ctx, experimentDir)
	if errors.Is(err, apierr.ErrNotFound) {
		return w.Workspace.MkdirsByPath(ctx, experimentDir)
	}
	if err != nil {
		return fmt.Errorf("failed to check experiment_directory %q: %w", experimentDir, err)
	}
	if info.ObjectType != workspace.ObjectTypeDirectory {
		return fmt.Errorf("experiment_directory %q is not a directory (object_type=%s)", experimentDir, info.ObjectType)
	}
	return nil
}
