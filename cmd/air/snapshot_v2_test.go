package aircmd

import (
	"archive/tar"
	"bytes"
	"compress/gzip"
	"context"
	"encoding/hex"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"maps"
	"net/http"
	"os"
	"os/exec"
	"path"
	"path/filepath"
	"slices"
	"strings"
	"sync"
	"sync/atomic"
	"testing"
	"time"

	"github.com/databricks/cli/libs/cmdio"
	"github.com/databricks/cli/libs/filer"
	"github.com/databricks/cli/libs/testserver"
	"github.com/databricks/databricks-sdk-go"
	"github.com/databricks/databricks-sdk-go/service/jobs"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// fixedMtime pins file times so fragment bytes, and therefore ids, do not depend on when the
// test ran.
var fixedMtime = time.Unix(1_700_000_000, 0)

// writeV2Tree writes files under a fresh repo directory with fixed modes and mtimes. The
// directory name is "code".
func writeV2Tree(t *testing.T, files map[string]string) string {
	t.Helper()
	repo := filepath.Join(t.TempDir(), "code")
	for rel, content := range files {
		writeRepoFile(t, repo, rel, content)
		full := filepath.Join(repo, filepath.FromSlash(rel))
		require.NoError(t, os.Chmod(full, 0o644))
		require.NoError(t, os.Chtimes(full, fixedMtime, fixedMtime))
	}
	return repo
}

// v2TestTree is a repo with enough files to spread over several buckets.
func v2TestTree(n int) map[string]string {
	files := make(map[string]string, n)
	for i := range n {
		files[fmt.Sprintf("pkg%d/mod%d.py", i%4, i)] = strings.Repeat(fmt.Sprintf("x = %d\n", i), 10+i)
	}
	return files
}

// v2IDs returns the member ids of the repo's snapshot packed with k members, as a set.
func v2IDs(t *testing.T, repo string, k int) map[string]bool {
	t.Helper()
	files, err := snapshotFiles(t.Context(), repo, nil, false)
	require.NoError(t, err)
	members, err := buildV2Members(t.Context(), t.TempDir(), repo, filepath.Base(repo), files, k)
	require.NoError(t, err)
	ids := make(map[string]bool, len(members))
	for _, m := range members {
		ids[m.ID] = true
	}
	return ids
}

// idChanges returns how many ids in after are new, and how many in before are gone. A member
// whose content changed contributes one of each.
func idChanges(before, after map[string]bool) (added, removed int) {
	for id := range after {
		if !before[id] {
			added++
		}
	}
	for id := range before {
		if !after[id] {
			removed++
		}
	}
	return added, removed
}

func TestKForTarSize(t *testing.T) {
	tests := []struct {
		name    string
		tarSize int64
		prevK   int
		want    int
	}{
		{name: "small repo is one member", tarSize: 1 << 20, want: 1},
		{name: "16 MiB steps to four", tarSize: 16 << 20, want: 4},
		{name: "just under 128 MiB stays four", tarSize: 128<<20 - 1, want: 4},
		{name: "128 MiB steps to sixteen", tarSize: 128 << 20, want: 16},
		{name: "shrinking keeps previous k", tarSize: 1 << 20, prevK: 16, want: 16},
		{name: "growth past previous k", tarSize: 200 << 20, prevK: 4, want: 16},
	}
	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			assert.Equal(t, tc.want, kForTarSize(tc.tarSize, tc.prevK))
		})
	}
}

// Same tree gives the same ids, and the ids are pinned so a change to the fragment encoding
// is visible in review (it re-uploads every member once).
// The same tree gives the same ids, from one run to the next and from a copy in another
// directory. Ids are not pinned because owner fields are part of the fragment, so they depend
// on the user that packages the tree.
func TestV2IDsAreDeterministic(t *testing.T) {
	first := v2IDs(t, writeV2Tree(t, v2TestTree(24)), v2Buckets)
	assert.Equal(t, first, v2IDs(t, writeV2Tree(t, v2TestTree(24)), v2Buckets))
	assert.Equal(t, first, v2IDs(t, writeV2Tree(t, v2TestTree(24)), v2Buckets))
}

