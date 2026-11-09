package aircmd

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"io"
	"io/fs"
	"path"
	"slices"
	"strings"
	"sync"
	"testing"
	"time"

	"github.com/databricks/cli/libs/filer"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// fakeClock is the time source for the GC and writer tests.
type fakeClock struct{ now time.Time }

func (c *fakeClock) Now() time.Time { return c.now }

func (c *fakeClock) Advance(d time.Duration) { c.now = c.now.Add(d) }

// memFile is one file in memFiler, stamped with the clock time of its last write.
type memFile struct {
	data  []byte
	mtime time.Time
}

// memFiler is an in-memory filer.Filer. Files are keyed by slash path relative to the store
// root, and ReadDir lists the direct children of a directory.
type memFiler struct {
	mu    sync.Mutex
	clock *fakeClock
	files map[string]memFile
	// writes counts writes by object name, for the writer tests.
	writes map[string]int
	// blockDir, when set, makes ReadDir of that directory wait for the context.
	blockDir string
}

func newMemFiler(clock *fakeClock) *memFiler {
	return &memFiler{clock: clock, files: map[string]memFile{}, writes: map[string]int{}}
}

func memKey(p string) string {
	return path.Clean(strings.TrimPrefix(p, "/"))
}

func (f *memFiler) Write(ctx context.Context, p string, r io.Reader, _ ...filer.WriteMode) error {
	data, err := io.ReadAll(r)
	if err != nil {
		return err
	}
	f.mu.Lock()
	defer f.mu.Unlock()
	f.files[memKey(p)] = memFile{data: data, mtime: f.clock.Now()}
	f.writes[path.Base(p)]++
	return nil
}

func (f *memFiler) Read(_ context.Context, p string) (io.ReadCloser, error) {
	f.mu.Lock()
	defer f.mu.Unlock()
	file, ok := f.files[memKey(p)]
	if !ok {
		return nil, &fs.PathError{Op: "read", Path: p, Err: fs.ErrNotExist}
	}
	return io.NopCloser(bytes.NewReader(file.data)), nil
}

func (f *memFiler) Delete(_ context.Context, p string, _ ...filer.DeleteMode) error {
	f.mu.Lock()
	defer f.mu.Unlock()
	if _, ok := f.files[memKey(p)]; !ok {
		return &fs.PathError{Op: "delete", Path: p, Err: fs.ErrNotExist}
	}
	delete(f.files, memKey(p))
	return nil
}

func (f *memFiler) Stat(_ context.Context, p string) (fs.FileInfo, error) {
	f.mu.Lock()
	defer f.mu.Unlock()
	file, ok := f.files[memKey(p)]
	if !ok {
		return nil, &fs.PathError{Op: "stat", Path: p, Err: fs.ErrNotExist}
	}
	return memInfo{name: path.Base(memKey(p)), size: int64(len(file.data)), mtime: file.mtime}, nil
}

func (f *memFiler) ReadDir(ctx context.Context, dir string) ([]fs.DirEntry, error) {
	if dir == f.blockDir {
		<-ctx.Done()
		return nil, ctx.Err()
	}
	f.mu.Lock()
	defer f.mu.Unlock()
	parent := memKey(dir)
	var entries []fs.DirEntry
	for k, file := range f.files {
		if path.Dir(k) != parent {
			continue
		}
		entries = append(entries, memEntry{info: memInfo{name: path.Base(k), size: int64(len(file.data)), mtime: file.mtime}})
	}
	slices.SortFunc(entries, func(a, b fs.DirEntry) int { return strings.Compare(a.Name(), b.Name()) })
	return entries, nil
}

func (f *memFiler) Mkdir(context.Context, string) error { return nil }

// names returns the keys currently in the store, sorted.
func (f *memFiler) names() []string {
	f.mu.Lock()
	defer f.mu.Unlock()
	var out []string
	for k := range f.files {
		out = append(out, k)
	}
	slices.Sort(out)
	return out
}

