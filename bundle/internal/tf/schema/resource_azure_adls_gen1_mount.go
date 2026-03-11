// Generated from Databricks Terraform provider schema. DO NOT EDIT.

package schema

type ResourceAzureAdlsGen1Mount struct {
	ClientId            string `json:"client_id"`
	ClientSecretKey     string `json:"client_secret_key"`
	ClientSecretScope   string `json:"client_secret_scope"`
	ClusterId           string `json:"cluster_id,omitempty"`
	Directory           string `json:"directory,omitempty"`
	Id                  string `json:"id,omitempty"`
	MountName           string `json:"mount_name"`
	Source              string `json:"source,omitempty"`
	SparkConfPrefix     string `json:"spark_conf_prefix,omitempty"`
	StorageResourceName string `json:"storage_resource_name"`
	TenantId            string `json:"tenant_id"`
}
