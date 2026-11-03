package direct

import (
	"context"
	"errors"
	"fmt"
	"maps"
	"slices"
	"strings"

	"github.com/databricks/cli/bundle/config"
	"github.com/databricks/cli/bundle/deployplan"
	"github.com/databricks/cli/bundle/direct/dresources"
	"github.com/databricks/cli/libs/diag"
	"github.com/databricks/cli/libs/logdiag"
	"github.com/databricks/cli/libs/retry"
	"github.com/databricks/cli/libs/structs/structdiff"
	"github.com/databricks/cli/libs/structs/structpath"
	"github.com/databricks/databricks-sdk-go/apierr"
)

// lifecycleBindIDs collects lifecycle.bind IDs by resource key. A nil configRoot
// (destroy) has no bind blocks.
func lifecycleBindIDs(configRoot *config.Root) map[string]string {
	result := map[string]string{}
	if configRoot == nil {
		return result
	}
	for _, group := range configRoot.Resources.AllResources() {
		for name, resource := range group.Resources {
			bind := resource.GetLifecycle().GetBind()
			if bind == nil {
				continue
			}
			result["resources."+group.Description.PluralName+"."+name] = bind.ID
		}
	}
	return result
}

// bindIDPath returns the config path of a resource's lifecycle.bind.id, used to point
// diagnostics at the offending entry.
func bindIDPath(resourceKey string) *structpath.PathNode {
	resourceType, resourceName := splitResourceKey(resourceKey)
	return structpath.NewPath(nil, "resources", resourceType, resourceName, "lifecycle", "bind", "id")
}

func (b *DeploymentBundle) logBindError(ctx context.Context, configRoot *config.Root, resourceKey, summary string) {
	path := bindIDPath(resourceKey)
	logdiag.LogDiag(ctx, diag.Diagnostic{
		Severity:  diag.Error,
		Summary:   summary,
		Locations: configRoot.GetLocationsOf(path),
		Paths:     []*structpath.PathNode{path},
	})
}

// validateBindConfig logs diagnostics for lifecycle.bind entries that conflict with the
// state. Returns false if any errors were logged.
func (b *DeploymentBundle) validateBindConfig(ctx context.Context, configRoot *config.Root, bindIDByKey map[string]string) bool {
	hasError := false
	keys := slices.Sorted(maps.Keys(bindIDByKey))

	// A resource that is already in state cannot be re-bound to a different ID:
	// lifecycle.bind adopts a resource for the first time. When the IDs match the bind
	// is a no-op and the resource just deploys normally.
	for _, key := range keys {
		bindID := bindIDByKey[key]
		dbentry, hasEntry := b.StateDB.GetResourceEntry(key)
		if !hasEntry || dbentry.ID == "" || dbentry.ID == bindID {
			continue
		}
		b.logBindError(ctx, configRoot, key, fmt.Sprintf("%s: resource already bound to ID %q, cannot bind as %q; remove lifecycle.bind or unbind the existing resource", key, dbentry.ID, bindID))
		hasError = true
	}

	// Two resources must not bind the same workspace ID. Otherwise both would be
	// adopted and two state entries would manage one remote resource. IDs are only
	// unique within a resource type (a job and an experiment can share a numeric ID),
	// so only resources of the same type can collide.
	keysByTypeAndID := map[string][]string{}
	for _, key := range keys {
		keysByTypeAndID[typeAndID(key, bindIDByKey[key])] = append(keysByTypeAndID[typeAndID(key, bindIDByKey[key])], key)
	}
	for _, key := range keys {
		bindID := bindIDByKey[key]
		others := slices.DeleteFunc(slices.Clone(keysByTypeAndID[typeAndID(key, bindID)]), func(k string) bool { return k == key })
		if len(others) == 0 {
			continue
		}
		b.logBindError(ctx, configRoot, key, fmt.Sprintf("%s has the same lifecycle.bind ID %q as %s; each workspace resource can only be bound once", key, bindID, strings.Join(others, ", ")))
		hasError = true
	}

	// A bind ID must not collide with another resource's ID in state. Such a collision
	// would let a bind silently redirect a delete/recreate/update_id at a resource the
	// user is also trying to import; reject it instead of choosing one over the other.
	stateKeys := slices.Sorted(maps.Keys(b.StateDB.Data.State))
	for _, key := range keys {
		bindID := bindIDByKey[key]
		for _, stateKey := range stateKeys {
			if stateKey == key || b.StateDB.Data.State[stateKey].ID != bindID || typeAndID(stateKey, bindID) != typeAndID(key, bindID) {
				continue
			}
			b.logBindError(ctx, configRoot, key, fmt.Sprintf("%s has the same lifecycle.bind ID %q as existing resource %q; remove lifecycle.bind or the conflicting resource", key, bindID, stateKey))
			hasError = true
		}
	}

	return !hasError
}

