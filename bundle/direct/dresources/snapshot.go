package dresources

import (
	"context"
	"fmt"
	"os"

	"github.com/databricks/cli/bundle/config/resources"
	"github.com/databricks/cli/libs/snapshot"
	"github.com/databricks/databricks-sdk-go"
)

type ResourceSnapshot struct {
	uploader *snapshot.SnapshotClient
}

type SnapshotState struct {
	RelativePath string              `json:"relative_path"`
	FullPath     string              `json:"full_path"`
	BundleID     string              `json:"bundle_id"`
	ACL          []snapshot.ACLEntry `json:"acl"`
	// ZipPath locates the bundle zip staged locally by the deploy pipeline. It is
	// small enough to round-trip through the plan file, so deploying from a plan
	// restores it directly (no re-injection needed). The file name is the content
	// hash, which is also the last component of RelativePath.
	ZipPath string `json:"zip_path"`
}

type SnapshotRemote struct {
	RelativePath string `json:"relative_path"`
	FullPath     string `json:"full_path"`
}

func (s *ResourceSnapshot) New(client *databricks.WorkspaceClient) *ResourceSnapshot {
	// Return a zero-value instance when client is nil (e.g. refschema introspection).
	if client == nil {
		return &ResourceSnapshot{
			uploader: nil,
		}
	}

	uploader, err := snapshot.NewSnapshotClient(client)
	if err != nil {
		panic(err)
	}

	return &ResourceSnapshot{
		uploader: uploader,
	}
}

func (s *ResourceSnapshot) PrepareState(input *resources.Snapshot) *SnapshotState {
	return &SnapshotState{
		RelativePath: input.RelativePath(),
		FullPath:     input.FullPath(),
		BundleID:     input.BundleID,
		ACL:          input.ACL,
		ZipPath:      input.ZipPath,
	}
}

func (s *ResourceSnapshot) RemapState(remote *SnapshotRemote) *SnapshotState {
	return &SnapshotState{
		RelativePath: remote.RelativePath,
		FullPath:     remote.FullPath,
		BundleID:     "",
		ACL:          nil,
		ZipPath:      "",
	}
}

func (s *ResourceSnapshot) DoRead(ctx context.Context, id string) (*SnapshotRemote, error) {
	info, err := s.uploader.Get(ctx, id)
	if err != nil {
		return nil, err
	}
	return &SnapshotRemote{
		RelativePath: id,
		FullPath:     info.Path,
	}, nil
}

func (s *ResourceSnapshot) DoCreate(ctx context.Context, state *SnapshotState) (string, *SnapshotRemote, error) {
	content, err := os.ReadFile(state.ZipPath)
	if err != nil {
		return "", nil, fmt.Errorf("reading snapshot zip: %w", err)
	}

	path := state.RelativePath
	info, err := s.uploader.Upload(ctx, path, state.BundleID, state.ACL, content)
	if err != nil {
		return "", nil, err
	}
	return path, &SnapshotRemote{RelativePath: path, FullPath: info.Path}, nil
}

// DoDelete is a no-op: a snapshot is immutable and cannot be deleted, so this only
// clears the state entry. DoUpdate is omitted entirely (the adapter treats it as
// optional) because every change recreates the snapshot. IsGone below keeps
// delete/destroy idempotent.
func (s *ResourceSnapshot) DoDelete(ctx context.Context, id string, state *SnapshotState) error {
	return nil
}

// IsGone treats a snapshot as already-deleted. The snapshot is immutable, so it can't be deleted.
func (s *ResourceSnapshot) IsGone(remote *SnapshotRemote) bool {
	return true
}
