package artifacts

import (
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

	"github.com/databricks/cli/bundle"
	"github.com/databricks/cli/bundle/config"
	"github.com/databricks/cli/bundle/config/mutator/aicode"
	"github.com/databricks/cli/bundle/deploy/files"
	"github.com/databricks/cli/libs/fileset"
	libsync "github.com/databricks/cli/libs/sync"
)

// buildTarballArtifact produces the gzipped tarball for a `type: tgz` artifact that
// DABs builds itself (no user `build` command). Every entry nests under a single
// top-level directory named for the artifact's code-source root (`path`), matching the
// AI Runtime's /databricks/code_source/<dir> extraction contract — the same layout
// aicode (for a directory code_source_path) and the air CLI produce. With `git` set the
// tarball snapshots that ref; otherwise it packs the working tree scoped to `include`.
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

// codeRoot returns the artifact's code-source root as a sync-root-relative path plus
// its directory name. dirName is the load-bearing top-level entry the runtime extracts
// to /databricks/code_source/<dir>.
func codeRoot(b *bundle.Bundle, a *config.Artifact) (relBase, dirName string, err error) {
	rel, err := filepath.Rel(b.SyncRootPath, a.Path)
	if err != nil {
		return "", "", fmt.Errorf("artifact path %q: %w", a.Path, err)
	}
	return filepath.ToSlash(rel), filepath.Base(a.Path), nil
}

// tarballFromInclude packs the working tree under the artifact's code-source root,
// optionally narrowed to `include` subpaths, using the bundle sync walker (filtering
// matches bundle file sync: .gitignore + sync.include/exclude). Entries are re-based
// under the code-source dir name via the shared aicode.BuildCodeSnapshot packer.
func tarballFromInclude(ctx context.Context, b *bundle.Bundle, a *config.Artifact, w io.Writer) error {
	relBase, dirName, err := codeRoot(b, a)
	if err != nil {
		return err
	}
	list, err := includeFiles(ctx, b, relBase, a.Include)
	if err != nil {
		return err
	}
	if len(list) == 0 {
		return fmt.Errorf("artifact tgz: no files to pack under %q (all excluded by .gitignore/sync.exclude, or empty)", a.Path)
	}
	_, err = aicode.BuildCodeSnapshot(b.SyncRoot, relBase, list, dirName, w)
	return err
}

// includeFiles lists the files to pack: the whole code-source root (relBase) when
// include is empty, else only the given subpaths (relative to relBase). The result is
// scoped to relBase so a force-added sync.include stray outside it is dropped.
func includeFiles(ctx context.Context, b *bundle.Bundle, relBase string, include []string) ([]fileset.File, error) {
	opts, err := files.GetSyncOptions(ctx, b)
	if err != nil {
		return nil, err
	}
	var paths []string
	if len(include) == 0 {
		paths = []string{relBase}
	} else {
		for _, p := range include {
			paths = append(paths, path.Join(relBase, p))
		}
	}
	fl, err := libsync.NewFileList(ctx, opts.WorktreeRoot, opts.LocalRoot, paths, opts.Include, opts.Exclude)
	if err != nil {
		return nil, err
	}
	all, err := fl.Files(ctx)
	if err != nil {
		return nil, err
	}
	if relBase == "." {
		return all, nil
	}
	prefix := relBase + "/"
	return slices.DeleteFunc(all, func(f fileset.File) bool {
		return !strings.HasPrefix(f.Relative, prefix)
	}), nil
}

// tarballFromGit snapshots a git ref via `git archive`, nesting every entry under the
// code-source dir name (--prefix) so the archive matches the include/aicode layout.
// Commit wins over Branch. `include`, when set, scopes the archived pathspecs.
func tarballFromGit(ctx context.Context, b *bundle.Bundle, a *config.Artifact, w io.Writer) error {
	ref := a.Git.Commit
	if ref == "" {
		ref = a.Git.Branch
	}
	if ref == "" {
		return errors.New("git artifact: specify git.commit or git.branch")
	}
	dirName := filepath.Base(a.Path)
	args := []string{"-C", b.SyncRootPath, "archive", "--format=tar.gz", "--prefix=" + dirName + "/", ref}
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
