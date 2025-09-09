// Generated from Databricks Terraform provider schema. DO NOT EDIT.

package schema

type ResourceServicePrincipalFederationPolicyOidcPolicy struct {
	Audiences    []string `json:"audiences,omitempty"`
	Issuer       string   `json:"issuer,omitempty"`
	JwksJson     string   `json:"jwks_json,omitempty"`
	JwksUri      string   `json:"jwks_uri,omitempty"`
	Subject      string   `json:"subject,omitempty"`
	SubjectClaim string   `json:"subject_claim,omitempty"`
}

type ResourceServicePrincipalFederationPolicy struct {
	CreateTime         string                                              `json:"create_time,omitempty"`
	Description        string                                              `json:"description,omitempty"`
	Name               string                                              `json:"name,omitempty"`
	OidcPolicy         *ResourceServicePrincipalFederationPolicyOidcPolicy `json:"oidc_policy,omitempty"`
	PolicyId           string                                              `json:"policy_id,omitempty"`
	ServicePrincipalId int                                                 `json:"service_principal_id,omitempty"`
	Uid                string                                              `json:"uid,omitempty"`
	UpdateTime         string                                              `json:"update_time,omitempty"`
}
