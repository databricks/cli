package deployplan

import (
	"cmp"
	"fmt"
	"slices"
	"strings"
	"sync"

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

	mutex sync.Mutex      `json:"-"`
	locks map[string]bool `json:"-"`
}

func NewPlan() *Plan {
	return &Plan{
		Plan:  make(map[string]*PlanEntry),
		locks: make(map[string]bool),
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

func (p *Plan) GetActions() []Action {
	actions := make([]Action, 0, len(p.Plan))
	for key, entry := range p.Plan {
		at := ActionTypeFromString(entry.Action)
		parts := strings.SplitN(strings.TrimPrefix(key, "resources."), ".", 2)
		if len(parts) < 2 {
			continue
		}
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

// LockEntry returns *PlanEntry; subsequent calls before UnlockEntry() with the same resourceKey will panic.
func (p *Plan) LockEntry(resourceKey string) *PlanEntry {
	p.mutex.Lock()
	defer p.mutex.Unlock()

	entry, ok := p.Plan[resourceKey]
	if ok {
		if p.locks[resourceKey] {
			panic(fmt.Sprintf("internal DAG error, concurrent access to %q", resourceKey))
		}
		p.locks[resourceKey] = true
		return entry
	}

	return nil
}

func (p *Plan) UnlockEntry(resourceKey string) {
	p.mutex.Lock()
	defer p.mutex.Unlock()
	p.locks[resourceKey] = false
}
