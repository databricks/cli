package terraform

import "strings"

type Plan struct {
	// Path to the plan
	Path string

	// If true, the plan is empty and applying it will not do anything
	IsEmpty bool
}

type Action struct {
	// Type and name of the resource
	ResourceType string `json:"resource_type"`
	ResourceName string `json:"resource_name"`

	Action ActionType `json:"action"`
}

func (a Action) String() string {
	// terraform resources have the databricks_ prefix, which is not needed.
	rtype := strings.TrimPrefix(a.ResourceType, "databricks_")
	return strings.Join([]string{" ", string(a.Action), rtype, a.ResourceName}, " ")
}

func (c Action) IsInplaceSupported() bool {
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
