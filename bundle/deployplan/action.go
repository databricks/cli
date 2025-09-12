package deployplan

import (
	"fmt"
	"strings"
)

type Action struct {
	// Full resource key in format "resources.<group>.<key>"
	Key string

	ActionType ActionType
}

func (a Action) String() string {
	// Backward compatible format: "resources.jobs.foo" -> "job foo"
	key := strings.TrimPrefix(a.Key, "resources.")
	key = strings.ReplaceAll(key, "s.", " ")
	key = strings.ReplaceAll(key, ".", " ")
	return fmt.Sprintf("  %s %s", a.ActionType.StringShort(), key)
}

// Implements cmdio.Event for cmdio.Log
func (a Action) IsInplaceSupported() bool {
	return false
}

type ActionType int

// Actions are ordered in increasing severity.
// If case of several options, action with highest severity wins.
// Note, Create/Delete are handled explicitly and never compared.
const (
	ActionTypeUnset ActionType = iota
	ActionTypeNoop
	ActionTypeResize
	ActionTypeUpdate
	ActionTypeUpdateWithID
	ActionTypeCreate
	ActionTypeRecreate
	ActionTypeDelete
)

var actionName = map[ActionType]string{
	ActionTypeNoop:         "noop",
	ActionTypeResize:       "resize",
	ActionTypeUpdate:       "update(id_stable)",
	ActionTypeUpdateWithID: "update(id_changes)",
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

func (a ActionType) IsNoop() bool {
	return a == ActionTypeNoop
}

func (a ActionType) KeepsID() bool {
	switch a {
	case ActionTypeCreate, ActionTypeUpdateWithID, ActionTypeRecreate, ActionTypeDelete:
		return false
	default:
		return true
	}
}

// StringShort short version of action string, without part in parens.
func (a ActionType) StringShort() string {
	items := strings.SplitN(actionName[a], "(", 2)
	return items[0]
}

// StringFull returns the string representation of the action type.
func (a ActionType) StringFull() string {
	return actionName[a]
}

func ActionTypeFromString(s string) ActionType {
	actionType, ok := nameToAction[s]
	if !ok {
		return ActionTypeUnset
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

// FilterGroup returns actions that match the specified group and any of the specified action types
func FilterGroup(changes []Action, group string, actionTypes ...ActionType) []Action {
	var result []Action

	// Create a set of action types for efficient lookup
	actionTypeSet := make(map[ActionType]bool)
	for _, actionType := range actionTypes {
		actionTypeSet[actionType] = true
	}

	for _, action := range changes {
		actionGroup, _ := ParseResourceKey(action.Key)
		if actionGroup == group && actionTypeSet[action.ActionType] {
			result = append(result, action)
		}
	}
	return result
}
