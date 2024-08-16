// Generated from Databricks Terraform provider schema. DO NOT EDIT.

package schema

type ResourceNotebook struct {
	ContentBase64 string `json:"content_base64,omitempty"`
	Format        string `json:"format,omitempty"`
	Id            string `json:"id,omitempty"`
	Language      string `json:"language,omitempty"`
	Md5           string `json:"md5,omitempty"`
	ObjectId      int    `json:"object_id,omitempty"`
	ObjectType    string `json:"object_type,omitempty"`
	Path          string `json:"path"`
	Source        string `json:"source,omitempty"`
	Url           string `json:"url,omitempty"`
	WorkspacePath string `json:"workspace_path,omitempty"`
}
