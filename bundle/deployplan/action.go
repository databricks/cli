package deployplan

import (
	"fmt"
	"strings"
)

type Action struct {
	// Full resource key, e.g. "resources.jobs.foo" or "resources.jobs.foo.permissions"
	ResourceKey string
	ActionType  ActionType
}

func (a Action) String() string {
	return fmt.Sprintf("  %s %s", a.ActionType.StringShort(), a.ResourceKey)
}

func (a Action) IsChildResource() bool {
	// Note, strictly speaking ResourceKey could be resources.jobs["my.job"] but
	// we have an assumption in many other places that it's always looks like "resources.jobs.my_job"
	items := strings.Split(a.ResourceKey, ".")
	return len(items) == 4
}

type ActionType string

// Actions are ordered in increasing severity.
// If case of several options, action with highest severity wins.
// Note, Create/Delete are handled explicitly and never compared.
const (
	Undefined    ActionType = ""
	Skip         ActionType = "skip"
	Resize       ActionType = "resize"
	Update       ActionType = "update"
	UpdateWithID ActionType = "update_id"
	Create       ActionType = "create"
	Recreate     ActionType = "recreate"
	Delete       ActionType = "delete"
)

var actionOrder = map[ActionType]int{
	Undefined:    0,
	Skip:         1,
	Resize:       2,
	Update:       3,
	UpdateWithID: 4,
	Create:       5,
	Recreate:     6,
	Delete:       7,
}

func (a ActionType) KeepsID() bool {
	switch a {
	case Create, UpdateWithID, Recreate, Delete:
		return false
	default:
		return true
	}
}

// StringShort short version of action string, without suffix
func (a ActionType) StringShort() string {
	items := strings.SplitN(string(a), "_", 2)
	return items[0]
}

// GetHigherAction returns the action with higher severity between a and b.
// Actions are ordered by severity in actionOrder map.
func GetHigherAction(a, b ActionType) ActionType {
	if actionOrder[a] > actionOrder[b] {
		return a
	}
	return b
}
