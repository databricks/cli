package direct

import (
	"context"
	"errors"
	"fmt"
	"slices"
	"strings"

	"github.com/databricks/cli/bundle/config"
	"github.com/databricks/cli/bundle/deployplan"
	"github.com/databricks/cli/bundle/direct/dresources"
	"github.com/databricks/cli/libs/diag"
	"github.com/databricks/cli/libs/dyn"
	"github.com/databricks/cli/libs/logdiag"
	"github.com/databricks/cli/libs/structs/structdiff"
	"github.com/databricks/databricks-sdk-go/apierr"
)

// validateBindConfig logs diagnostics for invalid bind entries. Returns false if any
// errors were logged.
func (b *DeploymentBundle) validateBindConfig(ctx context.Context, configRoot *config.Root, bindConfig config.Bind) bool {
	hasError := false
	targetName := configRoot.Bundle.Target

	// Bind blocks must reference resources that exist in the merged config.
	bindConfig.ForEach(func(resourceType, resourceName, bindID string) {
		key := "resources." + resourceType + "." + resourceName
		if _, ok := b.Plan.Plan[key]; ok {
			return
		}
		bindPath := dyn.NewPath(dyn.Key("targets"), dyn.Key(targetName), dyn.Key("bind"), dyn.Key(resourceType), dyn.Key(resourceName))
		logdiag.LogDiag(ctx, diag.Diagnostic{
			Severity:  diag.Error,
			Summary:   fmt.Sprintf("bind block references undefined resource %q; define it in the resources section or remove the bind block", key),
			Locations: configRoot.GetLocations(bindPath.String()),
			Paths:     []dyn.Path{bindPath},
		})
		hasError = true
	})

	// A resource that is already in state cannot be re-bound to a different ID: the
	// bind block is meant to import a resource for the first time. When the IDs match
	// the bind is a no-op and the resource just deploys normally.
	bindConfig.ForEach(func(resourceType, resourceName, bindID string) {
		key := "resources." + resourceType + "." + resourceName
		dbentry, hasEntry := b.StateDB.GetResourceEntry(key)
		if !hasEntry || dbentry.ID == "" || dbentry.ID == bindID {
			return
		}
		bindPath := dyn.NewPath(dyn.Key("targets"), dyn.Key(targetName), dyn.Key("bind"), dyn.Key(resourceType), dyn.Key(resourceName))
		logdiag.LogDiag(ctx, diag.Diagnostic{
			Severity:  diag.Error,
			Summary:   fmt.Sprintf("%s: resource already bound to ID %q, cannot bind as %q; remove the bind block or unbind the existing resource", key, dbentry.ID, bindID),
			Locations: configRoot.GetLocations(bindPath.String()),
			Paths:     []dyn.Path{bindPath},
		})
		hasError = true
	})

	// Two bind blocks must not point at the same workspace ID. Otherwise both would
	// be bound for the first time and create two state entries managing one remote
	// resource; reject the collision instead of silently double-managing it.
	bindKeysByID := map[string][]string{}
	bindConfig.ForEach(func(resourceType, resourceName, bindID string) {
		bindKey := "resources." + resourceType + "." + resourceName
		bindKeysByID[bindID] = append(bindKeysByID[bindID], bindKey)
	})
	bindConfig.ForEach(func(resourceType, resourceName, bindID string) {
		keys := bindKeysByID[bindID]
		if len(keys) < 2 {
			return
		}
		bindKey := "resources." + resourceType + "." + resourceName
		others := slices.DeleteFunc(slices.Clone(keys), func(k string) bool { return k == bindKey })
		bindPath := dyn.NewPath(dyn.Key("targets"), dyn.Key(targetName), dyn.Key("bind"), dyn.Key(resourceType), dyn.Key(resourceName))
		logdiag.LogDiag(ctx, diag.Diagnostic{
			Severity:  diag.Error,
			Summary:   fmt.Sprintf("bind block for %q has the same ID %q as bind block(s) for %s; each workspace resource can only be bound once", bindKey, bindID, strings.Join(others, ", ")),
			Locations: configRoot.GetLocations(bindPath.String()),
			Paths:     []dyn.Path{bindPath},
		})
		hasError = true
	})

	// A bind ID must not collide with another resource's ID in state. Such a collision
	// would let a bind silently redirect a delete/recreate/update_id at a resource the
	// user is also trying to import; reject it instead of choosing one over the other.
	stateKeys := make([]string, 0, len(b.StateDB.Data.State))
	for stateKey := range b.StateDB.Data.State {
		stateKeys = append(stateKeys, stateKey)
	}
	slices.Sort(stateKeys)

	bindConfig.ForEach(func(resourceType, resourceName, bindID string) {
		bindKey := "resources." + resourceType + "." + resourceName
		for _, stateKey := range stateKeys {
			stateEntry := b.StateDB.Data.State[stateKey]
			if stateKey == bindKey || stateEntry.ID != bindID {
				continue
			}
			bindPath := dyn.NewPath(dyn.Key("targets"), dyn.Key(targetName), dyn.Key("bind"), dyn.Key(resourceType), dyn.Key(resourceName))
			logdiag.LogDiag(ctx, diag.Diagnostic{
				Severity:  diag.Error,
				Summary:   fmt.Sprintf("bind block for %q has the same ID %q as existing resource %q; remove the bind block or the conflicting resource", bindKey, bindID, stateKey),
				Locations: configRoot.GetLocations(bindPath.String()),
				Paths:     []dyn.Path{bindPath},
			})
			hasError = true
		}
	})

	return !hasError
}

