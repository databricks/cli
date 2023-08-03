// Generated from Databricks Terraform provider schema. DO NOT EDIT.

package schema

type ResourceAccessControlRuleSetGrantRules struct {
	Principals []string `json:"principals,omitempty"`
	Role       string   `json:"role"`
}

type ResourceAccessControlRuleSet struct {
	Etag       string                                   `json:"etag,omitempty"`
	Id         string                                   `json:"id,omitempty"`
	Name       string                                   `json:"name"`
	GrantRules []ResourceAccessControlRuleSetGrantRules `json:"grant_rules,omitempty"`
}
