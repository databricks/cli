// Generated from Databricks Terraform provider schema. DO NOT EDIT.

package schema

type DataSourceCatalogCatalogInfoEffectivePredictiveOptimizationFlag struct {
	InheritedFromName string `json:"inherited_from_name,omitempty"`
	InheritedFromType string `json:"inherited_from_type,omitempty"`
	Value             string `json:"value"`
}

type DataSourceCatalogCatalogInfoProvisioningInfo struct {
	State string `json:"state,omitempty"`
}

type DataSourceCatalogCatalogInfo struct {
	BrowseOnly                          bool                                                             `json:"browse_only,omitempty"`
	CatalogType                         string                                                           `json:"catalog_type,omitempty"`
	Comment                             string                                                           `json:"comment,omitempty"`
	ConnectionName                      string                                                           `json:"connection_name,omitempty"`
	CreatedAt                           int                                                              `json:"created_at,omitempty"`
	CreatedBy                           string                                                           `json:"created_by,omitempty"`
	EnablePredictiveOptimization        string                                                           `json:"enable_predictive_optimization,omitempty"`
	FullName                            string                                                           `json:"full_name,omitempty"`
	IsolationMode                       string                                                           `json:"isolation_mode,omitempty"`
	MetastoreId                         string                                                           `json:"metastore_id,omitempty"`
	Name                                string                                                           `json:"name,omitempty"`
	Options                             map[string]string                                                `json:"options,omitempty"`
	Owner                               string                                                           `json:"owner,omitempty"`
	Properties                          map[string]string                                                `json:"properties,omitempty"`
	ProviderName                        string                                                           `json:"provider_name,omitempty"`
	SecurableType                       string                                                           `json:"securable_type,omitempty"`
	ShareName                           string                                                           `json:"share_name,omitempty"`
	StorageLocation                     string                                                           `json:"storage_location,omitempty"`
	StorageRoot                         string                                                           `json:"storage_root,omitempty"`
	UpdatedAt                           int                                                              `json:"updated_at,omitempty"`
	UpdatedBy                           string                                                           `json:"updated_by,omitempty"`
	EffectivePredictiveOptimizationFlag *DataSourceCatalogCatalogInfoEffectivePredictiveOptimizationFlag `json:"effective_predictive_optimization_flag,omitempty"`
	ProvisioningInfo                    *DataSourceCatalogCatalogInfoProvisioningInfo                    `json:"provisioning_info,omitempty"`
}

type DataSourceCatalog struct {
	Id          string                        `json:"id,omitempty"`
	Name        string                        `json:"name"`
	CatalogInfo *DataSourceCatalogCatalogInfo `json:"catalog_info,omitempty"`
}
