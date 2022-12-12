// Generated from Databricks Terraform provider schema. DO NOT EDIT.

package schema

type DataSourceShareObject struct {
	AddedAt        int    `json:"added_at,omitempty"`
	AddedBy        string `json:"added_by,omitempty"`
	Comment        string `json:"comment,omitempty"`
	DataObjectType string `json:"data_object_type"`
	Name           string `json:"name"`
	SharedAs       string `json:"shared_as,omitempty"`
}

type DataSourceShare struct {
	CreatedAt int                     `json:"created_at,omitempty"`
	CreatedBy string                  `json:"created_by,omitempty"`
	Id        string                  `json:"id,omitempty"`
	Name      string                  `json:"name,omitempty"`
	Object    []DataSourceShareObject `json:"object,omitempty"`
}
