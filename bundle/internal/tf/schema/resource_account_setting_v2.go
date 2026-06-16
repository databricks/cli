// Generated from Databricks Terraform provider schema. DO NOT EDIT.

package schema

type ResourceAccountSettingV2AibiDashboardEmbeddingAccessPolicy struct {
	AccessPolicyType string `json:"access_policy_type"`
}

type ResourceAccountSettingV2AibiDashboardEmbeddingApprovedDomains struct {
	ApprovedDomains []string `json:"approved_domains,omitempty"`
}

type ResourceAccountSettingV2AllowedAppsUserApiScopes struct {
	AllowedScopes []string `json:"allowed_scopes,omitempty"`
}

type ResourceAccountSettingV2AutomaticClusterUpdateWorkspaceEnablementDetails struct {
	ForcedForComplianceMode           bool `json:"forced_for_compliance_mode,omitempty"`
	UnavailableForDisabledEntitlement bool `json:"unavailable_for_disabled_entitlement,omitempty"`
	UnavailableForNonEnterpriseTier   bool `json:"unavailable_for_non_enterprise_tier,omitempty"`
}

type ResourceAccountSettingV2AutomaticClusterUpdateWorkspaceMaintenanceWindowWeekDayBasedScheduleWindowStartTime struct {
	Hours   int `json:"hours,omitempty"`
	Minutes int `json:"minutes,omitempty"`
}

type ResourceAccountSettingV2AutomaticClusterUpdateWorkspaceMaintenanceWindowWeekDayBasedSchedule struct {
	DayOfWeek       string                                                                                                       `json:"day_of_week,omitempty"`
	Frequency       string                                                                                                       `json:"frequency,omitempty"`
	WindowStartTime *ResourceAccountSettingV2AutomaticClusterUpdateWorkspaceMaintenanceWindowWeekDayBasedScheduleWindowStartTime `json:"window_start_time,omitempty"`
}

type ResourceAccountSettingV2AutomaticClusterUpdateWorkspaceMaintenanceWindow struct {
	WeekDayBasedSchedule *ResourceAccountSettingV2AutomaticClusterUpdateWorkspaceMaintenanceWindowWeekDayBasedSchedule `json:"week_day_based_schedule,omitempty"`
}

type ResourceAccountSettingV2AutomaticClusterUpdateWorkspace struct {
	CanToggle                       bool                                                                      `json:"can_toggle,omitempty"`
	Enabled                         bool                                                                      `json:"enabled,omitempty"`
	EnablementDetails               *ResourceAccountSettingV2AutomaticClusterUpdateWorkspaceEnablementDetails `json:"enablement_details,omitempty"`
	MaintenanceWindow               *ResourceAccountSettingV2AutomaticClusterUpdateWorkspaceMaintenanceWindow `json:"maintenance_window,omitempty"`
	RestartEvenIfNoUpdatesAvailable bool                                                                      `json:"restart_even_if_no_updates_available,omitempty"`
}

type ResourceAccountSettingV2BooleanVal struct {
	Value bool `json:"value,omitempty"`
}

type ResourceAccountSettingV2CollaborationPlatformConnectivity struct {
	Connectivity string `json:"connectivity"`
}

type ResourceAccountSettingV2EffectiveAibiDashboardEmbeddingAccessPolicy struct {
	AccessPolicyType string `json:"access_policy_type"`
}

type ResourceAccountSettingV2EffectiveAibiDashboardEmbeddingApprovedDomains struct {
	ApprovedDomains []string `json:"approved_domains,omitempty"`
}

type ResourceAccountSettingV2EffectiveAllowedAppsUserApiScopes struct {
	AllowedScopes []string `json:"allowed_scopes,omitempty"`
}

