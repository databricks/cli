package deploy

import (
	"bytes"
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"io/fs"
	"os"

	"github.com/databricks/cli/bundle"
	"github.com/databricks/cli/libs/filer"
	"github.com/databricks/cli/libs/log"
	"github.com/databricks/cli/libs/sync"
)

type statePull struct {
	filerFactory
}

func (s *statePull) Apply(ctx context.Context, b *bundle.Bundle) error {
	f, err := s.filerFactory(b)
	if err != nil {
		return err
	}

	// Download deployment state file from filer to local cache directory.
	log.Infof(ctx, "Opening remote deployment state file")
	remote, err := s.remoteState(ctx, f)
	if err != nil {
		log.Infof(ctx, "Unable to open remote deployment state file: %s", err)
		return err
	}
	if remote == nil {
		log.Infof(ctx, "Remote deployment state file does not exist")
		return nil
	}

	statePath, err := getPathToStateFile(ctx, b)
	if err != nil {
		return err
	}

	local, err := os.OpenFile(statePath, os.O_CREATE|os.O_RDWR, 0600)
	if err != nil {
		return err
	}
	defer local.Close()

	data := remote.Bytes()
	if !isLocalStateStale(local, bytes.NewReader(data)) {
		log.Infof(ctx, "Local deployment state is the same or newer, ignoring remote state")
		return nil
	}

	// Truncating the file before writing
	local.Truncate(0)
	local.Seek(0, 0)

	// Write file to disk.
	log.Infof(ctx, "Writing remote deployment state file to local cache directory")
	_, err = io.Copy(local, bytes.NewReader(data))
	if err != nil {
		return err
	}

	cacheDir, err := b.CacheDir(ctx)
	if err != nil {
		return fmt.Errorf("cannot get bundle cache directory: %w", err)
	}

	opts := &sync.SyncOptions{
		LocalPath:        b.Config.Path,
		RemotePath:       b.Config.Workspace.FilePath,
		SnapshotBasePath: cacheDir,
		Host:             b.WorkspaceClient().Config.Host,
	}

	snapshotPath, err := sync.SnapshotPath(opts)
	if err != nil {
		return err
	}

	if _, err := os.Stat(snapshotPath); err == nil {
		log.Infof(ctx, "Snapshot already exists, skipping creation")
		return nil
	}

	var state DeploymentState
	err = json.Unmarshal(data, &state)
	if err != nil {
		return err
	}

	// Create a new snapshot based on the deployment state file.
	log.Infof(ctx, "Creating new snapshot")
	snapshotState, err := sync.NewSnapshotState(state.Files.ToSlice())
	if err != nil {
		return err
	}

	snapshot := &sync.Snapshot{
		SnapshotPath:  snapshotPath,
		New:           true,
		Version:       sync.LatestSnapshotVersion,
		Host:          opts.Host,
		RemotePath:    opts.RemotePath,
		SnapshotState: snapshotState,
	}

	// Persist the snapshot to disk.
	log.Infof(ctx, "Persisting snapshot to disk")
	return snapshot.Save(ctx)
}

func (s *statePull) remoteState(ctx context.Context, f filer.Filer) (*bytes.Buffer, error) {
	// Download deployment state file from filer to local cache directory.
	remote, err := f.Read(ctx, DeploymentStateFileName)
	if err != nil {
		// On first deploy this file doesn't yet exist.
		if errors.Is(err, fs.ErrNotExist) {
			return nil, nil
		}
		return nil, err
	}

	defer remote.Close()

	var buf bytes.Buffer
	_, err = io.Copy(&buf, remote)
	if err != nil {
		return nil, err
	}

	return &buf, nil
}

func (s *statePull) Name() string {
	return "deploy:state-pull"
}

func StatePull() bundle.Mutator {
	return &statePull{stateFiler}
}
