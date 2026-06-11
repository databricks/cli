// Generated from Databricks Terraform provider schema. DO NOT EDIT.

package schema

type DataSourceWorkspaceSettingV2AibiDashboardEmbeddingAccessPolicy struct {
	AccessPolicyType string `json:"access_policy_type"`
}

type DataSourceWorkspaceSettingV2AibiDashboardEmbeddingApprovedDomains struct {
	ApprovedDomains []string `json:"approved_domains,omitempty"`
}

type DataSourceWorkspaceSettingV2AllowedAppsUserApiScopes struct {
	AllowedScopes []string `json:"allowed_scopes,omitempty"`
}

type DataSourceWorkspaceSettingV2AutomaticClusterUpdateWorkspaceEnablementDetails struct {
	ForcedForComplianceMode           bool `json:"forced_for_compliance_mode,omitempty"`
	UnavailableForDisabledEntitlement bool `json:"unavailable_for_disabled_entitlement,omitempty"`
	UnavailableForNonEnterpriseTier   bool `json:"unavailable_for_non_enterprise_tier,omitempty"`
}

type DataSourceWorkspaceSettingV2AutomaticClusterUpdateWorkspaceMaintenanceWindowWeekDayBasedScheduleWindowStartTime struct {
	Hours   int `json:"hours,omitempty"`
	Minutes int `json:"minutes,omitempty"`
}

type DataSourceWorkspaceSettingV2AutomaticClusterUpdateWorkspaceMaintenanceWindowWeekDayBasedSchedule struct {
	DayOfWeek       string                                                                                                           `json:"day_of_week,omitempty"`
	Frequency       string                                                                                                           `json:"frequency,omitempty"`
	WindowStartTime *DataSourceWorkspaceSettingV2AutomaticClusterUpdateWorkspaceMaintenanceWindowWeekDayBasedScheduleWindowStartTime `json:"window_start_time,omitempty"`
}

type DataSourceWorkspaceSettingV2AutomaticClusterUpdateWorkspaceMaintenanceWindow struct {
	WeekDayBasedSchedule *DataSourceWorkspaceSettingV2AutomaticClusterUpdateWorkspaceMaintenanceWindowWeekDayBasedSchedule `json:"week_day_based_schedule,omitempty"`
}

type DataSourceWorkspaceSettingV2AutomaticClusterUpdateWorkspace struct {
	CanToggle                       bool                                                                          `json:"can_toggle,omitempty"`
	Enabled                         bool                                                                          `json:"enabled,omitempty"`
	EnablementDetails               *DataSourceWorkspaceSettingV2AutomaticClusterUpdateWorkspaceEnablementDetails `json:"enablement_details,omitempty"`
	MaintenanceWindow               *DataSourceWorkspaceSettingV2AutomaticClusterUpdateWorkspaceMaintenanceWindow `json:"maintenance_window,omitempty"`
	RestartEvenIfNoUpdatesAvailable bool                                                                          `json:"restart_even_if_no_updates_available,omitempty"`
}

type DataSourceWorkspaceSettingV2BooleanVal struct {
	Value bool `json:"value,omitempty"`
}

type DataSourceWorkspaceSettingV2CollaborationPlatformConnectivity struct {
	Connectivity string `json:"connectivity"`
}

type DataSourceWorkspaceSettingV2EffectiveAibiDashboardEmbeddingAccessPolicy struct {
	AccessPolicyType string `json:"access_policy_type"`
}

type DataSourceWorkspaceSettingV2EffectiveAibiDashboardEmbeddingApprovedDomains struct {
	ApprovedDomains []string `json:"approved_domains,omitempty"`
}

type DataSourceWorkspaceSettingV2EffectiveAllowedAppsUserApiScopes struct {
	AllowedScopes []string `json:"allowed_scopes,omitempty"`
}

type DataSourceWorkspaceSettingV2EffectiveAutomaticClusterUpdateWorkspaceEnablementDetails struct {
	ForcedForComplianceMode           bool `json:"forced_for_compliance_mode,omitempty"`
	UnavailableForDisabledEntitlement bool `json:"unavailable_for_disabled_entitlement,omitempty"`
	UnavailableForNonEnterpriseTier   bool `json:"unavailable_for_non_enterprise_tier,omitempty"`
}

