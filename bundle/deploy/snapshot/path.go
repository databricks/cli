package snapshot

import (
	"archive/zip"
	"bytes"
	"context"
	"crypto/sha256"
	"encoding/hex"
	"fmt"
	"io"
	"os"
	"path/filepath"
	"slices"
	"time"

	"github.com/databricks/cli/bundle"
	"github.com/databricks/cli/libs/fileset"
	"github.com/databricks/cli/libs/git"
	"github.com/databricks/cli/libs/set"
)

// zipEpoch is a fixed timestamp used for all zip entries to make the zip content-addressed
// and reproducible: the same file content always produces the same hash regardless of when
// the zip was built or the file's mtime.
var zipEpoch = time.Date(2000, 1, 1, 0, 0, 0, 0, time.UTC)

// BundleZip builds the zip that is uploaded to the snapshot API.
// It contains:
//   - all files from the bundle sync root under the "files/" prefix,
//     selected with the same git-aware + include/exclude logic as files.Upload
//   - all built artifact files under the "artifacts/.internal/" prefix
//
// The snapshot ID is always IDFromContent(BundleZip(b)), ensuring the
// pre-calculated path and the uploaded path are derived from the same content.
func BundleZip(ctx context.Context, b *bundle.Bundle) ([]byte, error) {
	var buf bytes.Buffer
	zw := zip.NewWriter(&buf)

	if err := addSyncRootToZip(ctx, zw, b); err != nil {
		return nil, err
	}
	if err := addArtifactsToZip(zw, b); err != nil {
		return nil, err
	}

	if err := zw.Close(); err != nil {
		return nil, err
	}
	return buf.Bytes(), nil
}

// IDFromContent returns the SHA-256 hex digest of content.
func IDFromContent(content []byte) string {
	h := sha256.Sum256(content)
	return hex.EncodeToString(h[:])
}

// SnapshotID builds the bundle zip and returns its SHA-256 hex digest.
// Called after artifacts are built so that ApplyImmutableWorkspacePaths and
// snapshot.Upload both hash identical content.
func SnapshotID(ctx context.Context, b *bundle.Bundle) (string, error) {
	content, err := BundleZip(ctx, b)
	if err != nil {
		return "", err
	}
	return IDFromContent(content), nil
}

// syncFiles returns the list of files to include in the snapshot zip using the
// same git-aware include/exclude logic as files.Upload (libs/sync).
func syncFiles(ctx context.Context, b *bundle.Bundle) ([]fileset.File, error) {
	// Use git.NewFileSet so that .gitignore rules are respected, matching the
	// behaviour of the normal files.Upload sync path.
	// Avoid passing an empty/nil paths slice: git.NewFileSet forwards it to
	// fileset.New whose variadic default ("." if no args) is bypassed when the
	// caller explicitly passes a nil slice.  The SyncDefaultPath mutator always
	// sets Sync.Paths to ["."] in the normal pipeline; we replicate that here
	// so BundleZip works even when the bundle hasn't gone through the full pipeline.
	var gitFS *git.FileSet
	var err error
	if len(b.Config.Sync.Paths) > 0 {
		gitFS, err = git.NewFileSet(ctx, b.WorktreeRoot, b.SyncRoot, b.Config.Sync.Paths)
	} else {
		gitFS, err = git.NewFileSet(ctx, b.WorktreeRoot, b.SyncRoot)
	}
	if err != nil {
		return nil, fmt.Errorf("build file set: %w", err)
	}

	all := set.NewSetF(func(f fileset.File) string {
		return f.Relative
	})

	gitFiles, err := gitFS.Files()
	if err != nil {
		return nil, fmt.Errorf("list sync files: %w", err)
	}
	all.Add(gitFiles...)

	if len(b.Config.Sync.Include) > 0 {
		includeFS, err := fileset.NewGlobSet(b.SyncRoot, b.Config.Sync.Include)
		if err != nil {
			return nil, fmt.Errorf("build include set: %w", err)
		}
		include, err := includeFS.Files()
		if err != nil {
			return nil, fmt.Errorf("list include files: %w", err)
		}
		all.Add(include...)
	}

	if len(b.Config.Sync.Exclude) > 0 {
		excludeFS, err := fileset.NewGlobSet(b.SyncRoot, b.Config.Sync.Exclude)
		if err != nil {
			return nil, fmt.Errorf("build exclude set: %w", err)
		}
		exclude, err := excludeFS.Files()
		if err != nil {
			return nil, fmt.Errorf("list exclude files: %w", err)
		}
		for _, f := range exclude {
			all.Remove(f)
		}
	}

	files := all.Iter()
	// Sort for a stable zip (same content → same hash regardless of map iteration order).
	slices.SortFunc(files, func(a, b fileset.File) int {
		if a.Relative < b.Relative {
			return -1
		}
		if a.Relative > b.Relative {
			return 1
		}
		return 0
	})
	return files, nil
}

func addSyncRootToZip(ctx context.Context, zw *zip.Writer, b *bundle.Bundle) error {
	files, err := syncFiles(ctx, b)
	if err != nil {
		return err
	}

	for _, f := range files {
		rc, err := b.SyncRoot.Open(f.Relative)
		if err != nil {
			return fmt.Errorf("open %s: %w", f.Relative, err)
		}

		entryPath := filepath.ToSlash(f.Relative)
		h := &zip.FileHeader{
			Name:     "files/" + entryPath,
			Method:   zip.Deflate,
			Modified: zipEpoch,
		}
		w, err := zw.CreateHeader(h)
		if err != nil {
			rc.Close()
			return fmt.Errorf("zip entry for %s: %w", f.Relative, err)
		}
		_, err = io.Copy(w, rc)
		rc.Close()
		if err != nil {
			return fmt.Errorf("write %s: %w", f.Relative, err)
		}
	}
	return nil
}

func addArtifactsToZip(zw *zip.Writer, b *bundle.Bundle) error {
	for _, artifact := range b.Config.Artifacts {
		for _, af := range artifact.Files {
			source := af.Source
			if af.Patched != "" {
				source = af.Patched
			}
			// ".internal" matches libraries.InternalDirName so that ReplaceWithRemotePath
			// produces library paths that resolve correctly inside the snapshot.
			if err := addLocalFileToZip(zw, source, "artifacts/.internal"); err != nil {
				return err
			}
		}
	}
	return nil
}

func addLocalFileToZip(zw *zip.Writer, localPath, zipPrefix string) error {
	f, err := os.Open(localPath)
	if err != nil {
		return fmt.Errorf("open %s: %w", localPath, err)
	}
	defer f.Close()

	entryName := zipPrefix + "/" + filepath.Base(localPath)
	h := &zip.FileHeader{
		Name:     entryName,
		Method:   zip.Deflate,
		Modified: zipEpoch,
	}
	w, err := zw.CreateHeader(h)
	if err != nil {
		return fmt.Errorf("zip entry %s: %w", entryName, err)
	}
	_, err = io.Copy(w, f)
	return err
}
