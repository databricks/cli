package deployplan

import (
	"cmp"
	"slices"
	"strings"
)

type Plan struct {
	// Current version is zero which is omitted and has no backward compatibility guarantees
	PlanVersion int `json:"plan_version,omitempty"`
	// TODO:
	// - CliVersion  string               `json:"cli_version"`
	// - Copy Serial / Lineage from the state file
	Plan map[string]PlanEntry `json:"plan,omitzero"`
}

type PlanEntry struct {
	ID        string           `json:"id,omitempty"`
	DependsOn []DependsOnEntry `json:"depends_on,omitempty"`
	Action    string           `json:"action"`
	Fields    []Field          `json:"fields,omitempty"`
}

type DependsOnEntry struct {
	Node  string `json:"node"`
	Label string `json:"label,omitempty"`
}

type Field struct {
	Path   string `json:"path"`
	State  any    `json:"state,omitempty"`
	Config any    `json:"config"`
	Remote any    `json:"remote,omitempty"`
	Action string `json:"action"`
}

func (p Plan) GetActions() []Action {
	actions := make([]Action, 0, len(p.Plan))
	for key, entry := range p.Plan {
		if entry.Action == ActionTypeNoop.StringFull() {
			continue
		}
		parts := strings.SplitN(strings.TrimPrefix(key, "resources."), ".", 2)
		if len(parts) != 2 {
			continue
		}
		at := ActionTypeFromString(entry.Action)
		if at == ActionTypeUnset {
			continue
		}
		actions = append(actions, Action{
			ResourceNode: ResourceNode{Group: parts[0], Key: parts[1]},
			ActionType:   at,
		})
	}

	slices.SortFunc(actions, func(x, y Action) int {
		if c := cmp.Compare(x.Group, y.Group); c != 0 {
			return c
		}
		return cmp.Compare(x.Key, y.Key)
	})

	return actions
}
