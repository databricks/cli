// Generated from Databricks Terraform provider schema. DO NOT EDIT.

package schema

type ResourceRegisteredModel struct {
	CatalogName     string `json:"catalog_name"`
	Comment         string `json:"comment,omitempty"`
	Id              string `json:"id,omitempty"`
	Name            string `json:"name"`
	SchemaName      string `json:"schema_name"`
	StorageLocation string `json:"storage_location,omitempty"`
}
