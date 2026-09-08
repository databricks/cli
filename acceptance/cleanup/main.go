// Package main implements a standalone bundle cleanup program. It is invoked as a
// separate always()-triggered workflow job (see cli-isolated-tests.yml in
// databricks-eng/eng-dev-ecosystem) so it runs even when a test job times out or
// is cancelled — unlike a t.Cleanup, which go test skips in those cases.
package main

import (
	"context"
	"errors"
	"flag"
	"fmt"
	"log"
	"os"
	"os/exec"
	"path"
	"path/filepath"
	"regexp"
	"slices"
	"strings"
	"sync"
	"time"

	"github.com/databricks/databricks-sdk-go"
	"github.com/databricks/databricks-sdk-go/apierr"
	"github.com/databricks/databricks-sdk-go/service/iam"
	"github.com/databricks/databricks-sdk-go/service/workspace"
)

// ciRunID matches a valid GITHUB_RUN_ID (same cap as acceptance_test.go).
var ciRunID = regexp.MustCompile(`^[0-9]{1,11}$`)

func main() {
	var cliPath string
	flag.StringVar(&cliPath, "cli", "", "path to databricks CLI binary (required)")
	flag.Parse()

	if cliPath == "" {
		log.Fatal("-cli: path to databricks CLI binary is required")
	}

	runID := os.Getenv("GITHUB_RUN_ID")
	if !ciRunID.MatchString(runID) {
		log.Fatalf("GITHUB_RUN_ID %q is not a valid run id (must be 1-11 digits)", runID)
	}
	prefix := "ci" + runID + "x"

	if err := cleanBundles(context.Background(), cliPath, prefix); err != nil {
		log.Fatal(err)
	}
}

// cleanBundles finds every bundle this run deployed under the current user's
// ~/.bundle directory (identified by the run's prefix) and destroys each one.
func cleanBundles(ctx context.Context, execPath, prefix string) error {
	start := time.Now()

	w, err := databricks.NewWorkspaceClient()
	if err != nil {
		return fmt.Errorf("cannot create workspace client: %w", err)
	}

	me, err := w.CurrentUser.Me(ctx, iam.MeRequest{})
	if err != nil {
		return fmt.Errorf("cannot resolve current user: %w", err)
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
		for _, child := range listChildDirs(ctx, w, bundleRoot) {
			if strings.Contains(path.Base(child), prefix) {
				roots = append(roots, findDeploymentRoots(ctx, w, child)...)
			}
		}
	}
	slices.Sort(roots)

	log.Printf("bundle cleanup: found %d deployment(s) with prefix %q", len(roots), prefix)

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
			log.Printf("destroying %s", root)
			if out, err := destroyBundle(execPath, root); err != nil {
				log.Printf("destroy failed: %s\n%s", root, out)
				mu.Lock()
				failed = append(failed, root)
				mu.Unlock()
			}
		})
	}
	wg.Wait()

	slices.Sort(failed)
	log.Printf("bundle cleanup: destroyed %d/%d deployment(s) in %s", len(roots)-len(failed), len(roots), time.Since(start))
	if len(failed) > 0 {
		return fmt.Errorf("failed to destroy %d deployment(s): %s", len(failed), strings.Join(failed, ", "))
	}
	return nil
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

// listChildDirs returns the immediate subdirectory paths of dir. A missing dir
// (nothing was deployed under it) yields nil silently; any other listing error
// is logged loudly but does not stop the overall sweep.
func listChildDirs(ctx context.Context, w *databricks.WorkspaceClient, dir string) []string {
	objects, err := w.Workspace.ListAll(ctx, workspace.ListWorkspaceRequest{Path: dir})
	if err != nil {
		if !errors.Is(err, apierr.ErrNotFound) {
			log.Printf("WARNING: bundle cleanup incomplete, cannot list %s: %s", dir, err)
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
	dir, err := os.MkdirTemp("", "bundle-clean")
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
