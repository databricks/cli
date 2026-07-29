package acceptance_test

import (
	"context"
	"os"
	"os/exec"
	"path"
	"slices"
	"strconv"
	"strings"
	"testing"
	"time"

	"github.com/databricks/databricks-sdk-go"
	"github.com/databricks/databricks-sdk-go/service/iam"
	"github.com/databricks/databricks-sdk-go/service/workspace"
	"github.com/stretchr/testify/require"
)

// setupBundleCleanup arranges for every bundle deployed by this run to be
// destroyed once the suite finishes. It is a cloud-only operation: all cloud
// tests share one real workspace, whereas local tests each get a throwaway
// in-memory fake workspace with nothing to clean up.
//
// It ensures a per-run id is present (synthesizing one for non-CI cloud runs)
// so ciUniqueName stamps the "ci<runid>x" prefix onto every $UNIQUE_NAME. The
// cleanup then sweeps exactly the deployments carrying this run's prefix and
// nothing else, which is what makes destroying against the shared workspace
// safe even while other runs deploy concurrently.
func setupBundleCleanup(t *testing.T, execPath, cloudEnv string) {
	if cloudEnv == "" {
		return
	}

	runID := os.Getenv("GITHUB_RUN_ID")
	if runID == "" {
		// Non-CI cloud run (e.g. a developer using `deco env run`): mint a run id
		// so deployed bundles are identifiable and sweepable. Milliseconds stay
		// within ciUniqueName's 15-digit budget and are unique enough per run.
		runID = strconv.FormatInt(time.Now().UnixMilli(), 10)
		t.Setenv("GITHUB_RUN_ID", runID)
	}

	prefix := "ci" + runID + "x"
	// t.Context() is canceled once the test finishes, before cleanups run, so
	// derive a context that survives cancellation for the cleanup's API calls.
	ctx := context.WithoutCancel(t.Context())
	t.Cleanup(func() {
		cleanBundles(ctx, t, execPath, prefix)
	})
}

// cleanBundles finds every bundle this run deployed under the current user's
// ~/.bundle directory (identified by the run's prefix) and destroys each one,
// logging each deployment and the total time taken.
func cleanBundles(ctx context.Context, t *testing.T, execPath, prefix string) {
	start := time.Now()

	w, err := databricks.NewWorkspaceClient()
	require.NoError(t, err)

	me, err := w.CurrentUser.Me(ctx, iam.MeRequest{})
	require.NoError(t, err)

	bundleRoot := "/Workspace/Users/" + me.UserName + "/.bundle"

	// The run's prefix always appears in the first path segment under .bundle
	// (the bundle name, or the leaf of a workspace.root_path override), so match
	// there and only descend into this run's subtrees. This avoids walking the
	// thousands of directories other runs may have leaked under .bundle.
	var roots []string
	for _, child := range listChildDirs(ctx, w, bundleRoot) {
		if strings.Contains(path.Base(child), prefix) {
			roots = append(roots, findDeploymentRoots(ctx, w, child)...)
		}
	}
	slices.Sort(roots)

	t.Logf("%s bundle cleanup: found %d deployment(s) with prefix %q", time.Now().Format(time.RFC3339), len(roots), prefix)

	// Best-effort: attempt every deployment and report failures at the end, so
	// one stuck bundle doesn't leave the rest leaked.
	var failed []string
	for _, root := range roots {
		t.Logf("%s destroying %s", time.Now().Format(time.RFC3339), root)
		if out, err := destroyBundle(execPath, root); err != nil {
			t.Logf("%s destroy failed: %s\n%s", time.Now().Format(time.RFC3339), root, out)
			failed = append(failed, root)
		}
	}

	t.Logf("%s bundle cleanup: destroyed %d/%d deployment(s) in %s", time.Now().Format(time.RFC3339), len(roots)-len(failed), len(roots), time.Since(start))
	require.Empty(t, failed, "failed to destroy %d deployment(s): %s", len(failed), strings.Join(failed, ", "))
}

// findDeploymentRoots walks the workspace tree under dir and returns the paths
// of bundle deployment roots. A directory is a deployment root when it contains
// a "state" or "files" child, which the bundle deploy writes beneath the
// resolved workspace.root_path. This works regardless of whether the root is
// the default ~/.bundle/<name>/<target> or a custom ~/.bundle/<...> override.
func findDeploymentRoots(ctx context.Context, w *databricks.WorkspaceClient, dir string) []string {
	var childDirs []string
	for _, child := range listChildDirs(ctx, w, dir) {
		if base := path.Base(child); base == "state" || base == "files" {
			// dir is a deployment root; don't descend into its internals.
			return []string{dir}
		}
		childDirs = append(childDirs, child)
	}

	var roots []string
	for _, child := range childDirs {
		roots = append(roots, findDeploymentRoots(ctx, w, child)...)
	}
	return roots
}

// listChildDirs returns the immediate subdirectory paths of dir, or nil if dir
// does not exist (nothing was deployed under it).
func listChildDirs(ctx context.Context, w *databricks.WorkspaceClient, dir string) []string {
	objects, err := w.Workspace.ListAll(ctx, workspace.ListWorkspaceRequest{Path: dir})
	if err != nil {
		return nil
	}
	var dirs []string
	for _, o := range objects {
		if o.ObjectType == workspace.ObjectTypeDirectory {
			dirs = append(dirs, o.Path)
		}
	}
	return dirs
}

// destroyBundle destroys the bundle deployed at rootPath using the CLI binary,
// returning the combined output and any error. It writes a throwaway
// databricks.yml pinning workspace.root_path to the deployment root; destroy
// pulls the remote state from there (auto-detecting the engine) and deletes the
// resources and files. The bundle name and target are placeholders because
// root_path fully determines the deployment location. --force-lock overrides a
// stale deployment lock left by a test that was killed mid-deploy; these are
// known-leaked bundles, so there is no concurrent deployment to conflict with.
func destroyBundle(execPath, rootPath string) ([]byte, error) {
	dir, err := os.MkdirTemp("", "bundle-clean") //nolint:usetesting // runs in a cleanup, where t.TempDir is already removed
	if err != nil {
		return nil, err
	}
	defer os.RemoveAll(dir)

	databricksYML := "bundle:\n  name: cleanup\nworkspace:\n  root_path: " + rootPath + "\ntargets:\n  default: {}\n"
	if err := os.WriteFile(path.Join(dir, "databricks.yml"), []byte(databricksYML), 0o644); err != nil {
		return nil, err
	}

	cmd := exec.Command(execPath, "bundle", "destroy", "--target", "default", "--auto-approve", "--force-lock")
	cmd.Dir = dir
	cmd.Env = os.Environ()

	return cmd.CombinedOutput()
}
