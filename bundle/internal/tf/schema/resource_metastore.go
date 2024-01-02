// Generated from Databricks Terraform provider schema. DO NOT EDIT.

package schema

type ResourceMetastore struct {
	Cloud                                       string `json:"cloud,omitempty"`
	CreatedAt                                   int    `json:"created_at,omitempty"`
	CreatedBy                                   string `json:"created_by,omitempty"`
	DefaultDataAccessConfigId                   string `json:"default_data_access_config_id,omitempty"`
	DeltaSharingOrganizationName                string `json:"delta_sharing_organization_name,omitempty"`
	DeltaSharingRecipientTokenLifetimeInSeconds int    `json:"delta_sharing_recipient_token_lifetime_in_seconds,omitempty"`
	DeltaSharingScope                           string `json:"delta_sharing_scope,omitempty"`
	ForceDestroy                                bool   `json:"force_destroy,omitempty"`
	GlobalMetastoreId                           string `json:"global_metastore_id,omitempty"`
	Id                                          string `json:"id,omitempty"`
	MetastoreId                                 string `json:"metastore_id,omitempty"`
	Name                                        string `json:"name"`
	Owner                                       string `json:"owner,omitempty"`
	Region                                      string `json:"region,omitempty"`
	StorageRoot                                 string `json:"storage_root,omitempty"`
	StorageRootCredentialId                     string `json:"storage_root_credential_id,omitempty"`
	UpdatedAt                                   int    `json:"updated_at,omitempty"`
	UpdatedBy                                   string `json:"updated_by,omitempty"`
}