type DataSourceWorkspaceSettingV2EffectiveAutomaticClusterUpdateWorkspaceMaintenanceWindowWeekDayBasedScheduleWindowStartTime struct {
	Hours   int `json:"hours,omitempty"`
	Minutes int `json:"minutes,omitempty"`
}

type DataSourceWorkspaceSettingV2EffectiveAutomaticClusterUpdateWorkspaceMaintenanceWindowWeekDayBasedSchedule struct {
	DayOfWeek       string                                                                                                                    `json:"day_of_week,omitempty"`
	Frequency       string                                                                                                                    `json:"frequency,omitempty"`
	WindowStartTime *DataSourceWorkspaceSettingV2EffectiveAutomaticClusterUpdateWorkspaceMaintenanceWindowWeekDayBasedScheduleWindowStartTime `json:"window_start_time,omitempty"`
}

type DataSourceWorkspaceSettingV2EffectiveAutomaticClusterUpdateWorkspaceMaintenanceWindow struct {
	WeekDayBasedSchedule *DataSourceWorkspaceSettingV2EffectiveAutomaticClusterUpdateWorkspaceMaintenanceWindowWeekDayBasedSchedule `json:"week_day_based_schedule,omitempty"`
}

type DataSourceWorkspaceSettingV2EffectiveAutomaticClusterUpdateWorkspace struct {
	CanToggle                       bool                                                                                   `json:"can_toggle,omitempty"`
	Enabled                         bool                                                                                   `json:"enabled,omitempty"`
	EnablementDetails               *DataSourceWorkspaceSettingV2EffectiveAutomaticClusterUpdateWorkspaceEnablementDetails `json:"enablement_details,omitempty"`
	MaintenanceWindow               *DataSourceWorkspaceSettingV2EffectiveAutomaticClusterUpdateWorkspaceMaintenanceWindow `json:"maintenance_window,omitempty"`
	RestartEvenIfNoUpdatesAvailable bool                                                                                   `json:"restart_even_if_no_updates_available,omitempty"`
}

type DataSourceWorkspaceSettingV2EffectiveBooleanVal struct {
	Value bool `json:"value,omitempty"`
}

type DataSourceWorkspaceSettingV2EffectiveCollaborationPlatformConnectivity struct {
	Connectivity string `json:"connectivity"`
}

type DataSourceWorkspaceSettingV2EffectiveIntegerVal struct {
	Value int `json:"value,omitempty"`
}

type DataSourceWorkspaceSettingV2EffectiveOperationalEmailCustomRecipient struct {
	Email string `json:"email,omitempty"`
}

type DataSourceWorkspaceSettingV2EffectivePersonalCompute struct {
	Value string `json:"value,omitempty"`
}

type DataSourceWorkspaceSettingV2EffectiveRestrictWorkspaceAdmins struct {
	DisableGovTagCreation bool   `json:"disable_gov_tag_creation,omitempty"`
	Status                string `json:"status"`
}

type DataSourceWorkspaceSettingV2EffectiveStringVal struct {
	Value string `json:"value,omitempty"`
}

type DataSourceWorkspaceSettingV2IntegerVal struct {
	Value int `json:"value,omitempty"`
}

type DataSourceWorkspaceSettingV2OperationalEmailCustomRecipient struct {
	Email string `json:"email,omitempty"`
}

type DataSourceWorkspaceSettingV2PersonalCompute struct {
	Value string `json:"value,omitempty"`
}

type DataSourceWorkspaceSettingV2ProviderConfig struct {
	WorkspaceId string `json:"workspace_id,omitempty"`
}

type DataSourceWorkspaceSettingV2RestrictWorkspaceAdmins struct {
	DisableGovTagCreation bool   `json:"disable_gov_tag_creation,omitempty"`
	Status                string `json:"status"`
}

type DataSourceWorkspaceSettingV2StringVal struct {
	Value string `json:"value,omitempty"`
}

