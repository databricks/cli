package acceptance_test

import (
	"context"
	"errors"
	"os"
	"os/exec"
	"path"
	"path/filepath"
	"runtime"
	"slices"
	"strings"
	"sync"
	"testing"
	"time"

	"github.com/databricks/databricks-sdk-go"
	"github.com/databricks/databricks-sdk-go/apierr"
	"github.com/databricks/databricks-sdk-go/service/iam"
	"github.com/databricks/databricks-sdk-go/service/workspace"
	"github.com/stretchr/testify/require"
)

// TestCleanupLeakedBundles destroys every bundle a cloud run deployed. It is the
// run's cleanup step: unlike a t.Cleanup — which go test skips when the suite
// panics on -timeout or the CI job is killed at its time limit — this runs as a
// separate, always()-invoked job after every matrix leg finishes, so leaked
// deployments are reclaimed even when a test job times out.
//
// It sweeps the run-wide "ci<GITHUB_RUN_ID>x" prefix that ciUniqueName stamps into
// every $UNIQUE_NAME, so it matches all legs of the run. That is safe only because
// it runs once every leg has finished and no bundle is still live. It is gated
// behind CLEANUP_LEAKED_BUNDLES so it never runs inline in a normal suite, where a
// run-wide sweep would destroy sibling legs' live deployments.
func TestCleanupLeakedBundles(t *testing.T) {
	if os.Getenv("CLEANUP_LEAKED_BUNDLES") == "" {
		t.Skip("set CLEANUP_LEAKED_BUNDLES=1 to run the post-run bundle sweep")
	}
	if os.Getenv("CLOUD_ENV") == "" {
		t.Skip("bundle cleanup only applies to cloud runs")
	}

	prefix := ciRunPrefix()
	require.NotEmpty(t, prefix, "GITHUB_RUN_ID must be a valid run id so the sweep knows which bundles to destroy")

	cwd, err := os.Getwd()
	require.NoError(t, err)

	// Reuse a prebuilt CLI when given (-clipath), else build from the current source.
	execPath := CLIPath
	if execPath == "" {
		buildDir := getBuildDir(t, cwd, runtime.GOOS, runtime.GOARCH)
		execPath = BuildCLI(t, buildDir, "", runtime.GOOS, runtime.GOARCH)
	}

	cleanBundles(t.Context(), t, execPath, prefix)
}

