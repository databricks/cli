package deploy

import (
	"context"
	"encoding/json"
	"errors"
	"io/fs"
	"os"
	"path/filepath"
	"time"
	"uuid"

	"github.com/databricks/cli/bundle"
	"github.com/databricks/cli/internal/build"
	"github.com/databricks/cli/libs/atomicfile"
	"github.com/databricks/cli/libs/diag"
	"github.com/databricks/cli/libs/log"
)

type stateUpdate struct {
	// keepInternalFiles keeps listing the previous state's entries under the
	// target's .internal directory that are not on disk; see StateUpdateForSavedPlan.
	keepInternalFiles bool
}

func (s *stateUpdate) Name() string {
	return "deploy:state-update"
}

func (s *stateUpdate) Apply(ctx context.Context, b *bundle.Bundle) diag.Diagnostics {
	state, err := load(ctx, b)
	if err != nil {
		return diag.FromErr(err)
	}

	// Increment the state sequence.
	state.Seq = state.Seq + 1

	// Update timestamp.
	state.Timestamp = time.Now().UTC()

	// Update the CLI version and deployment state version.
	state.CliVersion = build.GetInfo().Version
	state.Version = DeploymentStateVersion

	// Update the state with the current list of synced files.
	fl, err := fromSlice(b.Files)
	if err != nil {
		return diag.FromErr(err)
	}
	if s.keepInternalFiles {
		_, internal := splitInternalFiles(b.Config.Bundle.Target, state.Files)
		fl = withKeptFiles(fl, internal)
	}
	state.Files = fl

	// Generate a UUID for the deployment, if one does not already exist
	if state.ID == uuid.Nil() {
		state.ID = uuid.New()
	}
	b.Metrics.DeploymentId = state.ID

	statePath, err := getPathToStateFile(ctx, b)
	if err != nil {
		return diag.FromErr(err)
	}
	data, err := json.Marshal(state)
	if err != nil {
		return diag.FromErr(err)
	}

	// Write the state back to the file.
	if err := atomicfile.Write(statePath, data, 0o600); err != nil {
		log.Infof(ctx, "Unable to write deployment state file: %s", err)
		return diag.FromErr(err)
	}

	return nil
}

// withKeptFiles appends the kept entries that the synced files don't already list.
func withKeptFiles(synced, kept Filelist) Filelist {
	seen := make(map[string]bool, len(synced))
	for _, f := range synced {
		seen[filepath.ToSlash(f.LocalPath)] = true
	}
	for _, f := range kept {
		if !seen[filepath.ToSlash(f.LocalPath)] {
			synced = append(synced, f)
		}
	}
	return synced
}

func StateUpdate() bundle.Mutator {
	return &stateUpdate{}
}

// StateUpdateForSavedPlan is StateUpdate for applying a saved plan. The apply skips
// the build that generates the files in the bundle's internal directory (such as
// python_wheel_wrapper notebooks), and StatePullForSavedPlan keeps their remote
// copies, so the state keeps listing them for a later deploy to delete.
func StateUpdateForSavedPlan() bundle.Mutator {
	return &stateUpdate{keepInternalFiles: true}
}

func load(ctx context.Context, b *bundle.Bundle) (*DeploymentState, error) {
	// If the file does not exist, return a new DeploymentState.
	statePath, err := getPathToStateFile(ctx, b)
	if err != nil {
		return nil, err
	}

	log.Infof(ctx, "Loading deployment state from %s", statePath)
	f, err := os.Open(statePath)
	if err != nil {
		if errors.Is(err, fs.ErrNotExist) {
			log.Infof(ctx, "No deployment state file found")
			return &DeploymentState{
				Version:    DeploymentStateVersion,
				CliVersion: build.GetInfo().Version,
			}, nil
		}
		return nil, err
	}
	defer f.Close()
	return loadState(f)
}
