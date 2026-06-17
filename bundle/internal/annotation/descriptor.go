package annotation

import "github.com/databricks/cli/internal/clijson"

type Descriptor struct {
	Description         string                         `json:"description,omitempty"`
	MarkdownDescription string                         `json:"markdown_description,omitempty"`
	Title               string                         `json:"title,omitempty"`
	Default             any                            `json:"default,omitempty"`
	Enum                []any                          `json:"enum,omitempty"`
	MarkdownExamples    string                         `json:"markdown_examples,omitempty"`
	DeprecationMessage  string                         `json:"deprecation_message,omitempty"`
	LaunchStage         clijson.LaunchStage            `json:"x-databricks-launch-stage,omitempty"`
	EnumLaunchStages    map[string]clijson.LaunchStage `json:"x-databricks-enum-launch-stages,omitempty"`
	EnumDescriptions    map[string]string              `json:"x-databricks-enum-descriptions,omitempty"`
	OutputOnly          *bool                          `json:"x-databricks-field-behaviors_output_only,omitempty"`
}

const Placeholder = "PLACEHOLDER"
