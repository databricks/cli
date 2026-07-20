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

// tarEpoch is stamped on every entry so identical content yields identical bytes
// (and SHA-256) regardless of mtimes, which is what makes the archive content-addressed.
var tarEpoch = time.Date(2000, 1, 1, 0, 0, 0, 0, time.UTC)

// appleDoublePrefix marks macOS AppleDouble metadata files, excluded to match the AIR CLI.
const appleDoublePrefix = "._"

// buildCodeSnapshot writes a reproducible gzipped tarball of files to out and returns
// its SHA-256. Each file at "<relBase>/<rest>" becomes "<prefix>/<rest>", so the archive
// expands to <prefix>/... matching the runtime's /databricks/code_source/<dir> contract.
func buildCodeSnapshot(syncRoot vfs.Path, relBase string, files []fileset.File, prefix string, out io.Writer) (string, error) {
	// Sort by relative path so the byte stream (and hash) is order-independent.
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
	// Re-base f.Relative (relative to syncRoot) under the code dir so it nests under prefix.
	rel := f.Relative
	if relBase != "." {
		trimmed, ok := strings.CutPrefix(rel, relBase+"/")
		if !ok {
			return nil // outside the code dir; the file list is scoped to it, so skip defensively
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

	// Only regular files are archived (symlinks are out of scope).
	if !info.Mode().IsRegular() {
		return nil
	}

	hdr := &tar.Header{
		Typeflag: tar.TypeReg,
		Name:     path.Join(prefix, rel),
		Size:     info.Size(),
		// Fixed mode + mtime for reproducibility; code runs via an interpreter, so exec bits don't matter.
		Mode:    0o644,
		ModTime: tarEpoch,
	}
	if err := tw.WriteHeader(hdr); err != nil {
		return fmt.Errorf("tar header for %s: %w", rel, err)
	}
	if _, err := io.Copy(tw, rc); err != nil {
		return fmt.Errorf("write %s: %w", rel, err)
	}
	return nil
}
