//go:build benchworkspace

// BenchmarkWalkers — minimal head-to-head comparison of /workspace/list-repo
// (recursive, server-side) vs a client-side parallel walker over the
// non-recursive /workspace/list endpoint.
//
// Always 100 files, distributed round-robin across the available leaves of
// each tree shape. Total file count is constant; only depth/breadth changes:
//
//	leaves=1   → 100 files in one dir         (flat)
//	leaves=4   → ~25 files per dir, depth 2
//	leaves=16  → ~6-7 files per dir, depth 4
//	leaves=64  → ~1-2 files per dir, depth 6
//
// The scaffold lives at a hardcoded persistent path:
//
//	/Users/$DATABRICKS_BENCH_USER/.tmp/sync-bench-walkers-fixture/
//
// First run populates it (~one-time cost). Subsequent runs reuse it, so
// repeated bench iterations are fast. To force a fresh scaffold, delete the
// fixture dir from the workspace.
//
// Run with 3 iterations per cell:
//
//	export DATABRICKS_BENCH_PROFILE=<profile>
//	export DATABRICKS_BENCH_USER=<email>
//	go test -tags benchworkspace -bench=BenchmarkWalkers$ -benchtime=3x -timeout=15m ./libs/sync/

package sync

import (
	"context"
	"fmt"
	"os"
	"path"
	"testing"

	"github.com/databricks/cli/libs/filer"
	"github.com/databricks/databricks-sdk-go"
	apiclient "github.com/databricks/databricks-sdk-go/client"
	"github.com/databricks/databricks-sdk-go/config"
)

const (
	walkersFilesPerCell = 100
	walkersFixtureName  = "sync-bench-walkers-fixture"
)

func BenchmarkWalkers(b *testing.B) {
	profile := os.Getenv("DATABRICKS_BENCH_PROFILE")
	user := os.Getenv("DATABRICKS_BENCH_USER")
	if profile == "" || user == "" {
		b.Skip("DATABRICKS_BENCH_PROFILE and DATABRICKS_BENCH_USER must be set")
	}
	wc, err := databricks.NewWorkspaceClient(&databricks.Config{Profile: profile})
	if err != nil {
		b.Fatalf("workspace client: %v", err)
	}
	apiC, err := apiclient.New(&config.Config{Profile: profile})
	if err != nil {
		b.Fatalf("api client: %v", err)
	}

	fixtureRoot := fmt.Sprintf("/Users/%s/.tmp/%s", user, walkersFixtureName)
	if err := wc.Workspace.MkdirsByPath(context.Background(), fixtureRoot); err != nil {
		b.Fatalf("mkdir fixture root: %v", err)
	}
	env := &benchEnv{wc: wc, apiClient: apiC, username: user, root: fixtureRoot}

	wfc, err := filer.NewWorkspaceFilesClient(wc, fixtureRoot)
	if err != nil {
		b.Fatalf("filer: %v", err)
	}
	lister := wfc.(*filer.WorkspaceFilesClient)

	cases := []struct {
		name  string
		shape treeShape
	}{
		{"leaves=1", treeShape{"flat", 0, 0}},
		{"leaves=4", treeShape{"small", 2, 2}},
		{"leaves=16", treeShape{"medium", 4, 2}},
		{"leaves=64", treeShape{"large", 6, 2}},
	}

	for _, c := range cases {
		dir := path.Join(fixtureRoot, c.name)
		items := walkersItemsRoundRobin(c.shape, walkersFilesPerCell)
		ensureWalkersFixture(b, env, lister, dir, items)

		b.Run(c.name+"/list-repo", func(b *testing.B) {
			for i := 0; i < b.N; i++ {
				if _, err := lister.ListWithSHAs(context.Background(), dir); err != nil {
					b.Fatalf("list-repo: %v", err)
				}
			}
		})
		b.Run(c.name+"/parallel-walk", func(b *testing.B) {
			for i := 0; i < b.N; i++ {
				if _, err := parallelWalk(context.Background(), apiC, dir, 8); err != nil {
					b.Fatalf("parallel-walk: %v", err)
				}
			}
		})
	}
}

// walkersItemsRoundRobin generates n file paths distributed round-robin
// across the leaves of the given shape (file 0 → leaf 0, file 1 → leaf 1, …,
// rolling over once every leaf has one file). Returns a (relPath → body)
// map suitable for passing to populate().
func walkersItemsRoundRobin(shape treeShape, n int) map[string][]byte {
	gen := generators()["file"]
	leaves := walkersLeafPaths(shape)
	out := make(map[string][]byte, n)
	for i := 0; i < n; i++ {
		leaf := leaves[i%len(leaves)]
		suf, body := gen(i)
		rel := path.Join(leaf, fmt.Sprintf("f-%d%s", i, suf))
		out[rel] = body
	}
	return out
}

// walkersLeafPaths returns every leaf-dir path for the given shape, in DFS
// order. For shape=flat returns [""], so files land directly under the cell
// root.
func walkersLeafPaths(shape treeShape) []string {
	if shape.depth == 0 {
		return []string{""}
	}
	var paths []string
	var walk func(prefix string, d int)
	walk = func(prefix string, d int) {
		if d == 0 {
			paths = append(paths, prefix)
			return
		}
		for b := 0; b < shape.branch; b++ {
			walk(path.Join(prefix, fmt.Sprintf("d%d", b)), d-1)
		}
	}
	walk("", shape.depth)
	return paths
}

// ensureWalkersFixture populates dir with items only if the existing fixture
// doesn't already match. Existence check is by file count under dir (using
// list-repo). Lets repeat bench runs reuse the scaffolding.
func ensureWalkersFixture(b *testing.B, env *benchEnv, lister *filer.WorkspaceFilesClient, dir string, items map[string][]byte) {
	existing, err := lister.ListWithSHAs(context.Background(), dir)
	if err == nil {
		fileCount := 0
		for _, o := range existing {
			if o.ContentSHA256Hex != "" {
				fileCount++
			}
		}
		if fileCount == len(items) {
			b.Logf("fixture %s reused (%d files)", dir, fileCount)
			return
		}
	}
	b.Logf("scaffolding fixture %s (%d files)", dir, len(items))
	populate(b, env, dir, items)
}