type DataSourceWorkspaceSettingV2 struct {
	AibiDashboardEmbeddingAccessPolicy             *DataSourceWorkspaceSettingV2AibiDashboardEmbeddingAccessPolicy             `json:"aibi_dashboard_embedding_access_policy,omitempty"`
	AibiDashboardEmbeddingApprovedDomains          *DataSourceWorkspaceSettingV2AibiDashboardEmbeddingApprovedDomains          `json:"aibi_dashboard_embedding_approved_domains,omitempty"`
	AllowedAppsUserApiScopes                       *DataSourceWorkspaceSettingV2AllowedAppsUserApiScopes                       `json:"allowed_apps_user_api_scopes,omitempty"`
	AutomaticClusterUpdateWorkspace                *DataSourceWorkspaceSettingV2AutomaticClusterUpdateWorkspace                `json:"automatic_cluster_update_workspace,omitempty"`
	BooleanVal                                     *DataSourceWorkspaceSettingV2BooleanVal                                     `json:"boolean_val,omitempty"`
	CollaborationPlatformConnectivity              *DataSourceWorkspaceSettingV2CollaborationPlatformConnectivity              `json:"collaboration_platform_connectivity,omitempty"`
	EffectiveAibiDashboardEmbeddingAccessPolicy    *DataSourceWorkspaceSettingV2EffectiveAibiDashboardEmbeddingAccessPolicy    `json:"effective_aibi_dashboard_embedding_access_policy,omitempty"`
	EffectiveAibiDashboardEmbeddingApprovedDomains *DataSourceWorkspaceSettingV2EffectiveAibiDashboardEmbeddingApprovedDomains `json:"effective_aibi_dashboard_embedding_approved_domains,omitempty"`
	EffectiveAllowedAppsUserApiScopes              *DataSourceWorkspaceSettingV2EffectiveAllowedAppsUserApiScopes              `json:"effective_allowed_apps_user_api_scopes,omitempty"`
	EffectiveAutomaticClusterUpdateWorkspace       *DataSourceWorkspaceSettingV2EffectiveAutomaticClusterUpdateWorkspace       `json:"effective_automatic_cluster_update_workspace,omitempty"`
	EffectiveBooleanVal                            *DataSourceWorkspaceSettingV2EffectiveBooleanVal                            `json:"effective_boolean_val,omitempty"`
	EffectiveCollaborationPlatformConnectivity     *DataSourceWorkspaceSettingV2EffectiveCollaborationPlatformConnectivity     `json:"effective_collaboration_platform_connectivity,omitempty"`
	EffectiveIntegerVal                            *DataSourceWorkspaceSettingV2EffectiveIntegerVal                            `json:"effective_integer_val,omitempty"`
	EffectiveOperationalEmailCustomRecipient       *DataSourceWorkspaceSettingV2EffectiveOperationalEmailCustomRecipient       `json:"effective_operational_email_custom_recipient,omitempty"`
	EffectivePersonalCompute                       *DataSourceWorkspaceSettingV2EffectivePersonalCompute                       `json:"effective_personal_compute,omitempty"`
	EffectiveRestrictWorkspaceAdmins               *DataSourceWorkspaceSettingV2EffectiveRestrictWorkspaceAdmins               `json:"effective_restrict_workspace_admins,omitempty"`
	EffectiveStringVal                             *DataSourceWorkspaceSettingV2EffectiveStringVal                             `json:"effective_string_val,omitempty"`
	IntegerVal                                     *DataSourceWorkspaceSettingV2IntegerVal                                     `json:"integer_val,omitempty"`
	Name                                           string                                                                      `json:"name"`
	OperationalEmailCustomRecipient                *DataSourceWorkspaceSettingV2OperationalEmailCustomRecipient                `json:"operational_email_custom_recipient,omitempty"`
	PersonalCompute                                *DataSourceWorkspaceSettingV2PersonalCompute                                `json:"personal_compute,omitempty"`
	ProviderConfig                                 *DataSourceWorkspaceSettingV2ProviderConfig                                 `json:"provider_config,omitempty"`
	RestrictWorkspaceAdmins                        *DataSourceWorkspaceSettingV2RestrictWorkspaceAdmins                        `json:"restrict_workspace_admins,omitempty"`
	StringVal                                      *DataSourceWorkspaceSettingV2StringVal                                      `json:"string_val,omitempty"`
}
