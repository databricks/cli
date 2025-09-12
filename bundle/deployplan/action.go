package deployplan

import (
	"fmt"
	"strings"
)

type Action struct {
	ResourceNode

	ActionType ActionType
}

func (a Action) String() string {
	typ, _ := strings.CutSuffix(a.Group, "s")
	return fmt.Sprintf("  %s %s %s", a.ActionType.StringShort(), typ, a.Key)
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

var ShortName = map[ActionType]string{
	ActionTypeUpdateWithID: "update",
}

// FullName provides stable identifiers for actions including update_with_id.
var FullName = map[ActionType]string{
	ActionTypeNoop:         "noop",
	ActionTypeResize:       "resize",
	ActionTypeUpdate:       "update",
	ActionTypeUpdateWithID: "update_with_id",
	ActionTypeCreate:       "create",
	ActionTypeRecreate:     "recreate",
	ActionTypeDelete:       "delete",
}

var FullNameReverse = map[string]ActionType{}

func init() {
	for k, v := range FullName {
		FullNameReverse[v] = k
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

// StringShort short version of action. Currently this only replaces "update_with_id" with "update".
// Note, we intentionally do not implement String() method to force explicit choice.
func (a ActionType) StringShort() string {
	result := ShortName[a]
	if result != "" {
		return result
	}
	return a.StringFull()
}

// StringFull returns the string representation of the action type.
func (a ActionType) StringFull() string {
	return FullName[a]
}

// ActionTypeFromString decodes a string to ActionType using FullName.
func ActionTypeFromString(s string) ActionType {
	actionType, ok := FullNameReverse[s]
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
		if action.Group == group && actionTypeSet[action.ActionType] {
			result = append(result, action)
		}
	}
	return result
}
