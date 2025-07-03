package statemgmt

import (
	"context"
	"encoding/json"
	"errors"
	"io"
	"io/fs"
	"os"

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
	return "statemgmt:state-pull"
}

func (l *statePull) remoteState(ctx context.Context, b *bundle.Bundle) (*tfState, []byte, error) {
	f, err := l.filerFactory(b)
	if err != nil {
		return nil, nil, err
	}

	r, err := f.Read(ctx, b.StateFilename())
	if err != nil {
		return nil, nil, err
	}
	defer r.Close()

	content, err := io.ReadAll(r)
	if err != nil {
		return nil, nil, err
	}

	state := &tfState{}
	err = json.Unmarshal(content, state)
	if err != nil {
		return nil, nil, err
	}

	return state, content, nil
}

func (l *statePull) localState(ctx context.Context, b *bundle.Bundle) (*tfState, error) {
	path, err := b.StateLocalPath(ctx)
	if err != nil {
		return nil, err
	}

	content, err := os.ReadFile(path)
	if err != nil {
		return nil, err
	}

	state := &tfState{}
	err = json.Unmarshal(content, state)
	if err != nil {
		return nil, err
	}

	return state, nil
}

func (l *statePull) Apply(ctx context.Context, b *bundle.Bundle) diag.Diagnostics {
	localStatePath, err := b.StateLocalPath(ctx)
	if err != nil {
		return diag.FromErr(err)
	}

	// Case: Remote state file does not exist. In this case we fallback to using the
	// local resources state. This allows users to change the "root_path" their bundle is
	// configured with.
	remoteState, remoteContent, err := l.remoteState(ctx, b)
	if errors.Is(err, fs.ErrNotExist) {
		log.Infof(ctx, "Remote state file does not exist. Using local resources state.")
		return nil
	}
	if err != nil {
		return diag.Errorf("failed to read remote state file: %v", err)
	}

	// Expected invariant: remote state file should have a lineage UUID. Error
	// if that's not the case.
	if remoteState.Lineage == "" {
		return diag.Errorf("remote state file does not have a lineage")
	}

	// Case: Local state file does not exist. In this case we should rely on the remote state file.
	localState, err := l.localState(ctx, b)
	if errors.Is(err, fs.ErrNotExist) {
		log.Infof(ctx, "Local state file does not exist. Using remote resources state.")
		err := os.WriteFile(localStatePath, remoteContent, 0o600)
		return diag.FromErr(err)
	}
	if err != nil {
		return diag.Errorf("failed to read local state file: %v", err)
	}

	// If the lineage does not match, the resources state files do not correspond to the same deployment.
	if localState.Lineage != remoteState.Lineage {
		log.Warnf(ctx, "Remote and local state lineages do not match (remote: %#v, local: %#v). Using remote resources state. Invalidating local resources state.", remoteState.Lineage, localState.Lineage)
		err := os.WriteFile(localStatePath, remoteContent, 0o600)
		return diag.FromErr(err)
	}

	// If the remote state is newer than the local state, we should use the remote state.
	if remoteState.Serial > localState.Serial {
		log.Infof(ctx, "Remote state is newer than local state (remote: %d, local: %d). Using remote resources state.", remoteState.Serial, localState.Serial)
		err := os.WriteFile(localStatePath, remoteContent, 0o600)
		return diag.FromErr(err)
	}

	// default: local state is newer or equal to remote state in terms of serial sequence.
	// It is also of the same lineage. Keep using the local state.
	return nil
}

func StatePull() bundle.Mutator {
	return &statePull{deploy.StateFiler}
}