// fixedV2 is a v2Decision with a fixed answer.
func fixedV2(v2 bool) v2Decision {
	return func(context.Context) (bool, error) { return v2, nil }
}

// entryHeader is the part of a tar header that must match between v1 and v2.
type entryHeader struct {
	Typeflag byte
	Mode     int64
	ModTime  int64
	Uid, Gid int
	Uname    string
	Gname    string
	Linkname string
	Size     int64
}

// tarHeaders reads every entry header from a tar stream.
func tarHeaders(t *testing.T, r io.Reader) map[string]entryHeader {
	t.Helper()
	out := map[string]entryHeader{}
	tr := tar.NewReader(r)
	for {
		hdr, err := tr.Next()
		if errors.Is(err, io.EOF) {
			return out
		}
		require.NoError(t, err)
		out[hdr.Name] = entryHeader{
			Typeflag: hdr.Typeflag,
			Mode:     hdr.Mode,
			ModTime:  hdr.ModTime.UnixNano(),
			Uid:      hdr.Uid,
			Gid:      hdr.Gid,
			Uname:    hdr.Uname,
			Gname:    hdr.Gname,
			Linkname: hdr.Linkname,
			Size:     hdr.Size,
		}
	}
}

// A v1 tarball and the v2 members of the same tree carry identical headers for every entry:
// name, mode, mtime, owner, type, link target and size. Only the packaging differs.
func TestV2AndV1EntryHeadersMatch(t *testing.T) {
	repo := writeV2Tree(t, v2TestTree(60))
	require.NoError(t, os.Symlink("pkg0/mod0.py", filepath.Join(repo, "link.py")))
	files, err := snapshotFiles(t.Context(), repo, nil, false)
	require.NoError(t, err)

	v1Path := filepath.Join(t.TempDir(), "v1.tar.gz")
	_, err = createPlainTarball(t.Context(), repo, v1Path, files)
	require.NoError(t, err)
	v1Data, err := os.ReadFile(v1Path)
	require.NoError(t, err)
	gz, err := gzip.NewReader(bytes.NewReader(v1Data))
	require.NoError(t, err)
	v1 := tarHeaders(t, gz)

	tmp := t.TempDir()
	members, err := buildV2Members(t.Context(), tmp, repo, filepath.Base(repo), files, v2Buckets)
	require.NoError(t, err)
	v2 := map[string]entryHeader{}
	for _, m := range members {
		f, err := os.Open(m.fragment)
		require.NoError(t, err)
		maps.Copy(v2, tarHeaders(t, f))
		require.NoError(t, f.Close())
	}

	require.Len(t, v2, len(v1))
	for name, want := range v1 {
		assert.Equal(t, want, v2[name], name)
	}
	assert.Equal(t, byte(tar.TypeSymlink), v1[path.Join(filepath.Base(repo), "link.py")].Typeflag)
}

// The manifest has exactly the agreed fields, and the end object is last.
func TestV2ManifestSchema(t *testing.T) {
	repo := writeV2Tree(t, v2TestTree(30))
	store := newV2Store(t, nil)
	files, err := snapshotFiles(t.Context(), repo, nil, false)
	require.NoError(t, err)
	var result snapshotResult
	require.NoError(t, uploadSnapshotV2(t.Context(), store.f, testSnapshotArtifactPath+"/.internal", repo, files, &result, time.Now()))

	rc, err := store.f.Read(t.Context(), path.Base(result.CodeSourcePath))
	require.NoError(t, err)
	defer rc.Close()
	var manifest map[string]json.RawMessage
	require.NoError(t, json.NewDecoder(rc).Decode(&manifest))
	keys := make([]string, 0, len(manifest))
	for k := range manifest {
		keys = append(keys, k)
	}
	slices.Sort(keys)
	assert.Equal(t, []string{"component", "format", "k", "members", "tar_size", "version"}, keys)

	var typed struct {
		Format  string     `json:"format"`
		Version int        `json:"version"`
		K       int        `json:"k"`
		Members []v2Member `json:"members"`
	}
	raw, err := json.Marshal(manifest)
	require.NoError(t, err)
	require.NoError(t, json.Unmarshal(raw, &typed))
	assert.Equal(t, "air-snapshot-manifest", typed.Format)
	assert.Equal(t, 2, typed.Version)
	require.NotEmpty(t, typed.Members)

	end, err := writeV2EndFragment(t.TempDir())
	require.NoError(t, err)
	assert.Equal(t, end.ID, typed.Members[len(typed.Members)-1].ID, "end object is last")
	for _, m := range typed.Members {
		assert.Len(t, m.ID, 32)
		assert.Positive(t, m.GzSize)
		assert.Positive(t, m.TarSize)
	}
}

