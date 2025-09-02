// Generated from Databricks Terraform provider schema. DO NOT EDIT.

package schema

type ResourceWorkspaceSettingV2AibiDashboardEmbeddingAccessPolicy struct {
	AccessPolicyType string `json:"access_policy_type"`
}

type ResourceWorkspaceSettingV2AibiDashboardEmbeddingApprovedDomains struct {
	ApprovedDomains []string `json:"approved_domains,omitempty"`
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

type ResourceWorkspaceSettingV2DefaultDataSecurityMode struct {
	Status string `json:"status"`
}

type ResourceWorkspaceSettingV2EffectiveAibiDashboardEmbeddingAccessPolicy struct {
	AccessPolicyType string `json:"access_policy_type"`
}

type ResourceWorkspaceSettingV2EffectiveAibiDashboardEmbeddingApprovedDomains struct {
	ApprovedDomains []string `json:"approved_domains,omitempty"`
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

type ResourceWorkspaceSettingV2EffectiveDefaultDataSecurityMode struct {
	Status string `json:"status"`
}

type ResourceWorkspaceSettingV2EffectiveIntegerVal struct {
	Value int `json:"value,omitempty"`
}

type ResourceWorkspaceSettingV2EffectivePersonalCompute struct {
	Value string `json:"value,omitempty"`
}

type ResourceWorkspaceSettingV2EffectiveRestrictWorkspaceAdmins struct {
	Status string `json:"status"`
}

type ResourceWorkspaceSettingV2EffectiveStringVal struct {
	Value string `json:"value,omitempty"`
}

type ResourceWorkspaceSettingV2IntegerVal struct {
	Value int `json:"value,omitempty"`
}

type ResourceWorkspaceSettingV2PersonalCompute struct {
	Value string `json:"value,omitempty"`
}

type ResourceWorkspaceSettingV2RestrictWorkspaceAdmins struct {
	Status string `json:"status"`
}

type ResourceWorkspaceSettingV2StringVal struct {
	Value string `json:"value,omitempty"`
}

type ResourceWorkspaceSettingV2 struct {
	AibiDashboardEmbeddingAccessPolicy             *ResourceWorkspaceSettingV2AibiDashboardEmbeddingAccessPolicy             `json:"aibi_dashboard_embedding_access_policy,omitempty"`
	AibiDashboardEmbeddingApprovedDomains          *ResourceWorkspaceSettingV2AibiDashboardEmbeddingApprovedDomains          `json:"aibi_dashboard_embedding_approved_domains,omitempty"`
	AutomaticClusterUpdateWorkspace                *ResourceWorkspaceSettingV2AutomaticClusterUpdateWorkspace                `json:"automatic_cluster_update_workspace,omitempty"`
	BooleanVal                                     *ResourceWorkspaceSettingV2BooleanVal                                     `json:"boolean_val,omitempty"`
	DefaultDataSecurityMode                        *ResourceWorkspaceSettingV2DefaultDataSecurityMode                        `json:"default_data_security_mode,omitempty"`
	EffectiveAibiDashboardEmbeddingAccessPolicy    *ResourceWorkspaceSettingV2EffectiveAibiDashboardEmbeddingAccessPolicy    `json:"effective_aibi_dashboard_embedding_access_policy,omitempty"`
	EffectiveAibiDashboardEmbeddingApprovedDomains *ResourceWorkspaceSettingV2EffectiveAibiDashboardEmbeddingApprovedDomains `json:"effective_aibi_dashboard_embedding_approved_domains,omitempty"`
	EffectiveAutomaticClusterUpdateWorkspace       *ResourceWorkspaceSettingV2EffectiveAutomaticClusterUpdateWorkspace       `json:"effective_automatic_cluster_update_workspace,omitempty"`
	EffectiveBooleanVal                            *ResourceWorkspaceSettingV2EffectiveBooleanVal                            `json:"effective_boolean_val,omitempty"`
	EffectiveDefaultDataSecurityMode               *ResourceWorkspaceSettingV2EffectiveDefaultDataSecurityMode               `json:"effective_default_data_security_mode,omitempty"`
	EffectiveIntegerVal                            *ResourceWorkspaceSettingV2EffectiveIntegerVal                            `json:"effective_integer_val,omitempty"`
	EffectivePersonalCompute                       *ResourceWorkspaceSettingV2EffectivePersonalCompute                       `json:"effective_personal_compute,omitempty"`
	EffectiveRestrictWorkspaceAdmins               *ResourceWorkspaceSettingV2EffectiveRestrictWorkspaceAdmins               `json:"effective_restrict_workspace_admins,omitempty"`
	EffectiveStringVal                             *ResourceWorkspaceSettingV2EffectiveStringVal                             `json:"effective_string_val,omitempty"`
	IntegerVal                                     *ResourceWorkspaceSettingV2IntegerVal                                     `json:"integer_val,omitempty"`
	Name                                           string                                                                    `json:"name,omitempty"`
	PersonalCompute                                *ResourceWorkspaceSettingV2PersonalCompute                                `json:"personal_compute,omitempty"`
	RestrictWorkspaceAdmins                        *ResourceWorkspaceSettingV2RestrictWorkspaceAdmins                        `json:"restrict_workspace_admins,omitempty"`
	StringVal                                      *ResourceWorkspaceSettingV2StringVal                                      `json:"string_val,omitempty"`
	WorkspaceId                                    string                                                                    `json:"workspace_id,omitempty"`
}
