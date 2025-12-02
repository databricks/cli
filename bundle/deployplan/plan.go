package deployplan

import (
	"cmp"
	"fmt"
	"slices"
	"sync"

	"github.com/databricks/cli/libs/structs/structpath"
	"github.com/databricks/cli/libs/structs/structvar"
)

type Plan struct {
	// Current version is zero which is omitted and has no backward compatibility guarantees
	PlanVersion int `json:"plan_version,omitempty"`
	// TODO:
	// - CliVersion  string               `json:"cli_version"`
	// - Copy Serial / Lineage from the state file
	// - Store a path to state file
	Plan map[string]*PlanEntry `json:"plan,omitzero"`

	mutex   sync.Mutex `json:"-"`
	lockmap lockmap    `json:"-"`
}

func NewPlan() *Plan {
	return &Plan{
		Plan:    make(map[string]*PlanEntry),
		lockmap: newLockmap(),
	}
}

type PlanEntry struct {
	ID          string               `json:"id,omitempty"`
	DependsOn   []DependsOnEntry     `json:"depends_on,omitempty"`
	Action      string               `json:"action,omitempty"`
	NewState    *structvar.StructVar `json:"new_state,omitempty"`
	RemoteState any                  `json:"remote_state,omitempty"`
	Changes     *Changes             `json:"changes,omitempty"`
}

type DependsOnEntry struct {
	Node  string `json:"node"`
	Label string `json:"label,omitempty"`
}

type Changes struct {
	Local  map[string]Trigger `json:"local,omitempty"`
	Remote map[string]Trigger `json:"remote,omitempty"`
}

type Trigger struct {
	Action string `json:"action"`
	Reason string `json:"reason,omitempty"`
}

// HasChange checks if there are any changes for fields with the given prefix.
// This function is path-aware and correctly handles path component boundaries.
// For example:
//   - HasChange("a") matches "a" and "a.b" but not "aa"
//   - HasChange("config") matches "config" and "config.name" but not "configuration"
//
// Note: This function does not support wildcard patterns.
func (c *Changes) HasChange(fieldPath string) bool {
	if c == nil {
		return false
	}

	for field := range c.Local {
		if structpath.HasPrefix(field, fieldPath) {
			return true
		}
	}

	for field := range c.Remote {
		if structpath.HasPrefix(field, fieldPath) {
			return true
		}
	}

	return false
}

func (p *Plan) GetActions() []Action {
	actions := make([]Action, 0, len(p.Plan))
	for key, entry := range p.Plan {
		at := ActionTypeFromString(entry.Action)
		actions = append(actions, Action{
			ResourceKey: key,
			ActionType:  at,
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