type memInfo struct {
	name  string
	size  int64
	mtime time.Time
}

func (i memInfo) Name() string       { return i.name }
func (i memInfo) Size() int64        { return i.size }
func (i memInfo) Mode() fs.FileMode  { return 0o644 }
func (i memInfo) ModTime() time.Time { return i.mtime }
func (i memInfo) IsDir() bool        { return false }
func (i memInfo) Sys() any           { return nil }

type memEntry struct{ info memInfo }

func (e memEntry) Name() string               { return e.info.name }
func (e memEntry) IsDir() bool                { return false }
func (e memEntry) Type() fs.FileMode          { return 0 }
func (e memEntry) Info() (fs.FileInfo, error) { return e.info, nil }

// v2TestID is a 32-hex object id for the fixtures.
func v2TestID(n int) string {
	return fmt.Sprintf("%032x", n)
}

// seedObject writes an object as of the clock's current time.
func seedObject(t *testing.T, f *memFiler, id string) {
	t.Helper()
	require.NoError(t, f.Write(t.Context(), v2ObjectName(id), bytes.NewReader([]byte("gz")), filer.OverwriteIfExists, filer.CreateParentDirectories))
}

// seedManifest writes a v2 manifest that references ids, as of the clock's current time.
func seedManifest(t *testing.T, f *memFiler, name string, ids ...string) {
	t.Helper()
	m := v2Manifest{Format: v2ManifestFormat, Version: v2ManifestVersion, Component: "code", K: 1}
	for _, id := range ids {
		m.Members = append(m.Members, v2Member{ID: id, GzSize: 2, TarSize: 1024})
	}
	data, err := json.Marshal(m)
	require.NoError(t, err)
	require.NoError(t, f.Write(t.Context(), name, bytes.NewReader(data), filer.OverwriteIfExists, filer.CreateParentDirectories))
}

func newGCClock() *fakeClock {
	return &fakeClock{now: time.Date(2026, 11, 1, 12, 0, 0, 0, time.UTC)}
}

// Constants must keep the reuse window below GC age by the margin the safety argument uses.
func TestV2GCWindowsKeepTheSafetyMargin(t *testing.T) {
	assert.Equal(t, 7*24*time.Hour, v2ReuseWindow)
	assert.Equal(t, 8*24*time.Hour, v2GCAge)
	assert.Equal(t, 30*24*time.Hour, v2ManifestLease)
	assert.Equal(t, 24*time.Hour, v2GCInterval)
	assert.Equal(t, 24*time.Hour, v2GCAge-v2ReuseWindow)
}

// An object reused at time T has an mtime above T - R, so it cannot be deleted before
// T + (A - R). A GC run that lists the store in that window keeps it, even before the reusing
// submit writes its manifest.
func TestV2GCRaceReusedObjectNotDeletableBeforeMargin(t *testing.T) {
	clock := newGCClock()
	f := newMemFiler(clock)
	id := v2TestID(1)

	// The object was written just inside the reuse window when the submit at T reuses it.
	reuse := clock.Now()
	clock.now = reuse.Add(-v2ReuseWindow + time.Second)
	seedObject(t, f, id)
	clock.now = reuse

	deadline := reuse.Add(v2GCAge - v2ReuseWindow)
	for _, at := range []time.Duration{0, time.Hour, 12 * time.Hour, deadline.Sub(reuse) - time.Minute} {
		plan, err := planV2GC(t.Context(), f, reuse.Add(at))
		require.NoError(t, err)
		assert.NotContains(t, plan.Objects, v2ObjectName(id), "deleted too early at +%v", at)
	}
	// Past the margin the same object is deletable, which bounds the window from above.
	plan, err := planV2GC(t.Context(), f, deadline.Add(2*time.Second))
	require.NoError(t, err)
	assert.Contains(t, plan.Objects, v2ObjectName(id))
}

