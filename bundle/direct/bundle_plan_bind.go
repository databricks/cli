package direct

import (
	"bytes"
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"maps"
	"reflect"
	"slices"
	"strings"

	"github.com/databricks/cli/bundle/config"
	"github.com/databricks/cli/bundle/deployplan"
	"github.com/databricks/cli/bundle/direct/dresources"
	"github.com/databricks/cli/libs/diag"
	"github.com/databricks/cli/libs/logdiag"
	"github.com/databricks/cli/libs/retry"
	"github.com/databricks/cli/libs/structs/structaccess"
	"github.com/databricks/cli/libs/structs/structdiff"
	"github.com/databricks/cli/libs/structs/structpath"
	"github.com/databricks/databricks-sdk-go"
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

// PrepareBindings reads and validates resources to adopt without modifying deployment state.
func (b *DeploymentBundle) PrepareBindings(ctx context.Context, client *databricks.WorkspaceClient, configRoot *config.Root, selected []string) (map[string]*deployplan.BindEntry, error) {
	b.StateDB.AssertOpenedForRead()
	bindIDs := lifecycleBindIDs(configRoot)
	if len(bindIDs) == 0 {
		return nil, nil
	}
	if !b.validateBindConfig(ctx, configRoot, bindIDs) {
		return nil, errors.New("bind validation failed")
	}
	if err := b.init(client); err != nil {
		return nil, err
	}
	configured, err := b.makePlan(ctx, configRoot, &b.StateDB.Data)
	if err != nil {
		return nil, err
	}
	if len(selected) > 0 {
		if err := configured.FilterToSelected(ctx, selected); err != nil {
			return nil, err
		}
	}
	bindings := make(map[string]*deployplan.BindEntry)
	for _, resourceKey := range slices.Sorted(maps.Keys(bindIDs)) {
		entry := configured.Plan[resourceKey]
		if entry == nil || b.StateDB.GetResourceID(resourceKey) != "" {
			continue
		}
		adapter, err := b.getAdapterForKey(resourceKey)
		if err != nil {
			return nil, err
		}
		remote, err := readBoundResource(ctx, adapter, resourceKey, bindIDs[resourceKey])
		if err != nil {
			return nil, err
		}
		state, err := adapter.RemapState(remote)
		if err != nil {
			return nil, fmt.Errorf("reading binding state for %s: %w", resourceKey, err)
		}
		configuredState, _ := b.StateCache.Load(resourceKey)
		state, err = prepareBindingState(adapter, state, configuredState.Value)
		if err != nil {
			return nil, fmt.Errorf("preparing binding for %s: %w", resourceKey, err)
		}
		encoded, err := json.Marshal(state)
		if err != nil {
			return nil, fmt.Errorf("serializing binding for %s: %w", resourceKey, err)
		}
		bindings[resourceKey] = &deployplan.BindEntry{
			ID: bindIDs[resourceKey], State: encoded, DependsOn: entry.DependsOn,
		}
	}
	return bindings, nil
}

// prepareBindingState imports the remote values of configured fields. Importing
// backend-only fields would claim ownership of defaults the bundle never set;
// importing desired values would hide changes that have not been applied yet.
func prepareBindingState(adapter *dresources.Adapter, remote, configured any) (any, error) {
	diff, err := structdiff.GetStructDiff(remote, configured, adapter.KeyedSlices())
	if err != nil {
		return nil, err
	}
	encoded, err := json.Marshal(remote)
	if err != nil {
		return nil, err
	}
	var state any
	decoder := json.NewDecoder(bytes.NewReader(encoded))
	decoder.UseNumber()
	if err := decoder.Decode(&state); err != nil {
		return nil, err
	}
	// Remove positional array entries from the end so earlier indices stay valid.
	for _, change := range slices.Backward(diff) {
		value := reflect.ValueOf(change.New)
		if value.IsValid() {
			switch value.Kind() {
			case reflect.Pointer, reflect.Map, reflect.Slice, reflect.Interface:
				if !value.IsNil() {
					continue
				}
			default:
				continue
			}
		}
		// Dashboards and Genie spaces persist the remote etag for drift detection.
		if change.Path.String() == "etag" {
			continue
		}
		if change.Path.IsRoot() {
			return nil, errors.New("cannot bind an empty resource configuration")
		}
		state, err = removeBindingField(state, change.Path.AsSlice())
		if err != nil {
			return nil, fmt.Errorf("removing unconfigured field %s: %w", change.Path, err)
		}
	}
	encoded, err = json.Marshal(state)
	if err != nil {
		return nil, err
	}
	typed, err := parseState(adapter.StateType(), encoded)
	if err != nil {
		return nil, err
	}
	return compactStateForSave(adapter, typed)
}

// removeBindingField removes a field from a JSON value, including keyed array
// elements such as tasks[task_key='main']. The paths come from structdiff.
func removeBindingField(value any, path []*structpath.PathNode) (any, error) {
	node := path[0]
	if key, ok := node.StringKey(); ok {
		object, ok := value.(map[string]any)
		if !ok {
			return nil, fmt.Errorf("expected an object at %s", node.Parent())
		}
		if len(path) == 1 {
			delete(object, key)
			return object, nil
		}
		child, err := removeBindingField(object[key], path[1:])
		if err != nil {
			return nil, err
		}
		object[key] = child
		return object, nil
	}
	sequence, ok := value.([]any)
	if !ok {
		return nil, fmt.Errorf("expected an array at %s", node.Parent())
	}
	index, indexed := node.Index()
	if !indexed {
		key, match, _ := node.KeyValue()
		index = slices.IndexFunc(sequence, func(element any) bool {
			field, _ := structaccess.GetByString(element, key)
			return field == match
		})
	}
	if index < 0 || index >= len(sequence) {
		return nil, fmt.Errorf("array element %s does not exist", node)
	}
	if len(path) == 1 {
		return slices.Delete(sequence, index, index+1), nil
	}
	child, err := removeBindingField(sequence[index], path[1:])
	if err != nil {
		return nil, err
	}
	sequence[index] = child
	return sequence, nil
}

func readBoundResource(ctx context.Context, adapter *dresources.Adapter, resourceKey, id string) (any, error) {
	remote, err := retry.With(ctx, retry.IsTransientRead, func() (any, error) {
		return adapter.DoRead(ctx, id)
	})
	if apierr.IsMissing(err) || (err == nil && adapter.IsGone(remote)) {
		return nil, fmt.Errorf("cannot bind %s: resource with ID %q does not exist in workspace", resourceKey, id)
	}
	if err != nil {
		return nil, fmt.Errorf("cannot bind %s: reading remote resource id=%q: %w", resourceKey, id, err)
	}
	return remote, nil
}

// CheckBindings verifies that resources from a saved bind phase still exist.
func (b *DeploymentBundle) CheckBindings(ctx context.Context, client *databricks.WorkspaceClient, bindings map[string]*deployplan.BindEntry) error {
	if err := b.init(client); err != nil {
		return err
	}
	for _, key := range slices.Sorted(maps.Keys(bindings)) {
		adapter, err := b.getAdapterForKey(key)
		if err != nil {
			return err
		}
		if _, err := readBoundResource(ctx, adapter, key, bindings[key].ID); err != nil {
			return err
		}
	}
	return nil
}

func (b *DeploymentBundle) checkBoundResource(ctx context.Context, resourceKey string, entry *deployplan.PlanEntry, adapter *dresources.Adapter) error {
	if !entry.Action.KeepsID() {
		return fmt.Errorf("cannot %s %s: lifecycle.bind requires preserving resource ID %q", entry.Action, resourceKey, entry.ID)
	}
	id := b.StateDB.GetResourceID(resourceKey)
	if id != entry.ID {
		return fmt.Errorf("cannot deploy %s: bound resource ID %q does not match state ID %q", resourceKey, entry.ID, id)
	}
	remote, err := retry.With(ctx, retry.IsTransientRead, func() (any, error) {
		return adapter.DoRead(ctx, id)
	})
	if apierr.IsMissing(err) || (err == nil && adapter.IsGone(remote)) {
		return fmt.Errorf("cannot deploy %s: bound resource with ID %q no longer exists in the workspace; remove lifecycle.bind to create a new resource", resourceKey, id)
	}
	if err != nil {
		return fmt.Errorf("reading bound resource %s: %w", resourceKey, err)
	}
	configured, ok := b.StateCache.Load(resourceKey)
	if !ok {
		return fmt.Errorf("cannot deploy %s: missing configured state for bound resource", resourceKey)
	}
	local, err := dresources.CompactState(adapter.ResourceConfig(), configured.Value)
	if err != nil {
		return err
	}
	comparable, err := adapter.RemapState(remote)
	if err != nil {
		return err
	}
	comparable, err = dresources.CompactState(adapter.ResourceConfig(), comparable)
	if err != nil {
		return err
	}
	diff, err := structdiff.GetStructDiff(comparable, local, adapter.KeyedSlices())
	if err != nil {
		return err
	}
	saved, _ := b.StateDB.GetResourceEntry(resourceKey)
	old, err := parseState(adapter.StateType(), saved.State)
	if err != nil {
		return err
	}
	changes, err := prepareChanges(ctx, adapter, nil, diff, old, comparable)
	if err != nil {
		return err
	}
	if err := addPerFieldActions(ctx, adapter, changes, configured.Value, remote); err != nil {
		return err
	}
	action := getMaxAction(changes)
	if action == deployplan.Recreate || action == deployplan.UpdateWithID {
		return fmt.Errorf("cannot deploy %s: %w", resourceKey, buildBindConflictError(action, changes))
	}
	return nil
}

// buildBindConflictError names the fields that would change the bound resource's ID.
func buildBindConflictError(action deployplan.ActionType, changes deployplan.Changes) error {
	problematicFields := make([]string, 0, len(changes))
	for fieldPath, change := range changes {
		if change.Action == action {
			problematicFields = append(problematicFields, fieldPath)
		}
	}
	slices.Sort(problematicFields)

	var headline string
	// buildBindConflictError is only invoked for Recreate / UpdateWithID; other
	// action types are guarded by the caller.
	switch action { //nolint:exhaustive
	case deployplan.Recreate:
		headline = "lifecycle.bind prevents recreating the bound resource"
	case deployplan.UpdateWithID:
		headline = "lifecycle.bind prevents changing the ID of the bound resource"
	}

	var msg strings.Builder
	msg.WriteString(headline + "\n\n")
	msg.WriteString("Fields requiring ")
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
	msg.WriteString("\nMatch these fields to the existing resource to preserve its ID")
	msg.WriteString("; remove lifecycle.bind to allow the ID to change")

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
