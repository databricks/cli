package aircmd

// Snapshot format v2 stores a plain_tar working tree as content-addressed members and a
// manifest, so an iteration re-uploads only the members whose files changed. The launcher
// side reads the manifest; the CLI emits v2 only when the backend advertises
// codeSnapshotV2Capability, and otherwise keeps writing v1 (see snapshot_upload.go).

import (
	"archive/tar"
	"bytes"
	"context"
	"crypto/sha256"
	"encoding/binary"
	"encoding/hex"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"io/fs"
	"os"
	"path"
	"path/filepath"
	"runtime"
	"slices"
	"strings"
	"sync/atomic"
	"time"

	"github.com/databricks/cli/libs/filer"
	"github.com/databricks/cli/libs/log"
	"golang.org/x/sync/errgroup"
)

// codeSnapshotV2Capability is the config:validate capability that enables v2.
const codeSnapshotV2Capability = "code_snapshot_v2"

// v2Decision reports whether the backend advertised v2. It blocks until config:validate has
// finished, so it is only called where the answer is needed.
type v2Decision func(ctx context.Context) (bool, error)

const (
	v2Buckets           = 16
	v2ManifestFormat    = "air-snapshot-manifest"
	v2ManifestVersion   = 2
	v2ManifestSuffix    = ".air.json"
	v2ObjectsDir        = "objects"
	v2UploadConcurrency = 8
	v2UploadAttempts    = 3
	v2UploadBackoff     = 200 * time.Millisecond
	v2KStepBytes        = 16 << 20
	v2K16StepBytes      = 128 << 20
	v2EndTarBytes       = 2 * tarBlockSize
)

// v2Manifest is the object at code_source_path. Members are in archive order and end with
// the end-of-archive member, so concatenating their gzip objects gives a valid v1 .tar.gz.
type v2Manifest struct {
	Format    string     `json:"format"`
	Version   int        `json:"version"`
	Component string     `json:"component"`
	K         int        `json:"k"`
	TarSize   int64      `json:"tar_size"`
	Members   []v2Member `json:"members"`
}

// v2Member is one object: the first 128 bits of sha256 over its uncompressed tar fragment.
type v2Member struct {
	ID      string `json:"id"`
	GzSize  int64  `json:"gz_size"`
	TarSize int64  `json:"tar_size"`

	fragment string // local uncompressed fragment, not part of the manifest
}

// v2Bucket assigns a file to one of v2Buckets by the hash of its path, so a file keeps its
// bucket across snapshots.
func v2Bucket(rel string) int {
	sum := sha256.Sum256([]byte(rel))
	return int(binary.BigEndian.Uint64(sum[:8]) % v2Buckets)
}

// kForTarSize returns how many members a snapshot of tarSize bytes uses, never fewer than
// prevK so that a shrinking repo keeps its members.
func kForTarSize(tarSize int64, prevK int) int {
	k := 1
	switch {
	case tarSize >= v2K16StepBytes:
		k = 16
	case tarSize >= v2KStepBytes:
		k = 4
	}
	return max(k, prevK)
}

// memberOf maps a bucket to one of k members. Buckets merge evenly, so each member of a
// k=4 snapshot holds whole buckets of the k=16 layout.
func memberOf(bucket, k int) int {
	return bucket / (v2Buckets / k)
}

// v2Header is the tar header for info. It carries the same owner fields as v1, so extraction
// restores the same ownership. Ids therefore depend on the machine that packaged the tree,
// and another machine re-uploads once.
func v2Header(info fs.FileInfo, name, linkTarget string) (*tar.Header, error) {
	hdr, err := tar.FileInfoHeader(info, linkTarget)
	if err != nil {
		return nil, err
	}
	hdr.Name = name
	return hdr, nil
}

// writeV2Fragment writes the tar entries for files to fragmentPath and returns the member id
// and the fragment's size. The fragment has complete entries only: no end-of-archive marker
// and no record padding, so it can be concatenated with other fragments.
func writeV2Fragment(fragmentPath, repoPath, component string, files []snapshotFile) (string, int64, error) {
	f, err := os.Create(fragmentPath)
	if err != nil {
		return "", 0, err
	}
	defer f.Close()

	hash := sha256.New()
	tw := tar.NewWriter(io.MultiWriter(f, hash))
	for _, file := range files {
		if err := writeV2Entry(tw, repoPath, component, file); err != nil {
			return "", 0, err
		}
	}
	if err := tw.Flush(); err != nil {
		return "", 0, fmt.Errorf("failed to write snapshot member: %w", err)
	}
	info, err := f.Stat()
	if err != nil {
		return "", 0, err
	}
	return hex.EncodeToString(hash.Sum(nil)[:16]), info.Size(), nil
}