// A one-file edit changes exactly one member id: the member holding the file.
func TestV2OneFileEditChangesOneID(t *testing.T) {
	files := v2TestTree(40)
	repo := writeV2Tree(t, files)
	before := v2IDs(t, repo, v2Buckets)

	edited := "x = changed\n"
	writeRepoFile(t, repo, "pkg1/mod5.py", edited)
	require.NoError(t, os.Chmod(filepath.Join(repo, "pkg1", "mod5.py"), 0o644))
	require.NoError(t, os.Chtimes(filepath.Join(repo, "pkg1", "mod5.py"), fixedMtime, fixedMtime))
	after := v2IDs(t, repo, v2Buckets)

	added, removed := idChanges(before, after)
	assert.Equal(t, 1, added, "one member id is new")
	assert.Equal(t, 1, removed, "one member id is gone")
	assert.Len(t, after, len(before))
}

// Add, remove and rename each change at most two ids.
func TestV2AddRemoveRenameChangeAtMostTwoIDs(t *testing.T) {
	base := writeV2Tree(t, v2TestTree(40))
	before := v2IDs(t, base, v2Buckets)

	t.Run("add", func(t *testing.T) {
		repo := writeV2Tree(t, v2TestTree(40))
		writeRepoFile(t, repo, "pkg2/new.py", "print()")
		added, _ := idChanges(before, v2IDs(t, repo, v2Buckets))
		assert.LessOrEqual(t, added, 2)
	})
	t.Run("remove", func(t *testing.T) {
		repo := writeV2Tree(t, v2TestTree(40))
		require.NoError(t, os.Remove(filepath.Join(repo, "pkg0", "mod0.py")))
		added, _ := idChanges(before, v2IDs(t, repo, v2Buckets))
		assert.LessOrEqual(t, added, 2)
	})
	t.Run("rename", func(t *testing.T) {
		repo := writeV2Tree(t, v2TestTree(40))
		require.NoError(t, os.Rename(filepath.Join(repo, "pkg1", "mod1.py"), filepath.Join(repo, "pkg1", "renamed.py")))
		added, _ := idChanges(before, v2IDs(t, repo, v2Buckets))
		assert.LessOrEqual(t, added, 2)
	})
}

// Concatenating the member objects in manifest order gives one archive with the full file
// set. Reading it with tar must not stop early, which is what record padding inside a
// fragment would cause.
func TestV2ConcatenatedMembersExtractTheFullSet(t *testing.T) {
	repo := writeV2Tree(t, v2TestTree(60))
	files, err := snapshotFiles(t.Context(), repo, nil, false)
	require.NoError(t, err)
	tmp := t.TempDir()
	members, err := buildV2Members(t.Context(), tmp, repo, filepath.Base(repo), files, v2Buckets)
	require.NoError(t, err)
	end, err := writeV2EndFragment(tmp)
	require.NoError(t, err)
	members = append(members, end)
	require.Greater(t, len(members), 2)

	var concat bytes.Buffer
	for _, m := range members {
		gzPath := filepath.Join(tmp, m.ID+".gz")
		require.NoError(t, compressRaw(t.Context(), m.fragment, gzPath))
		data, err := os.ReadFile(gzPath)
		require.NoError(t, err)
		concat.Write(data)
	}

	var want []string
	for _, f := range files {
		want = append(want, path.Join(filepath.Base(repo), filepath.ToSlash(f.rel)))
	}
	slices.Sort(want)

	gz, err := gzip.NewReader(bytes.NewReader(concat.Bytes()))
	require.NoError(t, err)
	got := tarNames(t, gz)
	slices.Sort(got)
	assert.Equal(t, want, got)

	// The same concatenation, read by the system tar, also lists the full set.
	tarBin, err := exec.LookPath("tar")
	if err != nil {
		t.Skip("tar not available")
	}
	archive := filepath.Join(tmp, "concat.tar.gz")
	require.NoError(t, os.WriteFile(archive, concat.Bytes(), 0o644))
	out, err := exec.CommandContext(t.Context(), tarBin, "-tzf", archive).Output()
	require.NoError(t, err)
	listed := strings.Fields(string(out))
	slices.Sort(listed)
	assert.Equal(t, want, listed)
}

