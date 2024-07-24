package plan

import "strings"

type Plan interface {
	// Path where the plan is persisted
	// Path() string

	// Get all actions of the given types
	ActionsByTypes(...ActionType) []Action

	// Whether the plan is empty, i.e. no actions
	IsEmpty() bool

	// Apply the plan, changing the underlying resources.
	Apply() error
}

type Action struct {
	rtype string
	rname string

	atype ActionType
}

func (a Action) String() string {
	return strings.Join([]string{" ", string(a.atype), a.rtype, a.rname}, " ")
}

func (c Action) IsInplaceSupported() bool {
	return false
}

type ActionType string

const (
	ActionTypeCreate   ActionType = "create"
	ActionTypeDelete   ActionType = "delete"
	ActionTypeUpdate   ActionType = "update"
	ActionTypeNoOp     ActionType = "no-op"
	ActionTypeRead     ActionType = "read"
	ActionTypeRecreate ActionType = "recreate"
)
