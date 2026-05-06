//go:build benchworkspace

// BenchmarkWalkers — minimal head-to-head comparison of /workspace/list-repo
// (recursive, server-side) vs a client-side parallel walker over the
// non-recursive /workspace/list endpoint.
//
// One cell per leaf-count: 1, 4, 16, 64. Full occupancy — exactly one file
// per leaf directory, so total files equals leaf count and every leaf is
// populated. Two sub-benches per cell: list-repo vs parallel-walk (8 workers).
//
// Run with 3 iterations per cell:
//
//	export DATABRICKS_BENCH_PROFILE=<profile>
//	export DATABRICKS_BENCH_USER=<email>
//	go test -tags benchworkspace -bench=BenchmarkWalkers$ -benchtime=3x -timeout=15m ./libs/sync/

package sync

import (
	"context"
	"path"
	"testing"

	"github.com/databricks/cli/libs/filer"
)

func BenchmarkWalkers(b *testing.B) {
	env := newBenchEnv(b)
	wfc, err := filer.NewWorkspaceFilesClient(env.wc, env.root)
	if err != nil {
		b.Fatalf("filer: %v", err)
	}
	lister := wfc.(*filer.WorkspaceFilesClient)

	// Shapes are sized so every leaf dir gets exactly one file at full
	// occupancy (files = leaves).
	cases := []struct {
		name   string
		shape  treeShape
		leaves int
	}{
		{"leaves=1", treeShape{"flat", 0, 0}, 1},
		{"leaves=4", treeShape{"small", 2, 2}, 4},
		{"leaves=16", treeShape{"medium", 4, 2}, 16},
		{"leaves=64", treeShape{"large", 6, 2}, 64},
	}

	for _, c := range cases {
		dir := path.Join(env.root, "walkers-"+c.name)
		populate(b, env, dir, itemsForCell(c.shape, c.leaves, "file"))
		_, _ = lister.ListWithSHAs(context.Background(), dir)

		b.Run(c.name+"/list-repo", func(b *testing.B) {
			for i := 0; i < b.N; i++ {
				if _, err := lister.ListWithSHAs(context.Background(), dir); err != nil {
					b.Fatalf("list-repo: %v", err)
				}
			}
		})
		b.Run(c.name+"/parallel-walk", func(b *testing.B) {
			for i := 0; i < b.N; i++ {
				if _, err := parallelWalk(context.Background(), env.apiClient, dir, 8); err != nil {
					b.Fatalf("parallel-walk: %v", err)
				}
			}
		})
	}
}
