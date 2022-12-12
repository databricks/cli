// Generated from Databricks Terraform provider schema. DO NOT EDIT.

package schema

type ResourceSecretScopeKeyvaultMetadata struct {
	DnsName    string `json:"dns_name"`
	ResourceId string `json:"resource_id"`
}

type ResourceSecretScope struct {
	BackendType            string                               `json:"backend_type,omitempty"`
	Id                     string                               `json:"id,omitempty"`
	InitialManagePrincipal string                               `json:"initial_manage_principal,omitempty"`
	Name                   string                               `json:"name"`
	KeyvaultMetadata       *ResourceSecretScopeKeyvaultMetadata `json:"keyvault_metadata,omitempty"`
}