// typeAndID keys a bind by resource type and workspace ID, the scope within which IDs
// are unique.
func typeAndID(resourceKey, bindID string) string {
	resourceType, _ := splitResourceKey(resourceKey)
	return resourceType + ":" + bindID
}

// handleBindPlan reads the remote resource for a resource that is being bound for
// the first time, computes the change set against the local config, and selects
// either Bind (no changes) or BindAndUpdate (config drift can be applied).
//
// Recreate / UpdateWithID actions imply destroying or replacing the workspace
// resource — incompatible with bind — so they surface as actionable errors.
// Returns false if an error was logged.
func (b *DeploymentBundle) handleBindPlan(ctx context.Context, resourceKey string, entry *deployplan.PlanEntry, adapter *dresources.Adapter) bool {
	bindID := entry.BindID
	errorPrefix := "cannot bind " + resourceKey

	remoteState, err := retry.With(ctx, retry.IsTransientRead, func() (any, error) {
		return adapter.DoRead(ctx, bindID)
	})
	if err != nil {
		if apierr.IsMissing(err) {
			logdiag.LogError(ctx, fmt.Errorf("%s: resource with ID %q does not exist in workspace", errorPrefix, bindID))
		} else {
			logdiag.LogError(ctx, fmt.Errorf("%s: reading remote resource id=%q: %w", errorPrefix, bindID, err))
		}
		return false
	}

	entry.RemoteState = remoteState
	b.RemoteStateCache.Store(resourceKey, remoteState)

	sv, ok := b.StateCache.Load(resourceKey)
	if !ok {
		logdiag.LogError(ctx, fmt.Errorf("%s: internal error: no state cache entry", errorPrefix))
		return false
	}

	remoteStateComparable, err := adapter.RemapState(remoteState)
	if err != nil {
		logdiag.LogError(ctx, fmt.Errorf("%s: interpreting remote state: %w", errorPrefix, err))
		return false
	}

	// Compact both sides on the same fields, so a hashed_fields field is a hash on
	// both sides of the diff. sv.Value keeps the full contents for the deploy.
	remoteStateComparable, err = dresources.CompactState(adapter.ResourceConfig(), remoteStateComparable)
	if err != nil {
		logdiag.LogError(ctx, fmt.Errorf("%s: compacting remote state: %w", errorPrefix, err))
		return false
	}
	localState, err := dresources.CompactState(adapter.ResourceConfig(), sv.Value)
	if err != nil {
		logdiag.LogError(ctx, fmt.Errorf("%s: compacting local state: %w", errorPrefix, err))
		return false
	}

	remoteDiff, err := structdiff.GetStructDiff(remoteStateComparable, localState, adapter.KeyedSlices())
	if err != nil {
		logdiag.LogError(ctx, fmt.Errorf("%s: diffing remote state: %w", errorPrefix, err))
		return false
	}

	// No "saved state" exists for a first-time bind; the diff is purely remote vs. config.
	entry.Changes, err = prepareChanges(ctx, adapter, nil, remoteDiff, nil, remoteStateComparable)
	if err != nil {
		logdiag.LogError(ctx, fmt.Errorf("%s: %w", errorPrefix, err))
		return false
	}

	err = addPerFieldActions(ctx, adapter, entry.Changes, sv.Value, remoteState)
	if err != nil {
		logdiag.LogError(ctx, fmt.Errorf("%s: classifying changes: %w", errorPrefix, err))
		return false
	}

	switch maxAction := getMaxAction(entry.Changes); maxAction {
	case deployplan.Skip, deployplan.Undefined:
		entry.Action = deployplan.Bind
	case deployplan.Update, deployplan.Resize:
		// Same guard as the in-state path: an update can only be applied if the
		// resource implements DoUpdate.
		if maxAction == deployplan.Update && !adapter.HasDoUpdate() {
			logdiag.LogError(ctx, fmt.Errorf("%s: resource does not support update action but plan produced update", errorPrefix))
			return false
		}
		entry.Action = deployplan.BindAndUpdate
	case deployplan.Recreate, deployplan.UpdateWithID:
		logdiag.LogError(ctx, buildBindConflictError(resourceKey, maxAction, entry.Changes, true))
		return false
	default:
		logdiag.LogError(ctx, fmt.Errorf("%s: internal error: unexpected action %q during bind planning", errorPrefix, maxAction))
		return false
	}
	return true
}

