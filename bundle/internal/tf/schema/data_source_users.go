// Generated from Databricks Terraform provider schema. DO NOT EDIT.

package schema

type DataSourceUsersProviderConfig struct {
	WorkspaceId string `json:"workspace_id"`
}

type DataSourceUsersUsersEmails struct {
	Display string `json:"display,omitempty"`
	Primary bool   `json:"primary,omitempty"`
	Ref     string `json:"ref,omitempty"`
	Type    string `json:"type,omitempty"`
	Value   string `json:"value,omitempty"`
}

type DataSourceUsersUsersEntitlements struct {
	Display string `json:"display,omitempty"`
	Primary bool   `json:"primary,omitempty"`
	Ref     string `json:"ref,omitempty"`
	Type    string `json:"type,omitempty"`
	Value   string `json:"value,omitempty"`
}

type DataSourceUsersUsersGroups struct {
	Display string `json:"display,omitempty"`
	Primary bool   `json:"primary,omitempty"`
	Ref     string `json:"ref,omitempty"`
	Type    string `json:"type,omitempty"`
	Value   string `json:"value,omitempty"`
}

type DataSourceUsersUsersName struct {
	FamilyName string `json:"family_name,omitempty"`
	GivenName  string `json:"given_name,omitempty"`
}

type DataSourceUsersUsersRoles struct {
	Display string `json:"display,omitempty"`
	Primary bool   `json:"primary,omitempty"`
	Ref     string `json:"ref,omitempty"`
	Type    string `json:"type,omitempty"`
	Value   string `json:"value,omitempty"`
}

type DataSourceUsersUsers struct {
	Active       bool                               `json:"active,omitempty"`
	DisplayName  string                             `json:"display_name,omitempty"`
	Emails       []DataSourceUsersUsersEmails       `json:"emails,omitempty"`
	Entitlements []DataSourceUsersUsersEntitlements `json:"entitlements,omitempty"`
	ExternalId   string                             `json:"external_id,omitempty"`
	Groups       []DataSourceUsersUsersGroups       `json:"groups,omitempty"`
	Id           string                             `json:"id,omitempty"`
	Name         *DataSourceUsersUsersName          `json:"name,omitempty"`
	Roles        []DataSourceUsersUsersRoles        `json:"roles,omitempty"`
	Schemas      []string                           `json:"schemas,omitempty"`
	UserName     string                             `json:"user_name,omitempty"`
}

type DataSourceUsers struct {
	Api             string                         `json:"api,omitempty"`
	ExtraAttributes string                         `json:"extra_attributes,omitempty"`
	Filter          string                         `json:"filter,omitempty"`
	ProviderConfig  *DataSourceUsersProviderConfig `json:"provider_config,omitempty"`
	Users           []DataSourceUsersUsers         `json:"users,omitempty"`
}
