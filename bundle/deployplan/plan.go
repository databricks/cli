package deployplan

import (
	"cmp"
	"encoding/json"
	"fmt"
	"os"
	"slices"
	"sync"

	"github.com/databricks/cli/internal/build"
	"github.com/databricks/cli/libs/structs/structpath"
	"github.com/databricks/cli/libs/structs/structvar"
)

const currentPlanVersion = 2

type Plan struct {
	PlanVersion int                   `json:"plan_version,omitempty"`
	CLIVersion  string                `json:"cli_version,omitempty"`
	Lineage     string                `json:"lineage,omitempty"`
	Serial      int                   `json:"serial,omitempty"`
	Plan        map[string]*PlanEntry `json:"plan,omitzero"`

	// NotSelected is the number of resources removed by FilterToSelected via the
	// --select flag. Serialized so the summary survives a deploy from a plan file
	// (--plan); used only for summary reporting.
	NotSelected int `json:"not_selected,omitempty"`

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

// NewPlanTerraform creates a new Plan for terraform engine without plan_version.
func NewPlanTerraform() *Plan {
	return &Plan{
		CLIVersion: build.GetInfo().Version,
		Plan:       make(map[string]*PlanEntry),
		lockmap:    newLockmap(),
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
	Gone        bool                     `json:"gone,omitempty"`
	NewState    *structvar.StructVarJSON `json:"new_state,omitempty"`
	RemoteState any                      `json:"remote_state,omitempty"`
	Changes     Changes                  `json:"changes,omitempty"`
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
	// ReasonPolicyManaged: the field is inside a cluster spec that has a policy_id, and the
	// config never declared it. A cluster policy supplies values server-side, so the backend
	// legitimately extends the spec beyond what the bundle asks for; that is not drift.
	ReasonPolicyManaged = "policy_managed"

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

// FilterToSelected reduces the plan to the nodes in selected (format "type.name",
// e.g. "jobs.my_job") plus their transitive dependencies as recorded in each
// entry's DependsOn field. Nodes not reachable from the selected set are removed.
func (p *Plan) FilterToSelected(selected []string) {
	before := len(p.Plan)

	// Convert "type.name" → "resources.type.name" (plan key format).
	queue := make([]string, 0, len(selected))
	reachable := make(map[string]struct{}, len(selected))
	for _, s := range selected {
		key := "resources." + s
		p.enqueueReachable(reachable, &queue, key)
		// Grants and permissions are modeled as separate plan nodes for internal
		// reasons, but the user cannot address them via --select. Pull them in as
		// part of the parent resource so selecting a resource applies its grants
		// and permissions too. The dependency edge runs sub-node → parent, so the
		// BFS below would never reach them from the parent otherwise.
		p.enqueueReachable(reachable, &queue, key+".grants")
		p.enqueueReachable(reachable, &queue, key+".permissions")
	}

	// BFS following DependsOn edges to include transitive dependencies.
	for len(queue) > 0 {
		key := queue[0]
		queue = queue[1:]
		for _, dep := range p.Plan[key].DependsOn {
			p.enqueueReachable(reachable, &queue, dep.Node)
		}
	}

	for key := range p.Plan {
		if _, ok := reachable[key]; !ok {
			delete(p.Plan, key)
		}
	}

	p.NotSelected = before - len(p.Plan)
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
