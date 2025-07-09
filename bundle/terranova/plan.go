package terranova

import (
	"context"
	"errors"
	"fmt"

	"github.com/databricks/cli/bundle"
	"github.com/databricks/cli/bundle/deployplan"
	"github.com/databricks/cli/bundle/terranova/tnresources"
	"github.com/databricks/cli/bundle/terranova/tnstate"
	"github.com/databricks/cli/libs/dyn"
	"github.com/databricks/cli/libs/structdiff"
	"github.com/databricks/cli/libs/utils"
	"github.com/databricks/databricks-sdk-go"
)

type Planner struct {
	client       *databricks.WorkspaceClient
	db           *tnstate.TerranovaState
	group        string
	resourceName string
}

func (d *Planner) Plan(ctx context.Context, inputConfig any) (deployplan.ActionType, error) {
	entry, hasEntry := d.db.GetResourceEntry(d.group, d.resourceName)

	resource, cfgType, err := tnresources.New(d.client, d.group, d.resourceName, inputConfig)
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

	// TODO: GetStructDiff should deal with cases where it comes across
	// unresolved variables (so it needs additional support for dyn.Value storage).
	// In some cases, it should introduce "update or recreate" action, since it does not know whether
	// field is going to be changed.
	localDiff, err := structdiff.GetStructDiff(savedState, config)
	if err != nil {
		return "", fmt.Errorf("comparing state and config: %w", err)
	}

	if len(localDiff) == 0 {
		return deployplan.ActionTypeNoop, nil
	}

	return resource.ClassifyChanges(localDiff), nil
}

func CalculateDeployActions(ctx context.Context, b *bundle.Bundle) ([]deployplan.Action, error) {
	if !b.DirectDeployment {
		panic("direct deployment required")
	}

	err := b.OpenResourceDatabase(ctx)
	if err != nil {
		return nil, err
	}

	client := b.WorkspaceClient()
	var actions []deployplan.Action

	state := b.ResourceDatabase.ExportState(ctx)

	_, err = dyn.MapByPattern(
		b.Config.Value(),
		dyn.NewPattern(dyn.Key("resources"), dyn.AnyKey(), dyn.AnyKey()),
		func(p dyn.Path, v dyn.Value) (dyn.Value, error) {
			group := p[1].Key()
			name := p[2].Key()

			groupState := state[group]
			delete(groupState, name)

			config, ok := b.GetResourceConfig(group, name)
			if !ok {
				return dyn.InvalidValue, fmt.Errorf("internal error: cannot get config for %s.%s", group, name)
			}

			pl := Planner{
				client:       client,
				db:           &b.ResourceDatabase,
				group:        group,
				resourceName: name,
			}

			actionType, err := pl.Plan(ctx, config)
			if err != nil {
				return dyn.InvalidValue, err
			}

			if actionType != deployplan.ActionTypeNoop {
				actions = append(actions, deployplan.Action{
					Group:      group,
					Name:       name,
					ActionType: actionType,
				})
			}

			return v, nil
		},
	)

	// Remained in state are resources that no longer present in the config
	for group, groupState := range state {
		for name := range groupState {
			actions = append(actions, deployplan.Action{
				Group:      group,
				Name:       name,
				ActionType: deployplan.ActionTypeDelete,
			})
		}
	}

	if err != nil {
		return nil, fmt.Errorf("while reading resources config: %w", err)
	}

	return actions, nil
}

func CalculateDestroyActions(ctx context.Context, b *bundle.Bundle) ([]deployplan.Action, error) {
	if !b.DirectDeployment {
		panic("direct deployment required")
	}

	err := b.OpenResourceDatabase(ctx)
	if err != nil {
		return nil, err
	}

	db := &b.ResourceDatabase
	db.AssertOpened()
	var actions []deployplan.Action

	for _, group := range utils.SortedKeys(db.Data.Resources) {
		groupData := db.Data.Resources[group]
		for _, name := range utils.SortedKeys(groupData) {
			actions = append(actions, deployplan.Action{
				Group:      group,
				Name:       name,
				ActionType: deployplan.ActionTypeDelete,
			})
		}
	}

	return actions, nil
}
