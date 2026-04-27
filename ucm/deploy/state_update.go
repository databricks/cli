package deploy

import (
	"context"
	"fmt"
	"time"

	"github.com/databricks/cli/internal/build"
	"github.com/databricks/cli/libs/diag"
	"github.com/databricks/cli/ucm"
	"github.com/google/uuid"
)

// StateUpdate returns a mutator that advances the locally-cached
// ucm-state.json to the next deployment: Seq incremented by one, Version
// pinned to StateVersion, CliVersion stamped from build.GetInfo, Timestamp
// set to now (UTC), and ID initialised to a fresh UUID when the loaded
// state's ID is the zero value.
//
// Mirrors bundle/deploy.StateUpdate. Run between plan/approval and Push so
// the bump is durable on disk before any remote write — a crash between
// StateUpdate and Push leaves the local cache one Seq ahead of the remote,
// which the next Pull reconciles.
func StateUpdate() ucm.Mutator {
	return &stateUpdate{}
}

type stateUpdate struct{}

func (s *stateUpdate) Name() string {
	return "deploy:state-update"
}

func (s *stateUpdate) Apply(ctx context.Context, u *ucm.Ucm) diag.Diagnostics {
	localDir := LocalStateDir(u)
	prev, err := readLocalState(localDir)
	if err != nil {
		return diag.FromErr(fmt.Errorf("ucm state: read local %s: %w", UcmStateFileName, err))
	}

	next := advanceState(prev)

	if err := writeLocalState(localDir, next); err != nil {
		return diag.FromErr(fmt.Errorf("ucm state: refresh local %s: %w", UcmStateFileName, err))
	}
	return nil
}

// advanceState returns a copy of prev with Seq, Version, CliVersion,
// Timestamp, and (if needed) ID advanced. The input state is not mutated so
// callers can compare before/after without aliasing.
func advanceState(prev *State) *State {
	next := *prev
	next.Version = StateVersion
	next.Seq = prev.Seq + 1
	next.CliVersion = build.GetInfo().Version
	next.Timestamp = time.Now().UTC()
	if next.ID == uuid.Nil {
		next.ID = uuid.New()
	}
	return &next
}
