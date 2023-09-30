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

	// The message to print after the template is successfully initalized
	// In line with our ymls, we use snake_case here.
	SuccessMessage string `json:"success_message,omitempty"`
}
