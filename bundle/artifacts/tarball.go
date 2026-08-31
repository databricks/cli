package artifacts

import (
	"archive/tar"
	"compress/gzip"
	"context"
	"errors"
	"fmt"
	"io"
	"os"
	"os/exec"
	"path/filepath"
	"slices"
	"strings"
	"time"

	"github.com/databricks/cli/bundle"
	"github.com/databricks/cli/bundle/config"
	"github.com/databricks/cli/bundle/deploy/files"
	"github.com/databricks/cli/libs/fileset"
	libsync "github.com/databricks/cli/libs/sync"
	"github.com/databricks/cli/libs/vfs"
)

// tarballEpoch stamps every entry so the archive is reproducible: identical
// contents produce identical bytes regardless of file mtimes. Mirrors
// aicode.buildCodeSnapshot (the two packers could later share one implementation).
var tarballEpoch = time.Date(2000, 1, 1, 0, 0, 0, 0, time.UTC)

// buildTarballArtifact produces the gzipped tarball for a `type: tgz` artifact that
// DABs builds itself (no user `build` command). When `git` is set the tarball is a
// snapshot of that ref; otherwise it packs the working tree scoped to `include`
// paths (gitignore-honored). The result is written to the artifact's single output
// file, which the normal artifact upload path then uploads and references.
func buildTarballArtifact(ctx context.Context, b *bundle.Bundle, name string, a *config.Artifact) error {
	if len(a.Files) != 1 {
		return fmt.Errorf("artifact %q: a tgz artifact needs exactly one `files` entry naming the output path", name)
	}
	out := a.Files[0].Source // made absolute by artifacts.Prepare
	if err := os.MkdirAll(filepath.Dir(out), 0o755); err != nil {
		return err
	}

	// Write to a temp file and rename on success, so a failed build never leaves a
	// partial tarball at `out`.
	tmp, err := os.CreateTemp(filepath.Dir(out), filepath.Base(out)+".*.tmp")
	if err != nil {
		return err
	}
	tmpName := tmp.Name()
	renamed := false
	defer func() {
		tmp.Close() // harmless double-close after the success path; closes fd on error paths
		if !renamed {
			os.Remove(tmpName)
		}
	}()

	if a.Git != nil {
		err = tarballFromGit(ctx, b, a, tmp)
	} else {
		err = tarballFromInclude(ctx, b, a, tmp)
	}
	if err != nil {
		return err
	}
	if err := tmp.Close(); err != nil {
		return err
	}
	if err := os.Rename(tmpName, out); err != nil {
		return err
	}
	renamed = true
	return nil
}

// tarballFromInclude packs the working tree, scoped to a.Include, using the bundle's
// sync walker so filtering matches bundle file sync (.gitignore + sync.include/exclude).
func tarballFromInclude(ctx context.Context, b *bundle.Bundle, a *config.Artifact, w io.Writer) error {
	opts, err := files.GetSyncOptions(ctx, b)
	if err != nil {
		return err
	}
	// a.Include is non-empty here: build.go only reaches include mode when len > 0.
	fl, err := libsync.NewFileList(ctx, opts.WorktreeRoot, opts.LocalRoot, a.Include, opts.Include, opts.Exclude)
	if err != nil {
		return err
	}
	list, err := fl.Files(ctx)
	if err != nil {
		return err
	}
	// Sort so the archive byte stream doesn't depend on walk order.
	slices.SortFunc(list, func(x, y fileset.File) int {
		return strings.Compare(x.Relative, y.Relative)
	})

	gzw := gzip.NewWriter(w)
	tw := tar.NewWriter(gzw)
	for _, file := range list {
		if err := addFileToTarball(tw, b.SyncRoot, file); err != nil {
			return err
		}
	}
	if err := tw.Close(); err != nil {
		return err
	}
	return gzw.Close()
}

// tarballFromGit snapshots a git ref via `git archive`, so the archive reflects the
// committed tree at that ref rather than the working tree. Commit wins over Branch.
func tarballFromGit(ctx context.Context, b *bundle.Bundle, a *config.Artifact, w io.Writer) error {
	ref := a.Git.Commit
	if ref == "" {
		ref = a.Git.Branch
	}
	if ref == "" {
		return errors.New("git artifact: specify git.commit or git.branch")
	}
	args := []string{"-C", b.SyncRootPath, "archive", "--format=tar.gz", ref}
	if len(a.Include) > 0 {
		args = append(args, "--")
		args = append(args, a.Include...)
	}
	cmd := exec.CommandContext(ctx, "git", args...)
	cmd.Stdout = w
	var stderr strings.Builder
	cmd.Stderr = &stderr
	if err := cmd.Run(); err != nil {
		return fmt.Errorf("git archive %s: %w: %s", ref, err, stderr.String())
	}
	return nil
}

func addFileToTarball(tw *tar.Writer, root vfs.Path, f fileset.File) error {
	rc, err := root.Open(f.Relative)
	if err != nil {
		return fmt.Errorf("open %s: %w", f.Relative, err)
	}
	defer rc.Close()

	info, err := rc.Stat()
	if err != nil {
		return fmt.Errorf("stat %s: %w", f.Relative, err)
	}
	// Only regular files; the walker never yields directories and symlinks are out
	// of scope for a code snapshot.
	if !info.Mode().IsRegular() {
		return nil
	}

	// Preserve the owner execute bit; normalize the rest.
	mode := int64(0o644)
	if info.Mode().Perm()&0o100 != 0 {
		mode = 0o755
	}
	hdr := &tar.Header{
		Typeflag: tar.TypeReg,
		Name:     f.Relative,
		Size:     info.Size(),
		Mode:     mode,
		ModTime:  tarballEpoch,
	}
	if err := tw.WriteHeader(hdr); err != nil {
		return fmt.Errorf("tar header for %s: %w", f.Relative, err)
	}
	if _, err := io.Copy(tw, rc); err != nil {
		return fmt.Errorf("write %s: %w", f.Relative, err)
	}
	return nil
}
