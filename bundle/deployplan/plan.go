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
	return fmt.Sprintf("  %s %s %s", a.ActionType, typ, a.Key)
}

// Implements cmdio.Event for cmdio.Log
func (a Action) IsInplaceSupported() bool {
	return false
}

// These enum values are superset to action types defined in the tfjson library.
// "recreate" maps to the tfjson.Actions.Replace() function.
// "update" and "update_with_id" maps to tfjson.Actions.Update() and so on. source:
// https://github.com/hashicorp/terraform-json/blob/0104004301ca8e7046d089cdc2e2db2179d225be/action.go#L14
type ActionType string

const (
	ActionTypeUnset        ActionType = ""
	ActionTypeNoop         ActionType = "noop"
	ActionTypeCreate       ActionType = "create"
	ActionTypeDelete       ActionType = "delete"
	ActionTypeUpdate       ActionType = "update"
	ActionTypeUpdateWithID ActionType = "update_with_id"
	ActionTypeRecreate     ActionType = "recreate"
)

var ShortName = map[ActionType]ActionType{
	ActionTypeUpdateWithID: ActionTypeUpdate,
}

func (a ActionType) KeepsID() bool {
	switch a {
	case ActionTypeCreate:
		return false
	case ActionTypeUpdateWithID:
		return false
	case ActionTypeRecreate:
		return false
	default:
		return true
	}
}

func (a ActionType) String() string {
	shortAction := ShortName[a]
	if shortAction != "" {
		return string(shortAction)
	}
	return string(a)
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
