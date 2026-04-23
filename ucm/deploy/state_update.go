package deploy

import (
	"time"

	"github.com/databricks/cli/internal/build"
	"github.com/google/uuid"
)

// StateUpdate returns a copy of prev advanced to the next deployment: Seq
// incremented by one, Version pinned to StateVersion, CliVersion stamped from
// build.GetInfo, Timestamp set to now (UTC), and ID initialised to a fresh
// UUID when prev.ID is the zero value. The input state is not mutated so
// callers can compare before/after without aliasing.
func StateUpdate(prev *State) *State {
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
