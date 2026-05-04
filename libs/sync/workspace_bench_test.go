//go:build benchworkspace

// Workspace benchmarks for libs/sync. Real-network — they hit a Databricks
// workspace and import / list / delete files there.
//
// Quickstart:
//
//	export DATABRICKS_BENCH_PROFILE=my-profile     # required; profile from .databrickscfg
//	export DATABRICKS_BENCH_USER=me@example.com    # required; remote dirs go under /Users/$DATABRICKS_BENCH_USER/.tmp/
//
//	go test -tags benchworkspace -bench=. -benchtime=5x -timeout=90m ./libs/sync/...
//
// Run a single group:
//
//	go test -tags benchworkspace -bench=BenchmarkListRepo$               -benchtime=10x -timeout=20m ./libs/sync/...
//	go test -tags benchworkspace -bench=BenchmarkListRepoByContent       -benchtime=10x -timeout=20m ./libs/sync/...
//	go test -tags benchworkspace -bench=BenchmarkListWalkers             -benchtime=5x  -timeout=30m ./libs/sync/...
//	go test -tags benchworkspace -bench=BenchmarkSyncRunOnceColdSnapshot -benchtime=5x  -timeout=30m ./libs/sync/...
//
// Filter further with -bench='BenchmarkListWalkers/shape=medium/N=200'.
//
// Each benchmark group creates and tears down a unique remote tree under
// /Users/$DATABRICKS_BENCH_USER/.tmp/sync-bench-<random>/ on the configured
// workspace. Use a scratch profile.

package sync

import (
	"context"
	"crypto/rand"
	"encoding/hex"
	"fmt"
	"net/http"
	"net/url"
	"os"
	"path"
	"path/filepath"
	"sort"
	stdsync "sync"
	"sync/atomic"
	"strings"
	"testing"

	"github.com/databricks/cli/libs/filer"
	"github.com/databricks/cli/libs/vfs"
	"github.com/databricks/databricks-sdk-go"
	apiclient "github.com/databricks/databricks-sdk-go/client"
	"github.com/databricks/databricks-sdk-go/config"
	"github.com/databricks/databricks-sdk-go/service/workspace"
)

// ----- shape: tree depth and branching used to spread files across dirs ----

type treeShape struct {
	name   string
	depth  int
	branch int // unused when depth == 0 (flat)
}

// shapes ranges from "flat" (no nesting) to "large" (deep tree, many dirs).
// Tweak these values if you want to explore broader/narrower mixes.
var shapes = []treeShape{
	{"flat", 0, 0},     // 1 leaf dir
	{"small", 2, 2},    // 4 leaf dirs
	{"medium", 4, 2},   // 16 leaf dirs
	{"large", 6, 2},    // 64 leaf dirs
}

// generatePaths returns n relative file paths arranged into a tree of the
// requested shape. Filenames are unique; no extension is added (callers add
// the right one for the content kind they're using).
func generatePaths(shape treeShape, n int) []string {
	if shape.depth == 0 {
		out := make([]string, n)
		for i := 0; i < n; i++ {
			out[i] = fmt.Sprintf("f-%05d", i)
		}
		return out
	}
	leaves := 1
	for i := 0; i < shape.depth; i++ {
		leaves *= shape.branch
	}
	filesPerLeaf := (n + leaves - 1) / leaves
	if filesPerLeaf < 1 {
		filesPerLeaf = 1
	}
	var paths []string
	var walk func(prefix string, d int)
	walk = func(prefix string, d int) {
		if d == 0 {
			for f := 0; f < filesPerLeaf && len(paths) < n; f++ {
				paths = append(paths, path.Join(prefix, fmt.Sprintf("f-%d", f)))
			}
			return
		}
		for b := 0; b < shape.branch; b++ {
			if len(paths) >= n {
				return
			}
			walk(path.Join(prefix, fmt.Sprintf("d%d", b)), d-1)
		}
	}
	walk("", shape.depth)
	return paths
}

// ----- environment + helpers ------------------------------------------------