// The end member is two zero blocks, and it is a constant id.
func TestV2EndMemberIsConstant(t *testing.T) {
	tmp := t.TempDir()
	a, err := writeV2EndFragment(tmp)
	require.NoError(t, err)
	b, err := writeV2EndFragment(t.TempDir())
	require.NoError(t, err)
	assert.Equal(t, a.ID, b.ID)
	assert.Len(t, a.ID, 32)
	_, err = hex.DecodeString(a.ID)
	require.NoError(t, err)
	assert.Equal(t, int64(v2EndTarBytes), a.TarSize)
}

// v2Store is a fake workspace store for the v2 tests. Writes are recorded by object name, and
// the handler fails a write when failWrite says so.
type v2Store struct {
	f         filer.Filer
	mu        sync.Mutex
	writes    map[string]int
	failWrite func(name string, attempt int) bool
	failed    atomic.Int32
	inFlight  atomic.Int32
	maxSeen   atomic.Int32
}

// setFailWrite changes which writes fail from now on.
func (s *v2Store) setFailWrite(failWrite func(name string, attempt int) bool) {
	s.mu.Lock()
	defer s.mu.Unlock()
	s.failWrite = failWrite
}

// resetWrites forgets the recorded writes.
func (s *v2Store) resetWrites() {
	s.mu.Lock()
	defer s.mu.Unlock()
	s.writes = map[string]int{}
}

func newV2Store(t *testing.T, failWrite func(name string, attempt int) bool) *v2Store {
	t.Helper()
	server := testserver.New(t)
	t.Cleanup(server.Close)
	s := &v2Store{writes: map[string]int{}, failWrite: failWrite}
	server.Handle("POST", "/api/2.0/workspace-files/import-file/{path...}", func(req testserver.Request) any {
		name := req.Vars["path"]
		s.mu.Lock()
		s.writes[path.Base(name)]++
		attempt := s.writes[path.Base(name)]
		fail := s.failWrite
		s.mu.Unlock()
		if strings.Contains(name, "/.internal/") {
			cur := s.inFlight.Add(1)
			for {
				seen := s.maxSeen.Load()
				if cur <= seen || s.maxSeen.CompareAndSwap(seen, cur) {
					break
				}
			}
			defer s.inFlight.Add(-1)
			time.Sleep(20 * time.Millisecond)
		}
		if fail != nil && fail(path.Base(name), attempt) {
			s.failed.Add(1)
			return testserver.Response{StatusCode: 500}
		}
		return req.Workspace.WorkspaceFilesImportFile(name, req.Body, req.URL.Query().Get("overwrite") == "true")
	})
	testserver.AddDefaultHandlers(server)
	w, err := databricks.NewWorkspaceClient(&databricks.Config{Host: server.URL, Token: "token"})
	require.NoError(t, err)
	s.f, err = filer.NewWorkspaceFilesClient(w, testSnapshotArtifactPath+"/.internal")
	require.NoError(t, err)
	return s
}

func (s *v2Store) written(name string) int {
	s.mu.Lock()
	defer s.mu.Unlock()
	return s.writes[name]
}

func (s *v2Store) totalWrites() int {
	s.mu.Lock()
	defer s.mu.Unlock()
	n := 0
	for _, c := range s.writes {
		n += c
	}
	return n
}

