package dresources

import (
	"context"
	"errors"
	"path"

	"github.com/databricks/cli/libs/snapshot"
	"github.com/databricks/databricks-sdk-go/apierr"
)

type Snapshot struct {
	remoteRoot string
	uploader   snapshot.SnapshotUploader
}

type SnapshotConfig struct {
	BundleID   string
	ACL        []snapshot.ACLEntry
	ZipContent []byte
}

type SnapshotState struct {
	RelativePath string              `json:"relative_path"`
	FullPath     string              `json:"full_path"`
	BundleID     string              `json:"bundle_id"`
	ACL          []snapshot.ACLEntry `json:"acl"`
	ZipContent   []byte              `json:"-"`
}

type SnapshotRemote struct {
	RelativePath string `json:"relative_path"`
	FullPath     string `json:"full_path"`
}

func (s *SnapshotConfig) RelativePath() string {
	return path.Join(s.BundleID, snapshot.HashFromContent(s.ZipContent))
}

func (s *SnapshotConfig) FullPath(remoteRoot string) string {
	return path.Join(remoteRoot, s.RelativePath())
}

func (s *Snapshot) New(uploader snapshot.SnapshotUploader, remoteRoot string) *Snapshot {
	return &Snapshot{
		remoteRoot: remoteRoot,
		uploader:   uploader,
	}
}

func (s *Snapshot) PrepareState(input *SnapshotConfig) *SnapshotState {
	return &SnapshotState{
		RelativePath: input.RelativePath(),
		FullPath:     input.FullPath(s.remoteRoot),
		BundleID:     input.BundleID,
		ACL:          input.ACL,
		ZipContent:   input.ZipContent,
	}
}

func (s *Snapshot) RemapState(remote *SnapshotRemote) *SnapshotState {
	return &SnapshotState{
		RelativePath: remote.RelativePath,
		FullPath:     remote.FullPath,
		BundleID:     "",
		ACL:          nil,
		ZipContent:   nil,
	}
}

func (s *Snapshot) DoRead(ctx context.Context, id string) (*SnapshotRemote, error) {
	fullPath := path.Join(s.remoteRoot, id)
	_, err := s.uploader.Get(ctx, fullPath)
	if err != nil {
		if errors.Is(err, apierr.ErrNotFound) {
			return &SnapshotRemote{
				RelativePath: id,
				FullPath:     "",
			}, nil
		}
		return nil, err
	}
	return &SnapshotRemote{
		RelativePath: id,
		FullPath:     fullPath,
	}, nil
}

func (s *Snapshot) DoCreate(ctx context.Context, state *SnapshotState) (string, *SnapshotRemote, error) {
	path := state.RelativePath
	info, err := s.uploader.Upload(ctx, path, state.BundleID, state.ACL, state.ZipContent)
	if err != nil {
		return "", nil, err
	}
	return path, &SnapshotRemote{RelativePath: path, FullPath: info.Path}, nil
}

func (s *Snapshot) DoUpdate(ctx context.Context, id string, newState *SnapshotState, entry *PlanEntry) (*SnapshotRemote, error) {
	return nil, nil
}

func (s *Snapshot) DoDelete(ctx context.Context, id string, state *SnapshotState) error {
	return nil
}
