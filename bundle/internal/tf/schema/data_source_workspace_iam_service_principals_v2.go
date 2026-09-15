// Generated from Databricks Terraform provider schema. DO NOT EDIT.

package schema

type DataSourceWorkspaceIamServicePrincipalsV2ProviderConfig struct {
	WorkspaceId string `json:"workspace_id,omitempty"`
}

type DataSourceWorkspaceIamServicePrincipalsV2ServicePrincipalsProviderConfig struct {
	WorkspaceId string `json:"workspace_id,omitempty"`
}

type DataSourceWorkspaceIamServicePrincipalsV2ServicePrincipals struct {
	AccountId          string                                                                    `json:"account_id,omitempty"`
	AccountSpStatus    string                                                                    `json:"account_sp_status,omitempty"`
	ApplicationId      string                                                                    `json:"application_id,omitempty"`
	DisplayName        string                                                                    `json:"display_name,omitempty"`
	ExternalId         string                                                                    `json:"external_id,omitempty"`
	ProviderConfig     *DataSourceWorkspaceIamServicePrincipalsV2ServicePrincipalsProviderConfig `json:"provider_config,omitempty"`
	ServicePrincipalId string                                                                    `json:"service_principal_id"`
}

type DataSourceWorkspaceIamServicePrincipalsV2 struct {
	Filter            string                                                       `json:"filter,omitempty"`
	PageSize          int                                                          `json:"page_size,omitempty"`
	ProviderConfig    *DataSourceWorkspaceIamServicePrincipalsV2ProviderConfig     `json:"provider_config,omitempty"`
	ServicePrincipals []DataSourceWorkspaceIamServicePrincipalsV2ServicePrincipals `json:"service_principals,omitempty"`
}
