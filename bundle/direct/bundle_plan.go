package direct

import (
	"context"
	"errors"
	"fmt"
	"maps"
	"reflect"
	"slices"
	"strings"

	"github.com/databricks/cli/bundle/config"
	"github.com/databricks/cli/bundle/deployplan"
	"github.com/databricks/cli/bundle/direct/dresources"
	"github.com/databricks/cli/bundle/direct/dstate"
	"github.com/databricks/cli/libs/dyn"
	"github.com/databricks/cli/libs/dyn/dynvar"
	"github.com/databricks/cli/libs/log"
	"github.com/databricks/cli/libs/logdiag"
	"github.com/databricks/cli/libs/structs/structaccess"
	"github.com/databricks/cli/libs/structs/structdiff"
	"github.com/databricks/cli/libs/structs/structpath"
	"github.com/databricks/cli/libs/structs/structvar"
	"github.com/databricks/cli/libs/utils"
	"github.com/databricks/databricks-sdk-go"
)

var errDelayed = errors.New("must be resolved after apply")

func (b *DeploymentBundle) init(client *databricks.WorkspaceClient) error {
	if b.Adapters != nil {
		return nil
	}
	var err error
	b.Adapters, err = dresources.InitAll(client)
	return err
}

func (b *DeploymentBundle) CalculatePlan(ctx context.Context, client *databricks.WorkspaceClient, configRoot *config.Root, statePath string) (*deployplan.Plan, error) {
	err := b.StateDB.Open(statePath)
	if err != nil {
		return nil, fmt.Errorf("failed to read state from %s: %w", statePath, err)
	}

	err = b.init(client)
	if err != nil {
		return nil, err
	}

	plan, err := b.makePlan(ctx, configRoot, &b.StateDB.Data)
	if err != nil {
		return nil, fmt.Errorf("reading config: %w", err)
	}

	b.Plan = plan

	g, err := makeGraph(plan)
	if err != nil {
		return nil, err
	}

	err = g.DetectCycle()
	if err != nil {
		return nil, err
	}

	// We're processing resources in DAG order because we're resolving references (that can be resolved at plan stage).
	g.Run(defaultParallelism, func(resourceKey string, failedDependency *string) bool {
		errorPrefix := "cannot plan " + resourceKey

		entry, err := plan.WriteLockEntry(resourceKey)
		if err != nil {
			logdiag.LogError(ctx, fmt.Errorf("%s: internal error: %w", resourceKey, err))
			return false
		}

		if entry == nil {
			logdiag.LogError(ctx, fmt.Errorf("%s: internal error: node not in graph", resourceKey))
			return false
		}

		defer plan.WriteUnlockEntry(resourceKey)

		if failedDependency != nil {
			logdiag.LogError(ctx, fmt.Errorf("%s: dependency failed: %s", errorPrefix, *failedDependency))
			return false
		}

		adapter, err := b.getAdapterForKey(resourceKey)
		if err != nil {
			logdiag.LogError(ctx, fmt.Errorf("%s: %w", errorPrefix, err))
			return false
		}

		if entry.Action == deployplan.ActionTypeDelete.String() {
			dbentry, hasEntry := b.StateDB.GetResourceEntry(resourceKey)
			if !hasEntry {
				logdiag.LogError(ctx, fmt.Errorf("%s: internal error, missing in state", errorPrefix))
				return false
			}

			remoteState, err := adapter.DoRead(ctx, dbentry.ID)
			if err != nil {
				if isResourceGone(err) {
					// no such resource
					plan.RemoveEntry(resourceKey)
				} else {
					log.Warnf(ctx, "cannot read %s id=%q: %s", resourceKey, dbentry.ID, err)
					return false
				}
			}

			entry.RemoteState = remoteState

			return true
		}

		// Process all references in the resource using Refs map
		// Refs maps path inside resource to references e.g. "${resources.jobs.foo.id} ${resources.jobs.foo.name}"
		if !b.resolveReferences(ctx, entry, errorPrefix, true) {
			return false
		}

		dbentry, hasEntry := b.StateDB.GetResourceEntry(resourceKey)
		if !hasEntry {
			entry.Action = deployplan.ActionTypeCreate.String()
			return true
		}

		if dbentry.ID == "" {
			logdiag.LogError(ctx, fmt.Errorf("%s: invalid state empty id", errorPrefix))
			return false
		}

		savedState, err := typeConvert(adapter.StateType(), dbentry.State)
		if err != nil {
			logdiag.LogError(ctx, fmt.Errorf("%s: interpreting state: %w", errorPrefix, err))
			return false
		}

		// Note, currently we're diffing static structs, not dynamic value.
		// This means for fields that contain references like ${resources.group.foo.id} we do one of the following:
		// for strings: comparing unresolved string like "${resoures.group.foo.id}" with actual object id. As long as IDs do not have ${...} format we're good.
		// for integers: compare 0 with actual object ID. As long as real object IDs are never 0 we're good.
		// Once we add non-id fields or add per-field details to "bundle plan", we must read dynamic data and deal with references as first class citizen.
		// This means distinguishing between 0 that are actually object ids and 0 that are there because typed struct integer cannot contain ${...} string.
		localDiff, err := structdiff.GetStructDiff(savedState, entry.NewState.Value, adapter.KeyedSlices())
		if err != nil {
			logdiag.LogError(ctx, fmt.Errorf("%s: diffing local state: %w", errorPrefix, err))
			return false
		}

		remoteState, err := adapter.DoRead(ctx, dbentry.ID)
		if err != nil {
			if isResourceGone(err) {
				remoteState = nil
			} else {
				logdiag.LogError(ctx, fmt.Errorf("%s: failed to read id=%q: %w", errorPrefix, dbentry.ID, err))
				return false
			}
		}

		localAction, localChangeMap := localChangesToTriggers(ctx, adapter, localDiff, remoteState)
		if localAction == deployplan.ActionTypeRecreate {
			entry.Action = localAction.String()
			if len(localChangeMap) > 0 {
				entry.Changes = &deployplan.Changes{
					Local: localChangeMap,
				}
			}
			return true
		}

		// We have a choice whether to include remoteState or remoteStateComparable from below.
		// Including remoteState because in the near future remoteState is expected to become a superset struct of remoteStateComparable
		entry.RemoteState = remoteState

		var remoteAction deployplan.ActionType
		var remoteChangeMap map[string]deployplan.Trigger

		if remoteState == nil {
			remoteAction = deployplan.ActionTypeCreate
		} else {
			remoteStateComparable, err := adapter.RemapState(remoteState)
			if err != nil {
				logdiag.LogError(ctx, fmt.Errorf("%s: failed to interpret remote state id=%q: %w", errorPrefix, dbentry.ID, err))
				return false
			}

			remoteDiff, err := structdiff.GetStructDiff(savedState, remoteStateComparable, adapter.KeyedSlices())
			if err != nil {
				logdiag.LogError(ctx, fmt.Errorf("%s: diffing remote state: %w", errorPrefix, err))
				return false
			}

			remoteAction, remoteChangeMap = interpretOldStateVsRemoteState(ctx, adapter, remoteDiff, remoteState)
		}

		action := max(localAction, remoteAction)
		if action == deployplan.ActionTypeSkip {
			// resource is not going to change, can use remoteState to resolve references
			b.RemoteStateCache.Store(resourceKey, remoteState)
		}

		// Validate that resources without DoUpdate don't have update actions
		if action == deployplan.ActionTypeUpdate && !adapter.HasDoUpdate() {
			logdiag.LogError(ctx, fmt.Errorf("%s: resource does not support update action but plan produced update", errorPrefix))
			return false
		}

		entry.Action = action.String()

		if len(localChangeMap) > 0 || len(remoteChangeMap) > 0 {
			entry.Changes = &deployplan.Changes{
				Local:  localChangeMap,
				Remote: remoteChangeMap,
			}
		}

		return true
	})

	if logdiag.HasError(ctx) {
		return nil, errors.New("planning failed")
	}

	for _, entry := range plan.Plan {
		if entry.Action == deployplan.ActionTypeSkipString {
			entry.NewState = nil
		}
	}

	return plan, nil
}

