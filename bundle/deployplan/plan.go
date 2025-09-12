package deployplan

import (
	"cmp"
	"encoding/json"
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
	// - Store a path to state file (relative or absolute?) I guess absolute makes sense.
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
	ID            string               `json:"id,omitempty"`
	DependsOn     []DependsOnEntry     `json:"depends_on,omitempty"`
	Action        string               `json:"action,omitempty"`
	NewState      *structvar.StructVar `json:"new_state,omitempty"`
	RemoteState   any                  `json:"remote_state,omitempty"`
	LocalChanges  map[string]Trigger   `json:"local_changes,omitempty"`
	RemoteChanges map[string]Trigger   `json:"remote_changes,omitempty"`
}

type DependsOnEntry struct {
	Node  string `json:"node"`
	Label string `json:"label,omitempty"`
}

type Trigger struct {
	Action string `json:"action"`
	Reason string `json:"reason,omitempty"`
}

func (p *Plan) GetActions() []Action {
	p.mutex.Lock()
	defer p.mutex.Unlock()

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

func (p *Plan) ReadAction(resourceKey string) string {
	p.mutex.Lock()
	defer p.mutex.Unlock()

	entry, ok := p.Plan[resourceKey]
	if !ok {
		return ""
	}

	return entry.Action
}

// ReadResolvedEntry returns the action and resolved config for the given resource key.
// It returns the action string, the resolved config, and any error encountered.
func (p *Plan) ReadResolvedConfig(resourceKey string) (any, error) {
	p.mutex.Lock()
	defer p.mutex.Unlock()

	entry, ok := p.Plan[resourceKey]
	if !ok {
		return nil, fmt.Errorf("resource %q not found in plan", resourceKey)
	}

	if p.locks[resourceKey] {
		panic(fmt.Sprintf("internal DAG error, concurrent access to %q", resourceKey))
	}

	if entry.NewState == nil {
		return nil, fmt.Errorf("invalid plan, resource %q is missing new_state", resourceKey)
	}

	// At this point it's an error to have unresolved deps
	if len(entry.NewState.Refs) > 0 {
		return nil, fmt.Errorf("unresolved deps for %q: %s", resourceKey, jsonDump(entry.NewState.Refs))
	}

	return entry.NewState.Config, nil
}

func jsonDump(obj map[string]string) string {
	bytes, err := json.MarshalIndent(obj, "", "  ")
	if err != nil {
		return err.Error()
	}
	return string(bytes)
}

/*

// ResolveRef resolves a reference in the specified resource's NewState.
// It finds the target entry and calls ResolveRef on its NewState to resolve the reference with the given value.
func (p *Plan) ResolveRef(resourceKey, reference string, value any) error {
	p.mutex.Lock()
	defer p.mutex.Unlock()

	entry, ok := p.Plan[resourceKey]
	if !ok {
		return fmt.Errorf("resource %q not found in plan", resourceKey)
	}

	if p.locks[resourceKey] {
		panic(fmt.Sprintf("internal DAG error, concurrent access to %q", resourceKey))
	}

	// XXX we actually set NewState to nil for no-op resources. We should have regular state be available here (passed as parameter)
	// we also should not call this for noop actions at all
	if entry.NewState == nil {
		return fmt.Errorf("resource %q has no NewState to resolve references in", resourceKey)
	}

	return entry.NewState.ResolveRef(reference, value)
}
*/
