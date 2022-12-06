// Generated from Databricks Terraform provider schema. DO NOT EDIT.

package schema

type ResourceAzureBlobMount struct {
	AuthType           string `json:"auth_type"`
	ClusterId          string `json:"cluster_id,omitempty"`
	ContainerName      string `json:"container_name"`
	Directory          string `json:"directory,omitempty"`
	Id                 string `json:"id,omitempty"`
	MountName          string `json:"mount_name"`
	Source             string `json:"source,omitempty"`
	StorageAccountName string `json:"storage_account_name"`
	TokenSecretKey     string `json:"token_secret_key"`
	TokenSecretScope   string `json:"token_secret_scope"`
}
