package aircmd

import (
	"context"
	"crypto/sha256"
	"encoding/hex"
	"errors"
	"fmt"
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
	"golang.org/x/sync/errgroup"
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
	email, err := currentUserEmail(ctx, w)
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

// launchIndexComponent names the local cache that maps a staged file set to the
// launch dir that holds it.
const (
	launchIndexComponent = "air-launch-dirs"
	launchIndexTTL       = 30 * 24 * time.Hour
)

// launchIndexKey identifies a staged file set: the workspace, the user's home it
// is staged under, and a hash of the files' names and bytes.
type launchIndexKey struct {
	Host    string
	Base    string
	Content string
}

func newLaunchIndex(ctx context.Context) *cache.Cache {
	return cache.NewCache(ctx, launchIndexComponent, launchIndexTTL, nil)
}

// launchContentHash hashes each file's name and length before its bytes, so
// different file splits cannot produce the same input.
func launchContentHash(items []uploadItem) string {
	h := sha256.New()
	for _, item := range items {
		fmt.Fprintf(h, "%s\x00%d\x00", item.name, len(item.data))
		h.Write(item.data)
	}
	return hex.EncodeToString(h.Sum(nil))
}

// launchFilesStaged reports whether every launch file is already in dir with its
// local size; a workspace edit that keeps the size is not detected. Any missing
// file or failed check is a miss, so the caller uploads again.
func launchFilesStaged(ctx context.Context, w *databricks.WorkspaceClient, dir string, items []uploadItem) bool {
	group, groupCtx := errgroup.WithContext(ctx)
	for _, item := range items {
		group.Go(func() error {
			info, err := w.Workspace.GetStatusByPath(groupCtx, path.Join(dir, item.name))
			if err != nil {
				return err
			}
			if info.Size != int64(len(item.data)) {
				return fmt.Errorf("%s is %d bytes in the workspace, %d locally", item.name, info.Size, len(item.data))
			}
			return nil
		})
	}
	return group.Wait() == nil
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
