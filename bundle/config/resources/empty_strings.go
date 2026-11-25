package resources

// EmptyStringSender is implemented by resource types with string fields that must be
// sent to the API even when they are empty. The DropEmptyStrings mutator otherwise drops
// empty strings from omitempty fields.
type EmptyStringSender interface {
	// FieldsSentWhenEmpty returns the keys of top-level string fields that are set to ""
	// when they are empty or absent, so that "" is sent to the API.
	FieldsSentWhenEmpty() []string
}
