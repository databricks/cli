// Generated from Databricks Terraform provider schema. DO NOT EDIT.

package schema

type ResourceSchema struct {
	CatalogName  string            `json:"catalog_name"`
	Comment      string            `json:"comment,omitempty"`
	ForceDestroy bool              `json:"force_destroy,omitempty"`
	Id           string            `json:"id,omitempty"`
	MetastoreId  string            `json:"metastore_id,omitempty"`
	Name         string            `json:"name"`
	Owner        string            `json:"owner,omitempty"`
	Properties   map[string]string `json:"properties,omitempty"`
}
