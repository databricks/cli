package resources

import (
	"context"
	"net/url"
	"path"
	"strings"

	"github.com/databricks/cli/libs/snapshot"
	"github.com/databricks/databricks-sdk-go"
)

// Snapshot is an internal resource that stores the bundle zip as an immutable
// workspace object. It is created by the deploy pipeline and is not intended
// to be declared in user-authored databricks.yml files.
//
// JSON tags are present because the direct-deploy engine serialises the in-memory
// state to a JSON plan file (resources.internal_immutable_snapshots.*).
type Snapshot struct {
	// BundleID is the stable UUID that identifies the bundle deployment, used
	// as the first path component of the snapshot workspace path.
	BundleID string `json:"bundle_id"`
	// ACL is the access control list applied to the uploaded snapshot, granting
	// CAN_READ to the deploying user and to every principal in bundle.permissions.
	ACL []snapshot.ACLEntry `json:"acl"`
	// ZipPath is the local path of the bundle zip staged by the deploy pipeline.
	// The file is named "<sha256>.zip", so its base name is the content hash used
	// as the snapshot's relative path. Storing the path (not the bytes) keeps the
	// zip off the config, the state, and the plan file, and lets DoCreate stream it
	// straight from disk at upload time.
	ZipPath string `json:"zip_path"`
	// RemoteRoot is the workspace root path returned by the snapshot rootpath
	// API (e.g. /Workspace/Users/<user>/.snapshots).
	RemoteRoot string `json:"remote_root"`

	Lifecycle Lifecycle `json:"-"`
}

func (s *Snapshot) RelativePath() string {
	hash := strings.TrimSuffix(path.Base(s.ZipPath), ".zip")
	return path.Join(s.BundleID, hash)
}

func (s *Snapshot) FullPath() string {
	return path.Join(s.RemoteRoot, s.RelativePath())
}

func (s *Snapshot) Exists(ctx context.Context, w *databricks.WorkspaceClient, name string) (bool, error) {
	_, err := w.Workspace.GetStatusByPath(ctx, s.FullPath())
	if err != nil {
		return false, err
	}
	return true, nil
}

func (s *Snapshot) ResourceDescription() ResourceDescription {
	return ResourceDescription{
		SingularName:  "internal_immutable_snapshot",
		PluralName:    "internal_immutable_snapshots",
		SingularTitle: "Internal Immutable Snapshot",
		PluralTitle:   "Internal Immutable Snapshots",
	}
}

func (s *Snapshot) GetName() string {
	return s.RelativePath()
}

func (s *Snapshot) GetURL() string {
	// A snapshot is a workspace folder owned by the project's service principal, so
	// a browser URL is constructible from its path. We don't surface one yet:
	// workspaceurls has no folder-path helper, and the path is only known during
	// deploy (its content hash is derived from the zip staged by the deploy pipeline).
	return ""
}

func (s *Snapshot) InitializeURL(_ url.URL) {
	// See GetURL: no browser URL is surfaced for the snapshot folder yet.
}

func (s *Snapshot) GetLifecycle() LifecycleConfig {
	return s.Lifecycle
}