func localChangesToTriggers(ctx context.Context, adapter *dresources.Adapter, diff []structdiff.Change, remoteState any) (deployplan.ActionType, map[string]deployplan.Trigger) {
	action := deployplan.ActionTypeSkip
	var m map[string]deployplan.Trigger

	for _, ch := range diff {
		fieldAction, err := adapter.ClassifyChange(ch, remoteState, true)
		if err != nil {
			logdiag.LogError(ctx, fmt.Errorf("internal error: failed to classify change: %w", err))
			continue
		}
		if fieldAction > action {
			action = fieldAction
		}
		if m == nil {
			m = make(map[string]deployplan.Trigger)
		}
		m[ch.Path.String()] = deployplan.Trigger{Action: fieldAction.String()}
	}

	return action, m
}

func interpretOldStateVsRemoteState(ctx context.Context, adapter *dresources.Adapter, diff []structdiff.Change, remoteState any) (deployplan.ActionType, map[string]deployplan.Trigger) {
	action := deployplan.ActionTypeSkip
	m := make(map[string]deployplan.Trigger)

	for _, ch := range diff {
		if ch.Old == nil && ch.Path.IsDotString() {
			// The field was not set by us, but comes from the remote state.
			// This could either be server-side default or a policy.
			// In any case, this is not a change we should react to.
			// Note, we only consider struct fields here. Adding/removing elements to/from maps and slices should trigger updates.
			m[ch.Path.String()] = deployplan.Trigger{
				Action: deployplan.ActionTypeSkipString,
				Reason: "server_side_default",
			}
			continue
		}
		fieldAction, err := adapter.ClassifyChange(ch, remoteState, false)
		if err != nil {
			logdiag.LogError(ctx, fmt.Errorf("internal error: failed to classify change: %w", err))
			continue
		}
		if fieldAction > action {
			action = fieldAction
		}
		m[ch.Path.String()] = deployplan.Trigger{Action: fieldAction.String()}
	}

	if len(m) == 0 {
		m = nil
	}

	return action, m
}

