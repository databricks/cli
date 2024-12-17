// Generated from Databricks Terraform provider schema. DO NOT EDIT.

package schema

type ResourceGrantsGrant struct {
	Principal  string   `json:"principal"`
	Privileges []string `json:"privileges"`
}

type ResourceGrants struct {
	Catalog           string                `json:"catalog,omitempty"`
	Credential        string                `json:"credential,omitempty"`
	ExternalLocation  string                `json:"external_location,omitempty"`
	ForeignConnection string                `json:"foreign_connection,omitempty"`
	Function          string                `json:"function,omitempty"`
	Id                string                `json:"id,omitempty"`
	Metastore         string                `json:"metastore,omitempty"`
	Model             string                `json:"model,omitempty"`
	Pipeline          string                `json:"pipeline,omitempty"`
	Recipient         string                `json:"recipient,omitempty"`
	Schema            string                `json:"schema,omitempty"`
	Share             string                `json:"share,omitempty"`
	StorageCredential string                `json:"storage_credential,omitempty"`
	Table             string                `json:"table,omitempty"`
	Volume            string                `json:"volume,omitempty"`
	Grant             []ResourceGrantsGrant `json:"grant,omitempty"`
}