type benchEnv struct {
	wc        *databricks.WorkspaceClient
	apiClient *apiclient.DatabricksClient
	username  string
	root      string // absolute workspace path, no trailing slash
}

func newBenchEnv(tb testing.TB) *benchEnv {
	tb.Helper()
	profile := os.Getenv("DATABRICKS_BENCH_PROFILE")
	user := os.Getenv("DATABRICKS_BENCH_USER")
	if profile == "" || user == "" {
		tb.Skip("DATABRICKS_BENCH_PROFILE and DATABRICKS_BENCH_USER must be set")
	}
	wc, err := databricks.NewWorkspaceClient(&databricks.Config{Profile: profile})
	if err != nil {
		tb.Fatalf("workspace client: %v", err)
	}
	c, err := apiclient.New(&config.Config{Profile: profile})
	if err != nil {
		tb.Fatalf("api client: %v", err)
	}
	tag := make([]byte, 4)
	_, _ = rand.Read(tag)
	root := fmt.Sprintf("/Users/%s/.tmp/sync-bench-%s", user, hex.EncodeToString(tag))
	if err := wc.Workspace.MkdirsByPath(context.Background(), root); err != nil {
		tb.Fatalf("mkdir root: %v", err)
	}
	tb.Cleanup(func() {
		_ = wc.Workspace.Delete(context.Background(), workspace.Delete{Path: root, Recursive: true})
	})
	return &benchEnv{wc: wc, apiClient: c, username: user, root: root}
}

// importRaw uploads body bytes via the legacy import-file endpoint. Used for
// setup only — what we benchmark is the listing / sync path, not the upload.
func importRaw(ctx context.Context, c *apiclient.DatabricksClient, absPath string, body []byte) error {
	urlPath := fmt.Sprintf("/api/2.0/workspace-files/import-file/%s?overwrite=true",
		url.PathEscape(strings.TrimLeft(absPath, "/")))
	return c.Do(ctx, http.MethodPost, urlPath, nil, nil, body, nil)
}

// populate creates all parent dirs under remoteDir, then uploads (relPath →
// body) entries via a 16-worker pool. Setup helper for benchmarks.
func populate(tb testing.TB, env *benchEnv, remoteDir string, items map[string][]byte) {
	tb.Helper()
	ctx := context.Background()
	if err := env.wc.Workspace.MkdirsByPath(ctx, remoteDir); err != nil {
		tb.Fatalf("mkdir %s: %v", remoteDir, err)
	}
	parents := map[string]struct{}{}
	for rel := range items {
		if d := path.Dir(rel); d != "." && d != "/" {
			parents[d] = struct{}{}
		}
	}
	parentList := make([]string, 0, len(parents))
	for d := range parents {
		parentList = append(parentList, d)
	}
	sort.Strings(parentList)
	for _, d := range parentList {
		if err := env.wc.Workspace.MkdirsByPath(ctx, path.Join(remoteDir, d)); err != nil {
			tb.Fatalf("mkdir %s: %v", d, err)
		}
	}
	type job struct {
		rel  string
		body []byte
	}
	jobs := make(chan job, len(items))
	for r, b := range items {
		jobs <- job{r, b}
	}
	close(jobs)
	var wg stdsync.WaitGroup
	var failed atomic.Int64
	for i := 0; i < 16; i++ {
		wg.Go(func() {
			for j := range jobs {
				if err := importRaw(ctx, env.apiClient, path.Join(remoteDir, j.rel), j.body); err != nil {
					failed.Add(1)
				}
			}
		})
	}
	wg.Wait()
	if f := failed.Load(); f > 0 {
		tb.Logf("warning: %d uploads failed during populate", f)
	}
}