// TODO: calling this "Local" is not right, it can resolve "id" and remote refrences for "skip" targets
func (b *DeploymentBundle) LookupReferenceLocal(ctx context.Context, path *structpath.PathNode) (any, error) {
	// TODO: Prefix(3) assumes resources.jobs.foo but not resources.jobs.foo.permissions
	targetResourceKey := path.Prefix(3).String()

	fieldPath := path.SkipPrefix(3)
	fieldPathS := fieldPath.String()

	targetEntry, err := b.Plan.ReadLockEntry(targetResourceKey)
	if err != nil {
		return nil, err
	}

	if targetEntry == nil {
		return nil, fmt.Errorf("internal error: %s: missing entry in the plan", targetResourceKey)
	}

	defer b.Plan.ReadUnlockEntry(targetResourceKey)

	targetAction := deployplan.ActionTypeFromString(targetEntry.Action)
	if targetAction == deployplan.ActionTypeUndefined {
		return nil, fmt.Errorf("internal error: %s: missing action in the plan", targetResourceKey)
	}

	if fieldPathS == "id" {
		if targetAction.KeepsID() {
			dbentry, hasEntry := b.StateDB.GetResourceEntry(targetResourceKey)
			idValue := dbentry.ID
			if !hasEntry || idValue == "" {
				return nil, errors.New("internal error: no db entry")
			}
			return idValue, nil
		}
		// id may change after deployment, this needs to be done later
		return nil, errDelayed
	}

	if targetEntry.NewState == nil {
		return nil, fmt.Errorf("internal error: %s: action is %q missing new_state", targetResourceKey, targetEntry.Action)
	}

	_, isUnresolved := targetEntry.NewState.Refs[fieldPathS]
	if isUnresolved {
		// The value that is requested is itself a reference; this means it will be resolved after apply
		return nil, errDelayed
	}

	localConfig := targetEntry.NewState.Value

	targetGroup := config.GetResourceTypeFromKey(targetResourceKey)
	adapter := b.Adapters[targetGroup]
	if adapter == nil {
		return nil, fmt.Errorf("internal error: %s: unknown resource type %q", targetResourceKey, targetGroup)
	}

	configValidErr := structaccess.Validate(reflect.TypeOf(localConfig), fieldPath)
	remoteValidErr := structaccess.Validate(adapter.RemoteType(), fieldPath)
	// Note: using adapter.RemoteType() over reflect.TypeOf(remoteState) because remoteState might be untyped nil

	if configValidErr != nil && remoteValidErr != nil {
		return nil, fmt.Errorf("schema mismatch: %w; %w", configValidErr, remoteValidErr)
	}

	if configValidErr == nil && remoteValidErr != nil {
		// The field is only present in local schema; it must be resolved here.
		value, err := structaccess.Get(localConfig, fieldPath)
		if err != nil {
			return nil, fmt.Errorf("field not set: %w", err)
		}

		return value, nil
	}

	if configValidErr != nil && remoteValidErr == nil {
		// The field is only present in remote state schema.
		if targetAction != deployplan.ActionTypeSkip {
			// The resource is going to be updated, so remoteState can change
			return nil, errDelayed
		}
		remoteState, ok := b.RemoteStateCache.Load(targetResourceKey)
		if ok {
			return structaccess.Get(remoteState, fieldPath)
		}
		return nil, errDelayed
	}

	// Field is present in both: try local, fallback to remote (if skip) then delayed.

	value, err := structaccess.Get(localConfig, fieldPath)

	if err == nil && value != nil {
		return value, nil
	}

	if targetAction == deployplan.ActionTypeSkip {
		remoteState, ok := b.RemoteStateCache.Load(targetResourceKey)
		if ok {
			return structaccess.Get(remoteState, fieldPath)
		}
	}

	return nil, errDelayed
}