// writeV2Entry adds one file to the tar stream, matching tarpack's rules: regular files and
// symlinks are archived, and other file types are skipped.
func writeV2Entry(tw *tar.Writer, repoPath, component string, file snapshotFile) error {
	name := filepath.Join(repoPath, file.rel)
	info, err := os.Lstat(name)
	if err != nil {
		return fmt.Errorf("stat %s: %w", name, err)
	}
	mode := info.Mode()
	if !mode.IsRegular() && mode&fs.ModeSymlink == 0 {
		return nil
	}
	link := ""
	if mode&fs.ModeSymlink != 0 {
		if link, err = os.Readlink(name); err != nil {
			return fmt.Errorf("readlink %s: %w", name, err)
		}
	}
	hdr, err := v2Header(info, path.Join(component, filepath.ToSlash(file.rel)), link)
	if err != nil {
		return fmt.Errorf("tar header for %s: %w", name, err)
	}
	if err := tw.WriteHeader(hdr); err != nil {
		return fmt.Errorf("write header for %s: %w", name, err)
	}
	if !mode.IsRegular() {
		return nil
	}
	f, err := os.Open(name)
	if err != nil {
		return fmt.Errorf("open %s: %w", name, err)
	}
	defer f.Close()
	if _, err := io.Copy(tw, f); err != nil {
		return fmt.Errorf("write %s: %w", name, err)
	}
	return nil
}

// buildV2Members groups files into k members and writes one fragment per non-empty member
// in tmp. Members are returned in index order. Empty members are omitted because a gzip
// member holding no tar entries is not a valid tar archive.
func buildV2Members(ctx context.Context, tmp, repoPath, component string, files []snapshotFile, k int) ([]v2Member, error) {
	groups := make([][]snapshotFile, k)
	for _, file := range files {
		m := memberOf(v2Bucket(filepath.ToSlash(file.rel)), k)
		groups[m] = append(groups[m], file)
	}
	for _, group := range groups {
		slices.SortFunc(group, func(a, b snapshotFile) int {
			return strings.Compare(filepath.ToSlash(a.rel), filepath.ToSlash(b.rel))
		})
	}

	members := make([]v2Member, k)
	var g errgroup.Group
	g.SetLimit(max(1, min(k, runtime.NumCPU())))
	for i := range k {
		g.Go(func() error {
			fragment := filepath.Join(tmp, fmt.Sprintf("fragment-%d.tar", i))
			id, size, err := writeV2Fragment(fragment, repoPath, component, groups[i])
			if err != nil {
				return err
			}
			members[i] = v2Member{ID: id, TarSize: size, fragment: fragment}
			return nil
		})
	}
	if err := g.Wait(); err != nil {
		return nil, err
	}
	return slices.DeleteFunc(members, func(m v2Member) bool { return m.TarSize == 0 }), nil
}

// writeV2EndFragment writes the constant end-of-archive member: two zero blocks.
func writeV2EndFragment(tmp string) (v2Member, error) {
	fragment := filepath.Join(tmp, "end.tar")
	if err := os.WriteFile(fragment, make([]byte, v2EndTarBytes), 0o644); err != nil {
		return v2Member{}, err
	}
	sum := sha256.Sum256(make([]byte, v2EndTarBytes))
	return v2Member{ID: hex.EncodeToString(sum[:16]), TarSize: v2EndTarBytes, fragment: fragment}, nil
}

// v2ObjectName is the object path of a member, relative to the snapshot directory.
func v2ObjectName(id string) string {
	return path.Join(v2ObjectsDir, id+".tar.gz")
}

// putV2Members makes every member an object in store and records its gzip size. At most
// v2UploadConcurrency members are compressed or uploaded at once. now is the time the snapshot
// is written, against which object age is judged.
func putV2Members(ctx context.Context, store filer.Filer, members []v2Member, tmp string, now time.Time) error {
	g, gctx := errgroup.WithContext(ctx)
	g.SetLimit(v2UploadConcurrency)
	for i := range members {
		g.Go(func() error {
			return putV2Member(gctx, store, &members[i], tmp, now)
		})
	}
	return g.Wait()
}

