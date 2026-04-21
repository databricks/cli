package resources

// TagValidationRule declares required tag keys (and optionally allowed values)
// for a set of securable types. Enforced by ucm/config/mutator.ValidateTags
// during `validate`, `plan`, and `policy-check`.
//
// Note: this is ucm's own policy engine, independent of any server-side UC
// tag-policy feature.
type TagValidationRule struct {
	// SecurableTypes selects which resource kinds this rule applies to
	// (e.g., ["catalog", "schema"]). M0 supports "catalog" and "schema".
	SecurableTypes []string `json:"securable_types"`

	// Required is the list of tag keys that must be present on every matching
	// securable.
	Required []string `json:"required,omitempty"`

	// AllowedValues restricts the values a given tag key may take. If a key is
	// not present here, any value is allowed.
	AllowedValues map[string][]string `json:"allowed_values,omitempty"`
}
