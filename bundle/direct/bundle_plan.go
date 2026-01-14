package direct

import (
	"context"
	"errors"
	"fmt"
	"maps"
	"os"
	"reflect"
	"slices"
	"strings"

	"github.com/databricks/cli/bundle/config"
	"github.com/databricks/cli/bundle/config/resources"
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

// ValidatePlanAgainstState validates that a plan's lineage and serial match the current state.
// This should be called early in the deployment process, before any file operations.
// If the plan has no lineage (first deployment), validation is skipped.
func ValidatePlanAgainstState(statePath string, plan *deployplan.Plan) error {
	// If plan has no lineage, this is a first deployment before any state exists
	// No validation needed
	if plan.Lineage == "" {
		return nil
	}

	var stateDB dstate.DeploymentState
	err := stateDB.Open(statePath)
	if err != nil {
		// If state file doesn't exist but plan has lineage, something is wrong
		if os.IsNotExist(err) {
			return fmt.Errorf("plan has lineage %q but state file does not exist at %s; the state may have been deleted", plan.Lineage, statePath)
		}
		return fmt.Errorf("reading state from %s: %w", statePath, err)
	}

	// Validate that the plan's lineage matches the current state's lineage
	if plan.Lineage != stateDB.Data.Lineage {
		return fmt.Errorf("plan lineage %q does not match state lineage %q; the state may have been modified by another process", plan.Lineage, stateDB.Data.Lineage)
	}

	// Validate that the plan's serial matches the current state's serial
	if plan.Serial != stateDB.Data.Serial {
		return fmt.Errorf("plan serial %d does not match state serial %d; the state has been modified since the plan was created. Please run 'bundle plan' again", plan.Serial, stateDB.Data.Serial)
	}

	return nil
}

// InitForApply initializes the DeploymentBundle for applying a pre-computed plan.
// This is used when --plan is specified to skip the planning phase.
func (b *DeploymentBundle) InitForApply(ctx context.Context, client *databricks.WorkspaceClient, statePath string, plan *deployplan.Plan) error {
	err := b.StateDB.Open(statePath)
	if err != nil {
		return fmt.Errorf("reading state from %s: %w", statePath, err)
	}

	err = b.init(client)
	if err != nil {
		return err
	}

	// Eagerly parse all StructVarJSON entries to catch parsing errors early.
	// When the plan is read from JSON, Value contains raw JSON bytes.
	// We parse them into typed structs and cache them for later use.
	for resourceKey, entry := range plan.Plan {
		if entry.NewState == nil || len(entry.NewState.Value) == 0 {
			continue
		}

		adapter, err := b.getAdapterForKey(resourceKey)
		if err != nil {
			return fmt.Errorf("converting plan entry %s: %w", resourceKey, err)
		}

		sv, err := entry.NewState.ToStructVar(adapter.StateType())
		if err != nil {
			return fmt.Errorf("loading plan entry %s: %w", resourceKey, err)
		}

		b.StructVarCache.Store(resourceKey, sv)
	}

	b.Plan = plan
	return nil
}

func (b *DeploymentBundle) CalculatePlan(ctx context.Context, client *databricks.WorkspaceClient, configRoot *config.Root, statePath string) (*deployplan.Plan, error) {
	err := b.StateDB.Open(statePath)
	if err != nil {
		return nil, fmt.Errorf("reading state from %s: %w", statePath, err)
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
			logdiag.LogError(ctx, fmt.Errorf("%s: internal error: %w", errorPrefix, err))
			return false
		}

		if entry == nil {
			logdiag.LogError(ctx, fmt.Errorf("%s: internal error: node not in graph", errorPrefix))
			return false
		}

		defer plan.WriteUnlockEntry(resourceKey)

		if failedDependency != nil {
			// TODO: this should be a warning
			logdiag.LogError(ctx, fmt.Errorf("%s: dependency failed: %s", errorPrefix, *failedDependency))
			return false
		}

		adapter, err := b.getAdapterForKey(resourceKey)
		if err != nil {
			logdiag.LogError(ctx, fmt.Errorf("%s: getting adapter: %w", errorPrefix, err))
			return false
		}

		if entry.Action == deployplan.Delete {
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
					log.Warnf(ctx, "reading %s id=%q: %s", resourceKey, dbentry.ID, err)
					return false
				}
			}

			entry.RemoteState = remoteState

			return true
		}

		// Process all references in the resource using Refs map
		// Refs maps path inside resource to references e.g. "${resources.jobs.foo.id} ${resources.jobs.foo.name}"
		if !b.resolveReferences(ctx, resourceKey, entry, errorPrefix, true) {
			return false
		}

		dbentry, hasEntry := b.StateDB.GetResourceEntry(resourceKey)
		if !hasEntry {
			entry.Action = deployplan.Create
			return true
		}

		if dbentry.ID == "" {
			logdiag.LogError(ctx, fmt.Errorf("%s: invalid state empty id", errorPrefix))
			return false
		}

		savedState, err := parseState(adapter.StateType(), dbentry.State)
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
		sv, ok := b.StructVarCache.Load(resourceKey)
		if !ok {
			logdiag.LogError(ctx, fmt.Errorf("%s: internal error: no state found for %q", errorPrefix, resourceKey))
			return false
		}
		localDiff, err := structdiff.GetStructDiff(savedState, sv.Value, adapter.KeyedSlices())
		if err != nil {
			logdiag.LogError(ctx, fmt.Errorf("%s: diffing local state: %w", errorPrefix, err))
			return false
		}

		remoteState, err := adapter.DoRead(ctx, dbentry.ID)
		if err != nil {
			if isResourceGone(err) {
				remoteState = nil
			} else {
				logdiag.LogError(ctx, fmt.Errorf("%s: reading id=%q: %w", errorPrefix, dbentry.ID, err))
				return false
			}
		}

		// We have a choice whether to include remoteState or remoteStateComparable from below.
		// Including remoteState because in the near future remoteState is expected to become a superset struct of remoteStateComparable
		entry.RemoteState = remoteState

		var action deployplan.ActionType
		var remoteDiff []structdiff.Change
		var remoteStateComparable any

		if remoteState != nil {
			remoteStateComparable, err = adapter.RemapState(remoteState)
			if err != nil {
				logdiag.LogError(ctx, fmt.Errorf("%s: interpreting remote state id=%q: %w", errorPrefix, dbentry.ID, err))
				return false
			}

			remoteDiff, err = structdiff.GetStructDiff(remoteStateComparable, sv.Value, adapter.KeyedSlices())
			if err != nil {
				logdiag.LogError(ctx, fmt.Errorf("%s: diffing remote state: %w", errorPrefix, err))
				return false
			}
		}

		entry.Changes, err = prepareChanges(ctx, adapter, localDiff, remoteDiff, savedState, remoteState != nil)
		if err != nil {
			logdiag.LogError(ctx, fmt.Errorf("%s: %w", errorPrefix, err))
			return false
		}

		err = addPerFieldActions(ctx, adapter, entry.Changes, remoteState)
		if err != nil {
			logdiag.LogError(ctx, fmt.Errorf("%s: classifying changes: %w", errorPrefix, err))
			return false
		}

		if remoteState == nil {
			// Even if local action is "recreate" which is higher than "create", we should still pick "create" here
			// because we know remote does not exist.
			action = deployplan.Create
		} else {
			action = getMaxAction(entry.Changes)
		}

		if action == deployplan.Skip {
			// resource is not going to change, can use remoteState to resolve references
			b.RemoteStateCache.Store(resourceKey, remoteState)
		}

		// Validate that resources without DoUpdate don't have update actions
		if action == deployplan.Update && !adapter.HasDoUpdate() {
			logdiag.LogError(ctx, fmt.Errorf("%s: resource does not support update action but plan produced update", errorPrefix))
			return false
		}

		entry.Action = action
		return true
	})

	if logdiag.HasError(ctx) {
		return nil, errors.New("planning failed")
	}

	for _, entry := range plan.Plan {
		if entry.Action == deployplan.Skip {
			entry.NewState = nil
		}
	}

	return plan, nil
}

