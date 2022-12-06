// Generated from Databricks Terraform provider schema. DO NOT EDIT.

package tf

type ResourceGrantsGrant struct {
	Principal  string   `json:"principal"`
	Privileges []string `json:"privileges"`
}

type ResourceGrants struct {
	Catalog           string                `json:"catalog,omitempty"`
	ExternalLocation  string                `json:"external_location,omitempty"`
	Function          string                `json:"function,omitempty"`
	Id                string                `json:"id,omitempty"`
	MaterializedView  string                `json:"materialized_view,omitempty"`
	Metastore         string                `json:"metastore,omitempty"`
	Schema            string                `json:"schema,omitempty"`
	Share             string                `json:"share,omitempty"`
	StorageCredential string                `json:"storage_credential,omitempty"`
	Table             string                `json:"table,omitempty"`
	View              string                `json:"view,omitempty"`
	Grant             []ResourceGrantsGrant `json:"grant,omitempty"`
}
