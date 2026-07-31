package resources

import (
	"path"

	"github.com/databricks/cli/libs/snapshot"
)

// Snapshot is the configuration for the snapshot resource.
// This is an internal resource that is used to store the snapshot of the bundle.
// It is not meant to be used by the user.
type Snapshot struct {
	BundleID   string              `json:"bundle_id"`
	ACL        []snapshot.ACLEntry `json:"acl"`
	ZipContent string              `json:"zip_content"`
}

func (s *Snapshot) RelativePath() string {
	return path.Join(s.BundleID, snapshot.HashFromContent([]byte(s.ZipContent)))
}

func (s *Snapshot) FullPath(remoteRoot string) string {
	return path.Join(remoteRoot, s.RelativePath())
}