func getMaxAction(m map[string]*deployplan.ChangeDesc) deployplan.ActionType {
	result := deployplan.Skip
	for _, ch := range m {
		result = deployplan.GetHigherAction(result, ch.Action)
	}
	return result
}

func prepareChanges(ctx context.Context, adapter *dresources.Adapter, localDiff, remoteDiff []structdiff.Change, oldState any, hasRemote bool) (deployplan.Changes, error) {
	m := make(deployplan.Changes)

	for _, ch := range localDiff {
		e := deployplan.ChangeDesc{
			Old: ch.Old,
			New: ch.New,
		}
		if hasRemote {
			// by default, assume e.Remote is the same as config; if not the case it'll be ovewritten below
			e.Remote = ch.New
		}
		m[ch.Path.String()] = &e
	}

	for _, ch := range remoteDiff {
		entry := m[ch.Path.String()]
		if entry == nil {
			// we have difference for remoteState but not difference for localState
			// from remoteDiff we can find out remote value (ch.Old) and new config value (ch.New) but we don't know oldState value
			oldStateVal, err := structaccess.Get(oldState, ch.Path)
			var notFound *structaccess.NotFoundError
			if err != nil && !errors.As(err, &notFound) {
				log.Debugf(ctx, "Constructing diff: accessing %q on %T: %s", ch.Path, oldState, err)
			}
			m[ch.Path.String()] = &deployplan.ChangeDesc{
				Old:    oldStateVal,
				New:    ch.New,
				Remote: ch.Old,
			}
		} else {
			entry.Remote = ch.Old
			if !structdiff.IsEqual(entry.New, ch.New) {
				// this is not fatal (may result in unexpected drift or undetected change but not incorrect deploy), but good to log this
				log.Warnf(ctx, "unexpected local and remote diffs (%T, %T); entry=%v ch=%v", entry.New, ch.New, entry, ch)
			}
		}
	}

	return m, nil
}

