// Generated from Databricks Terraform provider schema. DO NOT EDIT.

package schema

type DataSourceAccountIamServicePrincipalsV2ServicePrincipals struct {
	AccountId          string `json:"account_id,omitempty"`
	AccountSpStatus    string `json:"account_sp_status,omitempty"`
	ApplicationId      string `json:"application_id,omitempty"`
	DisplayName        string `json:"display_name,omitempty"`
	ExternalId         string `json:"external_id,omitempty"`
	ServicePrincipalId string `json:"service_principal_id"`
}

type DataSourceAccountIamServicePrincipalsV2 struct {
	Filter            string                                                     `json:"filter,omitempty"`
	PageSize          int                                                        `json:"page_size,omitempty"`
	ServicePrincipals []DataSourceAccountIamServicePrincipalsV2ServicePrincipals `json:"service_principals,omitempty"`
}
