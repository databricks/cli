package deployplan

import (
	"cmp"
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"iter"
	"maps"
	"os"
	"slices"
	"strings"
	"sync"

	"github.com/databricks/cli/internal/build"
	"github.com/databricks/cli/libs/structs/structpath"
	"github.com/databricks/cli/libs/structs/structvar"
)

const currentPlanVersion = 2

type Plan struct {
	PlanVersion int    `json:"plan_version,omitempty"`
	CLIVersion  string `json:"cli_version,omitempty"`
	Lineage     string `json:"lineage,omitempty"`
	Serial      int    `json:"serial,omitempty"`

	// Features are the state feature flags this plan was built against, mirroring the state
	// file's own "features" field. The stamps above exist because of a feature being set, so
	// deploy --plan rejects a plan whose features differ from the target's (see process.go).
	Features map[string]struct{} `json:"features,omitempty"`

	Plan map[string]*PlanEntry `json:"plan,omitzero"`

	// NotSelected is the number of resources removed by FilterToSelected via the
	// --select flag. Serialized so the summary survives a deploy from a plan file
	// (--plan); used only for summary reporting.
	NotSelected int `json:"not_selected,omitempty"`

	// WasFiltered is true when FilterToSelected was called on this plan, regardless
	// of how many resources were excluded. Serialized so process.go can distinguish
	// a plan that was pre-filtered at plan time (suppress the "--select ignored"
	// warning) from one that was not (the flag has no effect and should warn).
	WasFiltered bool `json:"was_filtered,omitempty"`

	// Selectors holds the raw --select values used to produce this filtered plan.
	// Serialized alongside WasFiltered so process.go can warn when --select at
	// deploy time differs from the selectors baked into the plan file.
	Selectors []string `json:"selectors,omitempty"`

	mutex   sync.Mutex `json:"-"`
	lockmap lockmap    `json:"-"`
}

// ActionCounts summarizes a plan's actions by category. A recreate counts as
// both a create and a delete, matching how plan and deploy report changes.
type ActionCounts struct {
	Create    int
	Change    int
	Delete    int
	Unchanged int
}

// CountActions tallies the plan's actions by category. Order is irrelevant to a
// tally, so it iterates the plan map directly rather than the sorted GetActions.
func (p *Plan) CountActions() ActionCounts {
	var c ActionCounts
	for _, entry := range p.Plan {
		switch entry.Action {
		case Create:
			c.Create++
		case Update, UpdateWithID, Resize:
			c.Change++
		case Delete:
			// A state-only delete touches nothing in the backend and only drops the
			// state entry, so it is not a real action: leave it out of the tally
			// entirely rather than misreport it as deleted or unchanged. This matches
			// how `bundle destroy` counts its own deletions.
			if entry.IsStateOnlyDelete() {
				continue
			}
			c.Delete++
		case Recreate:
			// A recreate counts as both a delete and a create.
			c.Delete++
			c.Create++
		case Skip, Undefined:
			c.Unchanged++
		}
	}
	return c
}

// NewPlanDirect creates a new Plan for direct engine with plan_version set.
func NewPlanDirect() *Plan {
	return &Plan{
		PlanVersion: currentPlanVersion,
		CLIVersion:  build.GetInfo().Version,
		Plan:        make(map[string]*PlanEntry),
		lockmap:     newLockmap(),
	}
}

// LoadPlanFromFile reads a plan from a JSON file.
func LoadPlanFromFile(path string) (*Plan, error) {
	file, err := os.Open(path)
	if err != nil {
		return nil, fmt.Errorf("reading plan file: %w", err)
	}
	defer file.Close()
	var plan Plan
	decoder := json.NewDecoder(file)
	decoder.DisallowUnknownFields()
	if err := decoder.Decode(&plan); err != nil {
		return nil, fmt.Errorf("parsing plan JSON: %w", err)
	}

	// Validate plan version
	if plan.PlanVersion != currentPlanVersion {
		return nil, fmt.Errorf("plan version mismatch: plan has version %d (generated with CLI %q), but current version is %d", plan.PlanVersion, plan.CLIVersion, currentPlanVersion)
	}

	// Initialize internal fields that are not serialized
	plan.lockmap = newLockmap()
	if plan.Plan == nil {
		plan.Plan = make(map[string]*PlanEntry)
	}
	return &plan, nil
}