func addPerFieldActions(ctx context.Context, adapter *dresources.Adapter, changes deployplan.Changes, remoteState any) error {
	fieldTriggers := adapter.FieldTriggers()

	for pathString, ch := range changes {
		path, err := structpath.Parse(pathString)
		if err != nil {
			return err
		}

		if ch.New == nil && ch.Old == nil && ch.Remote != nil && path.IsDotString() {
			// The field was not set by us, but comes from the remote state.
			// This could either be server-side default or a policy.
			// In any case, this is not a change we should react to.
			// Note, we only consider struct fields here. Adding/removing elements to/from maps and slices should trigger updates.
			ch.Action = deployplan.Skip
			ch.Reason = deployplan.ReasonServerSideDefault
		} else if structdiff.IsEqual(ch.Remote, ch.New) {
			ch.Action = deployplan.Skip
			ch.Reason = deployplan.ReasonRemoteAlreadySet
		} else if action, ok := fieldTriggers[pathString]; ok {
			// TODO: should we check prefixes instead?
			ch.Action = action
			ch.Reason = deployplan.ReasonFieldTriggers
		} else {
			ch.Action = deployplan.Update
		}

		err = adapter.OverrideChangeDesc(ctx, path, ch, remoteState)
		if err != nil {
			return fmt.Errorf("internal error: failed to classify change: %w", err)
		}
	}

	return nil
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

	targetAction := targetEntry.Action
	if targetAction == deployplan.Undefined {
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

	// Get StructVar from cache
	sv, ok := b.StructVarCache.Load(targetResourceKey)
	if !ok {
		return nil, fmt.Errorf("internal error: %s: missing cached StructVar", targetResourceKey)
	}

	_, isUnresolved := sv.Refs[fieldPathS]
	if isUnresolved {
		// The value that is requested is itself a reference; this means it will be resolved after apply
		return nil, errDelayed
	}

	localConfig := sv.Value

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
		if targetAction != deployplan.Skip {
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

	if targetAction == deployplan.Skip {
		remoteState, ok := b.RemoteStateCache.Load(targetResourceKey)
		if ok {
			return structaccess.Get(remoteState, fieldPath)
		}
	}

	return nil, errDelayed
}

// getStructVar returns the cached StructVar for the given resource key.
// The StructVar must have been eagerly loaded during plan creation or InitForApply.
func (b *DeploymentBundle) getStructVar(resourceKey string) (*structvar.StructVar, error) {
	sv, ok := b.StructVarCache.Load(resourceKey)
	if !ok {
		return nil, fmt.Errorf("internal error: StructVar not found in cache for %s", resourceKey)
	}
	return sv, nil
}

// resolveReferences processes all references in entry.NewState.Refs.
// If isLocal is true, uses LookupReferenceLocal (for planning phase).
// If isLocal is false, uses LookupReferenceRemote (for apply phase).
func (b *DeploymentBundle) resolveReferences(ctx context.Context, resourceKey string, entry *deployplan.PlanEntry, errorPrefix string, isLocal bool) bool {
	sv, err := b.getStructVar(resourceKey)
	if err != nil {
		logdiag.LogError(ctx, fmt.Errorf("%s: %w", errorPrefix, err))
		return false
	}

	var resolved bool
	for fieldPathStr, refString := range sv.Refs {
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

			err = sv.ResolveRef(ref, value)
			if err != nil {
				logdiag.LogError(ctx, fmt.Errorf("%s: cannot update %s with value of %q: %w", errorPrefix, fieldPathStr, ref, err))
				return false
			}
			resolved = true
		}
	}

	// Sync resolved values back to StructVarJSON for serialization
	if resolved {
		if err := sv.SyncToJSON(entry.NewState); err != nil {
			logdiag.LogError(ctx, fmt.Errorf("%s: cannot save state: %w", errorPrefix, err))
			return false
		}
	}

	return true
}

func (b *DeploymentBundle) makePlan(ctx context.Context, configRoot *config.Root, db *dstate.Database) (*deployplan.Plan, error) {
	p := deployplan.NewPlan()

	// Copy state metadata to plan for validation during apply
	p.Lineage = db.Lineage
	p.Serial = db.Serial

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
				return nil, err
			}
		}
	}

	slices.Sort(nodes)

	for _, node := range nodes {
		delete(existingKeys, node)

		prefix := "cannot plan " + node
		inputConfig, err := configRoot.GetResourceConfig(node)
		if err != nil {
			return nil, err
		}

		adapter, err := b.getAdapterForKey(node)
		if err != nil {
			return nil, fmt.Errorf("%s: %w", prefix, err)
		}

		baseRefs := map[string]string{}

		if strings.HasSuffix(node, ".permissions") {
			var inputConfigStructVar *structvar.StructVar
			var err error

			if strings.HasPrefix(node, "resources.secret_scopes.") {
				typedConfig, ok := inputConfig.(*[]resources.SecretScopePermission)
				if !ok {
					return nil, fmt.Errorf("%s: expected *[]resources.SecretScopePermission, got %T", prefix, inputConfig)
				}
				inputConfigStructVar, err = dresources.PrepareSecretScopeAclsInputConfig(*typedConfig, node)
			} else {
				inputConfigStructVar, err = dresources.PreparePermissionsInputConfig(inputConfig, node)
			}

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

		newState := &structvar.StructVar{
			Value: newStateConfig,
			Refs:  refs,
		}

		// Store in cache for use during planning phase
		b.StructVarCache.Store(node, newState)

		// Convert to JSON for serialization in plan
		newStateJSON, err := newState.ToJSON()
		if err != nil {
			return nil, fmt.Errorf("%s: cannot serialize state: %w", node, err)
		}

		e := deployplan.PlanEntry{
			DependsOn: dependsOn,
			NewState:  newStateJSON,
		}

		p.Plan[node] = &e
	}

	for n, entry := range existingKeys {
		if p.Plan[n] != nil {
			panic("unexpected node " + n)
		}

		p.Plan[n] = &deployplan.PlanEntry{
			Action:    deployplan.Delete,
			DependsOn: entry.DependsOn,
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