// generators returns body generators for each content kind, sized roughly to
// 200 bytes so list-repo response time isn't dominated by per-file content.
func generators() map[string]func(i int) (suffix string, body []byte) {
	pad := func(s string, n int) []byte {
		if n <= len(s) {
			return []byte(s[:n])
		}
		buf := make([]byte, n)
		copy(buf, s)
		for i := len(s); i < n; i++ {
			buf[i] = byte('a' + (i % 26))
		}
		return buf
	}
	return map[string]func(i int) (string, []byte){
		"file": func(i int) (string, []byte) {
			return ".txt", pad(fmt.Sprintf("plain file %d\n", i), 200)
		},
		"py-notebook": func(i int) (string, []byte) {
			return ".py", pad(fmt.Sprintf("# Databricks notebook source\nprint('%d')\n", i), 200)
		},
		"sql-notebook": func(i int) (string, []byte) {
			return ".sql", pad(fmt.Sprintf("-- Databricks notebook source\nSELECT %d;\n", i), 200)
		},
		"ipynb": func(i int) (string, []byte) {
			return ".ipynb", pad(fmt.Sprintf(`{"cells":[{"cell_type":"code","source":["# Databricks notebook source\n","print(%d)"],"outputs":[],"execution_count":null,"metadata":{}}],"metadata":{"kernelspec":{"display_name":"Python 3","language":"python","name":"python3"}},"nbformat":4,"nbformat_minor":4}`, i), 200)
		},
		"dashboard": func(i int) (string, []byte) {
			return ".lvdash.json", pad(`{"datasets":[],"pages":[{"name":"p","displayName":"P","layout":[]}]}`, 200)
		},
	}
}

// itemsForCell builds the (relPath → body) map for a (shape, n, kindKey) cell.
func itemsForCell(shape treeShape, n int, kindKey string) map[string][]byte {
	gen := generators()[kindKey]
	paths := generatePaths(shape, n)
	out := make(map[string][]byte, len(paths))
	for i, rel := range paths {
		suf, body := gen(i)
		out[rel+suf] = body
	}
	return out
}

// itemsMixed builds a map where each file is a different kind in a fixed
// rotation. Used by the content-mix benchmark.
func itemsMixed(shape treeShape, n int) map[string][]byte {
	gens := generators()
	kinds := []string{"file", "py-notebook", "ipynb", "dashboard", "sql-notebook"}
	paths := generatePaths(shape, n)
	out := make(map[string][]byte, len(paths))
	for i, rel := range paths {
		suf, body := gens[kinds[i%len(kinds)]](i)
		out[rel+suf] = body
	}
	return out
}

// ----- parallel-walk runner (test-only alternative to list-repo) ------------

// parallelWalk lists everything under root by issuing non-recursive
// /api/2.0/workspace/list calls and recursing into directories from the
// client side, with up to `workers` outstanding calls in flight. It serves as
// a baseline to compare against the recursive list-repo endpoint.
//
// Test-only: we do not ship this in production. The list-repo path is
// strictly faster for any nested tree (see benchmark numbers).
func parallelWalk(ctx context.Context, c *apiclient.DatabricksClient, root string, workers int) ([]filer.RemoteFileMetadata, error) {
	type listObject struct {
		ObjectType       string `json:"object_type"`
		Path             string `json:"path"`
		ContentSHA256Hex string `json:"content_sha256_hex"`
		HasWsfsMetadata  bool   `json:"has_wsfs_metadata"`
	}
	type listResp struct {
		Objects []listObject `json:"objects"`
	}

	sem := make(chan struct{}, workers)
	var (
		mu       stdsync.Mutex
		results  []filer.RemoteFileMetadata
		firstErr error
		wg       stdsync.WaitGroup
	)

	var walk func(dir string)
	walk = func(dir string) {
		defer wg.Done()
		sem <- struct{}{}
		defer func() { <-sem }()

		var resp listResp
		body := map[string]any{"path": dir, "return_wsfs_metadata": true}
		if err := c.Do(ctx, http.MethodGet, "/api/2.0/workspace/list", nil, nil, body, &resp); err != nil {
			mu.Lock()
			if firstErr == nil {
				firstErr = err
			}
			mu.Unlock()
			return
		}
		for _, o := range resp.Objects {
			if o.ObjectType == "DIRECTORY" {
				wg.Add(1)
				go walk(o.Path)
				continue
			}
			if o.ContentSHA256Hex == "" {
				continue
			}
			mu.Lock()
			results = append(results, filer.RemoteFileMetadata{
				Path:             o.Path,
				ContentSHA256Hex: o.ContentSHA256Hex,
				ObjectType:       o.ObjectType,
			})
			mu.Unlock()
		}
	}
	wg.Add(1)
	go walk(root)
	wg.Wait()
	return results, firstErr
}

