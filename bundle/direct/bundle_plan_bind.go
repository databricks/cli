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
)

// ApplyBindToPlan layers declarative bind blocks onto a plan that has already been
// produced by CalculatePlan. It validates the bind config and overrides plan
// entries for resources that are being bound for the first time.
//
// For each bind entry:
//   - If the resource has no state, override the planned Create with Bind or
//     BindAndUpdate by reading the remote resource and diffing against config.
//   - If the resource is already in state under the same ID, the bind is a no-op
//     (a previously-bound resource deploys normally).
//   - If the resource is in state under a different ID, surface an error.
func (b *DeploymentBundle) ApplyBindToPlan(ctx context.Context, configRoot *config.Root, bindConfig config.Bind) error {
	if bindConfig.IsEmpty() {
		return nil
	}

	if b.Plan == nil {
		return errors.New("internal error: ApplyBindToPlan called before CalculatePlan")
	}

	if !b.validateBindConfig(ctx, configRoot, bindConfig) {
		return errors.New("bind validation failed")
	}

	bindConfig.ForEach(func(resourceType, resourceName, bindID string) {
		resourceKey := "resources." + resourceType + "." + resourceName
		entry := b.Plan.Plan[resourceKey]

		dbentry, hasEntry := b.StateDB.GetResourceEntry(resourceKey)
		if hasEntry {
			if dbentry.ID != bindID {
				logdiag.LogError(ctx, fmt.Errorf("%s: resource already bound to ID %q, cannot bind as %q; remove the bind block or unbind the existing resource", resourceKey, dbentry.ID, bindID))
				return
			}
			// IDs match: leave the plan entry as-is so the resource deploys normally.
			return
		}

		entry.BindID = bindID
		adapter, err := b.getAdapterForKey(resourceKey)
		if err != nil {
			logdiag.LogError(ctx, fmt.Errorf("cannot plan %s: getting adapter: %w", resourceKey, err))
			return
		}

		b.handleBindPlan(ctx, resourceKey, entry, adapter)
	})

	if logdiag.HasError(ctx) {
		return errors.New("bind planning failed")
	}

	return nil
}

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

	remoteState, err := adapter.DoRead(ctx, bindID)
	if err != nil {
		if isResourceGone(err) {
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
