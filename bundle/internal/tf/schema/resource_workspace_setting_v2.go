// Generated from Databricks Terraform provider schema. DO NOT EDIT.

package schema

type ResourceWorkspaceSettingV2AibiDashboardEmbeddingAccessPolicy struct {
	AccessPolicyType string `json:"access_policy_type"`
}

type ResourceWorkspaceSettingV2AibiDashboardEmbeddingApprovedDomains struct {
	ApprovedDomains []string `json:"approved_domains,omitempty"`
}

type ResourceWorkspaceSettingV2AllowedAppsUserApiScopes struct {
	AllowedScopes []string `json:"allowed_scopes,omitempty"`
}

type ResourceWorkspaceSettingV2AutomaticClusterUpdateWorkspaceEnablementDetails struct {
	ForcedForComplianceMode           bool `json:"forced_for_compliance_mode,omitempty"`
	UnavailableForDisabledEntitlement bool `json:"unavailable_for_disabled_entitlement,omitempty"`
	UnavailableForNonEnterpriseTier   bool `json:"unavailable_for_non_enterprise_tier,omitempty"`
}

type ResourceWorkspaceSettingV2AutomaticClusterUpdateWorkspaceMaintenanceWindowWeekDayBasedScheduleWindowStartTime struct {
	Hours   int `json:"hours,omitempty"`
	Minutes int `json:"minutes,omitempty"`
}

type ResourceWorkspaceSettingV2AutomaticClusterUpdateWorkspaceMaintenanceWindowWeekDayBasedSchedule struct {
	DayOfWeek       string                                                                                                         `json:"day_of_week,omitempty"`
	Frequency       string                                                                                                         `json:"frequency,omitempty"`
	WindowStartTime *ResourceWorkspaceSettingV2AutomaticClusterUpdateWorkspaceMaintenanceWindowWeekDayBasedScheduleWindowStartTime `json:"window_start_time,omitempty"`
}

type ResourceWorkspaceSettingV2AutomaticClusterUpdateWorkspaceMaintenanceWindow struct {
	WeekDayBasedSchedule *ResourceWorkspaceSettingV2AutomaticClusterUpdateWorkspaceMaintenanceWindowWeekDayBasedSchedule `json:"week_day_based_schedule,omitempty"`
}

type ResourceWorkspaceSettingV2AutomaticClusterUpdateWorkspace struct {
	CanToggle                       bool                                                                        `json:"can_toggle,omitempty"`
	Enabled                         bool                                                                        `json:"enabled,omitempty"`
	EnablementDetails               *ResourceWorkspaceSettingV2AutomaticClusterUpdateWorkspaceEnablementDetails `json:"enablement_details,omitempty"`
	MaintenanceWindow               *ResourceWorkspaceSettingV2AutomaticClusterUpdateWorkspaceMaintenanceWindow `json:"maintenance_window,omitempty"`
	RestartEvenIfNoUpdatesAvailable bool                                                                        `json:"restart_even_if_no_updates_available,omitempty"`
}

type ResourceWorkspaceSettingV2BooleanVal struct {
	Value bool `json:"value,omitempty"`
}

type ResourceWorkspaceSettingV2CollaborationPlatformConnectivity struct {
	Connectivity string `json:"connectivity"`
}

type ResourceWorkspaceSettingV2EffectiveAibiDashboardEmbeddingAccessPolicy struct {
	AccessPolicyType string `json:"access_policy_type"`
}

type ResourceWorkspaceSettingV2EffectiveAibiDashboardEmbeddingApprovedDomains struct {
	ApprovedDomains []string `json:"approved_domains,omitempty"`
}

type ResourceWorkspaceSettingV2EffectiveAllowedAppsUserApiScopes struct {
	AllowedScopes []string `json:"allowed_scopes,omitempty"`
}

type ResourceWorkspaceSettingV2EffectiveAutomaticClusterUpdateWorkspaceEnablementDetails struct {
	ForcedForComplianceMode           bool `json:"forced_for_compliance_mode,omitempty"`
	UnavailableForDisabledEntitlement bool `json:"unavailable_for_disabled_entitlement,omitempty"`
	UnavailableForNonEnterpriseTier   bool `json:"unavailable_for_non_enterprise_tier,omitempty"`
}

