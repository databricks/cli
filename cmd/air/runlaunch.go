package aircmd

import (
	"context"
	"encoding/base64"
	"encoding/json"
	"errors"
	"fmt"
	"path"
	"strings"
	"uuid"

	"github.com/databricks/cli/libs/env"
	"github.com/databricks/databricks-sdk-go"
	"github.com/databricks/databricks-sdk-go/apierr"
	"github.com/databricks/databricks-sdk-go/config"
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

// userNameFromWorkspaceToken returns the sub claim of cfg's OAuth access token when this
// workspace issued it, or "" otherwise. It is only a guess: sub need not equal the SCIM userName.
func userNameFromWorkspaceToken(ctx context.Context, cfg *config.Config) string {
	token, err := cfg.GetTokenSource().Token(ctx)
	if err != nil {
		return ""
	}
	parts := strings.Split(token.AccessToken, ".")
	if len(parts) != 3 {
		return ""
	}
	payload, err := base64.RawURLEncoding.DecodeString(parts[1])
	if err != nil {
		return ""
	}
	var claims struct {
		Issuer  string `json:"iss"`
		Subject string `json:"sub"`
	}
	if json.Unmarshal(payload, &claims) != nil || claims.Issuer != cfg.Host+"/oidc" {
		return ""
	}
	return claims.Subject
}

// userWorkspaceDirOverride returns the per-user workspace directory set by env or profile, if any.
func userWorkspaceDirOverride(ctx context.Context, w *databricks.WorkspaceClient) (string, error) {
	if override := env.Get(ctx, userWorkspaceDirEnv); override != "" {
		return override, nil
	}
	override, err := profileValue(w.Config, "databricks_internal_user_workspace_dir")
	if err != nil {
		return "", fmt.Errorf("failed to read selected profile: %w", err)
	}
	return override, nil
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

// prospectiveLaunchPaths returns the workspace paths a submission would use without creating
// directories or uploading artifacts, after running validate on them.
//
// The user's directory comes from SCIM Me unless overridden. The access token usually names the
// user already, so validate first runs on that guess while SCIM Me is in flight, and again on
// SCIM's answer if it differs.
func prospectiveLaunchPaths(ctx context.Context, w *databricks.WorkspaceClient, cfg *runConfig, validate func(launchDir, commandPath string) error) (string, string, string, error) {
	runName := ""
	if cfg.MLflowRunName != nil {
		runName = *cfg.MLflowRunName
	}
	validated := func(base string) (string, string, string, error) {
		launchDir := cliLaunchDir(base, cfg.ExperimentName, runName)
		commandPath := path.Join(launchDir, commandScriptName)
		return base, launchDir, commandPath, validate(launchDir, commandPath)
	}

	override, err := userWorkspaceDirOverride(ctx, w)
	if err != nil {
		return "", "", "", err
	}
	if override != "" {
		return validated(override)
	}

	guess := userNameFromWorkspaceToken(ctx, w.Config)
	var email string
	var g errgroup.Group
	g.Go(func() (err error) {
		email, err = currentUserEmail(ctx, w)
		return err
	})
	var base, launchDir, commandPath string
	var guessErr error
	if guess != "" {
		base, launchDir, commandPath, guessErr = validated("/Workspace/Users/" + guess)
	}
	if err := g.Wait(); err != nil {
		return "", "", "", err
	}
	if guess == "" || email != guess {
		return validated("/Workspace/Users/" + email)
	}
	return base, launchDir, commandPath, guessErr
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
