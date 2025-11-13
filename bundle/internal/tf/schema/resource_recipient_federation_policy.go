// Generated from Databricks Terraform provider schema. DO NOT EDIT.

package schema

type ResourceRecipientFederationPolicyOidcPolicy struct {
	Audiences    []string `json:"audiences,omitempty"`
	Issuer       string   `json:"issuer"`
	Subject      string   `json:"subject"`
	SubjectClaim string   `json:"subject_claim"`
}

type ResourceRecipientFederationPolicy struct {
	Comment    string                                       `json:"comment,omitempty"`
	CreateTime string                                       `json:"create_time,omitempty"`
	Id         string                                       `json:"id,omitempty"`
	Name       string                                       `json:"name,omitempty"`
	OidcPolicy *ResourceRecipientFederationPolicyOidcPolicy `json:"oidc_policy,omitempty"`
	UpdateTime string                                       `json:"update_time,omitempty"`
}
