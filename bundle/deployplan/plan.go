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

	mutex   sync.Mutex `json:"-"`
	lockmap lockmap    `json:"-"`
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
	ID          string                   `json:"id,omitempty"`
	DependsOn   []DependsOnEntry         `json:"depends_on,omitempty"`
	Action      ActionType               `json:"action,omitempty"`
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

	// Special reason that results in removing this change from the plan
	ReasonDrop = "!drop"
)

// HasChange checks if there are any actionable changes for fields with the given prefix.
// Skipped changes (e.g. backend defaults or ignored remote changes) do not count, so a
// section composed only of suppressed changes is treated as unchanged. Callers use this to
// decide whether to issue a section update, and updating for a skip-only change is a no-op.
// This function is path-aware and correctly handles path component boundaries.
// For example:
//   - HasChange for path "a" matches "a" and "a.b" but not "aa"
//   - HasChange for path "config" matches "config" and "config.name" but not "configuration"
func (c *Changes) HasChange(fieldPath *structpath.PathNode) bool {
	if c == nil {
		return false
	}

	for field := range *c {
		if (*c)[field].Action == Skip {
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

func (p *Plan) RemoveEntry(resourceKey string) {
	p.mutex.Lock()
	defer p.mutex.Unlock()
	delete(p.Plan, resourceKey)
}

// FilterToSelected reduces the plan to the nodes in selected (format "type.name",
// e.g. "jobs.my_job") plus their transitive dependencies as recorded in each
// entry's DependsOn field. Nodes not reachable from the selected set are removed.
func (p *Plan) FilterToSelected(selected []string) {
	// Convert "type.name" → "resources.type.name" (plan key format).
	queue := make([]string, 0, len(selected))
	reachable := make(map[string]struct{}, len(selected))
	for _, s := range selected {
		key := "resources." + s
		if _, ok := p.Plan[key]; ok {
			reachable[key] = struct{}{}
			queue = append(queue, key)
		}
	}

	// BFS following DependsOn edges to include transitive dependencies.
	for len(queue) > 0 {
		key := queue[0]
		queue = queue[1:]
		for _, dep := range p.Plan[key].DependsOn {
			if _, seen := reachable[dep.Node]; !seen {
				if _, ok := p.Plan[dep.Node]; ok {
					reachable[dep.Node] = struct{}{}
					queue = append(queue, dep.Node)
				}
			}
		}
	}

	for key := range p.Plan {
		if _, ok := reachable[key]; !ok {
			delete(p.Plan, key)
		}
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