// buildBindConflictError formats a multi-line message that names the fields whose
// changes forced an incompatible action (Recreate / UpdateWithID) and shows the two
// concrete resolutions: drop the offending fields, or drop lifecycle.bind. firstBind
// tells a resource being adopted apart from one that is already bound in state.
func buildBindConflictError(resourceKey string, action deployplan.ActionType, changes deployplan.Changes, firstBind bool) error {
	problematicFields := make([]string, 0, len(changes))
	for fieldPath, change := range changes {
		if change.Action == action {
			problematicFields = append(problematicFields, fieldPath)
		}
	}
	// Stable order so the YAML examples are deterministic across runs.
	slices.Sort(problematicFields)

	resourceType, resourceName := splitResourceKey(resourceKey)

	var headline, effect, option1 string
	// buildBindConflictError is only invoked for Recreate / UpdateWithID; other
	// action types are guarded by the caller.
	switch { //nolint:exhaustive
	case action == deployplan.Recreate && firstBind:
		headline = "cannot bind " + resourceKey + ": cannot recreate resource with lifecycle.bind"
		effect = "This would destroy and recreate the existing workspace resource, changing its ID."
	case action == deployplan.Recreate:
		headline = "cannot plan " + resourceKey + ": lifecycle.bind prevents recreating the bound resource"
		effect = "This would destroy and recreate the bound workspace resource, changing its ID."
	case action == deployplan.UpdateWithID && firstBind:
		headline = "cannot bind " + resourceKey + ": cannot update resource ID with lifecycle.bind"
		effect = "This would replace the existing workspace resource with a new ID."
	case action == deployplan.UpdateWithID:
		headline = "cannot plan " + resourceKey + ": lifecycle.bind prevents changing the ID of the bound resource"
		effect = "This would replace the bound workspace resource with a new ID."
	}
	if firstBind {
		option1 = "Remove the problematic fields from your configuration to make this a bind-only operation:"
	} else {
		option1 = "Revert the problematic fields in your configuration to keep the bound resource as it is:"
	}

	var msg strings.Builder
	msg.WriteString(headline + "\n\n")
	msg.WriteString(effect + "\n\n")

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
	msg.WriteString("1. " + option1 + "\n\n")
	msg.WriteString("   resources:\n")
	fmt.Fprintf(&msg, "     %s:\n", resourceType)
	fmt.Fprintf(&msg, "       %s:\n", resourceName)
	msg.WriteString("         # Remove or comment out these fields:\n")
	for _, field := range problematicFields {
		fmt.Fprintf(&msg, "         # %s: ...\n", field)
	}
	msg.WriteString("\n")
	msg.WriteString("2. Remove lifecycle.bind if you want to allow the resource to be recreated/updated:\n\n")
	msg.WriteString("   resources:\n")
	fmt.Fprintf(&msg, "     %s:\n", resourceType)
	fmt.Fprintf(&msg, "       %s:\n", resourceName)
	msg.WriteString("         lifecycle:\n")
	msg.WriteString("           # Remove the bind block:\n")
	msg.WriteString("           # bind:\n")
	// No trailing newline: error strings must not end with one.
	msg.WriteString("           #   id: <resource_id>")

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