// A manifest younger than L keeps the objects it references, however old they are. An
// unreferenced object older than A is deleted.
func TestV2GCLeasedManifestKeepsItsObjects(t *testing.T) {
	clock := newGCClock()
	f := newMemFiler(clock)
	kept, orphan := v2TestID(1), v2TestID(2)

	seedObject(t, f, kept)
	seedObject(t, f, orphan)
	clock.Advance(20 * 24 * time.Hour)
	seedManifest(t, f, "code_aaaaaaaaaaaaaaaa.air.json", kept)
	clock.Advance(v2GCAge) // objects are now 28 days old, past A

	plan, err := planV2GC(t.Context(), f, clock.Now())
	require.NoError(t, err)
	assert.Empty(t, plan.Manifests)
	assert.Equal(t, []string{v2ObjectName(orphan)}, plan.Objects)
	assert.Equal(t, 1, plan.ObjectsKept)

	require.NoError(t, applyV2GC(t.Context(), f, plan, clock.Now()))
	assert.Contains(t, f.names(), v2ObjectName(kept))
	assert.NotContains(t, f.names(), v2ObjectName(orphan))
}

// A manifest older than L is deleted, and its objects go with it once older than A.
func TestV2GCAbandonedManifestAndItsObjectsAreDeleted(t *testing.T) {
	clock := newGCClock()
	f := newMemFiler(clock)
	id := v2TestID(3)
	seedObject(t, f, id)
	seedManifest(t, f, "code_bbbbbbbbbbbbbbbb.air.json", id)
	clock.Advance(v2ManifestLease + time.Hour)

	plan, err := planV2GC(t.Context(), f, clock.Now())
	require.NoError(t, err)
	assert.Equal(t, []string{"code_bbbbbbbbbbbbbbbb.air.json"}, plan.Manifests)
	assert.Equal(t, []string{v2ObjectName(id)}, plan.Objects)

	require.NoError(t, applyV2GC(t.Context(), f, plan, clock.Now()))
	assert.Empty(t, f.names())
}

// An object older than A with no manifest is an orphan from an interrupted submit: deleted. A
// younger one is kept.
func TestV2GCOrphansDeletedAfterGCAge(t *testing.T) {
	clock := newGCClock()
	f := newMemFiler(clock)
	old, young := v2TestID(4), v2TestID(5)
	seedObject(t, f, old)
	clock.Advance(v2GCAge + time.Hour)
	seedObject(t, f, young)
	clock.Advance(v2GCAge - 2*time.Hour)

	plan, err := planV2GC(t.Context(), f, clock.Now())
	require.NoError(t, err)
	assert.Equal(t, []string{v2ObjectName(old)}, plan.Objects)
	assert.Equal(t, 1, plan.ObjectsKept)
}

// Nothing outside the v2 layout is touched: v1 tarballs and sidecars, notes, and objects with
// names that are not 32 hex characters. Even ancient ones stay.
func TestV2GCLeavesOtherFilesAlone(t *testing.T) {
	clock := newGCClock()
	f := newMemFiler(clock)
	others := []string{
		"code_0123456789abcdef.tar.gz",
		"code_0123456789abcdef.tar.gz.idx.json",
		"notes.txt",
		"code_0123456789abcdef.air.json.bak",
		"objects/notahexnameatall.tar.gz",
		"objects/" + v2TestID(6) + ".gz",
		"objects/sub/" + v2TestID(7) + ".tar.gz",
	}
	for _, name := range others {
		require.NoError(t, f.Write(t.Context(), name, bytes.NewReader([]byte("x")), filer.CreateParentDirectories))
	}
	seedObject(t, f, v2TestID(8))
	clock.Advance(365 * 24 * time.Hour)

	plan, err := planV2GC(t.Context(), f, clock.Now())
	require.NoError(t, err)
	require.NoError(t, applyV2GC(t.Context(), f, plan, clock.Now()))

	assert.Equal(t, []string{v2ObjectName(v2TestID(8))}, plan.Objects)
	for _, name := range others {
		assert.Contains(t, f.names(), name)
	}
}

