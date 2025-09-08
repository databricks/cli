package terranova

import (
	"context"
	"errors"
	"fmt"

	"github.com/databricks/cli/bundle/deployplan"
	"github.com/databricks/cli/bundle/terranova/tnresources"
	"github.com/databricks/cli/bundle/terranova/tnstate"
	"github.com/databricks/cli/libs/structdiff"
	"github.com/databricks/databricks-sdk-go"
)

func (d *DeploymentUnit) Plan(ctx context.Context, client *databricks.WorkspaceClient, db *tnstate.TerranovaState, inputConfig any) (deployplan.ActionType, error) {
	result, err := d.plan(ctx, client, db, inputConfig)
	if err != nil {
		return deployplan.ActionTypeNoop, fmt.Errorf("planning: %s.%s: %w", d.Group, d.Key, err)
	}
	return result, err
}

func (d *DeploymentUnit) plan(ctx context.Context, client *databricks.WorkspaceClient, db *tnstate.TerranovaState, inputConfig any) (deployplan.ActionType, error) {
	entry, hasEntry := db.GetResourceEntry(d.Group, d.Key)
	if !hasEntry {
		return deployplan.ActionTypeCreate, nil
	}
	if entry.ID == "" {
		return "", errors.New("invalid state: empty id")
	}

	config, err := d.Adapter.PrepareConfig(inputConfig)
	if err != nil {
		return "", fmt.Errorf("reading config: %w", err)
	}

	savedState, err := typeConvert(d.Adapter.ConfigType(), entry.State)
	if err != nil {
		return "", fmt.Errorf("interpreting state: %w", err)
	}

	// Note, currently we're diffing static structs, not dynamic value.
	// This means for fields that contain references like ${resources.group.foo.id} we do one of the following:
	// for strings: comparing unresolved string like "${resoures.group.foo.id}" with actual object id. As long as IDs do not have ${...} format we're good.
	// for integers: compare 0 with actual object ID. As long as real object IDs are never 0 we're good.
	// Once we add non-id fields or add per-field details to "bundle plan", we must read dynamic data and deal with references as first class citizen.
	// This means distinguishing between 0 that are actually object ids and 0 that are there because typed struct integer cannot contain ${...} string.
	return calcDiff(d.Adapter, savedState, config)
}

// TODO: return struct that has action but also individual differences and their effect (e.g. recreate)
func calcDiff(adapter *tnresources.Adapter, savedState, config any) (deployplan.ActionType, error) {
	localDiff, err := structdiff.GetStructDiff(savedState, config)
	if err != nil {
		return "", err
	}

	if len(localDiff) == 0 {
		return deployplan.ActionTypeNoop, nil
	}

	if adapter.MustRecreate(localDiff) {
		return deployplan.ActionTypeRecreate, nil
	}

	if adapter.HasClassifyChanges() {
		result, err := adapter.ClassifyChanges(localDiff)
		if err != nil {
			return "", err
		}

		if result == deployplan.ActionTypeUpdateWithID && !adapter.HasDoUpdateWithID() {
			return "", errors.New("internal error: unexpected plan='update_with_id'")
		}

		return result, nil
	}

	return deployplan.ActionTypeUpdate, nil
}
