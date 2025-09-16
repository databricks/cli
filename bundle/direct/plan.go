package direct

import (
	"context"
	"errors"
	"fmt"
	"reflect"

	"github.com/databricks/cli/bundle/deployplan"
	"github.com/databricks/cli/bundle/direct/dresources"
	"github.com/databricks/cli/bundle/direct/dstate"
	"github.com/databricks/cli/libs/dyn"
	"github.com/databricks/cli/libs/structs/structdiff"
	"github.com/databricks/databricks-sdk-go"

	"github.com/databricks/cli/libs/dyn/dynvar"
	"github.com/databricks/cli/libs/log"
	"github.com/databricks/cli/libs/structs/structaccess"
)

func (d *DeploymentUnit) Plan(ctx context.Context, client *databricks.WorkspaceClient, db *dstate.DeploymentState, inputConfig any, localOnly, refresh bool) (deployplan.ActionType, error) {
	result, err := d.plan(ctx, client, db, inputConfig, localOnly, refresh)
	if err != nil {
		return deployplan.ActionTypeNoop, fmt.Errorf("planning: %s.%s: %w", d.Group, d.Key, err)
	}
	return result, err
}

func (d *DeploymentUnit) plan(ctx context.Context, client *databricks.WorkspaceClient, db *dstate.DeploymentState, inputConfig any, localOnly, refresh bool) (deployplan.ActionType, error) {
	entry, hasEntry := db.GetResourceEntry(d.Group, d.Key)
	if !hasEntry {
		return deployplan.ActionTypeCreate, nil
	}
	if entry.ID == "" {
		return deployplan.ActionTypeUnset, errors.New("invalid state: empty id")
	}

	newState, err := d.Adapter.PrepareState(inputConfig)
	if err != nil {
		return deployplan.ActionTypeUnset, fmt.Errorf("reading config: %w", err)
	}

	savedState, err := typeConvert(d.Adapter.StateType(), entry.State)
	if err != nil {
		return deployplan.ActionTypeUnset, fmt.Errorf("interpreting state: %w", err)
	}

	// Note, currently we're diffing static structs, not dynamic value.
	// This means for fields that contain references like ${resources.group.foo.id} we do one of the following:
	// for strings: comparing unresolved string like "${resoures.group.foo.id}" with actual object id. As long as IDs do not have ${...} format we're good.
	// for integers: compare 0 with actual object ID. As long as real object IDs are never 0 we're good.
	// Once we add non-id fields or add per-field details to "bundle plan", we must read dynamic data and deal with references as first class citizen.
	// This means distinguishing between 0 that are actually object ids and 0 that are there because typed struct integer cannot contain ${...} string.
	return calcDiff(d.Adapter, savedState, newState)
}

func (d *DeploymentUnit) refreshRemoteState(ctx context.Context, id string) error {
	if d.RemoteState != nil {
		return nil
	}

	// TODO retry failures
	remoteState, err := d.Adapter.DoRefresh(ctx, id)
	if err != nil {
		return fmt.Errorf("failed to refresh remote state id=%s: %w", id, err)
	}
	err = d.SetRemoteState(remoteState)
	if err != nil {
		return err
	}

	return nil
}

var ErrDelayed = errors.New("must be resolved after apply")