type PlanEntry struct {
	ID        string           `json:"id,omitempty"`
	DependsOn []DependsOnEntry `json:"depends_on,omitempty"`
	Action    ActionType       `json:"action,omitempty"`
	// Gone is set on Delete entries when planning confirmed the resource no longer
	// exists remotely. Applying such an entry only removes it from the state, without
	// calling the delete API, and approval prompts do not list it as a deletion.
	Gone bool `json:"gone,omitempty"`
	// StateOnly is set on Delete entries for resources that implement no DoDelete:
	// deleting them has no backend effect. Like Gone, applying such an entry only
	// removes it from the state and it is excluded from destructive-action prompts,
	// textual plan output and the deleted count — but unlike Gone it is a property of
	// the resource type, not of the current remote state, so planning skips the
	// remote read that Gone detection needs.
	StateOnly   bool                     `json:"state_only,omitempty"`
	NewState    *structvar.StructVarJSON `json:"new_state,omitempty"`
	RemoteState any                      `json:"remote_state,omitempty"`
	Changes     Changes                  `json:"changes,omitempty"`
}

// IsStateOnlyDelete reports whether applying this delete only drops the state entry
// without any backend call: the resource is already gone remotely (Gone) or has no
// delete operation (StateOnly). Such deletes are omitted from human output, excluded
// from the resource counts, and need no destructive-action approval.
func (e *PlanEntry) IsStateOnlyDelete() bool {
	return e.Gone || e.StateOnly
}

type DependsOnEntry struct {
	Node  string `json:"node"`
	Label string `json:"label,omitempty"`
}

type Changes map[string]*ChangeDesc

type ChangeDesc struct {
	Action ActionType `json:"action"`
	Reason string     `json:"reason,omitempty"`
	Old    any        `json:"old,omitempty"`
	New    any        `json:"new,omitempty"`
	Remote any        `json:"remote,omitempty"`
}

// Possible values for Reason field
const (
	ReasonBackendDefault   = "backend_default"
	ReasonAlias            = "alias"
	ReasonRemoteAlreadySet = "remote_already_set"
	ReasonEmpty            = "empty"
	ReasonCustom           = "custom"
	// ReasonMissingInRemote: field is not present in RemoteType (write-only / input-only).
	// Remote always appears nil, so treat the absence as a no-op when there is no local change.
	ReasonMissingInRemote = "missing_in_remote"
	// ReasonRemoteAddition: the field is a remote-only addition (absent from config, present
	// in the remote) inside an object whose gate is set (e.g. a cluster with a policy_id). The
	// backend may extend such an object beyond what the bundle declares, so the addition is not
	// treated as drift. We do not attribute the value to any particular source.
	ReasonRemoteAddition = "remote_addition"
	// ReasonImmutableBackendValue: an immutable (recreate_on_changes) field the config never
	// set but the backend populated (old and new nil, remote set). It is left in place rather
	// than recreated - the user does not manage it and the value cannot be changed anyway.
	ReasonImmutableBackendValue = "immutable_backend_value"
	// ReasonUnmanaged: the config does not declare the field at all, so the bundle does not manage
	// it and whatever the remote reports is not drift (e.g. cluster libraries without a libraries
	// section, where job runs install their task libraries cluster-wide).
	ReasonUnmanaged = "unmanaged"

	// Special reason that results in removing this change from the plan
	ReasonDrop = "!drop"
)

// HasChange checks if there are any actionable changes for fields with the given prefix.
// Suppressed changes (Action == Skip) are ignored, matching HasChangeExcept.
// This function is path-aware and correctly handles path component boundaries.
// For example:
//   - HasChange for path "a" matches "a" and "a.b" but not "aa"
//   - HasChange for path "config" matches "config" and "config.name" but not "configuration"
func (c *Changes) HasChange(fieldPath *structpath.PathNode) bool {
	if c == nil {
		return false
	}

	for field, change := range *c {
		if change.Action == Skip {
			continue
		}
		fieldNode, err := structpath.ParsePath(field)
		if err != nil {
			continue
		}
		if fieldNode.HasPrefix(fieldPath) {
			return true
		}
	}

	return false
}

// HasChangeExcept checks if there are any changes for fields with the given prefixes.
func (c *Changes) HasChangeExcept(prefixes ...string) bool {
	if c == nil {
		return false
	}
	for field := range *c {
		if !slices.Contains(prefixes, field) {
			if (*c)[field].Action != Skip {
				return true
			}
		}
	}
	return false
}

