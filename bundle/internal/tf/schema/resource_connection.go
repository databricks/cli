// Generated from Databricks Terraform provider schema. DO NOT EDIT.

package schema

type ResourceConnection struct {
	Comment        string            `json:"comment,omitempty"`
	ConnectionType string            `json:"connection_type"`
	Id             string            `json:"id,omitempty"`
	MetastoreId    string            `json:"metastore_id,omitempty"`
	Name           string            `json:"name"`
	Options        map[string]string `json:"options"`
	Owner          string            `json:"owner,omitempty"`
	Properties     map[string]string `json:"properties,omitempty"`
	ReadOnly       bool              `json:"read_only,omitempty"`
}
