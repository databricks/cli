package deployplan

import (
	"fmt"
	"strings"
)

type Plan struct {
	// TerraformPlanPath is the path to the plan from the terraform CLI
	TerraformPlanPath string

	// If true, the plan is empty and applying it will not do anything
	TerraformIsEmpty bool

	// List of actions to apply (direct deployment)
	Actions []Action
}

type Action struct {
	// Resource group in the config, e.g. "jobs", "pipelines" etc
	Group string

	// Key of the resource the config
	Name string

	ActionType ActionType
}

func (a Action) String() string {
	typ, _ := strings.CutSuffix(a.Group, "s")
	return fmt.Sprintf("  %s %s %s", a.ActionType, typ, a.Name)
}

// Implements cmdio.Event for cmdio.Log
func (a Action) IsInplaceSupported() bool {
	return false
}

// These enum values correspond to action types defined in the tfjson library.
// "recreate" maps to the tfjson.Actions.Replace() function.
// "update" maps to tfjson.Actions.Update() and so on. source:
// https://github.com/hashicorp/terraform-json/blob/0104004301ca8e7046d089cdc2e2db2179d225be/action.go#L14
type ActionType string

const (
	ActionTypeUnset    ActionType = ""
	ActionTypeNoop     ActionType = "noop"
	ActionTypeCreate   ActionType = "create"
	ActionTypeDelete   ActionType = "delete"
	ActionTypeUpdate   ActionType = "update"
	ActionTypeRecreate ActionType = "recreate"
)

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
