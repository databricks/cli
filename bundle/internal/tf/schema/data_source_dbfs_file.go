// Generated from Databricks Terraform provider schema. DO NOT EDIT.

package schema

type DataSourceDbfsFile struct {
	Content       string `json:"content,omitempty"`
	FileSize      int    `json:"file_size,omitempty"`
	Id            string `json:"id,omitempty"`
	LimitFileSize bool   `json:"limit_file_size"`
	Path          string `json:"path"`
}