func (p *Plan) GetActions() []Action {
	actions := make([]Action, 0, len(p.Plan))
	for key, entry := range p.Plan {
		actions = append(actions, Action{
			ResourceKey: key,
			ActionType:  entry.Action,
			Gone:        entry.Gone,
			StateOnly:   entry.StateOnly,
		})
	}

	slices.SortFunc(actions, func(x, y Action) int {
		return cmp.Compare(x.ResourceKey, y.ResourceKey)
	})

	return actions
}

func (p *Plan) WriteLockEntry(resourceKey string) (*PlanEntry, error) {
	p.mutex.Lock()
	defer p.mutex.Unlock()

	if p.lockmap.TryLock(resourceKey) {
		return p.Plan[resourceKey], nil
	}

	return nil, fmt.Errorf("write lock: concurrent access to %q", resourceKey)
}

func (p *Plan) ReadLockEntry(resourceKey string) (*PlanEntry, error) {
	p.mutex.Lock()
	defer p.mutex.Unlock()

	if p.lockmap.TryRLock(resourceKey) {
		return p.Plan[resourceKey], nil
	}
	return nil, fmt.Errorf("read lock: concurrent access to %q", resourceKey)
}

func (p *Plan) WriteUnlockEntry(resourceKey string) {
	p.mutex.Lock()
	defer p.mutex.Unlock()
	p.lockmap.Unlock(resourceKey)
}

func (p *Plan) ReadUnlockEntry(resourceKey string) {
	p.mutex.Lock()
	defer p.mutex.Unlock()
	p.lockmap.RUnlock(resourceKey)
}

// ResolveSelectors maps each --select selector to its plan key ("resources.type.name")
// against resourceKeys of that same form. A selector is a qualified "type.name" or an
// unqualified "name"; it errors on no match or, when unqualified, more than one.
// Duplicate resourceKeys (a resource present in both config and state) collapse, so an
// unqualified selector is never falsely ambiguous against itself.
func ResolveSelectors(resourceKeys iter.Seq[string], selectors []string) ([]string, error) {
	// Index top-level resource keys by name and as qualified "type.name". Keys look like
	// "resources.<type>.<name>" (names have no dots, see Action.IsChildResource), so
	// sub-nodes (4 parts: grants/permissions) are skipped as not --select-addressable.
	byName := map[string][]string{}
	known := map[string]struct{}{}
	for key := range resourceKeys {
		parts := strings.Split(key, ".")
		if len(parts) != 3 || parts[0] != "resources" {
			continue
		}
		typeName, name := parts[1], parts[2]
		// Internal resource types (e.g. internal_immutable_snapshots) are not user-
		// addressable; skip them so they neither resolve a selector nor make an
		// unqualified one ambiguous (nor leak an internal type into the error).
		if strings.HasPrefix(typeName, "internal_") {
			continue
		}
		qualified := typeName + "." + name
		if _, alreadySeen := known[qualified]; !alreadySeen {
			known[qualified] = struct{}{}
			byName[name] = append(byName[name], qualified)
		}
	}

	resolved := make([]string, 0, len(selectors))
	for _, s := range selectors {
		var qualified string
		if strings.Contains(s, ".") {
			if _, ok := known[s]; !ok {
				return nil, fmt.Errorf("no such resource: %s", s)
			}
			qualified = s
		} else {
			switch matches := byName[s]; len(matches) {
			case 0:
				return nil, fmt.Errorf("no such resource: %s", s)
			case 1:
				qualified = matches[0]
			default:
				slices.Sort(matches)
				return nil, fmt.Errorf("ambiguous resource: %s (can resolve to %s); use a qualified name to disambiguate", s, strings.Join(matches, ", "))
			}
		}
		resolved = append(resolved, "resources."+qualified)
	}
	return resolved, nil
}

