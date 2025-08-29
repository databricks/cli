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

	// If true, takes priority over 'DeprecationMessage'
	ForceNotDeprecated bool `json:"force_not_deprecated,omitempty"`
}

const Placeholder = "PLACEHOLDER"
