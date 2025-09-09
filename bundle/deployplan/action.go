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
	return fmt.Sprintf("  %s %s %s", a.ActionType.String(), typ, a.Key)
}

// Implements cmdio.Event for cmdio.Log
func (a Action) IsInplaceSupported() bool {
	return false
}

type ActionType uint8

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
	ActionTypeNoop:         "noop",
	ActionTypeResize:       "resize",
	ActionTypeUpdate:       "update",
	ActionTypeUpdateWithID: "update",
	ActionTypeCreate:       "create",
	ActionTypeRecreate:     "recreate",
	ActionTypeDelete:       "delete",
}

func (a ActionType) IsNoop() bool {
	return a == ActionTypeNoop
}

func (a ActionType) KeepsID() bool {
	switch a {
	case ActionTypeCreate, ActionTypeUpdateWithID, ActionTypeRecreate:
		return false
	default:
		return true
	}
}

func (a ActionType) String() string {
	return ShortName[a]
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
