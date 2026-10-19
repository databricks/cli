package direct

import (
	"context"
	"errors"
	"fmt"
	"strings"

	"github.com/databricks/cli/bundle/direct/dstate"
	"github.com/databricks/cli/libs/log"
	"github.com/databricks/cli/libs/structs/structaccess"
	"github.com/databricks/cli/libs/structs/structpath"
)

// ErrResourceAlreadyBound is returned when attempting to bind a resource
// that is already bound to a different ID in the state.
type ErrResourceAlreadyBound struct {
	ResourceKey string
	ExistingID  string
	NewID       string
}

func (e ErrResourceAlreadyBound) Error() string {
	if e.ExistingID != e.NewID {
		return fmt.Sprintf("Resource already managed\n\n"+
			"The bundle is already managing a resource for %s with ID '%s'.\n"+
			"To bind to a different resource with ID '%s', you must first unbind the existing resource.",
			e.ResourceKey, e.ExistingID, e.NewID)
	}
	return fmt.Sprintf("Resource already managed\n\nThe bundle is already managing resource for %s and it is bound to the requested ID %s.",
		e.ResourceKey, e.ExistingID)
}

// copyRemoteEtag copies the etag from remote state for resources that use etag-based drift
// detection (dashboards and genie spaces). The etag is not provided by the
// user; it comes from remote. If we don't store it in state, we won't
// detect remote drift correctly and the next plan shows a bogus update.
func copyRemoteEtag(resourceKey string, remoteState, newState any) {
	if (strings.Contains(resourceKey, ".dashboards.") || strings.Contains(resourceKey, ".genie_spaces.")) && remoteState != nil {
		etag, err := structaccess.Get(remoteState, structpath.NewStringKey(nil, "etag"))
		if err == nil && etag != nil {
			if etagStr, ok := etag.(string); ok && etagStr != "" {
				_ = structaccess.Set(newState, structpath.NewStringKey(nil, "etag"), etagStr)
			}
		}
	}
}

// Unbind removes a resource from direct engine state without deleting
// the workspace resource. Also removes associated permissions/grants entries.
func (b *DeploymentBundle) Unbind(ctx context.Context, statePath, resourceKey string) error {
	err := b.StateDB.Open(ctx, statePath, dstate.WithRecovery(true), dstate.WithWrite(true), dstate.WithDeploymentHistory(false), dstate.OpenDmsArgs{})
	if err != nil {
		if errors.Is(err, dstate.ErrUnsettingRecording) {
			return errors.New("unbind is not supported for a bundle target that records deployment history")
		}
		return err
	}

	// Delete the main resource
	err = b.StateDB.DeleteState(ctx, resourceKey, false)
	if err != nil {
		return err
	}

	log.Infof(ctx, "Unbound %s", resourceKey)

	// Also delete associated permissions/grants if they exist
	// These are stored as "resources.jobs.foo.permissions" or "resources.schemas.bar.grants"
	permissionsKey := resourceKey + ".permissions"
	grantsKey := resourceKey + ".grants"

	for key := range b.StateDB.Data.State {
		if key == permissionsKey || key == grantsKey || strings.HasPrefix(key, resourceKey+".") {
			err = b.StateDB.DeleteState(ctx, key, false)
			if err != nil {
				return err
			}
			log.Infof(ctx, "Unbound %s", key)
		}
	}

	_, err = b.StateDB.Finalize(ctx)
	return err
}
