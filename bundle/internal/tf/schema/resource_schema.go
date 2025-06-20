// Generated from Databricks Terraform provider schema. DO NOT EDIT.

package schema

type ResourceSchema struct {
	CatalogName                  string            `json:"catalog_name"`
	Comment                      string            `json:"comment,omitempty"`
	EnablePredictiveOptimization string            `json:"enable_predictive_optimization,omitempty"`
	ForceDestroy                 bool              `json:"force_destroy,omitempty"`
	Id                           string            `json:"id,omitempty"`
	MetastoreId                  string            `json:"metastore_id,omitempty"`
	Name                         string            `json:"name"`
	Owner                        string            `json:"owner,omitempty"`
	Properties                   map[string]string `json:"properties,omitempty"`
	SchemaId                     string            `json:"schema_id,omitempty"`
	StorageRoot                  string            `json:"storage_root,omitempty"`
}
