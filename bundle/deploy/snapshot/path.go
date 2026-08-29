package snapshot

import (
	"archive/zip"
	"bytes"
	"context"
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"fmt"
	"io"
	"os"
	"path/filepath"
	"slices"
	"time"

	"github.com/databricks/cli/bundle"
	"github.com/databricks/cli/bundle/deploy/files"
	"github.com/databricks/cli/libs/fileset"
	libsync "github.com/databricks/cli/libs/sync"
)

// zipEpoch is a fixed timestamp used for all zip entries to make the zip content-addressed
// and reproducible: the same file content always produces the same hash regardless of when
// the zip was built or the file's mtime.
var zipEpoch = time.Date(2000, 1, 1, 0, 0, 0, 0, time.UTC)

// metadataFileName is the name of the metadata file embedded in every snapshot zip.
// It captures a hash of the ACL so that a change to top-level permissions always produces
// a different snapshot ID — necessary because immutable snapshots cannot be re-permissioned
// after creation. The hash avoids embedding principal names in the stored artifact.
const metadataFileName = ".databricks/snapshot-metadata.json"

// snapshotMetadata is the structure written to metadataFileName inside the zip.
type snapshotMetadata struct {
	// PermissionsHash is the SHA-256 hex digest of the JSON-serialized ACL.
	PermissionsHash string `json:"permissions_hash"`
}

// BundleZip builds the zip that is uploaded to the snapshot API.
// It contains:
//   - all files from the bundle sync root under the "files/" prefix,
//     selected with the same git-aware + include/exclude logic as files.Upload
//   - all built artifact files under the "artifacts/.internal/" prefix
//   - a metadata file at metadataFileName that embeds the ACL so that any
//     change to top-level permissions forces a new snapshot ID
//
// The snapshot ID is always IDFromContent(BundleZip(b)), ensuring the
// pre-calculated path and the uploaded path are derived from the same content.
// The second return value is the number of sync-root files included in the zip.
func BundleZip(ctx context.Context, b *bundle.Bundle) ([]byte, int, error) {
	var buf bytes.Buffer
	zw := zip.NewWriter(&buf)

	fileCount, err := addSyncRootToZip(ctx, zw, b)
	if err != nil {
		return nil, 0, err
	}
	if err := addArtifactsToZip(zw, b); err != nil {
		return nil, 0, err
	}
	if err := addMetadataToZip(zw, BuildACL(b)); err != nil {
		return nil, 0, err
	}

	if err := zw.Close(); err != nil {
		return nil, 0, err
	}
	return buf.Bytes(), fileCount, nil
}

// addMetadataToZip writes the snapshot metadata file into the zip so that
// any change to the ACL changes the snapshot hash and forces a new snapshot.
func addMetadataToZip(zw *zip.Writer, acl []ACLEntry) error {
	aclJSON, err := json.Marshal(acl)
	if err != nil {
		return fmt.Errorf("marshal ACL for permissions hash: %w", err)
	}
	data, err := json.Marshal(snapshotMetadata{PermissionsHash: IDFromContent(aclJSON)})
	if err != nil {
		return fmt.Errorf("marshal snapshot metadata: %w", err)
	}
	h := &zip.FileHeader{
		Name:     metadataFileName,
		Method:   zip.Deflate,
		Modified: zipEpoch,
	}
	w, err := zw.CreateHeader(h)
	if err != nil {
		return fmt.Errorf("zip entry for %s: %w", metadataFileName, err)
	}
	_, err = w.Write(data)
	return err
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
	content, _, err := BundleZip(ctx, b)
	if err != nil {
		return "", err
	}
	return IDFromContent(content), nil
}

// addSyncRootToZip returns the number of files added from the sync root.
func addSyncRootToZip(ctx context.Context, zw *zip.Writer, b *bundle.Bundle) (int, error) {
	opts, err := files.GetSyncOptions(ctx, b)
	if err != nil {
		return 0, err
	}
	fl, err := libsync.NewFileList(ctx, opts.WorktreeRoot, opts.LocalRoot, opts.Paths, opts.Include, opts.Exclude)
	if err != nil {
		return 0, fmt.Errorf("build file set: %w", err)
	}
	fileList, err := fl.Files(ctx)
	if err != nil {
		return 0, err
	}
	// Sort for a stable zip (same content → same hash regardless of iteration order).
	slices.SortFunc(fileList, func(a, b fileset.File) int {
		if a.Relative < b.Relative {
			return -1
		}
		if a.Relative > b.Relative {
			return 1
		}
		return 0
	})

	for _, f := range fileList {
		rc, err := b.SyncRoot.Open(f.Relative)
		if err != nil {
			return 0, fmt.Errorf("open %s: %w", f.Relative, err)
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
			return 0, fmt.Errorf("zip entry for %s: %w", f.Relative, err)
		}
		_, err = io.Copy(w, rc)
		rc.Close()
		if err != nil {
			return 0, fmt.Errorf("write %s: %w", f.Relative, err)
		}
	}
	return len(fileList), nil
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
