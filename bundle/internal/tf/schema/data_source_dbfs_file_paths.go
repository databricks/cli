// Generated from Databricks Terraform provider schema. DO NOT EDIT.

package schema

type DataSourceDbfsFilePaths struct {
	Id        string `json:"id,omitempty"`
	Path      string `json:"path"`
	PathList  []any  `json:"path_list,omitempty"`
	Recursive bool   `json:"recursive"`
}