// putV2Member writes one member as an object unless a fresh copy already exists. An object
// younger than v2ReuseWindow is reused, since its id is its content. An older one is written
// again, which refreshes its mtime and keeps GC from deleting it under a reusing submit.
func putV2Member(ctx context.Context, store filer.Filer, m *v2Member, tmp string, now time.Time) error {
	name := v2ObjectName(m.ID)
	info, err := store.Stat(ctx, name)
	if err == nil && now.Sub(info.ModTime()) < v2ReuseWindow {
		m.GzSize = info.Size()
		return nil
	}
	if err != nil && !errors.Is(err, fs.ErrNotExist) {
		return fmt.Errorf("failed to check snapshot object %s: %w", name, err)
	}

	gzPath := filepath.Join(tmp, m.ID+".tar.gz")
	if err := compressRaw(ctx, m.fragment, gzPath); err != nil {
		return fmt.Errorf("failed to compress snapshot member %s: %w", m.ID, err)
	}
	info, err = os.Stat(gzPath)
	if err != nil {
		return err
	}
	m.GzSize = info.Size()
	return withV2Retry(ctx, func() error {
		f, err := os.Open(gzPath)
		if err != nil {
			return err
		}
		defer f.Close()
		return store.Write(ctx, name, f, filer.OverwriteIfExists, filer.CreateParentDirectories)
	})
}

// renewV2Manifest renews an existing manifest whose objects are all present, by rewriting it
// unchanged. It reports false when the manifest is missing, unreadable, or references a missing
// object. The snapshot is then written in full, which also repairs a damaged manifest.
func renewV2Manifest(ctx context.Context, store filer.Filer, name string) (v2Manifest, bool, error) {
	rc, err := store.Read(ctx, name)
	if errors.Is(err, fs.ErrNotExist) {
		return v2Manifest{}, false, nil
	}
	if err != nil {
		return v2Manifest{}, false, fmt.Errorf("failed to read snapshot manifest %s: %w", name, err)
	}
	data, err := io.ReadAll(rc)
	rc.Close()
	if err != nil {
		return v2Manifest{}, false, fmt.Errorf("failed to read snapshot manifest %s: %w", name, err)
	}
	var manifest v2Manifest
	if err := json.Unmarshal(data, &manifest); err != nil {
		log.Debugf(ctx, "snapshot manifest %s is unreadable, rewriting it: %v", name, err)
		return v2Manifest{}, false, nil
	}

	var missing atomic.Bool
	g, gctx := errgroup.WithContext(ctx)
	g.SetLimit(v2UploadConcurrency)
	for _, m := range manifest.Members {
		g.Go(func() error {
			_, err := store.Stat(gctx, v2ObjectName(m.ID))
			if errors.Is(err, fs.ErrNotExist) {
				missing.Store(true)
				return nil
			}
			return err
		})
	}
	if err := g.Wait(); err != nil {
		return v2Manifest{}, false, fmt.Errorf("failed to check snapshot objects: %w", err)
	}
	if missing.Load() {
		return v2Manifest{}, false, nil
	}
	if err := store.Write(ctx, name, bytes.NewReader(data), filer.OverwriteIfExists, filer.CreateParentDirectories); err != nil {
		return v2Manifest{}, false, fmt.Errorf("failed to renew snapshot manifest %s: %w", name, err)
	}
	return manifest, true, nil
}

// withV2Retry runs op up to v2UploadAttempts times with exponential backoff. Context
// cancellation is never retried.
func withV2Retry(ctx context.Context, op func() error) error {
	var err error
	for attempt := range v2UploadAttempts {
		if err = op(); err == nil {
			return nil
		}
		if ctx.Err() != nil || attempt == v2UploadAttempts-1 {
			break
		}
		log.Debugf(ctx, "snapshot upload attempt %d failed, retrying: %v", attempt+1, err)
		select {
		case <-time.After(v2UploadBackoff << attempt):
		case <-ctx.Done():
			return ctx.Err()
		}
	}
	return err
}

