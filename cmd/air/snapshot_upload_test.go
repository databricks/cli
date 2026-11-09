package aircmd

import (
	"encoding/json"
	"path"
	"path/filepath"
	"strings"
	"sync"
	"testing"

	"github.com/databricks/cli/libs/cmdio"
	"github.com/databricks/cli/libs/filer"
	"github.com/databricks/cli/libs/testserver"
	"github.com/databricks/databricks-sdk-go"
	"github.com/databricks/databricks-sdk-go/service/jobs"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// uploadRecorder records which snapshot objects the fake workspace receives. The import
// handler runs concurrently with the sidecar and tarball writes, so it is mutex-guarded.
type uploadRecorder struct {
	mu      sync.Mutex
	written map[string]bool
	index   segmentIndex
}

// reset forgets prior writes so a phase can assert on its own uploads.
func (r *uploadRecorder) reset() {
	r.mu.Lock()
	defer r.mu.Unlock()
	r.written = map[string]bool{}
}

// names returns the snapshot object names written since the last reset.
func (r *uploadRecorder) names() map[string]bool {
	r.mu.Lock()
	defer r.mu.Unlock()
	out := map[string]bool{}
	for k := range r.written {
		out[k] = true
	}
	return out
}

func (r *uploadRecorder) record(name string, body []byte) {
	r.mu.Lock()
	defer r.mu.Unlock()
	r.written[name] = true
	if strings.HasSuffix(name, segmentIndexSuffix) {
		_ = json.Unmarshal(body, &r.index)
	}
}

// newUploadTestWorkspace returns a fake workspace that records snapshot object writes.
func newUploadTestWorkspace(t *testing.T) (*databricks.WorkspaceClient, *uploadRecorder) {
	t.Helper()
	server := testserver.New(t)
	t.Cleanup(server.Close)
	server.Handle("POST", "/api/2.2/jobs/runs/submit", func(req testserver.Request) any {
		return jobs.SubmitRunResponse{RunId: 555}
	})
	rec := &uploadRecorder{written: map[string]bool{}}
	server.Handle("POST", "/api/2.0/workspace-files/import-file/{path...}", func(req testserver.Request) any {
		p := req.Vars["path"]
		if strings.Contains(p, "/.air/repo_snapshots/") {
			rec.record(path.Base(p), req.Body)
		}
		return req.Workspace.WorkspaceFilesImportFile(p, req.Body, req.URL.Query().Get("overwrite") == "true")
	})
	stubValidateConfig(server)
	testserver.AddDefaultHandlers(server)
	w, err := databricks.NewWorkspaceClient(&databricks.Config{Host: server.URL, Token: "token"})
	require.NoError(t, err)
	return w, rec
}

// A plain-tar snapshot writes its index sidecar and tarball, and a later submit of the
// same tree is a cache hit that writes nothing.
func TestUploadSnapshotPlainTarWritesIndexAndTarball(t *testing.T) {
	w, rec := newUploadTestWorkspace(t)
	repo := filepath.Join(t.TempDir(), "src")
	writeRepoFile(t, repo, "train.py", "print()")
	snap, cfgPath := loadSnapshotConfig(t, repo)

	ctx := cmdio.MockDiscard(t.Context())
	sidecarStore, sidecarBase := testSidecarStore(t, w)
	first, err := uploadSnapshot(ctx, w, snap, cfgPath, testSnapshotArtifactPath, sidecarStore, sidecarBase, false)
	require.NoError(t, err)

	tarName := path.Base(first.CodeSourcePath)
	assert.Equal(t, map[string]bool{tarName: true, tarName + segmentIndexSuffix: true}, rec.names())
	assert.Equal(t, segmentFormat, rec.index.Format)
	assert.Equal(t, "src", rec.index.Component)
	require.NotNil(t, first.SizeBytes)
	assert.Equal(t, *first.SizeBytes, rec.index.Size)
	assert.Len(t, rec.index.Members, 1)

	rec.reset()
	second, err := uploadSnapshot(ctx, w, snap, cfgPath, testSnapshotArtifactPath, sidecarStore, sidecarBase, false)
	require.NoError(t, err)
	assert.Equal(t, first.CodeSourcePath, second.CodeSourcePath)
	assert.Empty(t, rec.names(), "cache hit should write neither the index nor the tarball")
}

// A tarball without its index sidecar is a cache miss: both objects are written again.
func TestUploadSnapshotPlainTarMissingSidecarReuploadsBoth(t *testing.T) {
	w, rec := newUploadTestWorkspace(t)
	repo := filepath.Join(t.TempDir(), "src")
	writeRepoFile(t, repo, "train.py", "print()")
	snap, cfgPath := loadSnapshotConfig(t, repo)

	ctx := cmdio.MockDiscard(t.Context())
	sidecarStore, sidecarBase := testSidecarStore(t, w)
	first, err := uploadSnapshot(ctx, w, snap, cfgPath, testSnapshotArtifactPath, sidecarStore, sidecarBase, false)
	require.NoError(t, err)
	tarName := path.Base(first.CodeSourcePath)

	uploadDir, err := filer.NewWorkspaceFilesClient(w, testSnapshotArtifactPath+"/.internal")
	require.NoError(t, err)
	require.NoError(t, uploadDir.Delete(ctx, tarName+segmentIndexSuffix))

	rec.reset()
	second, err := uploadSnapshot(ctx, w, snap, cfgPath, testSnapshotArtifactPath, sidecarStore, sidecarBase, false)
	require.NoError(t, err)
	assert.Equal(t, first.CodeSourcePath, second.CodeSourcePath)
	assert.Equal(t, map[string]bool{tarName: true, tarName + segmentIndexSuffix: true}, rec.names())
}

// loadSnapshotConfig writes a run config whose code_source is repo and returns its snapshot
// config and path.
func loadSnapshotConfig(t *testing.T, repo string) (*snapshotSourceConfig, string) {
	t.Helper()
	cfgPath := writeConfigFile(t, "run.yaml", minimalConfig+`
code_source:
  type: snapshot
  snapshot:
    root_path: `+repo+`
`)
	loaded, err := loadRunConfig(cfgPath)
	require.NoError(t, err)
	return loaded.CodeSource.Snapshot, cfgPath
}
