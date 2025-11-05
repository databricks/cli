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
	// Backward compatible format: "resources.jobs.foo" -> "job foo"
	key := strings.TrimPrefix(a.ResourceKey, "resources.")
	key = strings.ReplaceAll(key, "s.", " ")
	key = strings.ReplaceAll(key, ".", " ")
	return fmt.Sprintf("  %s %s", a.ActionType.StringShort(), key)
}

type ActionType int

// Actions are ordered in increasing severity.
// If case of several options, action with highest severity wins.
// Note, Create/Delete are handled explicitly and never compared.
const (
	ActionTypeUndefined ActionType = iota
	ActionTypeSkip
	ActionTypeResize
	ActionTypeUpdate
	ActionTypeUpdateWithID
	ActionTypeCreate
	ActionTypeRecreate
	ActionTypeDelete
)

const ActionTypeSkipString = "skip"

var actionName = map[ActionType]string{
	ActionTypeSkip:         ActionTypeSkipString,
	ActionTypeResize:       "resize",
	ActionTypeUpdate:       "update",
	ActionTypeUpdateWithID: "update_id",
	ActionTypeCreate:       "create",
	ActionTypeRecreate:     "recreate",
	ActionTypeDelete:       "delete",
}

var nameToAction = map[string]ActionType{}

func init() {
	for k, v := range actionName {
		if _, ok := nameToAction[v]; ok {
			panic("duplicate action string: " + v)
		}
		nameToAction[v] = k
	}
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
	items := strings.SplitN(actionName[a], "_", 2)
	return items[0]
}

// String returns the string representation of the action type.
func (a ActionType) String() string {
	return actionName[a]
}

func ActionTypeFromString(s string) ActionType {
	actionType, ok := nameToAction[s]
	if !ok {
		return ActionTypeUndefined
	}
	return actionType
}

// Filter returns actions that match the specified action type
func Filter(changes []Action, actionType ActionType) []Action {
	var result []Action
	for _, action := range changes {
		if action.ActionType == actionType {
			result = append(result, action)
		}
	}
	return result
}