// ----- benchmarks ----------------------------------------------------------

// BenchmarkListRepo measures /workspace/list-repo cost vs (shape, N) at fixed
// content (plain files).
func BenchmarkListRepo(b *testing.B) {
	env := newBenchEnv(b)
	wfc, err := filer.NewWorkspaceFilesClient(env.wc, env.root)
	if err != nil {
		b.Fatalf("filer: %v", err)
	}
	lister := wfc.(*filer.WorkspaceFilesClient)

	counts := []int{10, 100, 500}
	for _, shape := range shapes {
		for _, n := range counts {
			b.Run(fmt.Sprintf("shape=%s/N=%d", shape.name, n), func(b *testing.B) {
				dir := path.Join(env.root, fmt.Sprintf("listrepo-%s-%d", shape.name, n))
				populate(b, env, dir, itemsForCell(shape, n, "file"))
				_, _ = lister.ListWithSHAs(context.Background(), dir)
				b.ResetTimer()
				for i := 0; i < b.N; i++ {
					if _, err := lister.ListWithSHAs(context.Background(), dir); err != nil {
						b.Fatalf("list: %v", err)
					}
				}
			})
		}
	}
}

// BenchmarkListRepoByContent measures /workspace/list-repo cost across
// content mixes, at every shape, fixed N=200.
func BenchmarkListRepoByContent(b *testing.B) {
	const N = 200
	env := newBenchEnv(b)
	wfc, err := filer.NewWorkspaceFilesClient(env.wc, env.root)
	if err != nil {
		b.Fatalf("filer: %v", err)
	}
	lister := wfc.(*filer.WorkspaceFilesClient)

	contents := []string{"file", "py-notebook", "ipynb", "sql-notebook", "dashboard"}
	for _, shape := range shapes {
		for _, kind := range contents {
			b.Run(fmt.Sprintf("shape=%s/content=%s", shape.name, kind), func(b *testing.B) {
				dir := path.Join(env.root, fmt.Sprintf("content-%s-%s", shape.name, kind))
				populate(b, env, dir, itemsForCell(shape, N, kind))
				_, _ = lister.ListWithSHAs(context.Background(), dir)
				b.ResetTimer()
				for i := 0; i < b.N; i++ {
					if _, err := lister.ListWithSHAs(context.Background(), dir); err != nil {
						b.Fatalf("list: %v", err)
					}
				}
			})
		}
		b.Run(fmt.Sprintf("shape=%s/content=mixed", shape.name), func(b *testing.B) {
			dir := path.Join(env.root, fmt.Sprintf("content-%s-mixed", shape.name))
			populate(b, env, dir, itemsMixed(shape, N))
			_, _ = lister.ListWithSHAs(context.Background(), dir)
			b.ResetTimer()
			for i := 0; i < b.N; i++ {
				if _, err := lister.ListWithSHAs(context.Background(), dir); err != nil {
					b.Fatalf("list: %v", err)
				}
			}
		})
	}
}