// FilterToSelected reduces the plan to the resources named in selected (resolved via
// ResolveSelectors) plus the others that must change with them. The closure follows
// edges by action, like the apply graph (makeGraph): a deployed resource pulls in its
// dependencies (forward), a deleted one its dependents (reverse) so they are removed
// together. A delete selection errors if a live resource (not itself being deleted)
// still references the resource to be deleted; removing it would break the live resource.
func (p *Plan) FilterToSelected(_ context.Context, selected []string) error {
	before := len(p.Plan)

	resolved, err := ResolveSelectors(maps.Keys(p.Plan), selected)
	if err != nil {
		return err
	}

	// Only build reverse edges when at least one selected resource is being deleted,
	// since non-delete selections only follow forward DependsOn edges.
	var dependents map[string][]string
	for _, key := range resolved {
		if e, ok := p.Plan[key]; ok && e.Action == Delete {
			dependents = p.reverseEdges()
			break
		}
	}

	queue := make([]string, 0, len(selected))
	reachable := make(map[string]struct{}, len(selected))
	for _, key := range resolved {
		p.enqueueReachable(reachable, &queue, key)
		// Grants and permissions are modeled as separate plan nodes for internal
		// reasons, but the user cannot address them via --select. Pull them in as
		// part of the parent resource so selecting a resource applies its grants
		// and permissions too. The dependency edge runs sub-node → parent, so the
		// BFS below would never reach them from the parent otherwise.
		p.enqueueReachable(reachable, &queue, key+".grants")
		p.enqueueReachable(reachable, &queue, key+".permissions")
	}

	// BFS: deleted nodes pull in their dependents (also being deleted); deployed
	// nodes pull in their dependencies. Collect errors for live dependents of a
	// delete selection before returning, so the user sees all conflicts at once.
	var errs []string
	for len(queue) > 0 {
		key := queue[0]
		queue = queue[1:]
		if p.Plan[key].Action == Delete {
			for _, dependent := range dependents[key] {
				// Sub-nodes (grants/permissions, 4-part key) are not user-facing dependents
				// and never trigger the live-dependent error. If they are themselves being
				// deleted, they must be pulled in alongside their parent.
				if strings.Count(dependent, ".") >= 3 {
					if p.Plan[dependent].Action == Delete {
						p.enqueueReachable(reachable, &queue, dependent)
					}
					continue
				}
				if p.Plan[dependent].Action == Delete {
					// Dependent is also being deleted: pull it in.
					p.enqueueReachable(reachable, &queue, dependent)
				} else {
					// Dependent is live (still in config): deleting key would leave
					// it broken. Report a user-facing error with both names.
					deleted := strings.TrimPrefix(key, "resources.")
					live := strings.TrimPrefix(dependent, "resources.")
					errs = append(errs, fmt.Sprintf("cannot delete %s: %s still references it; remove the reference or also remove %s from the config", deleted, live, live))
				}
			}
		} else {
			for _, dep := range p.Plan[key].DependsOn {
				p.enqueueReachable(reachable, &queue, dep.Node)
			}
		}
	}
	if len(errs) > 0 {
		slices.Sort(errs)
		return errors.New(strings.Join(errs, "\n"))
	}

	for key := range p.Plan {
		if _, ok := reachable[key]; !ok {
			delete(p.Plan, key)
		}
	}

	p.NotSelected = before - len(p.Plan)
	p.WasFiltered = true
	p.Selectors = slices.Clone(selected)
	return nil
}

// reverseEdges returns the reverse of the plan's DependsOn edges: for each node, the
// nodes that depend on it.
func (p *Plan) reverseEdges() map[string][]string {
	dependents := map[string][]string{}
	for key, entry := range p.Plan {
		for _, dep := range entry.DependsOn {
			dependents[dep.Node] = append(dependents[dep.Node], key)
		}
	}
	return dependents
}

// enqueueReachable marks key as reachable and appends it to queue, if key exists
// in the plan and has not been seen before. Missing or already-seen keys are ignored.
func (p *Plan) enqueueReachable(reachable map[string]struct{}, queue *[]string, key string) {
	if _, seen := reachable[key]; seen {
		return
	}
	if _, ok := p.Plan[key]; ok {
		reachable[key] = struct{}{}
		*queue = append(*queue, key)
	}
}

type lockmap struct {
	state map[string]int
}

func newLockmap() lockmap {
	return lockmap{
		state: make(map[string]int),
	}
}

func (p *lockmap) TryLock(resourceKey string) bool {
	if p.state[resourceKey] == 0 {
		p.state[resourceKey] = -1
		return true
	}
	return false
}

func (p *lockmap) Unlock(resourceKey string) {
	if p.state[resourceKey] == -1 {
		p.state[resourceKey] = 0
	}
}

func (p *lockmap) TryRLock(resourceKey string) bool {
	if p.state[resourceKey] >= 0 {
		p.state[resourceKey] += 1
		return true
	}
	return false
}

func (p *lockmap) RUnlock(resourceKey string) {
	if p.state[resourceKey] > 0 {
		p.state[resourceKey] -= 1
	}
}
