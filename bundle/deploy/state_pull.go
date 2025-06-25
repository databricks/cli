package deploy

import (
	"bytes"
	"context"
	"encoding/json"
	"errors"
	"io"
	"io/fs"
	"os"

	"github.com/databricks/cli/bundle"
	"github.com/databricks/cli/bundle/deploy/files"
	"github.com/databricks/cli/libs/diag"
	"github.com/databricks/cli/libs/filer"
	"github.com/databricks/cli/libs/log"
	"github.com/databricks/cli/libs/sync"
)

type statePull struct {
	filerFactory FilerFactory
}

func (s *statePull) Apply(ctx context.Context, b *bundle.Bundle) diag.Diagnostics {
	f, err := s.filerFactory(b)
	if err != nil {
		return diag.FromErr(err)
	}

	// Download deployment state file from filer to local cache directory.
	log.Infof(ctx, "Opening remote deployment state file")
	remote, err := s.remoteState(ctx, f)
	if err != nil {
		log.Infof(ctx, "Unable to open remote deployment state file: %s", err)
		return diag.FromErr(err)
	}
	if remote == nil {
		log.Infof(ctx, "Remote deployment state file does not exist")
		return nil
	}

	statePath, err := getPathToStateFile(ctx, b)
	if err != nil {
		return diag.FromErr(err)
	}

	local, err := os.OpenFile(statePath, os.O_CREATE|os.O_RDWR, 0o600)
	if err != nil {
		return diag.FromErr(err)
	}
	defer local.Close()

	data := remote.Bytes()
	err = validateRemoteStateCompatibility(bytes.NewReader(data))
	if err != nil {
		return diag.FromErr(err)
	}

	if !isLocalStateStale(local, bytes.NewReader(data)) {
		log.Infof(ctx, "Local deployment state is the same or newer, ignoring remote state")
		return nil
	}

	// Truncating the file before writing
	err = local.Truncate(0)
	if err != nil {
		return diag.FromErr(err)
	}
	_, err = local.Seek(0, 0)
	if err != nil {
		return diag.FromErr(err)
	}

	// Write file to disk.
	log.Infof(ctx, "Writing remote deployment state file to local cache directory")
	_, err = io.Copy(local, bytes.NewReader(data))
	if err != nil {
		return diag.FromErr(err)
	}

	var state DeploymentState
	err = json.Unmarshal(data, &state)
	if err != nil {
		return diag.FromErr(err)
	}

	// Create a new snapshot based on the deployment state file.
	opts, err := files.GetSyncOptions(ctx, b)
	if err != nil {
		return diag.FromErr(err)
	}

	log.Infof(ctx, "Creating new snapshot")
	snapshot, err := sync.NewSnapshot(state.Files.toSlice(b.SyncRoot), opts)
	if err != nil {
		return diag.FromErr(err)
	}

	// Persist the snapshot to disk.
	log.Infof(ctx, "Persisting snapshot to disk")
	return diag.FromErr(snapshot.Save(ctx))
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

// StatePull returns a mutator that pulls the deployment state from the Databricks workspace
func StatePull() bundle.Mutator {
	return &statePull{StateFiler}
}
