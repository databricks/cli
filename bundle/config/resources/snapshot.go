package resources

import (
	"context"
	"net/url"
	"path"

	"github.com/databricks/cli/libs/snapshot"
	"github.com/databricks/databricks-sdk-go"
)

// Snapshot is an internal resource that stores the bundle zip as an immutable
// workspace object. It is created by the deploy pipeline and is not intended
// to be declared in user-authored databricks.yml files.
//
// JSON tags are present because the direct-deploy engine serialises the in-memory
// state to a JSON plan file (resources.internal_immutable_snapshots.*). Fields
// that must not leak into the plan file use json:"-".
type Snapshot struct {
	// BundleID is the stable UUID that identifies the bundle deployment, used
	// as the first path component of the snapshot workspace path.
	BundleID string `json:"bundle_id"`
	// ACL is the access control list applied to the uploaded snapshot, granting
	// CAN_READ to the deploying user and to every principal in bundle.permissions.
	ACL []snapshot.ACLEntry `json:"acl"`
	// ZipContent holds the raw zip bytes of the bundle source tree. It is
	// populated just before upload. The counterpart SnapshotState.ZipContent
	// carries json:"-" so the zip bytes never reach the plan file; SyncZipContent
	// re-injects them from here when deploying from a plan.
	ZipContent string `json:"zip_content"`
	// RemoteRoot is the workspace root path returned by the snapshot rootpath
	// API (e.g. /Workspace/Users/<user>/.snapshots).
	RemoteRoot string `json:"remote_root"`

	Lifecycle Lifecycle `json:"-"`
}

func (s *Snapshot) RelativePath() string {
	return path.Join(s.BundleID, snapshot.HashFromContent([]byte(s.ZipContent)))
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
	// Skipping URL initialization for snapshots
	return ""
}

func (s *Snapshot) InitializeURL(_ url.URL) {
	// Secret scopes do not have a URL
}

func (s *Snapshot) GetLifecycle() LifecycleConfig {
	return s.Lifecycle
}