type ResourceWorkspaceSettingV2EffectiveAutomaticClusterUpdateWorkspaceMaintenanceWindowWeekDayBasedScheduleWindowStartTime struct {
	Hours   int `json:"hours,omitempty"`
	Minutes int `json:"minutes,omitempty"`
}

type ResourceWorkspaceSettingV2EffectiveAutomaticClusterUpdateWorkspaceMaintenanceWindowWeekDayBasedSchedule struct {
	DayOfWeek       string                                                                                                                  `json:"day_of_week,omitempty"`
	Frequency       string                                                                                                                  `json:"frequency,omitempty"`
	WindowStartTime *ResourceWorkspaceSettingV2EffectiveAutomaticClusterUpdateWorkspaceMaintenanceWindowWeekDayBasedScheduleWindowStartTime `json:"window_start_time,omitempty"`
}

type ResourceWorkspaceSettingV2EffectiveAutomaticClusterUpdateWorkspaceMaintenanceWindow struct {
	WeekDayBasedSchedule *ResourceWorkspaceSettingV2EffectiveAutomaticClusterUpdateWorkspaceMaintenanceWindowWeekDayBasedSchedule `json:"week_day_based_schedule,omitempty"`
}

type ResourceWorkspaceSettingV2EffectiveAutomaticClusterUpdateWorkspace struct {
	CanToggle                       bool                                                                                 `json:"can_toggle,omitempty"`
	Enabled                         bool                                                                                 `json:"enabled,omitempty"`
	EnablementDetails               *ResourceWorkspaceSettingV2EffectiveAutomaticClusterUpdateWorkspaceEnablementDetails `json:"enablement_details,omitempty"`
	MaintenanceWindow               *ResourceWorkspaceSettingV2EffectiveAutomaticClusterUpdateWorkspaceMaintenanceWindow `json:"maintenance_window,omitempty"`
	RestartEvenIfNoUpdatesAvailable bool                                                                                 `json:"restart_even_if_no_updates_available,omitempty"`
}

type ResourceWorkspaceSettingV2EffectiveBooleanVal struct {
	Value bool `json:"value,omitempty"`
}

type ResourceWorkspaceSettingV2EffectiveCollaborationPlatformConnectivity struct {
	Connectivity string `json:"connectivity"`
}

type ResourceWorkspaceSettingV2EffectiveIntegerVal struct {
	Value int `json:"value,omitempty"`
}

type ResourceWorkspaceSettingV2EffectiveOperationalEmailCustomRecipient struct {
	Email string `json:"email,omitempty"`
}

type ResourceWorkspaceSettingV2EffectivePersonalCompute struct {
	Value string `json:"value,omitempty"`
}

type ResourceWorkspaceSettingV2EffectiveRestrictWorkspaceAdmins struct {
	DisableGovTagCreation bool   `json:"disable_gov_tag_creation,omitempty"`
	Status                string `json:"status"`
}

type ResourceWorkspaceSettingV2EffectiveStringVal struct {
	Value string `json:"value,omitempty"`
}

type ResourceWorkspaceSettingV2IntegerVal struct {
	Value int `json:"value,omitempty"`
}

type ResourceWorkspaceSettingV2OperationalEmailCustomRecipient struct {
	Email string `json:"email,omitempty"`
}

type ResourceWorkspaceSettingV2PersonalCompute struct {
	Value string `json:"value,omitempty"`
}

type ResourceWorkspaceSettingV2ProviderConfig struct {
	WorkspaceId string `json:"workspace_id,omitempty"`
}

type ResourceWorkspaceSettingV2RestrictWorkspaceAdmins struct {
	DisableGovTagCreation bool   `json:"disable_gov_tag_creation,omitempty"`
	Status                string `json:"status"`
}

type ResourceWorkspaceSettingV2StringVal struct {
	Value string `json:"value,omitempty"`
}

