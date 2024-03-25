// Generated from Databricks Terraform provider schema. DO NOT EDIT.

package schema

type ResourceFile struct {
	ContentBase64      string `json:"content_base64,omitempty"`
	FileSize           int    `json:"file_size,omitempty"`
	Id                 string `json:"id,omitempty"`
	Md5                string `json:"md5,omitempty"`
	ModificationTime   string `json:"modification_time,omitempty"`
	Path               string `json:"path"`
	RemoteFileModified bool   `json:"remote_file_modified,omitempty"`
	Source             string `json:"source,omitempty"`
}
