package aicode

import (
	"archive/tar"
	"compress/gzip"
	"context"
	"crypto/sha256"
	"encoding/hex"
	"fmt"
	"io"
	"path"
	"slices"
	"strings"
	"time"

	"github.com/databricks/cli/libs/fileset"
	"github.com/databricks/cli/libs/git"
	"github.com/databricks/cli/libs/vfs"
)

// tarEpoch is a fixed modification time stamped on every tar entry so the
// archive is content-addressed: identical file contents always produce an
// identical archive (and therefore an identical SHA-256) regardless of the
// files' mtimes or when the archive was built. This is what makes the cache
// key in packageAndUpload stable across deploys and keeps acceptance output
// deterministic.
var tarEpoch = time.Date(2000, 1, 1, 0, 0, 0, 0, time.UTC)

// appleDoublePrefix is the basename prefix of macOS AppleDouble metadata files.
// The AIR CLI excludes these (they sort before the real top-level entry and add
// useless per-file metadata); we match that behavior so archives built on macOS
// and Linux are identical.
const appleDoublePrefix = "._"

// buildTarball writes a reproducible gzipped tarball of the directory rooted at
// codeRoot to out and returns its SHA-256 hex digest. Every entry is prefixed
// with prefix (the code directory's basename) so the archive expands to
// <prefix>/... on extraction, matching the AIR CLI's `--prefix=<dir>/` layout
// and the runtime's /databricks/code_source/<dir> contract.
//
// The file list is gitignore-aware: it honors the code directory's .gitignore
// and always excludes .git and .databricks (via [git.NewFileSetAtRoot]), the
// same walker that backs bundle file sync.
func buildTarball(ctx context.Context, codeRoot vfs.Path, prefix string, out io.Writer) (string, error) {
	fsys, err := git.NewFileSetAtRoot(ctx, codeRoot)
	if err != nil {
		return "", err
	}

	fileList, err := fsys.Files()
	if err != nil {
		return "", err
	}

	// Sort by relative path so the archive byte stream (and thus its hash) does
	// not depend on filesystem walk order.
	slices.SortFunc(fileList, func(a, b fileset.File) int {
		return strings.Compare(a.Relative, b.Relative)
	})

	hash := sha256.New()
	gzw := gzip.NewWriter(io.MultiWriter(out, hash))
	tw := tar.NewWriter(gzw)

	for _, f := range fileList {
		if err := addFileToTar(tw, codeRoot, f, prefix); err != nil {
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

func addFileToTar(tw *tar.Writer, codeRoot vfs.Path, f fileset.File, prefix string) error {
	// fileset.File.Relative is already slash-separated, so path.Base is correct.
	if strings.HasPrefix(path.Base(f.Relative), appleDoublePrefix) {
		return nil
	}

	rc, err := codeRoot.Open(f.Relative)
	if err != nil {
		return fmt.Errorf("open %s: %w", f.Relative, err)
	}
	defer rc.Close()

	info, err := rc.Stat()
	if err != nil {
		return fmt.Errorf("stat %s: %w", f.Relative, err)
	}

	// Only regular files are archived. The gitignore-aware walker never yields
	// directories, and symlinks inside a code snapshot are out of scope for v1.
	if !info.Mode().IsRegular() {
		return nil
	}

	hdr := &tar.Header{
		Typeflag: tar.TypeReg,
		Name:     path.Join(prefix, f.Relative),
		Size:     info.Size(),
		// Normalize permissions to a fixed 0644 and zero the mtime so the archive
		// is reproducible across machines. Executable bits are intentionally
		// dropped; the runtime invokes code via an interpreter, not by executing
		// files from the snapshot directly.
		Mode:    0o644,
		ModTime: tarEpoch,
	}
	if err := tw.WriteHeader(hdr); err != nil {
		return fmt.Errorf("tar header for %s: %w", f.Relative, err)
	}
	if _, err := io.Copy(tw, rc); err != nil {
		return fmt.Errorf("write %s: %w", f.Relative, err)
	}
	return nil
}