type ResourceWorkspaceSettingV2 struct {
	AibiDashboardEmbeddingAccessPolicy             *ResourceWorkspaceSettingV2AibiDashboardEmbeddingAccessPolicy             `json:"aibi_dashboard_embedding_access_policy,omitempty"`
	AibiDashboardEmbeddingApprovedDomains          *ResourceWorkspaceSettingV2AibiDashboardEmbeddingApprovedDomains          `json:"aibi_dashboard_embedding_approved_domains,omitempty"`
	AllowedAppsUserApiScopes                       *ResourceWorkspaceSettingV2AllowedAppsUserApiScopes                       `json:"allowed_apps_user_api_scopes,omitempty"`
	AutomaticClusterUpdateWorkspace                *ResourceWorkspaceSettingV2AutomaticClusterUpdateWorkspace                `json:"automatic_cluster_update_workspace,omitempty"`
	BooleanVal                                     *ResourceWorkspaceSettingV2BooleanVal                                     `json:"boolean_val,omitempty"`
	CollaborationPlatformConnectivity              *ResourceWorkspaceSettingV2CollaborationPlatformConnectivity              `json:"collaboration_platform_connectivity,omitempty"`
	EffectiveAibiDashboardEmbeddingAccessPolicy    *ResourceWorkspaceSettingV2EffectiveAibiDashboardEmbeddingAccessPolicy    `json:"effective_aibi_dashboard_embedding_access_policy,omitempty"`
	EffectiveAibiDashboardEmbeddingApprovedDomains *ResourceWorkspaceSettingV2EffectiveAibiDashboardEmbeddingApprovedDomains `json:"effective_aibi_dashboard_embedding_approved_domains,omitempty"`
	EffectiveAllowedAppsUserApiScopes              *ResourceWorkspaceSettingV2EffectiveAllowedAppsUserApiScopes              `json:"effective_allowed_apps_user_api_scopes,omitempty"`
	EffectiveAutomaticClusterUpdateWorkspace       *ResourceWorkspaceSettingV2EffectiveAutomaticClusterUpdateWorkspace       `json:"effective_automatic_cluster_update_workspace,omitempty"`
	EffectiveBooleanVal                            *ResourceWorkspaceSettingV2EffectiveBooleanVal                            `json:"effective_boolean_val,omitempty"`
	EffectiveCollaborationPlatformConnectivity     *ResourceWorkspaceSettingV2EffectiveCollaborationPlatformConnectivity     `json:"effective_collaboration_platform_connectivity,omitempty"`
	EffectiveIntegerVal                            *ResourceWorkspaceSettingV2EffectiveIntegerVal                            `json:"effective_integer_val,omitempty"`
	EffectiveOperationalEmailCustomRecipient       *ResourceWorkspaceSettingV2EffectiveOperationalEmailCustomRecipient       `json:"effective_operational_email_custom_recipient,omitempty"`
	EffectivePersonalCompute                       *ResourceWorkspaceSettingV2EffectivePersonalCompute                       `json:"effective_personal_compute,omitempty"`
	EffectiveRestrictWorkspaceAdmins               *ResourceWorkspaceSettingV2EffectiveRestrictWorkspaceAdmins               `json:"effective_restrict_workspace_admins,omitempty"`
	EffectiveStringVal                             *ResourceWorkspaceSettingV2EffectiveStringVal                             `json:"effective_string_val,omitempty"`
	IntegerVal                                     *ResourceWorkspaceSettingV2IntegerVal                                     `json:"integer_val,omitempty"`
	Name                                           string                                                                    `json:"name,omitempty"`
	OperationalEmailCustomRecipient                *ResourceWorkspaceSettingV2OperationalEmailCustomRecipient                `json:"operational_email_custom_recipient,omitempty"`
	PersonalCompute                                *ResourceWorkspaceSettingV2PersonalCompute                                `json:"personal_compute,omitempty"`
	ProviderConfig                                 *ResourceWorkspaceSettingV2ProviderConfig                                 `json:"provider_config,omitempty"`
	RestrictWorkspaceAdmins                        *ResourceWorkspaceSettingV2RestrictWorkspaceAdmins                        `json:"restrict_workspace_admins,omitempty"`
	StringVal                                      *ResourceWorkspaceSettingV2StringVal                                      `json:"string_val,omitempty"`
}