// An interrupted upload writes its objects but never the manifest, so the snapshot is not
// visible. A retry finishes it without rewriting the objects that already landed.
func TestV2InterruptedUploadLeavesNoManifest(t *testing.T) {
	repo := writeV2Tree(t, v2TestTree(30))
	files, err := snapshotFiles(t.Context(), repo, nil, false)
	require.NoError(t, err)
	manifestName := "code_" + computePlainTarKey(files)[:16] + v2ManifestSuffix

	store := newV2Store(t, func(name string, _ int) bool { return name == manifestName })
	uploadPath := testSnapshotArtifactPath + "/.internal"
	var result snapshotResult
	require.Error(t, uploadSnapshotV2(t.Context(), store.f, uploadPath, repo, files, &result, time.Now()))
	_, err = store.f.Stat(t.Context(), manifestName)
	assert.ErrorIs(t, err, os.ErrNotExist)
	assert.Positive(t, store.totalWrites(), "the objects were written before the manifest failed")

	// The retry finds the objects in the store and writes only the manifest.
	store.setFailWrite(nil)
	store.resetWrites()
	result = snapshotResult{}
	require.NoError(t, uploadSnapshotV2(t.Context(), store.f, uploadPath, repo, files, &result, time.Now()))
	assert.Equal(t, 1, store.totalWrites(), "only the manifest is written on retry")
	assert.Equal(t, 1, store.written(manifestName))
}

// A transient object write failure is retried and the upload succeeds.
func TestV2RetriesTransientWriteFailure(t *testing.T) {
	repo := writeV2Tree(t, v2TestTree(30))
	files, err := snapshotFiles(t.Context(), repo, nil, false)
	require.NoError(t, err)
	failFirst := func(name string, attempt int) bool {
		return strings.HasSuffix(name, ".tar.gz") && attempt == 1
	}
	store := newV2Store(t, failFirst)
	var result snapshotResult
	require.NoError(t, uploadSnapshotV2(t.Context(), store.f, testSnapshotArtifactPath+"/.internal", repo, files, &result, time.Now()))
	// Every object's first write failed, so each was retried and the upload still succeeded.
	assert.Positive(t, store.failed.Load())
}

// At most v2UploadConcurrency members are compressed or uploaded at once, and uploads do
// overlap.
func TestV2UploadConcurrencyIsBounded(t *testing.T) {
	const members = 3 * v2UploadConcurrency
	tmp := t.TempDir()
	list := make([]v2Member, members)
	for i := range list {
		fragment := filepath.Join(tmp, fmt.Sprintf("f%d.tar", i))
		content := make([]byte, 1024)
		content[0] = byte(i)
		require.NoError(t, os.WriteFile(fragment, content, 0o644))
		sum := bytes.Repeat([]byte{byte(i + 1)}, 16)
		list[i] = v2Member{ID: hex.EncodeToString(sum), TarSize: 1024, fragment: fragment}
	}
	store := newV2Store(t, nil)
	require.NoError(t, putV2Members(t.Context(), store.f, list, tmp, time.Now()))

	assert.LessOrEqual(t, int(store.maxSeen.Load()), v2UploadConcurrency)
	assert.Greater(t, int(store.maxSeen.Load()), 1, "uploads should overlap")
	for _, m := range list {
		assert.Positive(t, m.GzSize)
	}
}

// A snapshot that is already in the store is found by its manifest name and writes nothing.
func TestV2UnchangedSnapshotWritesNothing(t *testing.T) {
	repo := writeV2Tree(t, v2TestTree(30))
	store := newV2Store(t, nil)
	uploadPath := testSnapshotArtifactPath + "/.internal"
	files, err := snapshotFiles(t.Context(), repo, nil, false)
	require.NoError(t, err)

	var first snapshotResult
	require.NoError(t, uploadSnapshotV2(t.Context(), store.f, uploadPath, repo, files, &first, time.Now()))
	before := store.totalWrites()
	assert.Positive(t, before)

	var second snapshotResult
	require.NoError(t, uploadSnapshotV2(t.Context(), store.f, uploadPath, repo, files, &second, time.Now()))
	assert.Equal(t, first.CodeSourcePath, second.CodeSourcePath)
	// The only write is the manifest rewrite that renews its lease. No object is written.
	assert.Equal(t, before+1, store.totalWrites(), "unchanged resubmit renews the manifest and nothing else")
	assert.Equal(t, 2, store.written(path.Base(second.CodeSourcePath)), "written once by each submit")
	assert.Equal(t, new(int64(0)), second.UploadDurationMs)
	assert.Equal(t, first.SizeBytes, second.SizeBytes)
}

