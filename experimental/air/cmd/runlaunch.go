package aircmd

import (
	"context"
	"errors"
	"fmt"
	"path"
	"strings"

	"github.com/databricks/cli/libs/env"
	"github.com/databricks/databricks-sdk-go"
	"github.com/databricks/databricks-sdk-go/apierr"
	"github.com/databricks/databricks-sdk-go/service/iam"
	"github.com/databricks/databricks-sdk-go/service/workspace"
	"github.com/google/uuid"
)

// userWorkspaceDirEnv overrides the per-user workspace directory; mirrors the
// Python CLI's DATABRICKS_INTERNAL_USER_WORKSPACE_DIR escape hatch.
const userWorkspaceDirEnv = "DATABRICKS_INTERNAL_USER_WORKSPACE_DIR"

// currentUserEmail returns the authenticated user's email (works for any domain).
func currentUserEmail(ctx context.Context, w *databricks.WorkspaceClient) (string, error) {
	me, err := w.CurrentUser.Me(ctx, iam.MeRequest{})
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
	unique := strings.ReplaceAll(uuid.NewString(), "-", "")[:16]
	return path.Join(base, ".air", "cli_launch", experiment, run+"_"+unique)
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