type ResourceAccountSettingV2EffectiveAutomaticClusterUpdateWorkspaceEnablementDetails struct {
	ForcedForComplianceMode           bool `json:"forced_for_compliance_mode,omitempty"`
	UnavailableForDisabledEntitlement bool `json:"unavailable_for_disabled_entitlement,omitempty"`
	UnavailableForNonEnterpriseTier   bool `json:"unavailable_for_non_enterprise_tier,omitempty"`
}

type ResourceAccountSettingV2EffectiveAutomaticClusterUpdateWorkspaceMaintenanceWindowWeekDayBasedScheduleWindowStartTime struct {
	Hours   int `json:"hours,omitempty"`
	Minutes int `json:"minutes,omitempty"`
}

type ResourceAccountSettingV2EffectiveAutomaticClusterUpdateWorkspaceMaintenanceWindowWeekDayBasedSchedule struct {
	DayOfWeek       string                                                                                                                `json:"day_of_week,omitempty"`
	Frequency       string                                                                                                                `json:"frequency,omitempty"`
	WindowStartTime *ResourceAccountSettingV2EffectiveAutomaticClusterUpdateWorkspaceMaintenanceWindowWeekDayBasedScheduleWindowStartTime `json:"window_start_time,omitempty"`
}

type ResourceAccountSettingV2EffectiveAutomaticClusterUpdateWorkspaceMaintenanceWindow struct {
	WeekDayBasedSchedule *ResourceAccountSettingV2EffectiveAutomaticClusterUpdateWorkspaceMaintenanceWindowWeekDayBasedSchedule `json:"week_day_based_schedule,omitempty"`
}

type ResourceAccountSettingV2EffectiveAutomaticClusterUpdateWorkspace struct {
	CanToggle                       bool                                                                               `json:"can_toggle,omitempty"`
	Enabled                         bool                                                                               `json:"enabled,omitempty"`
	EnablementDetails               *ResourceAccountSettingV2EffectiveAutomaticClusterUpdateWorkspaceEnablementDetails `json:"enablement_details,omitempty"`
	MaintenanceWindow               *ResourceAccountSettingV2EffectiveAutomaticClusterUpdateWorkspaceMaintenanceWindow `json:"maintenance_window,omitempty"`
	RestartEvenIfNoUpdatesAvailable bool                                                                               `json:"restart_even_if_no_updates_available,omitempty"`
}

type ResourceAccountSettingV2EffectiveBooleanVal struct {
	Value bool `json:"value,omitempty"`
}

type ResourceAccountSettingV2EffectiveCollaborationPlatformConnectivity struct {
	Connectivity string `json:"connectivity"`
}

type ResourceAccountSettingV2EffectiveIntegerVal struct {
	Value int `json:"value,omitempty"`
}

type ResourceAccountSettingV2EffectiveOperationalEmailCustomRecipient struct {
	Email string `json:"email,omitempty"`
}

type ResourceAccountSettingV2EffectivePersonalCompute struct {
	Value string `json:"value,omitempty"`
}

type ResourceAccountSettingV2EffectiveRestrictWorkspaceAdmins struct {
	DisableGovTagCreation bool   `json:"disable_gov_tag_creation,omitempty"`
	Status                string `json:"status"`
}

type ResourceAccountSettingV2EffectiveStringVal struct {
	Value string `json:"value,omitempty"`
}

type ResourceAccountSettingV2IntegerVal struct {
	Value int `json:"value,omitempty"`
}

type ResourceAccountSettingV2OperationalEmailCustomRecipient struct {
	Email string `json:"email,omitempty"`
}

type ResourceAccountSettingV2PersonalCompute struct {
	Value string `json:"value,omitempty"`
}

type ResourceAccountSettingV2RestrictWorkspaceAdmins struct {
	DisableGovTagCreation bool   `json:"disable_gov_tag_creation,omitempty"`
	Status                string `json:"status"`
}

type ResourceAccountSettingV2StringVal struct {
	Value string `json:"value,omitempty"`
}

