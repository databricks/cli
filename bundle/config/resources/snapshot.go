package resources

import (
	"context"
	"net/url"
	"path"

	"github.com/databricks/cli/libs/snapshot"
	"github.com/databricks/databricks-sdk-go"
)

// Snapshot is the configuration for the snapshot resource.
// This is an internal resource that is used to store the snapshot of the bundle.
// It is not meant to be used by the user.
type Snapshot struct {
	BundleID   string              `json:"bundle_id"`
	ACL        []snapshot.ACLEntry `json:"acl"`
	ZipContent string              `json:"zip_content"`
	RemoteRoot string              `json:"remote_root"`

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
		SingularName:  "snapshot",
		PluralName:    "snapshots",
		SingularTitle: "Snapshot",
		PluralTitle:   "Snapshots",
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
