// Generated from Databricks Terraform provider schema. DO NOT EDIT.

package schema

type ResourceComplianceSecurityProfileWorkspaceSettingComplianceSecurityProfileWorkspace struct {
	ComplianceStandards []string `json:"compliance_standards"`
	IsEnabled           bool     `json:"is_enabled"`
}

type ResourceComplianceSecurityProfileWorkspaceSetting struct {
	Etag                               string                                                                               `json:"etag,omitempty"`
	Id                                 string                                                                               `json:"id,omitempty"`
	SettingName                        string                                                                               `json:"setting_name,omitempty"`
	ComplianceSecurityProfileWorkspace *ResourceComplianceSecurityProfileWorkspaceSettingComplianceSecurityProfileWorkspace `json:"compliance_security_profile_workspace,omitempty"`
}