// handleBindPlan reads the remote resource for a resource that is being bound for
// the first time, computes the change set against the local config, and selects
// either Bind (no changes) or BindAndUpdate (config drift can be applied).
//
// Recreate / UpdateWithID actions imply destroying or replacing the workspace
// resource — incompatible with bind — so they surface as actionable errors.
func (b *DeploymentBundle) handleBindPlan(ctx context.Context, resourceKey string, entry *deployplan.PlanEntry, adapter *dresources.Adapter) {
	bindID := entry.BindID
	errorPrefix := "cannot bind " + resourceKey

	remoteState, err := retryOnTransient(ctx, func() (any, error) {
		return adapter.DoRead(ctx, bindID)
	})
	if err != nil {
		if apierr.IsMissing(err) {
			logdiag.LogError(ctx, fmt.Errorf("%s: resource with ID %q does not exist in workspace", errorPrefix, bindID))
		} else {
			logdiag.LogError(ctx, fmt.Errorf("%s: reading remote resource id=%q: %w", errorPrefix, bindID, err))
		}
		return
	}

	entry.RemoteState = remoteState
	b.RemoteStateCache.Store(resourceKey, remoteState)

	sv, ok := b.StateCache.Load(resourceKey)
	if !ok {
		logdiag.LogError(ctx, fmt.Errorf("%s: internal error: no state cache entry", errorPrefix))
		return
	}

	remoteStateComparable, err := adapter.RemapState(remoteState)
	if err != nil {
		logdiag.LogError(ctx, fmt.Errorf("%s: interpreting remote state: %w", errorPrefix, err))
		return
	}

	remoteDiff, err := structdiff.GetStructDiff(remoteStateComparable, sv.Value, adapter.KeyedSlices())
	if err != nil {
		logdiag.LogError(ctx, fmt.Errorf("%s: diffing remote state: %w", errorPrefix, err))
		return
	}

	// No "saved state" exists for a first-time bind; the diff is purely remote vs. config.
	entry.Changes, err = prepareChanges(ctx, adapter, nil, remoteDiff, nil, remoteStateComparable)
	if err != nil {
		logdiag.LogError(ctx, fmt.Errorf("%s: %w", errorPrefix, err))
		return
	}

	err = addPerFieldActions(ctx, adapter, entry.Changes, remoteState)
	if err != nil {
		logdiag.LogError(ctx, fmt.Errorf("%s: classifying changes: %w", errorPrefix, err))
		return
	}

	switch maxAction := getMaxAction(entry.Changes); maxAction {
	case deployplan.Skip, deployplan.Undefined:
		entry.Action = deployplan.Bind
	case deployplan.Update, deployplan.Resize:
		entry.Action = deployplan.BindAndUpdate
	case deployplan.Recreate, deployplan.UpdateWithID:
		logdiag.LogError(ctx, buildBindConflictError(errorPrefix, maxAction, entry.Changes, resourceKey))
	default:
		logdiag.LogError(ctx, fmt.Errorf("%s: internal error: unexpected action %q during bind planning", errorPrefix, maxAction))
	}
}

