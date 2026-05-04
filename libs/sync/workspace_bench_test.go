//go:build benchworkspace

// Workspace benchmarks for libs/sync. Real-network — they hit a Databricks
// workspace and import / list / delete files there.
//
// Quickstart:
//
//	export DATABRICKS_BENCH_PROFILE=my-profile     # required; profile from .databrickscfg
//	export DATABRICKS_BENCH_USER=me@example.com    # required; remote dirs go under /Users/$DATABRICKS_BENCH_USER/.tmp/
//
//	go test -tags benchworkspace -bench=. -benchtime=5x -timeout=60m ./libs/sync/...
//
// To run a single benchmark group:
//
//	go test -tags benchworkspace -bench=BenchmarkListRepoByCount -benchtime=10x -timeout=20m ./libs/sync/...
//	go test -tags benchworkspace -bench=BenchmarkListRepoByContent -benchtime=10x -timeout=20m ./libs/sync/...
//	go test -tags benchworkspace -bench=BenchmarkSyncRunOnceColdSnapshot -benchtime=5x -timeout=30m ./libs/sync/...
//
// Use a scratch profile / scratch user — every run creates and deletes a
// fresh /Users/$DATABRICKS_BENCH_USER/.tmp/sync-bench-<random>/ tree.

package sync

import (
	"bytes"
	"context"
	"crypto/rand"
	"encoding/hex"
	"fmt"
	"net/http"
	"net/url"
	"os"
	"path"
	"path/filepath"
	"strings"
	"sync"
	"sync/atomic"
	"testing"

	"github.com/databricks/cli/libs/filer"
	"github.com/databricks/cli/libs/vfs"
	"github.com/databricks/databricks-sdk-go"
	apiclient "github.com/databricks/databricks-sdk-go/client"
	"github.com/databricks/databricks-sdk-go/config"
	"github.com/databricks/databricks-sdk-go/service/workspace"
)

// benchEnv groups the workspace state a benchmark needs.
type benchEnv struct {
	wc        *databricks.WorkspaceClient
	apiClient *apiclient.DatabricksClient
	profile   string
	username  string
	root      string // absolute, no trailing slash
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
	return &benchEnv{wc: wc, apiClient: c, profile: profile, username: user, root: root}
}

// importRaw uploads body bytes to the given absolute workspace path via the
// legacy /workspace-files/import-file endpoint. Used during benchmark setup
// (we want population to be cheap and predictable; what we're benchmarking
// is the listing / sync, not the upload).
func importRaw(ctx context.Context, c *apiclient.DatabricksClient, absPath string, body []byte) error {
	urlPath := fmt.Sprintf("/api/2.0/workspace-files/import-file/%s?overwrite=true",
		url.PathEscape(strings.TrimLeft(absPath, "/")))
	return c.Do(ctx, http.MethodPost, urlPath, nil, nil, body, nil)
}

// populate uploads the given (relative path -> body) map under remoteDir,
// using a fixed-size worker pool. Returns once all uploads finish.
func populate(tb testing.TB, env *benchEnv, remoteDir string, items map[string][]byte) {
	tb.Helper()
	ctx := context.Background()
	if err := env.wc.Workspace.MkdirsByPath(ctx, remoteDir); err != nil {
		tb.Fatalf("mkdir %s: %v", remoteDir, err)
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
	var wg sync.WaitGroup
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
	if failed.Load() > 0 {
		tb.Logf("warning: %d uploads failed during populate", failed.Load())
	}
}

// generators returns a body-by-kind table. The bodies are sized to roughly
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

// BenchmarkListRepoByCount measures /workspace/list-repo cost at varying file
// counts (all plain files), to characterize how the call scales with N.
func BenchmarkListRepoByCount(b *testing.B) {
	env := newBenchEnv(b)
	wfc, err := filer.NewWorkspaceFilesClient(env.wc, env.root)
	if err != nil {
		b.Fatalf("filer: %v", err)
	}
	lister := wfc.(*filer.WorkspaceFilesClient)

	gen := generators()["file"]
	for _, n := range []int{10, 100, 500, 1000} {
		b.Run(fmt.Sprintf("N=%d", n), func(b *testing.B) {
			dir := path.Join(env.root, fmt.Sprintf("count-%d", n))
			items := make(map[string][]byte, n)
			for i := 0; i < n; i++ {
				suf, body := gen(i)
				items[fmt.Sprintf("file-%05d%s", i, suf)] = body
			}
			populate(b, env, dir, items)
			// Warm-up
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

// BenchmarkListRepoByContent measures /workspace/list-repo cost when the
// directory has different mixes of object types. Each scenario has a fixed
// number of files; only the content mix varies.
func BenchmarkListRepoByContent(b *testing.B) {
	const N = 200
	env := newBenchEnv(b)
	wfc, err := filer.NewWorkspaceFilesClient(env.wc, env.root)
	if err != nil {
		b.Fatalf("filer: %v", err)
	}
	lister := wfc.(*filer.WorkspaceFilesClient)

	gen := generators()
	scenarios := map[string]func(i int) (string, []byte){
		"all-files":         gen["file"],
		"all-py-notebooks":  gen["py-notebook"],
		"all-ipynb":         gen["ipynb"],
		"all-sql-notebooks": gen["sql-notebook"],
		"all-dashboards":    gen["dashboard"],
	}
	mixedKinds := []string{"file", "py-notebook", "ipynb", "dashboard", "sql-notebook"}
	scenarios["mixed"] = func(i int) (string, []byte) {
		return gen[mixedKinds[i%len(mixedKinds)]](i)
	}

	for name, g := range scenarios {
		b.Run(name, func(b *testing.B) {
			dir := path.Join(env.root, "content-"+name)
			items := make(map[string][]byte, N)
			for i := 0; i < N; i++ {
				suf, body := g(i)
				items[fmt.Sprintf("item-%05d%s", i, suf)] = body
			}
			populate(b, env, dir, items)
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

// BenchmarkSyncRunOnceColdSnapshot times an end-to-end Sync.RunOnce against
// a workspace that already has the bundle's files (the CI-cold-runner case).
// Sub-benchmarks compare with-Layer-3 vs without-Layer-3 to show the speedup
// the remote-SHA filter delivers when nothing has actually changed.
func BenchmarkSyncRunOnceColdSnapshot(b *testing.B) {
	env := newBenchEnv(b)
	user, err := env.wc.CurrentUser.Me(context.Background())
	if err != nil {
		b.Fatalf("Me: %v", err)
	}

	for _, n := range []int{20, 100, 500} {
		b.Run(fmt.Sprintf("N=%d", n), func(b *testing.B) {
			localDir := b.TempDir()
			gen := generators()["file"]
			for i := 0; i < n; i++ {
				suf, body := gen(i)
				p := filepath.Join(localDir, fmt.Sprintf("file-%05d%s", i, suf))
				if err := os.WriteFile(p, body, 0o644); err != nil {
					b.Fatalf("write %s: %v", p, err)
				}
			}
			remoteDir := path.Join(env.root, fmt.Sprintf("sync-N%d", n))
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

			// Pre-warm: one initial sync so the workspace has the files.
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

// keep tests in this file linkable even without the bench tag
var _ = bytes.NewReader
