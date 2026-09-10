// Generated from Databricks Terraform provider schema. DO NOT EDIT.

package schema

type DataSourceWorkspaceIamExternalServicePrincipalV2ProviderConfig struct {
	WorkspaceId string `json:"workspace_id,omitempty"`
}

type DataSourceWorkspaceIamExternalServicePrincipalV2 struct {
	AccountId                  string                                                          `json:"account_id,omitempty"`
	AccountSpStatus            string                                                          `json:"account_sp_status,omitempty"`
	ApplicationId              string                                                          `json:"application_id,omitempty"`
	DisplayName                string                                                          `json:"display_name,omitempty"`
	ExternalServicePrincipalId string                                                          `json:"external_service_principal_id,omitempty"`
	InternalId                 string                                                          `json:"internal_id,omitempty"`
	Name                       string                                                          `json:"name"`
	ProviderConfig             *DataSourceWorkspaceIamExternalServicePrincipalV2ProviderConfig `json:"provider_config,omitempty"`
}
