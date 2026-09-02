package resources

import (
	"context"
	"net/url"
	"path"
	"strings"

	"github.com/databricks/cli/libs/snapshot"
	"github.com/databricks/databricks-sdk-go"
)

const (
	// SnapshotKey is the config key of the singleton immutable-folder snapshot. The
	// "immutable" map key is assigned in bundle/deploy/snapshot.PlanUpload; the plural
	// segment must match ResourceDescription().PluralName below.
	SnapshotKey = "resources.internal_immutable_snapshots.immutable"

	// SnapshotFullPathRef is a ${...} reference to the snapshot's deployed workspace
	// path (FullPath). Mutators point workspace.file_path / workspace.artifact_path and
	// translated resource paths at it so they resolve to the snapshot after deploy.
	SnapshotFullPathRef = "${" + SnapshotKey + ".full_path}"

	// SnapshotResourceKey is the config key of the singleton immutable-folder snapshot.
	// The "immutable" map key is assigned in bundle/deploy/snapshot.PlanUpload; the plural
	// segment must match ResourceDescription().PluralName below.
	SnapshotResourceKey = "immutable"
)

// Snapshot is an internal resource that stores the bundle zip as an immutable
// workspace object. It is created by the deploy pipeline and is not intended
// to be declared in user-authored databricks.yml files.
//
// JSON tags are present because the direct-deploy engine serialises the in-memory
// state to a JSON plan file (resources.internal_immutable_snapshots.*).
//
// RemoteRoot, RelativePath and FullPath are all workspace (remote) paths and are
// related, not independent: FullPath = RemoteRoot + "/" + RelativePath. RemoteRoot is
// the shared parent, RelativePath is this snapshot's location within it (and the
// resource ID), and FullPath is the composed absolute path that downstream resources
// reference and that the API echoes back on read.
type Snapshot struct {
	// BundleID is the stable UUID that identifies the bundle deployment, used
	// as the first path component of RelativePath.
	// Example: "1a2b3c4d-....".
	BundleID string `json:"bundle_id"`
	// ACL is the access control list applied to the uploaded snapshot, granting
	// CAN_READ to the deploying user and to every principal in bundle.permissions.
	ACL []snapshot.ACLEntry `json:"acl"`
	// ZipPath is the local path of the bundle zip staged by the deploy pipeline.
	// The file is named "<sha256>.zip", so its base name is the content hash used
	// as the last component of RelativePath. Storing the path (not the bytes) keeps
	// the zip off the config, the state, and the plan file, and lets DoCreate stream
	// it straight from disk at upload time.
	// Example: "/.../.databricks/bundle/default/snapshots/e3b0c4...zip".
	ZipPath string `json:"zip_path"`
	// RemoteRoot is the workspace directory that holds every snapshot for this user,
	// returned by the snapshot rootpath API.
	// Example: "/Workspace/Users/<user>/.snapshots".
	RemoteRoot string `json:"remote_root"`

	Lifecycle Lifecycle `json:"-"`
}

// RelativePath is the snapshot's location under RemoteRoot: "<bundle_id>/<zip_hash>".
// It is the resource ID; the last component is the zip content hash (the ZipPath base
// name), which makes the pre-computed path match the uploaded content.
// Example: "1a2b3c4d-.../e3b0c4...".
func (s *Snapshot) RelativePath() string {
	hash := strings.TrimSuffix(path.Base(s.ZipPath), ".zip")
	return path.Join(s.BundleID, hash)
}

// FullPath is the absolute workspace path of the snapshot: RemoteRoot + "/" + RelativePath.
// Example: "/Workspace/Users/<user>/.snapshots/1a2b3c4d-.../e3b0c4...".
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