// BenchmarkListWalkers compares /workspace/list-repo (recursive, server-side)
// against the test-only parallel client-side walk, across (shape, N, workers).
// Plain-file content; the goal is to characterize the listing strategies
// themselves.
func BenchmarkListWalkers(b *testing.B) {
	env := newBenchEnv(b)
	wfc, err := filer.NewWorkspaceFilesClient(env.wc, env.root)
	if err != nil {
		b.Fatalf("filer: %v", err)
	}
	lister := wfc.(*filer.WorkspaceFilesClient)

	counts := []int{100, 500}
	workerCounts := []int{8, 32}
	for _, shape := range shapes {
		for _, n := range counts {
			dir := path.Join(env.root, fmt.Sprintf("walkers-%s-%d", shape.name, n))
			populate(b, env, dir, itemsForCell(shape, n, "file"))
			_, _ = lister.ListWithSHAs(context.Background(), dir)

			b.Run(fmt.Sprintf("shape=%s/N=%d/impl=list-repo", shape.name, n), func(b *testing.B) {
				for i := 0; i < b.N; i++ {
					if _, err := lister.ListWithSHAs(context.Background(), dir); err != nil {
						b.Fatalf("list-repo: %v", err)
					}
				}
			})
			for _, w := range workerCounts {
				w := w
				b.Run(fmt.Sprintf("shape=%s/N=%d/impl=parallel-w%d", shape.name, n, w), func(b *testing.B) {
					for i := 0; i < b.N; i++ {
						if _, err := parallelWalk(context.Background(), env.apiClient, dir, w); err != nil {
							b.Fatalf("parallel-walk: %v", err)
						}
					}
				})
			}
		}
	}
}

// BenchmarkSyncRunOnceColdSnapshot times an end-to-end Sync.RunOnce against a
// pre-warmed remote (the CI-cold-runner case). Sub-benchmarks compare with-
// and without-Layer-3 across (shape, N) cells. Plain-file content.
func BenchmarkSyncRunOnceColdSnapshot(b *testing.B) {
	env := newBenchEnv(b)
	user, err := env.wc.CurrentUser.Me(context.Background())
	if err != nil {
		b.Fatalf("Me: %v", err)
	}

	counts := []int{20, 100}
	for _, shape := range shapes {
		for _, n := range counts {
			b.Run(fmt.Sprintf("shape=%s/N=%d", shape.name, n), func(b *testing.B) {
				localDir := b.TempDir()
				gen := generators()["file"]
				for i, rel := range generatePaths(shape, n) {
					suf, body := gen(i)
					p := filepath.Join(localDir, filepath.FromSlash(rel)+suf)
					if err := os.MkdirAll(filepath.Dir(p), 0o755); err != nil {
						b.Fatalf("mkdir: %v", err)
					}
					if err := os.WriteFile(p, body, 0o644); err != nil {
						b.Fatalf("write %s: %v", p, err)
					}
				}
				remoteDir := path.Join(env.root, fmt.Sprintf("sync-%s-%d", shape.name, n))
				if err := env.wc.Workspace.MkdirsByPath(context.Background(), remoteDir); err != nil {
					b.Fatalf("mkdir: %v", err)
				}

				runOnce := func(b *testing.B, withLayer3 bool) {
					_ = os.RemoveAll(filepath.Join(localDir, ".databricks"))
					localRoot := vfs.MustNew(localDir)
					snapBase := b.TempDir()
					s, err := New(context.Background(), SyncOptions{
						WorktreeRoot:     localRoot,
						LocalRoot:        localRoot,
						Paths:            []string{"."},
						RemotePath:       remoteDir,
						SnapshotBasePath: snapBase,
						WorkspaceClient:  env.wc,
						CurrentUser:      user,
					})
					if err != nil {
						b.Fatalf("Sync.New: %v", err)
					}
					if !withLayer3 {
						s.remoteFilter = nil
					}
					if _, err := s.RunOnce(context.Background()); err != nil {
						b.Fatalf("RunOnce: %v", err)
					}
					s.Close()
				}

				// Pre-warm so the workspace already has every file.
				runOnce(b, true)
				b.ResetTimer()

				b.Run("with-layer3", func(b *testing.B) {
					for i := 0; i < b.N; i++ {
						runOnce(b, true)
					}
				})
				b.Run("without-layer3", func(b *testing.B) {
					for i := 0; i < b.N; i++ {
						runOnce(b, false)
					}
				})
			})
		}
	}
}
