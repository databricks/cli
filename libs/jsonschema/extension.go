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

	// Welcome message to print before prompting the user for input
	WelcomeMessage string `json:"welcome_message,omitempty"`

	// The message to print after the template is successfully initalized
	SuccessMessage string `json:"success_message,omitempty"`

	// PatternMatchFailureMessage is a user defined message that is displayed to the
	// user if a JSON schema pattern match fails.
	PatternMatchFailureMessage string `json:"pattern_match_failure_message,omitempty"`

	// Set the minimum semver version of this CLI to validate when loading this schema.
	// If the CLI version is less than this value, then validation for this
	// schema will fail.
	MinDatabricksCliVersion string `json:"min_databricks_cli_version,omitempty"`

	// Skip prompting if this schema is satisfied by the configuration already present. In
	// that case the default value of the property is used instead.
	SkipPromptIf *Schema `json:"skip_prompt_if,omitempty"`

	// Version of the schema. This is used to determine if the schema is
	// compatible with the current CLI version.
	Version *int `json:"version,omitempty"`

	// Preview indicates launch stage (e.g. PREVIEW).
	//
	// This field indicates whether the associated field is part of a private preview feature.
	// Currently, it is used exclusively by Python code generation to exclude certain fields
	// from the generated Sphinx documentation.
	Preview string `json:"x-databricks-preview,omitempty"`

	// This field is not in JSON schema spec, but it is supported in VSCode and in the Databricks Workspace
	// It is used to provide a rich description of the field in the hover tooltip.
	// https://code.visualstudio.com/docs/languages/json#_use-rich-formatting-in-hovers
	// Also it can be used in documentation generation.
	MarkdownDescription string `json:"markdownDescription,omitempty"`

	// This field is not in the JSON schema spec, but it is supported in VSCode
	// It is used to provide a warning for deprecated fields
	DeprecationMessage string `json:"deprecationMessage,omitempty"`

	// This field is not in the JSON schema spec, but it is supported in VSCode
	// It hides a property from IntelliSense (autocomplete suggestions).
	DoNotSuggest bool `json:"doNotSuggest,omitempty"`
}
