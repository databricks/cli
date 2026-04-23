package metadata

import (
	"context"
	"time"

	"github.com/databricks/cli/internal/build"
	"github.com/databricks/cli/libs/log"
	"github.com/databricks/cli/ucm"
	"github.com/databricks/cli/ucm/deploy"
	"github.com/google/uuid"
)

// TODO(#83-followup): stamp a UUID into direct.State so DeploymentID is
// non-empty on direct-engine deploys. Until then Compute returns an empty
// DeploymentID whenever the active engine is direct, because direct's State
// has no ID field and never writes ucm-state.json.

// Compute builds the Metadata record for u. The DeploymentID is read from the
// local ucm-state.json cache so the blob ties to the same State.ID that Push
// just wrote; when the local cache is absent (first-run before Pull) or
// malformed DeploymentID is left empty rather than failing — the metadata
// blob is informational and must not block deploy.
//
// Compute mirrors Upload's nil-guard on u so the two functions, which are
// always called as a pair, share identical preconditions.
func Compute(ctx context.Context, u *ucm.Ucm) Metadata {
	if u == nil {
		return Metadata{}
	}
	md := Metadata{
		Version:    Version,
		CliVersion: build.GetInfo().Version,
		Ucm: UcmMeta{
			Name:   u.Config.Ucm.Name,
			Target: u.Config.Ucm.Target,
		},
		Timestamp: time.Now().UTC(),
	}

	if id := readDeploymentID(ctx, u); id != uuid.Nil {
		md.DeploymentID = id.String()
	}
	return md
}

// readDeploymentID returns the State.ID stored in the local ucm-state.json,
// or uuid.Nil when the file is absent, unreadable, or carries a nil ID.
// Delegates to deploy.LoadLocalState so the metadata package doesn't keep its
// own partial-JSON decoder in lock-step with the deploy state schema.
func readDeploymentID(ctx context.Context, u *ucm.Ucm) uuid.UUID {
	state, ok, err := deploy.LoadLocalState(ctx, u)
	if err != nil {
		log.Debugf(ctx, "ucm metadata: local state unreadable: %v", err)
		return uuid.Nil
	}
	if !ok || state.ID == uuid.Nil {
		return uuid.Nil
	}
	return state.ID
}
