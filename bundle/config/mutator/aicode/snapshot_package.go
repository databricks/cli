package aicode

import (
	"archive/tar"
	"compress/gzip"
	"crypto/sha256"
	"encoding/hex"
	"fmt"
	"io"
	"path"
	"slices"
	"strings"
	"time"

	"github.com/databricks/cli/libs/fileset"
	"github.com/databricks/cli/libs/vfs"
)

// tarEpoch is a fixed modification time stamped on every tar entry so the archive
// is content-addressed: identical file contents always produce identical bytes
// (and therefore an identical SHA-256), regardless of file mtimes or when the
// archive was built. This is what lets an unchanged code directory resolve to the
// same uploaded filename across deploys and skip re-upload. The technique mirrors
// bundle/deploy/snapshot/path.go (which does the same for the immutable-folder zip).
// Reproducible per platform, not across them (see addFileToArchive).
var tarEpoch = time.Date(2000, 1, 1, 0, 0, 0, 0, time.UTC)

// appleDoublePrefix is the basename prefix of macOS AppleDouble metadata files.
// The AIR CLI excludes these; we match it so macOS archives carry no extra entries.
const appleDoublePrefix = "._"

// BuildCodeSnapshot writes a reproducible gzipped tarball of the given files to out
// and returns its SHA-256 hex digest. syncRoot is the root the files' Relative paths
// are against (the bundle sync root); relBase is the code directory relative to that
// root; prefix is the archive's top-level directory name. Each file at
// "<relBase>/<rest>" is written to the archive as "<prefix>/<rest>", so the archive
// expands to <prefix>/... — matching the runtime's /databricks/code_source/<dir>
// extraction contract.
//
// The file list is produced by the bundle's sync walker, so it honors .gitignore
// (including nested files) and the top-level sync.include/exclude globs — the same
// filtering as bundle file sync.
func BuildCodeSnapshot(syncRoot vfs.Path, relBase string, files []fileset.File, prefix string, out io.Writer) (string, error) {
	// Sort by relative path so the archive byte stream (and thus its hash) does not
	// depend on iteration order.
	slices.SortFunc(files, func(a, b fileset.File) int {
		return strings.Compare(a.Relative, b.Relative)
	})

	hash := sha256.New()
	gzw := gzip.NewWriter(io.MultiWriter(out, hash))
	tw := tar.NewWriter(gzw)

	for _, f := range files {
		if err := addFileToArchive(tw, syncRoot, relBase, f, prefix); err != nil {
			return "", err
		}
	}

	if err := tw.Close(); err != nil {
		return "", err
	}
	if err := gzw.Close(); err != nil {
		return "", err
	}
	return hex.EncodeToString(hash.Sum(nil)), nil
}

func addFileToArchive(tw *tar.Writer, syncRoot vfs.Path, relBase string, f fileset.File, prefix string) error {
	// f.Relative is relative to syncRoot and slash-separated. Re-base it under the
	// code directory so the entry nests under the archive prefix.
	rel := f.Relative
	if relBase != "." {
		trimmed, ok := strings.CutPrefix(rel, relBase+"/")
		if !ok {
			// Not under the code dir; the sync file list is scoped to it, so this
			// should not happen, but skip defensively rather than mis-place a file.
			return nil
		}
		rel = trimmed
	}

	if strings.HasPrefix(path.Base(rel), appleDoublePrefix) {
		return nil
	}

	rc, err := syncRoot.Open(f.Relative)
	if err != nil {
		return fmt.Errorf("open %s: %w", f.Relative, err)
	}
	defer rc.Close()

	info, err := rc.Stat()
	if err != nil {
		return fmt.Errorf("stat %s: %w", f.Relative, err)
	}

	// Only regular files are archived. The walker never yields directories, and
	// symlinks inside a code snapshot are out of scope.
	if !info.Mode().IsRegular() {
		return nil
	}

	// Preserve the owner execute bit so a bundled helper stays executable, and
	// normalize the rest to a canonical mode. Windows has no execute bit, so files are
	// archived 0644 there and the archive hash differs from a Unix-built one.
	mode := int64(0o644)
	if info.Mode().Perm()&0o100 != 0 {
		mode = 0o755
	}
	hdr := &tar.Header{
		Typeflag: tar.TypeReg,
		Name:     path.Join(prefix, rel),
		Size:     info.Size(),
		Mode:     mode,
		ModTime:  tarEpoch,
	}
	if err := tw.WriteHeader(hdr); err != nil {
		return fmt.Errorf("tar header for %s: %w", rel, err)
	}
	if _, err := io.Copy(tw, rc); err != nil {
		return fmt.Errorf("write %s: %w", rel, err)
	}
	return nil
}
