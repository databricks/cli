package statemgmt

import (
	"context"
	"encoding/json"
	"errors"
	"io"
	"io/fs"
	"os"
	"path/filepath"

	"github.com/databricks/cli/bundle"
	"github.com/databricks/cli/bundle/deploy"
	"github.com/databricks/cli/libs/diag"
	"github.com/databricks/cli/libs/log"
)

type tfState struct {
	Serial  int64  `json:"serial"`
	Lineage string `json:"lineage"`
}

type statePull struct {
	filerFactory deploy.FilerFactory
}

func (l *statePull) Name() string {
	return "state:pull"
}

func (l *statePull) remoteState(ctx context.Context, b *bundle.Bundle) (*tfState, []byte, error) {
	f, err := l.filerFactory(b)
	if err != nil {
		return nil, nil, err
	}

	r, err := f.Read(ctx, b.StateFileName())
	if errors.Is(err, fs.ErrNotExist) {
		// Fallback to legacy file name for backwards compatibility.
		const legacyStateFile = "terraform.tfstate"
		r, err = f.Read(ctx, legacyStateFile)
	}
	if err != nil {
		return nil, nil, err
	}
	defer r.Close()

	content, err := io.ReadAll(r)
	if err != nil {
		return nil, nil, err
	}

	state := &tfState{}
	if err := json.Unmarshal(content, state); err != nil {
		return nil, nil, err
	}

	return state, content, nil
}

func (l *statePull) localState(ctx context.Context, b *bundle.Bundle) (*tfState, error) {
	dir, err := b.CacheDir(ctx, "terraform")
	if err != nil {
		return nil, err
	}

	content, err := os.ReadFile(filepath.Join(dir, b.StateFileName()))
	if err != nil {
		return nil, err
	}

	state := &tfState{}
	if err := json.Unmarshal(content, state); err != nil {
		return nil, err
	}

	return state, nil
}

func (l *statePull) Apply(ctx context.Context, b *bundle.Bundle) diag.Diagnostics {
	dir, err := b.CacheDir(ctx, "terraform")
	if err != nil {
		return diag.FromErr(err)
	}
	localStatePath := filepath.Join(dir, b.StateFileName())

	remoteState, remoteContent, err := l.remoteState(ctx, b)
	if errors.Is(err, fs.ErrNotExist) {
		log.Infof(ctx, "Remote state file does not exist. Using local state.")
		return nil
	}
	if err != nil {
		return diag.Errorf("failed to read remote state file: %v", err)
	}

	if remoteState.Lineage == "" {
		return diag.Errorf("remote state file does not have a lineage")
	}

	localState, err := l.localState(ctx, b)
	if errors.Is(err, fs.ErrNotExist) {
		log.Infof(ctx, "Local state file does not exist. Using remote state.")
		if err := os.WriteFile(localStatePath, remoteContent, 0o600); err != nil {
			return diag.FromErr(err)
		}
		return nil
	}
	if err != nil {
		return diag.Errorf("failed to read local state file: %v", err)
	}

	if localState.Lineage != remoteState.Lineage {
		log.Infof(ctx, "Remote and local state lineages do not match. Using remote state. Invalidating local state.")
		if err := os.WriteFile(localStatePath, remoteContent, 0o600); err != nil {
			return diag.FromErr(err)
		}
		return nil
	}

	if remoteState.Serial > localState.Serial {
		log.Infof(ctx, "Remote state is newer than local state. Using remote state.")
		if err := os.WriteFile(localStatePath, remoteContent, 0o600); err != nil {
			return diag.FromErr(err)
		}
		return nil
	}

	// Local state is newer or equal; nothing to do.
	return nil
}

// StatePull returns a mutator that ensures the local bundle state is up-to-date
// with the state stored in the workspace.
func StatePull() bundle.Mutator {
	return &statePull{deploy.StateFiler}
}
