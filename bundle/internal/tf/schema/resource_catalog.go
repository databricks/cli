// Generated from Databricks Terraform provider schema. DO NOT EDIT.

package schema

type ResourceCatalogEffectivePredictiveOptimizationFlag struct {
	InheritedFromName string `json:"inherited_from_name,omitempty"`
	InheritedFromType string `json:"inherited_from_type,omitempty"`
	Value             string `json:"value"`
}

type ResourceCatalogManagedEncryptionSettingsAzureEncryptionSettings struct {
	AzureCmkAccessConnectorId string `json:"azure_cmk_access_connector_id,omitempty"`
	AzureCmkManagedIdentityId string `json:"azure_cmk_managed_identity_id,omitempty"`
	AzureTenantId             string `json:"azure_tenant_id"`
}

type ResourceCatalogManagedEncryptionSettings struct {
	AzureKeyVaultKeyId      string                                                           `json:"azure_key_vault_key_id,omitempty"`
	CustomerManagedKeyId    string                                                           `json:"customer_managed_key_id,omitempty"`
	AzureEncryptionSettings *ResourceCatalogManagedEncryptionSettingsAzureEncryptionSettings `json:"azure_encryption_settings,omitempty"`
}

type ResourceCatalogProviderConfig struct {
	WorkspaceId string `json:"workspace_id,omitempty"`
}

type ResourceCatalogProvisioningInfo struct {
	State string `json:"state,omitempty"`
}

type ResourceCatalog struct {
	BrowseOnly                          bool                                                `json:"browse_only,omitempty"`
	CatalogType                         string                                              `json:"catalog_type,omitempty"`
	Comment                             string                                              `json:"comment,omitempty"`
	ConnectionName                      string                                              `json:"connection_name,omitempty"`
	CreatedAt                           int                                                 `json:"created_at,omitempty"`
	CreatedBy                           string                                              `json:"created_by,omitempty"`
	EnablePredictiveOptimization        string                                              `json:"enable_predictive_optimization,omitempty"`
	ForceDestroy                        bool                                                `json:"force_destroy,omitempty"`
	FullName                            string                                              `json:"full_name,omitempty"`
	Id                                  string                                              `json:"id,omitempty"`
	IsolationMode                       string                                              `json:"isolation_mode,omitempty"`
	MetastoreId                         string                                              `json:"metastore_id,omitempty"`
	Name                                string                                              `json:"name,omitempty"`
	Options                             map[string]string                                   `json:"options,omitempty"`
	Owner                               string                                              `json:"owner,omitempty"`
	Properties                          map[string]string                                   `json:"properties,omitempty"`
	ProviderName                        string                                              `json:"provider_name,omitempty"`
	SecurableType                       string                                              `json:"securable_type,omitempty"`
	ShareName                           string                                              `json:"share_name,omitempty"`
	StorageLocation                     string                                              `json:"storage_location,omitempty"`
	StorageRoot                         string                                              `json:"storage_root,omitempty"`
	UpdatedAt                           int                                                 `json:"updated_at,omitempty"`
	UpdatedBy                           string                                              `json:"updated_by,omitempty"`
	EffectivePredictiveOptimizationFlag *ResourceCatalogEffectivePredictiveOptimizationFlag `json:"effective_predictive_optimization_flag,omitempty"`
	ManagedEncryptionSettings           *ResourceCatalogManagedEncryptionSettings           `json:"managed_encryption_settings,omitempty"`
	ProviderConfig                      *ResourceCatalogProviderConfig                      `json:"provider_config,omitempty"`
	ProvisioningInfo                    *ResourceCatalogProvisioningInfo                    `json:"provisioning_info,omitempty"`
}
