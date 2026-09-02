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
	"path"
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

// tarballEpoch stamps every entry so the archive is reproducible: identical contents
// produce identical bytes regardless of file mtimes.
var tarballEpoch = time.Date(2000, 1, 1, 0, 0, 0, 0, time.UTC)

// buildTarballArtifact produces the gzipped tarball for a `type: tgz` artifact that
// DABs builds itself (no user `build` command). Archive entries are the packed files
// named relative to the artifact's `path`. With `git` set the tarball snapshots that
// ref; otherwise it packs the working tree scoped to `include`. The result is written
// to the artifact's single output file, which the normal artifact upload path uploads
// and references.
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
	defer func() {
		tmp.Close()
		os.Remove(tmpName) // no-op once renamed into place
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
	return os.Rename(tmpName, out)
}

// codeRel returns the artifact's `path` as a slash-separated path relative to the sync
// root ("." when they are the same). include paths and archive entry names are relative
// to it.
func codeRel(b *bundle.Bundle, a *config.Artifact) (string, error) {
	rel, err := filepath.Rel(b.SyncRootPath, a.Path)
	if err != nil {
		return "", fmt.Errorf("artifact path %q: %w", a.Path, err)
	}
	return filepath.ToSlash(rel), nil
}

// tarballFromInclude packs the working tree under the artifact's `path`, optionally
// narrowed to `include` subpaths (relative to `path`). Entries are named relative to
// `path`. The bundle's sync walker is used so .gitignore is honored, but the
// bundle-wide sync.include/sync.exclude are deliberately NOT applied — they scope
// bundle file sync, not a code artifact.
func tarballFromInclude(ctx context.Context, b *bundle.Bundle, a *config.Artifact, w io.Writer) error {
	relBase, err := codeRel(b, a)
	if err != nil {
		return err
	}
	opts, err := files.GetSyncOptions(ctx, b)
	if err != nil {
		return err
	}
	paths := []string{relBase}
	if len(a.Include) > 0 {
		paths = paths[:0]
		for _, inc := range a.Include {
			paths = append(paths, path.Join(relBase, filepath.ToSlash(inc)))
		}
	}
	// nil include/exclude: only .gitignore filters the walk, not the bundle sync globs.
	fl, err := libsync.NewFileList(ctx, opts.WorktreeRoot, opts.LocalRoot, paths, nil, nil)
	if err != nil {
		return err
	}
	list, err := fl.Files(ctx)
	if err != nil {
		return err
	}
	if relBase != "." {
		prefix := relBase + "/"
		list = slices.DeleteFunc(list, func(f fileset.File) bool {
			return !strings.HasPrefix(f.Relative, prefix)
		})
	}
	if len(list) == 0 {
		return fmt.Errorf("artifact tgz: no files to pack under %q (empty, gitignored, or no `include` match)", a.Path)
	}
	// Sort so the archive byte stream doesn't depend on walk order.
	slices.SortFunc(list, func(x, y fileset.File) int {
		return strings.Compare(x.Relative, y.Relative)
	})

	gzw := gzip.NewWriter(w)
	tw := tar.NewWriter(gzw)
	for _, file := range list {
		if err := addFileToTarball(tw, b.SyncRoot, relBase, file); err != nil {
			return err
		}
	}
	if err := tw.Close(); err != nil {
		return err
	}
	return gzw.Close()
}

// tarballFromGit snapshots a git ref via `git archive`, so the archive reflects the
// committed tree at that ref. Entries are named relative to `path` (the tree at
// `<ref>:<path>`); `include`, when set, scopes the archived pathspecs. Commit wins over
// Branch.
func tarballFromGit(ctx context.Context, b *bundle.Bundle, a *config.Artifact, w io.Writer) error {
	ref := a.Git.Commit
	if ref == "" {
		ref = a.Git.Branch
	}
	if ref == "" {
		return errors.New("git artifact: specify git.commit or git.branch")
	}
	relBase, err := codeRel(b, a)
	if err != nil {
		return err
	}
	treeish := ref
	if relBase != "." {
		// The tree at <ref>:<path>, so entries come out relative to `path`.
		treeish = ref + ":" + relBase
	}
	args := []string{"-C", b.SyncRootPath, "archive", "--format=tar.gz", treeish}
	if len(a.Include) > 0 {
		args = append(args, "--")
		args = append(args, a.Include...)
	}
	cmd := exec.CommandContext(ctx, "git", args...)
	cmd.Stdout = w
	var stderr strings.Builder
	cmd.Stderr = &stderr
	if err := cmd.Run(); err != nil {
		return fmt.Errorf("git archive %s: %w: %s", treeish, err, stderr.String())
	}
	return nil
}

// addFileToTarball writes f (a sync-root-relative file) to the archive under a name
// relative to relBase, so entries are relative to the artifact's `path`.
func addFileToTarball(tw *tar.Writer, root vfs.Path, relBase string, f fileset.File) error {
	name := f.Relative
	if relBase != "." {
		trimmed, ok := strings.CutPrefix(name, relBase+"/")
		if !ok {
			// Outside the code root; the walk is scoped to it, so this shouldn't
			// happen, but skip defensively rather than mis-place a file.
			return nil
		}
		name = trimmed
	}

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
		Name:     name,
		Size:     info.Size(),
		Mode:     mode,
		ModTime:  tarballEpoch,
	}
	if err := tw.WriteHeader(hdr); err != nil {
		return fmt.Errorf("tar header for %s: %w", name, err)
	}
	if _, err := io.Copy(tw, rc); err != nil {
		return fmt.Errorf("write %s: %w", name, err)
	}
	return nil
}
