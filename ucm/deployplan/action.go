// Package deployplan is ucm's fork of bundle/deployplan.
//
// Forked rather than imported per cmd/ucm/CLAUDE.md: the bundle package is
// upstream-owned and will evolve, so pinning to its internals would break on
// every upstream sync. The on-disk / JSON shape is kept compatible with DAB
// so that consumers like `ucm plan -o json` produce the same structure as
// `bundle plan -o json` for the fields ucm exercises.
package deployplan

import (
	"fmt"
	"strings"
)

type Action struct {
	ResourceKey string
	ActionType  ActionType
}

// String returns the DAB-style "  <action> <key>" form used by plan output.
func (a Action) String() string {
	return fmt.Sprintf("  %s %s", a.ActionType.StringShort(), a.ResourceKey)
}

type ActionType string

const (
	Undefined    ActionType = ""
	Skip         ActionType = "skip"
	Resize       ActionType = "resize"
	Update       ActionType = "update"
	UpdateWithID ActionType = "update_id"
	Create       ActionType = "create"
	Recreate     ActionType = "recreate"
	Delete       ActionType = "delete"
)

var actionOrder = map[ActionType]int{
	Undefined:    0,
	Skip:         1,
	Resize:       2,
	Update:       3,
	UpdateWithID: 4,
	Create:       5,
	Recreate:     6,
	Delete:       7,
}

// KeepsID reports whether an action preserves the resource's existing id.
func (a ActionType) KeepsID() bool {
	switch a {
	case Create, UpdateWithID, Recreate, Delete:
		return false
	default:
		return true
	}
}

// StringShort returns the verb without the "_..." suffix (e.g. "update_id" → "update").
func (a ActionType) StringShort() string {
	items := strings.SplitN(string(a), "_", 2)
	return items[0]
}

// GetHigherAction returns the action with higher severity between a and b.
func GetHigherAction(a, b ActionType) ActionType {
	if actionOrder[a] > actionOrder[b] {
		return a
	}
	return b
}