// A manifest renewed between the plan and the apply is kept, and so is an object rewritten by a
// submit in that window.
func TestV2GCApplyRechecksAgeBeforeDelete(t *testing.T) {
	clock := newGCClock()
	f := newMemFiler(clock)
	id := v2TestID(9)
	seedObject(t, f, id)
	seedManifest(t, f, "code_cccccccccccccccc.air.json")
	clock.Advance(v2ManifestLease + time.Hour)

	plan, err := planV2GC(t.Context(), f, clock.Now())
	require.NoError(t, err)
	require.Contains(t, plan.Objects, v2ObjectName(id))
	require.Contains(t, plan.Manifests, "code_cccccccccccccccc.air.json")

	// A submit rewrites the object and the manifest before the apply runs.
	seedObject(t, f, id)
	seedManifest(t, f, "code_cccccccccccccccc.air.json")
	require.NoError(t, applyV2GC(t.Context(), f, plan, clock.Now()))
	assert.Contains(t, f.names(), v2ObjectName(id))
	assert.Contains(t, f.names(), "code_cccccccccccccccc.air.json")
}

// Off by default: with the variable unset the hook returns before touching the workspace, so
// nothing is listed or deleted. Any value that is not a true-like string is off too.
func TestV2GCOffByDefaultDoesNothing(t *testing.T) {
	t.Setenv(snapshotGCEnv, "")
	assert.False(t, gcEnabled(t.Context()))

	cfg := &runConfig{CodeSource: &codeSourceConfig{Snapshot: &snapshotSourceConfig{}}}
	// A nil workspace would panic if the hook got past the gate.
	gcAfterSubmit(t.Context(), nil, cfg, "/Workspace/Users/me")
}

func TestV2GCEnabledByEnv(t *testing.T) {
	t.Setenv(snapshotGCEnv, "true")
	assert.True(t, gcEnabled(t.Context()))
	t.Setenv(snapshotGCEnv, "maybe")
	assert.False(t, gcEnabled(t.Context()))
}

// The time box bounds a run: when listing blocks past the deadline, the run returns and
// deletes nothing, since the references it could not read might keep any object.
func TestV2GCTimeBoxStopsWithoutDeleting(t *testing.T) {
	clock := newGCClock()
	f := newMemFiler(clock)
	seedObject(t, f, v2TestID(10))
	clock.Advance(v2GCAge + time.Hour)
	f.blockDir = v2ObjectsDir

	ctx, cancel := context.WithTimeout(t.Context(), 50*time.Millisecond)
	defer cancel()
	start := time.Now()
	runV2GC(ctx, f, clock.Now())
	assert.Less(t, time.Since(start), 2*time.Second)
	assert.Contains(t, f.names(), v2ObjectName(v2TestID(10)))
}

// The GC runs at most once per v2GCInterval per store: a run within the interval of the marker
// does nothing, and one after it runs again.
func TestV2GCMarkerThrottlesRuns(t *testing.T) {
	clock := newGCClock()
	f := newMemFiler(clock)
	base := clock.Now()
	// seedOrphan writes an object that is already older than A at the base time.
	seedOrphan := func(n int) {
		clock.now = base.Add(-(v2GCAge + time.Hour))
		seedObject(t, f, v2TestID(n))
		clock.now = base
	}

	seedOrphan(11)
	runV2GC(t.Context(), f, base)
	assert.NotContains(t, f.names(), v2ObjectName(v2TestID(11)), "the first run is due")

	seedOrphan(12)
	runV2GC(t.Context(), f, base.Add(time.Hour))
	assert.Contains(t, f.names(), v2ObjectName(v2TestID(12)), "throttled within the interval")

	runV2GC(t.Context(), f, base.Add(v2GCInterval+time.Hour))
	assert.NotContains(t, f.names(), v2ObjectName(v2TestID(12)), "runs again after the interval")
}