// buildBindConflictError formats a multi-line message that names the fields whose
// changes forced an incompatible action (Recreate / UpdateWithID) and shows the two
// concrete resolutions: drop the offending fields, or drop the bind block.
func buildBindConflictError(errorPrefix string, action deployplan.ActionType, changes deployplan.Changes, resourceKey string) error {
	problematicFields := make([]string, 0, len(changes))
	for fieldPath, change := range changes {
		if change.Action == action {
			problematicFields = append(problematicFields, fieldPath)
		}
	}
	// Stable order so the YAML examples are deterministic across runs.
	slices.Sort(problematicFields)

	resourceType, resourceName := splitResourceKey(resourceKey)

	var msg strings.Builder
	// buildBindConflictError is only invoked for Recreate / UpdateWithID; other
	// action types are guarded by the caller.
	switch action { //nolint:exhaustive
	case deployplan.Recreate:
		msg.WriteString(errorPrefix + ": cannot recreate resource with bind block\n\n")
		msg.WriteString("This would destroy and recreate the existing workspace resource, changing its ID.\n\n")
	case deployplan.UpdateWithID:
		msg.WriteString(errorPrefix + ": cannot update resource ID with bind block\n\n")
		msg.WriteString("This would replace the existing workspace resource with a new ID.\n\n")
	}

	msg.WriteString("The following fields cannot be modified because they require ")
	if action == deployplan.Recreate {
		msg.WriteString("resource recreation:\n")
	} else {
		msg.WriteString("ID changes:\n")
	}
	for _, field := range problematicFields {
		if reason := changes[field].Reason; reason != "" {
			fmt.Fprintf(&msg, "  - %s (%s)\n", field, reason)
		} else {
			fmt.Fprintf(&msg, "  - %s\n", field)
		}
	}
	msg.WriteString("\n")

	msg.WriteString("To resolve this issue, you have two options:\n\n")
	msg.WriteString("1. Remove the problematic fields from your configuration to make this a bind-only operation:\n\n")
	msg.WriteString("   resources:\n")
	fmt.Fprintf(&msg, "     %s:\n", resourceType)
	fmt.Fprintf(&msg, "       %s:\n", resourceName)
	msg.WriteString("         # Remove or comment out these fields:\n")
	for _, field := range problematicFields {
		fmt.Fprintf(&msg, "         # %s: ...\n", field)
	}
	msg.WriteString("\n")
	msg.WriteString("2. Remove the bind block if you want to allow the resource to be recreated/updated:\n\n")
	msg.WriteString("   targets:\n")
	msg.WriteString("     <target_name>:\n")
	msg.WriteString("       # Remove the bind block:\n")
	msg.WriteString("       # bind:\n")
	fmt.Fprintf(&msg, "       #   %s:\n", resourceType)
	fmt.Fprintf(&msg, "       #     %s:\n", resourceName)
	msg.WriteString("       #       id: <resource_id>\n")

	return errors.New(msg.String())
}

// splitResourceKey extracts the resource type and name from a key like
// "resources.jobs.foo" → ("jobs", "foo").
func splitResourceKey(resourceKey string) (resourceType, resourceName string) {
	parts := strings.SplitN(resourceKey, ".", 3)
	if len(parts) < 3 {
		return "", ""
	}
	return parts[1], parts[2]
}
