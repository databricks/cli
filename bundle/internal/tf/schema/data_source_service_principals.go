// Generated from Databricks Terraform provider schema. DO NOT EDIT.

package schema

type DataSourceServicePrincipalsServicePrincipals struct {
	AclPrincipalId string `json:"acl_principal_id,omitempty"`
	Active         bool   `json:"active,omitempty"`
	ApplicationId  string `json:"application_id,omitempty"`
	DisplayName    string `json:"display_name,omitempty"`
	ExternalId     string `json:"external_id,omitempty"`
	Home           string `json:"home,omitempty"`
	Id             string `json:"id,omitempty"`
	Repos          string `json:"repos,omitempty"`
	ScimId         string `json:"scim_id,omitempty"`
	SpId           string `json:"sp_id,omitempty"`
}

type DataSourceServicePrincipals struct {
	ApplicationIds      []string                                       `json:"application_ids,omitempty"`
	DisplayNameContains string                                         `json:"display_name_contains,omitempty"`
	Id                  string                                         `json:"id,omitempty"`
	ServicePrincipals   []DataSourceServicePrincipalsServicePrincipals `json:"service_principals,omitempty"`
}
