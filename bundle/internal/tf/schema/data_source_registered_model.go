// Generated from Databricks Terraform provider schema. DO NOT EDIT.

package schema

type DataSourceRegisteredModel struct {
	FullName       string `json:"full_name"`
	IncludeAliases bool   `json:"include_aliases,omitempty"`
	IncludeBrowse  bool   `json:"include_browse,omitempty"`
	ModelInfo      any    `json:"model_info,omitempty"`
}
