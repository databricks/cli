// Generated from Databricks Terraform provider schema. DO NOT EDIT.

package schema

type ResourceDbfsFile struct {
	ContentBase64 string `json:"content_base64,omitempty"`
	DbfsPath      string `json:"dbfs_path,omitempty"`
	FileSize      int    `json:"file_size,omitempty"`
	Id            string `json:"id,omitempty"`
	Md5           string `json:"md5,omitempty"`
	Path          string `json:"path"`
	Source        string `json:"source,omitempty"`
}
