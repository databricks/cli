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
	libsync "github.com/databricks/cli/libs/sync"
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

func addSyncRootToZip(ctx context.Context, zw *zip.Writer, b *bundle.Bundle) error {
	files, err := libsync.GetFileList(ctx, libsync.SyncOptions{
		WorktreeRoot: b.WorktreeRoot,
		LocalRoot:    b.SyncRoot,
		Paths:        b.Config.Sync.Paths,
		Include:      b.Config.Sync.Include,
		Exclude:      b.Config.Sync.Exclude,
	})
	if err != nil {
		return err
	}
	// Sort for a stable zip (same content → same hash regardless of iteration order).
	slices.SortFunc(files, func(a, b fileset.File) int {
		if a.Relative < b.Relative {
			return -1
		}
		if a.Relative > b.Relative {
			return 1
		}
		return 0
	})

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
