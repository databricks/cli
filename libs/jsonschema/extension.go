package jsonschema

// Extension defines our custom JSON schema extensions.
//
// JSON schema supports custom extensions through vocabularies:
// https://json-schema.org/understanding-json-schema/reference/schema.html#vocabularies.
// We don't (yet?) define a meta-schema for the extensions below.
// It's not a big issue because the reach/scope of these extensions is limited.
type Extension struct {
	// Order defines the order of a field with respect to other fields.
	// If not defined, the field is ordered alphabetically after all fields
	// that do have an order defined.
	Order *int `json:"order,omitempty"`

	// PatternMatchFailureMessage is a user defined message that is displayed to the
	// user if a JSON schema pattern match fails.
	PatternMatchFailureMessage string `json:"pattern_match_failure_message,omitempty"`
}