// A submit reuses an object younger than R and writes a full object again once it is older.
// The writer's reuse rule is the other half of the safety margin.
func TestV2WriterReusesOnlyFreshObjects(t *testing.T) {
	clock := newGCClock()
	f := newMemFiler(clock)
	repo := writeV2Tree(t, v2TestTree(30))
	files, err := snapshotFiles(t.Context(), repo, nil, false)
	require.NoError(t, err)
	uploadPath := "/Workspace/x/.internal"

	var result snapshotResult
	require.NoError(t, uploadSnapshotV2(t.Context(), f, uploadPath, repo, files, &result, clock.Now()))
	objectWrites := func() int {
		n := 0
		for name, c := range f.writes {
			if strings.HasSuffix(name, ".tar.gz") {
				n += c
			}
		}
		return n
	}
	first := objectWrites()
	require.Positive(t, first)

	// Drop the manifest so the next submit takes the full path and considers each object.
	require.NoError(t, f.Delete(t.Context(), path.Base(result.CodeSourcePath)))
	clock.Advance(v2ReuseWindow - time.Hour)
	require.NoError(t, uploadSnapshotV2(t.Context(), f, uploadPath, repo, files, &result, clock.Now()))
	assert.Equal(t, first, objectWrites(), "fresh objects are reused")

	require.NoError(t, f.Delete(t.Context(), path.Base(result.CodeSourcePath)))
	clock.Advance(2 * time.Hour)
	require.NoError(t, uploadSnapshotV2(t.Context(), f, uploadPath, repo, files, &result, clock.Now()))
	assert.Equal(t, 2*first, objectWrites(), "objects older than R are written again")
}

// A rerun of an unchanged snapshot renews the manifest's mtime, which is its lease.
func TestV2RenewalRefreshesManifestMtime(t *testing.T) {
	clock := newGCClock()
	f := newMemFiler(clock)
	repo := writeV2Tree(t, v2TestTree(30))
	files, err := snapshotFiles(t.Context(), repo, nil, false)
	require.NoError(t, err)
	uploadPath := "/Workspace/x/.internal"

	var result snapshotResult
	require.NoError(t, uploadSnapshotV2(t.Context(), f, uploadPath, repo, files, &result, clock.Now()))
	name := path.Base(result.CodeSourcePath)

	clock.Advance(20 * 24 * time.Hour)
	require.NoError(t, uploadSnapshotV2(t.Context(), f, uploadPath, repo, files, &result, clock.Now()))
	info, err := f.Stat(t.Context(), name)
	require.NoError(t, err)
	assert.True(t, info.ModTime().Equal(clock.Now()), "manifest mtime %v, want %v", info.ModTime(), clock.Now())
}

// An unchanged rerun whose object is missing rewrites it, rather than renewing a manifest that
// points at nothing.
func TestV2RenewalRepairsMissingObject(t *testing.T) {
	clock := newGCClock()
	f := newMemFiler(clock)
	repo := writeV2Tree(t, v2TestTree(30))
	files, err := snapshotFiles(t.Context(), repo, nil, false)
	require.NoError(t, err)
	uploadPath := "/Workspace/x/.internal"

	var result snapshotResult
	require.NoError(t, uploadSnapshotV2(t.Context(), f, uploadPath, repo, files, &result, clock.Now()))
	var manifest v2Manifest
	rc, err := f.Read(t.Context(), path.Base(result.CodeSourcePath))
	require.NoError(t, err)
	require.NoError(t, json.NewDecoder(rc).Decode(&manifest))
	rc.Close()
	victim := v2ObjectName(manifest.Members[0].ID)
	require.NoError(t, f.Delete(t.Context(), victim))

	require.NoError(t, uploadSnapshotV2(t.Context(), f, uploadPath, repo, files, &result, clock.Now()))
	assert.Contains(t, f.names(), victim)
	// Written once by the first submit and once more to repair it.
	assert.Equal(t, 2, f.writes[path.Base(victim)])
}