// The manifest records the K it was written with, and the next snapshot of the same
// component starts from it, so a shrinking repo keeps its members.
func TestV2KRatchetsFromPreviousManifest(t *testing.T) {
	repo := writeV2Tree(t, v2TestTree(30))
	store := newV2Store(t, nil)
	uploadPath := testSnapshotArtifactPath + "/.internal"

	// A previous snapshot of the same component with 16 members.
	prev := v2Manifest{Format: v2ManifestFormat, Version: v2ManifestVersion, Component: "code", K: 16}
	data, err := json.Marshal(prev)
	require.NoError(t, err)
	require.NoError(t, store.f.Write(t.Context(), "code_0123456789abcdef.air.json", bytes.NewReader(data), filer.OverwriteIfExists, filer.CreateParentDirectories))

	k, err := previousV2K(t.Context(), store.f, "code")
	require.NoError(t, err)
	assert.Equal(t, 16, k)

	files, err := snapshotFiles(t.Context(), repo, nil, false)
	require.NoError(t, err)
	var result snapshotResult
	require.NoError(t, uploadSnapshotV2(t.Context(), store.f, uploadPath, repo, files, &result, time.Now()))

	rc, err := store.f.Read(t.Context(), path.Base(result.CodeSourcePath))
	require.NoError(t, err)
	defer rc.Close()
	var written v2Manifest
	require.NoError(t, json.NewDecoder(rc).Decode(&written))
	assert.Equal(t, 16, written.K)
	assert.Equal(t, v2ManifestFormat, written.Format)
	assert.Equal(t, "code", written.Component)
	// Empty buckets are omitted, and the end member is always last.
	assert.Equal(t, int64(v2EndTarBytes), written.Members[len(written.Members)-1].TarSize)
	assert.LessOrEqual(t, len(written.Members), 16+1)
}

// The fetched capabilities come from the raw validate response, and absent or empty ones
// are reported as none.
func TestValidateConfigReturnsCapabilities(t *testing.T) {
	tests := []struct {
		name string
		body any
		want []string
	}{
		{name: "advertised", body: map[string]any{"capabilities": []string{codeSnapshotV2Capability}}, want: []string{codeSnapshotV2Capability}},
		{name: "absent", body: map[string]any{}, want: nil},
	}
	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			server := testserver.New(t)
			t.Cleanup(server.Close)
			server.Handle("POST", "/api/2.0/ai-training/config:validate", func(req testserver.Request) any {
				return tc.body
			})
			testserver.AddDefaultHandlers(server)
			w, err := databricks.NewWorkspaceClient(&databricks.Config{Host: server.URL, Token: "token"})
			require.NoError(t, err)

			unavailable, caps, err := preflightValidate(t.Context(), w, baseRunConfig(), "/Workspace/Users/me/cmd.sh", nil, "token")
			require.NoError(t, err)
			assert.False(t, unavailable)
			assert.Equal(t, tc.want, caps)
		})
	}
}

