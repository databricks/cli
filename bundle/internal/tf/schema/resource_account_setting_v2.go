// Generated from Databricks Terraform provider schema. DO NOT EDIT.

package schema

type ResourceAccountSettingV2AibiDashboardEmbeddingAccessPolicy struct {
	AccessPolicyType string `json:"access_policy_type"`
}

type ResourceAccountSettingV2AibiDashboardEmbeddingApprovedDomains struct {
	ApprovedDomains []string `json:"approved_domains,omitempty"`
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

type ResourceAccountSettingV2EffectiveAibiDashboardEmbeddingAccessPolicy struct {
	AccessPolicyType string `json:"access_policy_type"`
}

type ResourceAccountSettingV2EffectiveAibiDashboardEmbeddingApprovedDomains struct {
	ApprovedDomains []string `json:"approved_domains,omitempty"`
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

type ResourceAccountSettingV2EffectiveIntegerVal struct {
	Value int `json:"value,omitempty"`
}

type ResourceAccountSettingV2EffectivePersonalCompute struct {
	Value string `json:"value,omitempty"`
}

type ResourceAccountSettingV2EffectiveRestrictWorkspaceAdmins struct {
	Status string `json:"status"`
}

type ResourceAccountSettingV2EffectiveStringVal struct {
	Value string `json:"value,omitempty"`
}

type ResourceAccountSettingV2IntegerVal struct {
	Value int `json:"value,omitempty"`
}

type ResourceAccountSettingV2PersonalCompute struct {
	Value string `json:"value,omitempty"`
}

type ResourceAccountSettingV2RestrictWorkspaceAdmins struct {
	Status string `json:"status"`
}

type ResourceAccountSettingV2StringVal struct {
	Value string `json:"value,omitempty"`
}

type ResourceAccountSettingV2 struct {
	AibiDashboardEmbeddingAccessPolicy             *ResourceAccountSettingV2AibiDashboardEmbeddingAccessPolicy             `json:"aibi_dashboard_embedding_access_policy,omitempty"`
	AibiDashboardEmbeddingApprovedDomains          *ResourceAccountSettingV2AibiDashboardEmbeddingApprovedDomains          `json:"aibi_dashboard_embedding_approved_domains,omitempty"`
	AutomaticClusterUpdateWorkspace                *ResourceAccountSettingV2AutomaticClusterUpdateWorkspace                `json:"automatic_cluster_update_workspace,omitempty"`
	BooleanVal                                     *ResourceAccountSettingV2BooleanVal                                     `json:"boolean_val,omitempty"`
	EffectiveAibiDashboardEmbeddingAccessPolicy    *ResourceAccountSettingV2EffectiveAibiDashboardEmbeddingAccessPolicy    `json:"effective_aibi_dashboard_embedding_access_policy,omitempty"`
	EffectiveAibiDashboardEmbeddingApprovedDomains *ResourceAccountSettingV2EffectiveAibiDashboardEmbeddingApprovedDomains `json:"effective_aibi_dashboard_embedding_approved_domains,omitempty"`
	EffectiveAutomaticClusterUpdateWorkspace       *ResourceAccountSettingV2EffectiveAutomaticClusterUpdateWorkspace       `json:"effective_automatic_cluster_update_workspace,omitempty"`
	EffectiveBooleanVal                            *ResourceAccountSettingV2EffectiveBooleanVal                            `json:"effective_boolean_val,omitempty"`
	EffectiveIntegerVal                            *ResourceAccountSettingV2EffectiveIntegerVal                            `json:"effective_integer_val,omitempty"`
	EffectivePersonalCompute                       *ResourceAccountSettingV2EffectivePersonalCompute                       `json:"effective_personal_compute,omitempty"`
	EffectiveRestrictWorkspaceAdmins               *ResourceAccountSettingV2EffectiveRestrictWorkspaceAdmins               `json:"effective_restrict_workspace_admins,omitempty"`
	EffectiveStringVal                             *ResourceAccountSettingV2EffectiveStringVal                             `json:"effective_string_val,omitempty"`
	IntegerVal                                     *ResourceAccountSettingV2IntegerVal                                     `json:"integer_val,omitempty"`
	Name                                           string                                                                  `json:"name,omitempty"`
	PersonalCompute                                *ResourceAccountSettingV2PersonalCompute                                `json:"personal_compute,omitempty"`
	RestrictWorkspaceAdmins                        *ResourceAccountSettingV2RestrictWorkspaceAdmins                        `json:"restrict_workspace_admins,omitempty"`
	StringVal                                      *ResourceAccountSettingV2StringVal                                      `json:"string_val,omitempty"`
}
