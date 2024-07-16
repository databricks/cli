// Generated from Databricks Terraform provider schema. DO NOT EDIT.

package schema

type ResourceDirectory struct {
	DeleteRecursive bool   `json:"delete_recursive,omitempty"`
	Id              string `json:"id,omitempty"`
	ObjectId        int    `json:"object_id,omitempty"`
	Path            string `json:"path"`
	WorkspacePath   string `json:"workspace_path,omitempty"`
}