// submitWithValidate runs a plain working-tree submit against a validate stub that returns
// validateBody, and returns the bodies of the snapshot objects written and the submitted
// code_source_path.
func submitWithValidate(t *testing.T, validateBody any, repo string) (map[string][]byte, string) {
	t.Helper()
	server := testserver.New(t)
	t.Cleanup(server.Close)

	var got jobs.SubmitRun
	server.Handle("POST", "/api/2.2/jobs/runs/submit", func(req testserver.Request) any {
		require.NoError(t, json.Unmarshal(req.Body, &got))
		return jobs.SubmitRunResponse{RunId: 555}
	})
	server.Handle("POST", "/api/2.0/ai-training/config:validate", func(req testserver.Request) any {
		return validateBody
	})
	var mu sync.Mutex
	bodies := map[string][]byte{}
	server.Handle("POST", "/api/2.0/workspace-files/import-file/{path...}", func(req testserver.Request) any {
		p := req.Vars["path"]
		if strings.Contains(p, "/.air/repo_snapshots/") {
			mu.Lock()
			bodies[path.Base(p)] = append([]byte(nil), req.Body...)
			mu.Unlock()
		}
		return req.Workspace.WorkspaceFilesImportFile(p, req.Body, req.URL.Query().Get("overwrite") == "true")
	})
	testserver.AddDefaultHandlers(server)
	w, err := databricks.NewWorkspaceClient(&databricks.Config{Host: server.URL, Token: "token"})
	require.NoError(t, err)

	cfgPath := writeConfigFile(t, "run.yaml", minimalConfig+`
code_source:
  type: snapshot
  snapshot:
    root_path: `+repo+`
`)
	loaded, err := loadRunConfig(cfgPath)
	require.NoError(t, err)
	_, _, err = submitWorkload(cmdio.MockDiscard(t.Context()), w, loaded, cfgPath, "idem", false)
	require.NoError(t, err)
	return bodies, got.Tasks[0].AiRuntimeTask.CodeSourcePath
}

// A validate response without the capability (the current backend) produces the same v1
// tarball as before v2 existed, byte for byte, and no manifest.
func TestSubmitWithoutCapabilityKeepsV1ByteIdentical(t *testing.T) {
	repo := writeV2Tree(t, v2TestTree(20))
	bodies, codeSourcePath := submitWithValidate(t, validateConfigResponse{}, repo)

	tarName := path.Base(codeSourcePath)
	assert.True(t, strings.HasSuffix(tarName, ".tar.gz"), codeSourcePath)
	for name := range bodies {
		assert.False(t, strings.HasSuffix(name, v2ManifestSuffix), "no manifest without the capability: %s", name)
	}

	files, err := snapshotFiles(t.Context(), repo, nil, false)
	require.NoError(t, err)
	want := filepath.Join(t.TempDir(), "want.tar.gz")
	_, err = createPlainTarball(t.Context(), repo, want, files)
	require.NoError(t, err)
	wantBytes, err := os.ReadFile(want)
	require.NoError(t, err)
	assert.Equal(t, wantBytes, bodies[tarName])
}

// With the capability, the snapshot is a manifest plus member objects, and no v1 tarball.
func TestSubmitWithCapabilityWritesV2Manifest(t *testing.T) {
	repo := writeV2Tree(t, v2TestTree(20))
	validate := map[string]any{"capabilities": []string{codeSnapshotV2Capability}}
	bodies, codeSourcePath := submitWithValidate(t, validate, repo)

	assert.True(t, strings.HasSuffix(codeSourcePath, v2ManifestSuffix), codeSourcePath)
	manifestName := path.Base(codeSourcePath)
	raw, ok := bodies[manifestName]
	require.True(t, ok, "manifest uploaded")
	var manifest v2Manifest
	require.NoError(t, json.Unmarshal(raw, &manifest))
	assert.Equal(t, v2ManifestFormat, manifest.Format)
	assert.Equal(t, v2ManifestVersion, manifest.Version)
	files, err := snapshotFiles(t.Context(), repo, nil, false)
	require.NoError(t, err)
	assert.NotContains(t, bodies, "code_"+computePlainTarKey(files)[:16]+".tar.gz", "no v1 tarball with the capability")
}

// When validation is unavailable, no capability is known, so the submit keeps writing v1.
func TestSubmitWithValidateUnavailableKeepsV1(t *testing.T) {
	repo := writeV2Tree(t, v2TestTree(20))
	bodies, codeSourcePath := submitWithValidate(t, testserver.Response{StatusCode: http.StatusServiceUnavailable}, repo)

	assert.True(t, strings.HasSuffix(path.Base(codeSourcePath), ".tar.gz"), codeSourcePath)
	for name := range bodies {
		assert.False(t, strings.HasSuffix(name, v2ManifestSuffix), "no manifest without a validated capability: %s", name)
	}
}