// cleanBundles finds every bundle this run deployed under the current user's
// ~/.bundle directory (identified by the run's prefix) and destroys each one,
// logging each deployment and the total time taken.
func cleanBundles(ctx context.Context, t *testing.T, execPath, prefix string) {
	start := time.Now()

	// Cleanup never fails the test (see the WARNING note below), so on any error
	// that prevents sweeping, log loudly and return rather than require-failing.
	w, err := databricks.NewWorkspaceClient()
	if err != nil {
		t.Logf("WARNING: bundle cleanup skipped, cannot create client: %s", err)
		return
	}

	me, err := w.CurrentUser.Me(ctx, iam.MeRequest{})
	if err != nil {
		t.Logf("WARNING: bundle cleanup skipped, cannot resolve current user: %s", err)
		return
	}

	// Tests deploy under the user's home .bundle by default, but some set
	// workspace.root_path under /Shared (e.g. resources/jobs/shared-root-path),
	// so sweep both. A "/Shared/..." root_path is normalized to "/Workspace/Shared/..."
	// by prependWorkspacePrefix, so the swept path uses that form.
	bundleRoots := []string{
		"/Workspace/Users/" + me.UserName + "/.bundle",
		"/Workspace/Shared/" + me.UserName + "/.bundle",
	}

	// The run's prefix always appears in the first path segment under .bundle
	// (the bundle name, or the leaf of a workspace.root_path override), so match
	// there and only descend into this run's subtrees. This avoids walking the
	// thousands of directories other runs may have leaked under .bundle.
	var roots []string
	for _, bundleRoot := range bundleRoots {
		for _, child := range listChildDirs(ctx, t, w, bundleRoot) {
			if strings.Contains(path.Base(child), prefix) {
				roots = append(roots, findDeploymentRoots(ctx, t, w, child)...)
			}
		}
	}
	slices.Sort(roots)

	t.Logf("%s bundle cleanup: found %d deployment(s) with prefix %q", time.Now().Format(time.RFC3339), len(roots), prefix)

	// Each destroy shells out to a separate `bundle destroy` (auth + state pull +
	// deletes), so run them concurrently. Each is network-bound (not CPU-bound),
	// so cap at a fixed 20 rather than by GOMAXPROCS: it parallelizes the API
	// waits while staying well under the workspace rate limit. This is
	// best-effort, not fail-fast: a failed destroy is recorded and the rest still
	// run, so a plain WaitGroup with a semaphore fits better than errgroup (whose
	// error short-circuit we would not use).
	const maxConcurrentDestroys = 20
	sem := make(chan struct{}, maxConcurrentDestroys)
	var wg sync.WaitGroup
	var mu sync.Mutex
	var failed []string
	for _, root := range roots {
		sem <- struct{}{}
		wg.Go(func() {
			defer func() { <-sem }()
			t.Logf("%s destroying %s", time.Now().Format(time.RFC3339), root)
			if out, err := destroyBundle(execPath, root); err != nil {
				t.Logf("%s destroy failed: %s\n%s", time.Now().Format(time.RFC3339), root, out)
				mu.Lock()
				failed = append(failed, root)
				mu.Unlock()
			}
		})
	}
	wg.Wait()

	slices.Sort(failed)
	t.Logf("%s bundle cleanup: destroyed %d/%d deployment(s) in %s", time.Now().Format(time.RFC3339), len(roots)-len(failed), len(roots), time.Since(start))

	// Do not fail on a cleanup failure: this is best-effort housekeeping that runs
	// after the product tests, so a transient destroy failure should not turn the
	// cleanup step red and mask the real test signal. Log loudly instead; leaked
	// deployments are reclaimed by the periodic prefix sweep (sweep_test_resources.py).
	if len(failed) > 0 {
		t.Logf("WARNING: bundle cleanup failed to destroy %d deployment(s), leaked until swept: %s", len(failed), strings.Join(failed, ", "))
	}
}

// findDeploymentRoots walks the workspace tree under dir and returns the paths
// of bundle deployment roots. A directory is a deployment root when it contains
// a "state" or "files" child, which the bundle deploy writes beneath the
// resolved workspace.root_path. This works regardless of whether the root is
// the default ~/.bundle/<name>/<target> or a custom ~/.bundle/<...> override.
func findDeploymentRoots(ctx context.Context, t *testing.T, w *databricks.WorkspaceClient, dir string) []string {
	var childDirs []string
	for _, child := range listChildDirs(ctx, t, w, dir) {
		if base := path.Base(child); base == "state" || base == "files" {
			// dir is a deployment root; don't descend into its internals.
			return []string{dir}
		}
		childDirs = append(childDirs, child)
	}

	var roots []string
	for _, child := range childDirs {
		roots = append(roots, findDeploymentRoots(ctx, t, w, child)...)
	}
	return roots
}

// listChildDirs returns the immediate subdirectory paths of dir. A missing dir
// (nothing was deployed under it) yields nil silently; any other listing error
// is logged loudly (it means the sweep under dir is incomplete) but does not
// fail the test, since cleanup runs in the root t.Cleanup.
func listChildDirs(ctx context.Context, t *testing.T, w *databricks.WorkspaceClient, dir string) []string {
	objects, err := w.Workspace.ListAll(ctx, workspace.ListWorkspaceRequest{Path: dir})
	if err != nil {
		if !errors.Is(err, apierr.ErrNotFound) {
			t.Logf("WARNING: bundle cleanup incomplete, cannot list %s: %s", dir, err)
		}
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
	if err := os.WriteFile(filepath.Join(dir, "databricks.yml"), []byte(databricksYML), 0o644); err != nil {
		return nil, err
	}

	cmd := exec.Command(execPath, "bundle", "destroy", "--target", "default", "--auto-approve", "--force-lock")
	cmd.Dir = dir
	cmd.Env = os.Environ()

	return cmd.CombinedOutput()
}
