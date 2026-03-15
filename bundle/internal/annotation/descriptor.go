package annotation

type Descriptor struct {
	Description         string `json:"description,omitempty"`
	MarkdownDescription string `json:"markdown_description,omitempty"`
	Title               string `json:"title,omitempty"`
	Default             any    `json:"default,omitempty"`
	Enum                []any  `json:"enum,omitempty"`
	MarkdownExamples    string `json:"markdown_examples,omitempty"`
	DeprecationMessage  string `json:"deprecation_message,omitempty"`
	Preview             string `json:"x-databricks-preview,omitempty"`
	// OutputOnly is stored as a string "true" rather than a bool to ensure
	// consistent YAML serialization (literal block style treats bools as strings).
	OutputOnly string `json:"x-databricks-field-behaviors_output_only,omitempty"`
}

const Placeholder = "PLACEHOLDER"
