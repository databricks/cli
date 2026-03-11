// Generated from Databricks Terraform provider schema. DO NOT EDIT.

package schema

type DataSourceMetastoreMetastoreInfo struct {
	Cloud                                       string `json:"cloud,omitempty"`
	CreatedAt                                   int    `json:"created_at,omitempty"`
	CreatedBy                                   string `json:"created_by,omitempty"`
	DefaultDataAccessConfigId                   string `json:"default_data_access_config_id,omitempty"`
	DeltaSharingOrganizationName                string `json:"delta_sharing_organization_name,omitempty"`
	DeltaSharingRecipientTokenLifetimeInSeconds int    `json:"delta_sharing_recipient_token_lifetime_in_seconds,omitempty"`
	DeltaSharingScope                           string `json:"delta_sharing_scope,omitempty"`
	ExternalAccessEnabled                       bool   `json:"external_access_enabled,omitempty"`
	GlobalMetastoreId                           string `json:"global_metastore_id,omitempty"`
	MetastoreId                                 string `json:"metastore_id,omitempty"`
	Name                                        string `json:"name,omitempty"`
	Owner                                       string `json:"owner,omitempty"`
	PrivilegeModelVersion                       string `json:"privilege_model_version,omitempty"`
	Region                                      string `json:"region,omitempty"`
	StorageRoot                                 string `json:"storage_root,omitempty"`
	StorageRootCredentialId                     string `json:"storage_root_credential_id,omitempty"`
	StorageRootCredentialName                   string `json:"storage_root_credential_name,omitempty"`
	UpdatedAt                                   int    `json:"updated_at,omitempty"`
	UpdatedBy                                   string `json:"updated_by,omitempty"`
}

type DataSourceMetastore struct {
	Id            string                            `json:"id,omitempty"`
	MetastoreId   string                            `json:"metastore_id,omitempty"`
	Name          string                            `json:"name,omitempty"`
	Region        string                            `json:"region,omitempty"`
	MetastoreInfo *DataSourceMetastoreMetastoreInfo `json:"metastore_info,omitempty"`
}
