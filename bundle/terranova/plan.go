package terranova

import (
	"context"
	"errors"
	"fmt"

	"github.com/databricks/cli/bundle/deployplan"
	"github.com/databricks/cli/bundle/terranova/tnstate"
	"github.com/databricks/cli/libs/structdiff"
	"github.com/databricks/databricks-sdk-go"
)

type Planner struct {
	client       *databricks.WorkspaceClient
	db           *tnstate.TerranovaState
	group        string
	resourceName string
	settings     ResourceSettings
}

func (d *Planner) Plan(ctx context.Context, inputConfig any) (deployplan.ActionType, error) {
	result, err := d.plan(ctx, inputConfig)
	if err != nil {
		return deployplan.ActionTypeNoop, fmt.Errorf("planning: %s.%s: %w", d.group, d.resourceName, err)
	}
	return result, err
}

func (d *Planner) plan(_ context.Context, inputConfig any) (deployplan.ActionType, error) {
	entry, hasEntry := d.db.GetResourceEntry(d.group, d.resourceName)

	resource, cfgType, err := New(d.client, d.group, d.resourceName, inputConfig)
	if err != nil {
		return "", err
	}

	config := resource.Config()

	if !hasEntry {
		return deployplan.ActionTypeCreate, nil
	}

	oldID := entry.ID
	if oldID == "" {
		return "", errors.New("invalid state: empty id")
	}

	savedState, err := typeConvert(cfgType, entry.State)
	if err != nil {
		return "", fmt.Errorf("interpreting state: %w", err)
	}

	// Note, currently we're diffing static structs, not dynamic value.
	// This means for fields that contain references like ${resources.group.foo.id} we do one of the following:
	// for strings: comparing unresolved string like "${resoures.group.foo.id}" with actual object id. As long as IDs do not have ${...} format we're good.
	// for integers: compare 0 with actual object ID. As long as real object IDs are never 0 we're good.
	// Once we add non-id fields or add per-field details to "bundle plan", we must read dynamic data and deal with references as first class citizen.
	// This means distinguishing between 0 that are actually object ids and 0 that are there because typed struct integer cannot contain ${...} string.
	return calcDiff(d.settings, resource, savedState, config)
}

func calcDiff(settings ResourceSettings, resource IResource, savedState, config any) (deployplan.ActionType, error) {
	localDiff, err := structdiff.GetStructDiff(savedState, config)
	if err != nil {
		return "", err
	}

	if len(localDiff) == 0 {
		return deployplan.ActionTypeNoop, nil
	}

	if settings.MustRecreate(localDiff) {
		return deployplan.ActionTypeRecreate, nil
	}

	customClassify, hasCustomClassify := resource.(IResourceCustomClassify)

	if hasCustomClassify {
		_, hasUpdateWithID := resource.(IResourceUpdatesID)

		result := customClassify.ClassifyChanges(localDiff)
		if result == deployplan.ActionTypeRecreate && !settings.RecreateAllowed {
			return "", errors.New("internal error: unexpected plan='recreate'")
		}

		if result == deployplan.ActionTypeUpdateWithID && !hasUpdateWithID {
			return "", errors.New("internal error: unexpected plan='update_with_id'")
		}

		return result, nil
	}

	return deployplan.ActionTypeUpdate, nil
}
