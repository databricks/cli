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

// KeepsState is what a failure claims: mark it failed, name the resource it failed on, and
// leave state alone. State means the resource is as it was written; no state means a delete
// went through and nothing replaced it, so the resource really is gone and the deployment
// should say so. The id is not optional: the service refuses an update to a delete operation
// that does not name the resource.
const KeepsState = FieldErrorMessage | FieldResourceID | FieldStatus

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
