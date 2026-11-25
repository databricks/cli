package deploy

import (
	"bytes"
	"context"
	"encoding/json"
	"errors"
	"io"
	"io/fs"
	"os"
	"path/filepath"
	"strings"

	"github.com/databricks/cli/bundle"
	"github.com/databricks/cli/bundle/deploy/files"
	"github.com/databricks/cli/libs/atomicfile"
	"github.com/databricks/cli/libs/diag"
	"github.com/databricks/cli/libs/filer"
	"github.com/databricks/cli/libs/log"
	"github.com/databricks/cli/libs/sync"
)

type statePull struct {
	filerFactory FilerFactory

	// keepInternalFiles leaves files under the bundle's internal directory out of
	// the restored snapshot, so the file sync does not delete their remote copies.
	keepInternalFiles bool
}

func (s *statePull) Apply(ctx context.Context, b *bundle.Bundle) diag.Diagnostics {
	f, err := s.filerFactory(ctx, b)
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

	data := remote.Bytes()
	err = validateRemoteStateCompatibility(bytes.NewReader(data))
	if err != nil {
		return diag.FromErr(err)
	}

	// A missing or unreadable local file counts as stale, so we write the remote copy.
	localData, _ := os.ReadFile(statePath)
	if !isLocalStateStale(bytes.NewReader(localData), bytes.NewReader(data)) {
		log.Infof(ctx, "Local deployment state is the same or newer, ignoring remote state")
		return nil
	}

	// Write file to disk.
	log.Infof(ctx, "Writing remote deployment state file to local cache directory")
	if err := atomicfile.Write(statePath, data, 0o600); err != nil {
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

	fileList := state.Files
	if s.keepInternalFiles {
		fileList = withoutInternalFiles(b.Config.Bundle.Target, fileList)
	}

	log.Infof(ctx, "Creating new snapshot")
	snapshot, err := sync.NewSnapshot(fileList.toSlice(b.SyncRoot), opts)
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

// withoutInternalFiles drops the entries under a "<target>/.internal" directory. It
// matches that shape anywhere in the path rather than this process's InternalDir,
// because the deploy that wrote the state may have used another cache directory
// (DATABRICKS_BUNDLE_TMP).
func withoutInternalFiles(target string, fl Filelist) Filelist {
	marker := "/" + target + "/" + bundle.InternalDirName + "/"
	var out Filelist
	for _, f := range fl {
		if !strings.Contains("/"+filepath.ToSlash(f.LocalPath), marker) {
			out = append(out, f)
		}
	}
	return out
}

// StatePull returns a mutator that pulls the deployment state from the Databricks workspace
func StatePull() bundle.Mutator {
	return &statePull{filerFactory: StateFiler}
}

// StatePullForSavedPlan is StatePull for applying a saved plan. Applying a plan skips
// the build, which is what generates the files in the bundle's internal directory
// (such as python_wheel_wrapper notebooks), while the plan's resources still refer to
// them; their remote copies must survive the file sync.
func StatePullForSavedPlan() bundle.Mutator {
	return &statePull{filerFactory: StateFiler, keepInternalFiles: true}
}
