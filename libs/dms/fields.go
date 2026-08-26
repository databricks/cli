package dms

import "strings"

// Fields is the set of operation fields an update writes, sent as its update mask. The
// service rejects any other path, so this is the whole vocabulary.
type Fields uint8

const (
	FieldState Fields = 1 << iota
	FieldErrorMessage
	FieldResourceID
	FieldStatus
)

// DescribesResource is what a write that says how the resource looks claims: every field
// an update may change.
const DescribesResource = FieldState | FieldErrorMessage | FieldResourceID | FieldStatus

// ClearsState is what a write that leaves no resource claims: a delete, and the delete half of
// a recreate. State is named so the service reads the absent value as a clear, which is what
// drops the resource from the deployment. It cannot name resource_id either: the service counts
// only a state it was given as recording one, so an id alongside a cleared state is refused.
const ClearsState = FieldState | FieldErrorMessage | FieldStatus

// KeepsState is what a failure claims: mark it failed and leave state alone. State means
// the resource is as it was written; no state means a delete went through and nothing
// replaced it, so the resource really is gone and the deployment should say so.
//
// It cannot name resource_id: the service requires state in any mask that names the id, and
// naming state here would overwrite what the resource last recorded.
const KeepsState = FieldErrorMessage | FieldStatus

// wireNames pairs each field with its name on the wire, in the order a mask lists them.
var wireNames = []struct {
	field Fields
	name  string
}{
	{FieldState, "state"},
	{FieldErrorMessage, "error_message"},
	{FieldResourceID, "resource_id"},
	{FieldStatus, "status"},
}

// Has reports whether f contains every field in other.
func (f Fields) Has(other Fields) bool {
	return f&other == other
}

// Mask renders f as the update_mask the service expects, always in the same order.
func (f Fields) Mask() string {
	names := make([]string, 0, len(wireNames))
	for _, w := range wireNames {
		if f.Has(w.field) {
			names = append(names, w.name)
		}
	}
	return strings.Join(names, ",")
}
