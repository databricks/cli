// Generated from Databricks Terraform provider schema. DO NOT EDIT.

package schema

type ResourceAzureAdlsGen2Mount struct {
	ClientId             string `json:"client_id"`
	ClientSecretKey      string `json:"client_secret_key"`
	ClientSecretScope    string `json:"client_secret_scope"`
	ClusterId            string `json:"cluster_id,omitempty"`
	ContainerName        string `json:"container_name"`
	Directory            string `json:"directory,omitempty"`
	Environment          string `json:"environment,omitempty"`
	Id                   string `json:"id,omitempty"`
	InitializeFileSystem bool   `json:"initialize_file_system"`
	MountName            string `json:"mount_name"`
	Source               string `json:"source,omitempty"`
	StorageAccountName   string `json:"storage_account_name"`
	TenantId             string `json:"tenant_id"`
}
