// Generated from Databricks Terraform provider schema. DO NOT EDIT.

package schema

type ResourceEnhancedSecurityMonitoringWorkspaceSettingEnhancedSecurityMonitoringWorkspace struct {
	IsEnabled bool `json:"is_enabled"`
}

type ResourceEnhancedSecurityMonitoringWorkspaceSetting struct {
	Etag                                string                                                                                 `json:"etag,omitempty"`
	Id                                  string                                                                                 `json:"id,omitempty"`
	SettingName                         string                                                                                 `json:"setting_name,omitempty"`
	EnhancedSecurityMonitoringWorkspace *ResourceEnhancedSecurityMonitoringWorkspaceSettingEnhancedSecurityMonitoringWorkspace `json:"enhanced_security_monitoring_workspace,omitempty"`
}
