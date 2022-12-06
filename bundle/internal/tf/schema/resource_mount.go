// Generated from Databricks Terraform provider schema. DO NOT EDIT.

package schema

type ResourceMountAbfs struct {
	ClientId             string `json:"client_id"`
	ClientSecretKey      string `json:"client_secret_key"`
	ClientSecretScope    string `json:"client_secret_scope"`
	ContainerName        string `json:"container_name,omitempty"`
	Directory            string `json:"directory,omitempty"`
	InitializeFileSystem bool   `json:"initialize_file_system"`
	StorageAccountName   string `json:"storage_account_name,omitempty"`
	TenantId             string `json:"tenant_id,omitempty"`
}

type ResourceMountAdl struct {
	ClientId            string `json:"client_id"`
	ClientSecretKey     string `json:"client_secret_key"`
	ClientSecretScope   string `json:"client_secret_scope"`
	Directory           string `json:"directory,omitempty"`
	SparkConfPrefix     string `json:"spark_conf_prefix,omitempty"`
	StorageResourceName string `json:"storage_resource_name,omitempty"`
	TenantId            string `json:"tenant_id,omitempty"`
}

type ResourceMountGs struct {
	BucketName     string `json:"bucket_name"`
	ServiceAccount string `json:"service_account,omitempty"`
}

type ResourceMountS3 struct {
	BucketName      string `json:"bucket_name"`
	InstanceProfile string `json:"instance_profile,omitempty"`
}

type ResourceMountWasb struct {
	AuthType           string `json:"auth_type"`
	ContainerName      string `json:"container_name,omitempty"`
	Directory          string `json:"directory,omitempty"`
	StorageAccountName string `json:"storage_account_name,omitempty"`
	TokenSecretKey     string `json:"token_secret_key"`
	TokenSecretScope   string `json:"token_secret_scope"`
}

type ResourceMount struct {
	ClusterId      string             `json:"cluster_id,omitempty"`
	EncryptionType string             `json:"encryption_type,omitempty"`
	ExtraConfigs   map[string]string  `json:"extra_configs,omitempty"`
	Id             string             `json:"id,omitempty"`
	Name           string             `json:"name,omitempty"`
	ResourceId     string             `json:"resource_id,omitempty"`
	Source         string             `json:"source,omitempty"`
	Uri            string             `json:"uri,omitempty"`
	Abfs           *ResourceMountAbfs `json:"abfs,omitempty"`
	Adl            *ResourceMountAdl  `json:"adl,omitempty"`
	Gs             *ResourceMountGs   `json:"gs,omitempty"`
	S3             *ResourceMountS3   `json:"s3,omitempty"`
	Wasb           *ResourceMountWasb `json:"wasb,omitempty"`
}
