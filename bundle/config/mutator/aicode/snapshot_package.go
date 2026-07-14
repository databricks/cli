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

// tarEpoch is a fixed modification time stamped on every tar entry so the archive
// is content-addressed: identical file contents always produce identical bytes
// (and therefore an identical SHA-256), regardless of file mtimes or when the
// archive was built. This is what lets an unchanged code directory resolve to the
// same uploaded filename across deploys and skip re-upload. The technique mirrors
// bundle/deploy/snapshot/path.go (which does the same for the immutable-folder zip).
var tarEpoch = time.Date(2000, 1, 1, 0, 0, 0, 0, time.UTC)

// appleDoublePrefix is the basename prefix of macOS AppleDouble metadata files.
// The AIR CLI excludes these; we match it so archives are identical on macOS and Linux.
const appleDoublePrefix = "._"

// buildCodeSnapshot writes a reproducible gzipped tarball of the directory at
// codeDir to out and returns its SHA-256 hex digest. Every entry is prefixed with
// prefix (the code directory's basename) so the archive expands to <prefix>/... —
// matching the runtime's /databricks/code_source/<dir> extraction contract.
//
// The file list is gitignore-aware via [git.NewFileSetAtRoot]: it honors the code
// directory's .gitignore (including nested .gitignore files) and always excludes
// .git and .databricks — the same walker that backs bundle file sync. This is why
// packaging a large tree only archives the tracked payload rather than venvs,
// caches, and build outputs.
func buildCodeSnapshot(ctx context.Context, codeDir vfs.Path, prefix string, out io.Writer) (string, error) {
	fsys, err := git.NewFileSetAtRoot(ctx, codeDir)
	if err != nil {
		return "", err
	}

	files, err := fsys.Files()
	if err != nil {
		return "", err
	}

	// Sort by relative path so the archive byte stream (and thus its hash) does not
	// depend on filesystem walk order.
	slices.SortFunc(files, func(a, b fileset.File) int {
		return strings.Compare(a.Relative, b.Relative)
	})

	hash := sha256.New()
	gzw := gzip.NewWriter(io.MultiWriter(out, hash))
	tw := tar.NewWriter(gzw)

	for _, f := range files {
		if err := addFileToArchive(tw, codeDir, f, prefix); err != nil {
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

func addFileToArchive(tw *tar.Writer, codeDir vfs.Path, f fileset.File, prefix string) error {
	// fileset.File.Relative is slash-separated, so path.Base is correct here.
	if strings.HasPrefix(path.Base(f.Relative), appleDoublePrefix) {
		return nil
	}

	rc, err := codeDir.Open(f.Relative)
	if err != nil {
		return fmt.Errorf("open %s: %w", f.Relative, err)
	}
	defer rc.Close()

	info, err := rc.Stat()
	if err != nil {
		return fmt.Errorf("stat %s: %w", f.Relative, err)
	}

	// Only regular files are archived. The gitignore-aware walker never yields
	// directories, and symlinks inside a code snapshot are out of scope.
	if !info.Mode().IsRegular() {
		return nil
	}

	hdr := &tar.Header{
		Typeflag: tar.TypeReg,
		Name:     path.Join(prefix, f.Relative),
		Size:     info.Size(),
		// Normalize permissions and zero the mtime so the archive is reproducible
		// across machines. The runtime invokes code via an interpreter, not by
		// executing files from the snapshot, so execute bits are not preserved.
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
