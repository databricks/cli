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
	ActionTypeUndefined    ActionType = ""
	ActionTypeSkip         ActionType = "skip"
	ActionTypeResize       ActionType = "resize"
	ActionTypeUpdate       ActionType = "update"
	ActionTypeUpdateWithID ActionType = "update_id"
	ActionTypeCreate       ActionType = "create"
	ActionTypeRecreate     ActionType = "recreate"
	ActionTypeDelete       ActionType = "delete"
)

var actionOrder = map[ActionType]int{
	ActionTypeUndefined:    0,
	ActionTypeSkip:         1,
	ActionTypeResize:       2,
	ActionTypeUpdate:       3,
	ActionTypeUpdateWithID: 4,
	ActionTypeCreate:       5,
	ActionTypeRecreate:     6,
	ActionTypeDelete:       7,
}

func (a ActionType) KeepsID() bool {
	switch a {
	case ActionTypeCreate, ActionTypeUpdateWithID, ActionTypeRecreate, ActionTypeDelete:
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
