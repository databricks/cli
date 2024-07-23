// Generated from Databricks Terraform provider schema. DO NOT EDIT.

package schema

type DataSourceSchemaSchemaInfoEffectivePredictiveOptimizationFlag struct {
	InheritedFromName string `json:"inherited_from_name,omitempty"`
	InheritedFromType string `json:"inherited_from_type,omitempty"`
	Value             string `json:"value"`
}

type DataSourceSchemaSchemaInfo struct {
	BrowseOnly                          bool                                                           `json:"browse_only,omitempty"`
	CatalogName                         string                                                         `json:"catalog_name,omitempty"`
	CatalogType                         string                                                         `json:"catalog_type,omitempty"`
	Comment                             string                                                         `json:"comment,omitempty"`
	CreatedAt                           int                                                            `json:"created_at,omitempty"`
	CreatedBy                           string                                                         `json:"created_by,omitempty"`
	EnablePredictiveOptimization        string                                                         `json:"enable_predictive_optimization,omitempty"`
	FullName                            string                                                         `json:"full_name,omitempty"`
	MetastoreId                         string                                                         `json:"metastore_id,omitempty"`
	Name                                string                                                         `json:"name,omitempty"`
	Owner                               string                                                         `json:"owner,omitempty"`
	Properties                          map[string]string                                              `json:"properties,omitempty"`
	SchemaId                            string                                                         `json:"schema_id,omitempty"`
	StorageLocation                     string                                                         `json:"storage_location,omitempty"`
	StorageRoot                         string                                                         `json:"storage_root,omitempty"`
	UpdatedAt                           int                                                            `json:"updated_at,omitempty"`
	UpdatedBy                           string                                                         `json:"updated_by,omitempty"`
	EffectivePredictiveOptimizationFlag *DataSourceSchemaSchemaInfoEffectivePredictiveOptimizationFlag `json:"effective_predictive_optimization_flag,omitempty"`
}

type DataSourceSchema struct {
	Id         string                      `json:"id,omitempty"`
	Name       string                      `json:"name"`
	SchemaInfo *DataSourceSchemaSchemaInfo `json:"schema_info,omitempty"`
}
