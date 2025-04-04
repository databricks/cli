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
	Deprecated          string `json:"deprecated,omitempty"`
}

const Placeholder = "PLACEHOLDER"