func (d *DeploymentUnit) ResolveReferenceLocalOrRemote(ctx context.Context, db *dstate.DeploymentState, reference string, actionType deployplan.ActionType, config any) (any, error) {
	path, ok := dynvar.PureReferenceToPath(reference)
	if !ok || len(path) <= 3 || path[0].Key() != "resources" || path[1].Key() != d.Group || path[2].Key() != d.Key {
		return nil, fmt.Errorf("internal error: expected reference to resources.%s.%s, got %q", d.Group, d.Key, reference)
	}

	fieldPath := path[3:]

	if fieldPath.String() == "id" {
		if actionType.KeepsID() {
			entry, hasEntry := db.GetResourceEntry(d.Group, d.Key)
			idValue := entry.ID
			if !hasEntry || idValue == "" {
				return nil, errors.New("internal error: no db entry")
			}
			return idValue, nil
		}
		// id may change after deployment, this needs to be done later
		return nil, ErrDelayed
	}

	configValidErr := structaccess.Validate(reflect.TypeOf(config), fieldPath)
	remoteValidErr := structaccess.Validate(d.Adapter.RemoteType(), fieldPath)

	if configValidErr != nil && remoteValidErr != nil {
		return nil, fmt.Errorf("schema mismatch: %w; %w", configValidErr, remoteValidErr)
	}

	if configValidErr == nil && remoteValidErr != nil {
		// The field is only present in local schema; it must be resolved here.
		value, err := structaccess.Get(config, fieldPath)
		if err != nil {
			return nil, fmt.Errorf("field not set: %w", err)
		}

		return value, nil
	}

	if configValidErr != nil && remoteValidErr == nil {
		// The field is only present in remote state schema.
		// If resource is unchanged in this plan, we can proceed with resolution.
		// If resource is going to change, we need to postpone this until deploy.
		if !actionType.IsNoop() {
			return nil, ErrDelayed
		}

		return d.ReadRemoteStateField(ctx, db, fieldPath)

	}

	// Field is present in both: try local, fallback to remote.

	value, err := structaccess.Get(config, fieldPath)

	if err == nil && value != nil {
		return value, nil
	}

	if !actionType.IsNoop() {
		log.Infof(ctx, "Reference %q not found in config, delaying resolution after apply: %s", reference, err)
		// Can only proceed with remote resolution if resource is not going to be deployed
		return nil, ErrDelayed
	}

	log.Infof(ctx, "Reference %q not found in config, trying remote state: %s", reference, err)
	return d.ReadRemoteStateField(ctx, db, fieldPath)
}

func (d *DeploymentUnit) ResolveReferenceRemote(ctx context.Context, db *dstate.DeploymentState, reference string) (any, error) {
	path, ok := dynvar.PureReferenceToPath(reference)
	if !ok || len(path) <= 3 || path[0].Key() != "resources" || path[1].Key() != d.Group || path[2].Key() != d.Key {
		return nil, fmt.Errorf("internal error: expected reference to resources.%s.%s, got %q", d.Group, d.Key, reference)
	}

	fieldPath := path[3:]

	// Handle "id" field separately - read from state, not remote state
	if fieldPath.String() == "id" {
		entry, hasEntry := db.GetResourceEntry(d.Group, d.Key)
		if !hasEntry || entry.ID == "" {
			return nil, fmt.Errorf("internal error: no state entry or empty ID for %s.%s", d.Group, d.Key)
		}
		return entry.ID, nil
	}

	// Read other fields from remote state
	return d.ReadRemoteStateField(ctx, db, fieldPath)
}

func (d *DeploymentUnit) ReadRemoteStateField(ctx context.Context, db *dstate.DeploymentState, fieldPath dyn.Path) (any, error) {
	// We have options:
	// 1) Rely on the cached value; refresh if not cached.
	// 2) Always refresh, read the value.
	// 3) Consider this "unknown/variable" that always triggers a diff.
	// Long term we'll do (1), for now going with (2).
	// Not considering (3) because it would result in bad plans.

	entry, _ := db.GetResourceEntry(d.Group, d.Key)
	if entry.ID == "" {
		return nil, errors.New("internal error: Missing state entry")
	}

	err := d.refreshRemoteState(ctx, entry.ID)
	if err != nil {
		return nil, err
	}

	// Use remote state tracked by deployer
	remoteState := d.RemoteState
	if remoteState == nil {
		return nil, errors.New("no remote state available")
	}
	// remoteState cannot be nil there; but if it is, structaccess.Get will return an appropriate error

	value, errRemote := structaccess.Get(remoteState, fieldPath)
	if errRemote != nil {
		return nil, fmt.Errorf("field not set in remote state: %w", errRemote)
	}

	return value, nil
}

// TODO: return struct that has action but also individual differences and their effect (e.g. recreate)
func calcDiff(adapter *dresources.Adapter, savedState, config any) (deployplan.ActionType, error) {
	localDiff, err := structdiff.GetStructDiff(savedState, config)
	if err != nil {
		return deployplan.ActionTypeUnset, err
	}

	if len(localDiff) == 0 {
		return deployplan.ActionTypeNoop, nil
	}

	result := adapter.ClassifyByTriggers(localDiff)

	if result == deployplan.ActionTypeUpdateWithID && !adapter.HasDoUpdateWithID() {
		return deployplan.ActionTypeUnset, errors.New("internal error: unexpected plan='update_with_id'")
	}

	return result, nil
}
