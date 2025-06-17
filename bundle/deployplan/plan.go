package deployplan

import (
	"fmt"
	"strings"
)

type Plan struct {
	// Path to the plan
	Path string

	// If true, the plan is empty and applying it will not do anything
	IsEmpty bool
}

type Action struct {
	// Resource group in the config, e.g. "jobs", "pipelines" etc
	Group string

	// Key of the resource the config
	Name string

	Action ActionType
}

func (a Action) String() string {
	typ, _ := strings.CutSuffix(a.Group, "s")
	return fmt.Sprintf("  %s %s %s", a.Action, typ, a.Name)
}

// IsInplaceSupported returns false since deployplan actions are simple log messages and
// should always be appended rather than rendered in-place.
func (a Action) IsInplaceSupported() bool {
	return false
}

// These enum values correspond to action types defined in the tfjson library.
// "recreate" maps to the tfjson.Actions.Replace() function.
// "update" maps to tfjson.Actions.Update() and so on. source:
// https://github.com/hashicorp/terraform-json/blob/0104004301ca8e7046d089cdc2e2db2179d225be/action.go#L14
type ActionType string

const (
	ActionTypeCreate   ActionType = "create"
	ActionTypeDelete   ActionType = "delete"
	ActionTypeUpdate   ActionType = "update"
	ActionTypeNoOp     ActionType = "no-op"
	ActionTypeRead     ActionType = "read"
	ActionTypeRecreate ActionType = "recreate"
)

// Filter returns actions that match the specified action type
func Filter(changes []Action, actionType ActionType) []Action {
	var result []Action
	for _, action := range changes {
		if action.Action == actionType {
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
		if action.Group == group && actionTypeSet[action.Action] {
			result = append(result, action)
		}
	}
	return result
}