// resolveReferences processes all references in entry.NewState.Refs.
// If isLocal is true, uses LookupReferenceLocal (for planning phase).
// If isLocal is false, uses LookupReferenceRemote (for apply phase).
func (b *DeploymentBundle) resolveReferences(ctx context.Context, entry *deployplan.PlanEntry, errorPrefix string, isLocal bool) bool {
	for fieldPathStr, refString := range entry.NewState.Refs {
		refs, ok := dynvar.NewRef(dyn.V(refString))
		if !ok {
			logdiag.LogError(ctx, fmt.Errorf("%s: cannot parse %q", errorPrefix, refString))
			return false
		}

		for _, pathString := range refs.References() {
			ref := "${" + pathString + "}"
			targetPath, err := structpath.Parse(pathString)
			if err != nil {
				logdiag.LogError(ctx, fmt.Errorf("%s: cannot parse reference %q: %w", errorPrefix, ref, err))
				return false
			}

			var value any
			if isLocal {
				value, err = b.LookupReferenceLocal(ctx, targetPath)
				if err != nil {
					if errors.Is(err, errDelayed) {
						continue
					}
					logdiag.LogError(ctx, fmt.Errorf("%s: cannot resolve %q: %w", errorPrefix, ref, err))
					return false
				}
			} else {
				value, err = b.LookupReferenceRemote(ctx, targetPath)
				if err != nil {
					logdiag.LogError(ctx, fmt.Errorf("%s: cannot resolve %q: %w", errorPrefix, ref, err))
					return false
				}
			}

			err = entry.NewState.ResolveRef(ref, value)
			if err != nil {
				logdiag.LogError(ctx, fmt.Errorf("%s: cannot update %s with value of %q: %w", errorPrefix, fieldPathStr, ref, err))
				return false
			}
		}
	}
	return true
}

