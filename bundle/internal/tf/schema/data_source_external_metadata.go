// Generated from Databricks Terraform provider schema. DO NOT EDIT.

package schema

type DataSourceExternalMetadata struct {
	Columns     []string          `json:"columns,omitempty"`
	CreateTime  string            `json:"create_time,omitempty"`
	CreatedBy   string            `json:"created_by,omitempty"`
	Description string            `json:"description,omitempty"`
	EntityType  string            `json:"entity_type"`
	Id          string            `json:"id,omitempty"`
	MetastoreId string            `json:"metastore_id,omitempty"`
	Name        string            `json:"name"`
	Owner       string            `json:"owner,omitempty"`
	Properties  map[string]string `json:"properties,omitempty"`
	SystemType  string            `json:"system_type"`
	UpdateTime  string            `json:"update_time,omitempty"`
	UpdatedBy   string            `json:"updated_by,omitempty"`
	Url         string            `json:"url,omitempty"`
}
