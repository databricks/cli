// Generated from Databricks Terraform provider schema. DO NOT EDIT.

package schema

type ResourceGitCredential struct {
	Force                bool   `json:"force,omitempty"`
	GitProvider          string `json:"git_provider"`
	GitUsername          string `json:"git_username,omitempty"`
	Id                   string `json:"id,omitempty"`
	IsDefaultForProvider bool   `json:"is_default_for_provider,omitempty"`
	Name                 string `json:"name,omitempty"`
	PersonalAccessToken  string `json:"personal_access_token,omitempty"`
}
