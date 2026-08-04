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
//
// Reproducible for a given platform, not across platforms: entry modes carry the
// POSIX execute bit, which Windows does not have (see addFileToArchive).
var tarEpoch = time.Date(2000, 1, 1, 0, 0, 0, 0, time.UTC)

// appleDoublePrefix is the basename prefix of macOS AppleDouble metadata files.
// The AIR CLI excludes these; we match it so a macOS archive does not carry entries
// a Linux archive of the same sources lacks.
const appleDoublePrefix = "._"

// buildCodeSnapshot writes a reproducible gzipped tarball of the given files to out
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
func buildCodeSnapshot(syncRoot vfs.Path, relBase string, files []fileset.File, prefix string, out io.Writer) (string, error) {
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

	// Preserve the owner execute bit so a bundled helper the user invokes (e.g. a
	// ./run.sh called from command.sh) stays executable, but normalize the rest to a
	// canonical mode. Deriving the mode from the file's own bits keeps the archive
	// reproducible per platform (same input file -> same mode); mtime is zeroed for
	// the same reason.
	//
	// This is the one part of the archive that is NOT cross-platform identical:
	// Windows has no POSIX execute bit, so every file is archived 0644 there and a
	// Windows-built archive of the same sources hashes differently from a Unix-built
	// one (only affecting a deploy of the same bundle from both platforms, which just
	// re-uploads under a different content-addressed name). A helper that relies on
	// its own execute bit therefore needs to be invoked through its interpreter
	// (`bash run.sh`) to work when deployed from Windows.
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
