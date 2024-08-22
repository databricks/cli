// Generated from Databricks Terraform provider schema. DO NOT EDIT.

package schema

type DataSourceNotebook struct {
	Content       string `json:"content,omitempty"`
	Format        string `json:"format"`
	Id            string `json:"id,omitempty"`
	Language      string `json:"language,omitempty"`
	ObjectId      int    `json:"object_id,omitempty"`
	ObjectType    string `json:"object_type,omitempty"`
	Path          string `json:"path"`
	WorkspacePath string `json:"workspace_path,omitempty"`
}