type ResourceAccountSettingV2 struct {
	AibiDashboardEmbeddingAccessPolicy             *ResourceAccountSettingV2AibiDashboardEmbeddingAccessPolicy             `json:"aibi_dashboard_embedding_access_policy,omitempty"`
	AibiDashboardEmbeddingApprovedDomains          *ResourceAccountSettingV2AibiDashboardEmbeddingApprovedDomains          `json:"aibi_dashboard_embedding_approved_domains,omitempty"`
	AllowedAppsUserApiScopes                       *ResourceAccountSettingV2AllowedAppsUserApiScopes                       `json:"allowed_apps_user_api_scopes,omitempty"`
	AutomaticClusterUpdateWorkspace                *ResourceAccountSettingV2AutomaticClusterUpdateWorkspace                `json:"automatic_cluster_update_workspace,omitempty"`
	BooleanVal                                     *ResourceAccountSettingV2BooleanVal                                     `json:"boolean_val,omitempty"`
	CollaborationPlatformConnectivity              *ResourceAccountSettingV2CollaborationPlatformConnectivity              `json:"collaboration_platform_connectivity,omitempty"`
	EffectiveAibiDashboardEmbeddingAccessPolicy    *ResourceAccountSettingV2EffectiveAibiDashboardEmbeddingAccessPolicy    `json:"effective_aibi_dashboard_embedding_access_policy,omitempty"`
	EffectiveAibiDashboardEmbeddingApprovedDomains *ResourceAccountSettingV2EffectiveAibiDashboardEmbeddingApprovedDomains `json:"effective_aibi_dashboard_embedding_approved_domains,omitempty"`
	EffectiveAllowedAppsUserApiScopes              *ResourceAccountSettingV2EffectiveAllowedAppsUserApiScopes              `json:"effective_allowed_apps_user_api_scopes,omitempty"`
	EffectiveAutomaticClusterUpdateWorkspace       *ResourceAccountSettingV2EffectiveAutomaticClusterUpdateWorkspace       `json:"effective_automatic_cluster_update_workspace,omitempty"`
	EffectiveBooleanVal                            *ResourceAccountSettingV2EffectiveBooleanVal                            `json:"effective_boolean_val,omitempty"`
	EffectiveCollaborationPlatformConnectivity     *ResourceAccountSettingV2EffectiveCollaborationPlatformConnectivity     `json:"effective_collaboration_platform_connectivity,omitempty"`
	EffectiveIntegerVal                            *ResourceAccountSettingV2EffectiveIntegerVal                            `json:"effective_integer_val,omitempty"`
	EffectiveOperationalEmailCustomRecipient       *ResourceAccountSettingV2EffectiveOperationalEmailCustomRecipient       `json:"effective_operational_email_custom_recipient,omitempty"`
	EffectivePersonalCompute                       *ResourceAccountSettingV2EffectivePersonalCompute                       `json:"effective_personal_compute,omitempty"`
	EffectiveRestrictWorkspaceAdmins               *ResourceAccountSettingV2EffectiveRestrictWorkspaceAdmins               `json:"effective_restrict_workspace_admins,omitempty"`
	EffectiveStringVal                             *ResourceAccountSettingV2EffectiveStringVal                             `json:"effective_string_val,omitempty"`
	IntegerVal                                     *ResourceAccountSettingV2IntegerVal                                     `json:"integer_val,omitempty"`
	Name                                           string                                                                  `json:"name,omitempty"`
	OperationalEmailCustomRecipient                *ResourceAccountSettingV2OperationalEmailCustomRecipient                `json:"operational_email_custom_recipient,omitempty"`
	PersonalCompute                                *ResourceAccountSettingV2PersonalCompute                                `json:"personal_compute,omitempty"`
	RestrictWorkspaceAdmins                        *ResourceAccountSettingV2RestrictWorkspaceAdmins                        `json:"restrict_workspace_admins,omitempty"`
	StringVal                                      *ResourceAccountSettingV2StringVal                                      `json:"string_val,omitempty"`
}