// previousV2K returns the member count of the newest manifest for component in store, or 0
// when there is none. K only ratchets up, so the newest manifest is enough.
func previousV2K(ctx context.Context, store filer.Filer, component string) (int, error) {
	entries, err := store.ReadDir(ctx, ".")
	if errors.Is(err, fs.ErrNotExist) {
		return 0, nil
	}
	if err != nil {
		return 0, fmt.Errorf("failed to list snapshots: %w", err)
	}
	var newest fs.DirEntry
	var newestTime time.Time
	for _, entry := range entries {
		name := entry.Name()
		if !strings.HasPrefix(name, component+"_") || !strings.HasSuffix(name, v2ManifestSuffix) {
			continue
		}
		info, err := entry.Info()
		if err != nil {
			continue
		}
		if newest == nil || info.ModTime().After(newestTime) {
			newest, newestTime = entry, info.ModTime()
		}
	}
	if newest == nil {
		return 0, nil
	}
	rc, err := store.Read(ctx, newest.Name())
	if err != nil {
		return 0, fmt.Errorf("failed to read snapshot manifest %s: %w", newest.Name(), err)
	}
	defer rc.Close()
	var manifest v2Manifest
	if err := json.NewDecoder(rc).Decode(&manifest); err != nil {
		return 0, fmt.Errorf("failed to parse snapshot manifest %s: %w", newest.Name(), err)
	}
	return manifest.K, nil
}

// uploadSnapshotV2 writes the working-tree snapshot in v2 form and sets result from it. The
// manifest is written last, after every object succeeded, so a visible manifest always refers
// to complete objects. An unchanged snapshot is found by its manifest name alone.
func uploadSnapshotV2(ctx context.Context, store filer.Filer, uploadPath, repoPath string, files []snapshotFile, result *snapshotResult, now time.Time) error {
	component := filepath.Base(repoPath)
	manifestName := component + "_" + computePlainTarKey(files)[:16] + v2ManifestSuffix
	remote := path.Join(uploadPath, manifestName)

	// An unchanged snapshot renews its manifest's lease when all its objects are present. Renewal
	// is a rewrite, which is what keeps GC from treating the manifest as abandoned.
	previous, renewed, err := renewV2Manifest(ctx, store, manifestName)
	if err != nil {
		return err
	}
	if renewed {
		log.Debugf(ctx, "snapshot upload skipped; reusing %s", remote)
		var gzTotal int64
		for _, m := range previous.Members {
			gzTotal += m.GzSize
		}
		result.CodeSourcePath = remote
		result.PackagingDurationMs = new(int64(0))
		result.UploadDurationMs = new(int64(0))
		result.SizeBytes = &gzTotal
		return nil
	}

	prevK, err := previousV2K(ctx, store, component)
	if err != nil {
		return err
	}
	tarTotal := int64(v2EndTarBytes)
	for _, file := range files {
		tarTotal += file.tarSize()
	}
	k := kForTarSize(tarTotal, prevK)

	tmp, err := os.MkdirTemp("", "air-snapshot-v2-*")
	if err != nil {
		return err
	}
	defer os.RemoveAll(tmp)

	packagingStart := time.Now()
	members, err := buildV2Members(ctx, tmp, repoPath, component, files, k)
	if err != nil {
		return err
	}
	end, err := writeV2EndFragment(tmp)
	if err != nil {
		return err
	}
	members = append(members, end)
	result.PackagingDurationMs = new(time.Since(packagingStart).Milliseconds())

	uploadStart := time.Now()
	if err := putV2Members(ctx, store, members, tmp, now); err != nil {
		return err
	}
	manifest := v2Manifest{Format: v2ManifestFormat, Version: v2ManifestVersion, Component: component, K: k, Members: members}
	var gzTotal int64
	for _, m := range members {
		manifest.TarSize += m.TarSize
		gzTotal += m.GzSize
	}
	data, err := json.Marshal(manifest)
	if err != nil {
		return fmt.Errorf("failed to encode snapshot manifest: %w", err)
	}
	if err := withV2Retry(ctx, func() error {
		return store.Write(ctx, manifestName, bytes.NewReader(data), filer.OverwriteIfExists, filer.CreateParentDirectories)
	}); err != nil {
		return fmt.Errorf("failed to upload snapshot manifest %s: %w", manifestName, err)
	}
	result.UploadDurationMs = new(time.Since(uploadStart).Milliseconds())
	result.SizeBytes = &gzTotal
	result.CodeSourcePath = remote
	return nil
}