func (b *DeploymentBundle) makePlan(ctx context.Context, configRoot *config.Root, db *dstate.Database) (*deployplan.Plan, error) {
	p := deployplan.NewPlan()

	// Collect and sort nodes first, because MapByPattern gives them in randomized order
	var nodes []string

	existingKeys := maps.Clone(db.State)

	patterns := []dyn.Pattern{
		dyn.NewPattern(dyn.Key("resources"), dyn.AnyKey(), dyn.AnyKey()),
		dyn.NewPattern(dyn.Key("resources"), dyn.AnyKey(), dyn.AnyKey(), dyn.Key("permissions")),
		dyn.NewPattern(dyn.Key("resources"), dyn.AnyKey(), dyn.AnyKey(), dyn.Key("grants")),
	}

	// Walk?
	if configRoot != nil {
		for _, pat := range patterns {
			_, err := dyn.MapByPattern(
				configRoot.Value(),
				pat,
				func(p dyn.Path, v dyn.Value) (dyn.Value, error) {
					s := p.String()
					resourceType := config.GetResourceTypeFromKey(s)
					if resourceType == "" {
						return v, fmt.Errorf("cannot parse resource key: %q", s)
					}
					_, ok := dresources.SupportedResources[resourceType]
					if !ok {
						return v, fmt.Errorf("unsupported resource type: %s", resourceType)
					}

					nodes = append(nodes, s)
					return dyn.InvalidValue, nil
				},
			)
			if err != nil {
				return nil, fmt.Errorf("reading config: %w", err)
			}
		}
	}

	slices.Sort(nodes)

	for _, node := range nodes {
		delete(existingKeys, node)

		prefix := "cannot plan " + node
		inputConfig, err := configRoot.GetResourceConfig(node)
		if err != nil {
			return nil, fmt.Errorf("%s: reading config: %s", prefix, err)
		}

		adapter, err := b.getAdapterForKey(node)
		if err != nil {
			return nil, fmt.Errorf("%s: %w", prefix, err)
		}

		baseRefs := map[string]string{}

		if strings.HasSuffix(node, ".permissions") {
			inputConfigStructVar, err := dresources.PreparePermissionsInputConfig(inputConfig, node)
			if err != nil {
				return nil, err
			}
			inputConfig = inputConfigStructVar.Value
			baseRefs = inputConfigStructVar.Refs
		} else if strings.HasSuffix(node, ".grants") {
			inputConfigStructVar, err := dresources.PrepareGrantsInputConfig(inputConfig, node)
			if err != nil {
				return nil, err
			}
			inputConfig = inputConfigStructVar.Value
			baseRefs = inputConfigStructVar.Refs
		}

		newStateConfig, err := adapter.PrepareState(inputConfig)
		if err != nil {
			return nil, fmt.Errorf("%s: %w", prefix, err)
		}

		// Note, we're extracting references in input config but resolving them in newState.Config which is PrepareState(inputConfig)
		// This means input and state must be compatible: input can have more fields, but existing fields should not be moved
		// This means one cannot refer to fields not present in state (e.g. ${resources.jobs.foo.permissions})

		refs, err := extractReferences(configRoot.Value(), node)
		if err != nil {
			return nil, fmt.Errorf("failed to read references from config for %s: %w", node, err)
		}

		maps.Copy(refs, baseRefs)

		var dependsOn []deployplan.DependsOnEntry
		for _, reference := range refs {
			ref, ok := dynvar.NewRef(dyn.V(reference))
			if !ok {
				continue
			}

			for _, targetPath := range ref.References() {
				targetPathParsed, err := dyn.NewPathFromString(targetPath)
				if err != nil {
					return nil, fmt.Errorf("parsing %q: %w", targetPath, err)
				}

				targetNodeDP, _ := config.GetNodeAndType(targetPathParsed)
				targetNode := targetNodeDP.String()

				fullRef := "${" + targetPath + "}"

				found := false
				for _, dep := range dependsOn {
					if dep.Node == targetNode && dep.Label == fullRef {
						found = true
						break
					}
				}
				if !found {
					dependsOn = append(dependsOn, deployplan.DependsOnEntry{
						Node:  targetNode,
						Label: fullRef,
					})
				}
			}
		}

		slices.SortFunc(dependsOn, func(a, b deployplan.DependsOnEntry) int {
			if a.Node != b.Node {
				return strings.Compare(a.Node, b.Node)
			}
			return strings.Compare(a.Label, b.Label)
		})

		e := deployplan.PlanEntry{
			DependsOn: dependsOn,
			NewState: &structvar.StructVar{
				Value: newStateConfig,
				Refs:  refs,
			},
		}

		p.Plan[node] = &e
	}

	for n := range existingKeys {
		if p.Plan[n] != nil {
			panic("unexpected node " + n)
		}
		p.Plan[n] = &deployplan.PlanEntry{
			Action: deployplan.ActionTypeDelete.String(),
		}
	}

	return p, nil
}

func extractReferences(root dyn.Value, node string) (map[string]string, error) {
	nodeType := config.GetResourceTypeFromKey(node)
	refs := make(map[string]string)

	path, err := dyn.NewPathFromString(node)
	if err != nil {
		return nil, fmt.Errorf("internal error: bad node key: %q: %w", node, err)
	}

	val, err := dyn.GetByPath(root, path)
	if err != nil {
		return nil, err
	}

	err = dyn.WalkReadOnly(val, func(p dyn.Path, v dyn.Value) error {
		fullPath := append(path, p...)
		targetType := config.GetResourceTypeFromKey(fullPath.String())
		if targetType != nodeType {
			// Make sure these are associated with different nodes:
			// resources.jobs.foo...
			// resources.jobs.foo.permissions...
			// resources.jobs.foo.grants...
			return nil
		}
		ref, ok := dynvar.NewRef(v)
		if !ok {
			return nil
		}
		// Store the original string that contains references, not individual references
		refs[p.String()] = ref.Str
		return nil
	})
	if err != nil {
		return nil, fmt.Errorf("parsing refs: %w", err)
	}
	return refs, nil
}

func (b *DeploymentBundle) getAdapterForKey(resourceKey string) (*dresources.Adapter, error) {
	group := config.GetResourceTypeFromKey(resourceKey)
	if group == "" {
		return nil, fmt.Errorf("internal error: bad node: %s", resourceKey)
	}

	adapter, ok := b.Adapters[group]
	if !ok {
		return nil, fmt.Errorf("resource type %q not supported, available: %s", group, strings.Join(utils.SortedKeys(b.Adapters), ", "))
	}

	return adapter, nil
}
