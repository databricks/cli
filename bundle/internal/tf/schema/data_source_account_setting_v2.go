// Generated from Databricks Terraform provider schema. DO NOT EDIT.

package schema

type DataSourceAccountSettingV2AibiDashboardEmbeddingAccessPolicy struct {
	AccessPolicyType string `json:"access_policy_type"`
}

type DataSourceAccountSettingV2AibiDashboardEmbeddingApprovedDomains struct {
	ApprovedDomains []string `json:"approved_domains,omitempty"`
}

type DataSourceAccountSettingV2AutomaticClusterUpdateWorkspaceEnablementDetails struct {
	ForcedForComplianceMode           bool `json:"forced_for_compliance_mode,omitempty"`
	UnavailableForDisabledEntitlement bool `json:"unavailable_for_disabled_entitlement,omitempty"`
	UnavailableForNonEnterpriseTier   bool `json:"unavailable_for_non_enterprise_tier,omitempty"`
}

type DataSourceAccountSettingV2AutomaticClusterUpdateWorkspaceMaintenanceWindowWeekDayBasedScheduleWindowStartTime struct {
	Hours   int `json:"hours,omitempty"`
	Minutes int `json:"minutes,omitempty"`
}

type DataSourceAccountSettingV2AutomaticClusterUpdateWorkspaceMaintenanceWindowWeekDayBasedSchedule struct {
	DayOfWeek       string                                                                                                         `json:"day_of_week,omitempty"`
	Frequency       string                                                                                                         `json:"frequency,omitempty"`
	WindowStartTime *DataSourceAccountSettingV2AutomaticClusterUpdateWorkspaceMaintenanceWindowWeekDayBasedScheduleWindowStartTime `json:"window_start_time,omitempty"`
}

type DataSourceAccountSettingV2AutomaticClusterUpdateWorkspaceMaintenanceWindow struct {
	WeekDayBasedSchedule *DataSourceAccountSettingV2AutomaticClusterUpdateWorkspaceMaintenanceWindowWeekDayBasedSchedule `json:"week_day_based_schedule,omitempty"`
}

type DataSourceAccountSettingV2AutomaticClusterUpdateWorkspace struct {
	CanToggle                       bool                                                                        `json:"can_toggle,omitempty"`
	Enabled                         bool                                                                        `json:"enabled,omitempty"`
	EnablementDetails               *DataSourceAccountSettingV2AutomaticClusterUpdateWorkspaceEnablementDetails `json:"enablement_details,omitempty"`
	MaintenanceWindow               *DataSourceAccountSettingV2AutomaticClusterUpdateWorkspaceMaintenanceWindow `json:"maintenance_window,omitempty"`
	RestartEvenIfNoUpdatesAvailable bool                                                                        `json:"restart_even_if_no_updates_available,omitempty"`
}

type DataSourceAccountSettingV2BooleanVal struct {
	Value bool `json:"value,omitempty"`
}

type DataSourceAccountSettingV2DefaultDataSecurityMode struct {
	Status string `json:"status"`
}

type DataSourceAccountSettingV2EffectiveAibiDashboardEmbeddingAccessPolicy struct {
	AccessPolicyType string `json:"access_policy_type"`
}

type DataSourceAccountSettingV2EffectiveAibiDashboardEmbeddingApprovedDomains struct {
	ApprovedDomains []string `json:"approved_domains,omitempty"`
}

type DataSourceAccountSettingV2EffectiveAutomaticClusterUpdateWorkspaceEnablementDetails struct {
	ForcedForComplianceMode           bool `json:"forced_for_compliance_mode,omitempty"`
	UnavailableForDisabledEntitlement bool `json:"unavailable_for_disabled_entitlement,omitempty"`
	UnavailableForNonEnterpriseTier   bool `json:"unavailable_for_non_enterprise_tier,omitempty"`
}

type DataSourceAccountSettingV2EffectiveAutomaticClusterUpdateWorkspaceMaintenanceWindowWeekDayBasedScheduleWindowStartTime struct {
	Hours   int `json:"hours,omitempty"`
	Minutes int `json:"minutes,omitempty"`
}

type DataSourceAccountSettingV2EffectiveAutomaticClusterUpdateWorkspaceMaintenanceWindowWeekDayBasedSchedule struct {
	DayOfWeek       string                                                                                                                  `json:"day_of_week,omitempty"`
	Frequency       string                                                                                                                  `json:"frequency,omitempty"`
	WindowStartTime *DataSourceAccountSettingV2EffectiveAutomaticClusterUpdateWorkspaceMaintenanceWindowWeekDayBasedScheduleWindowStartTime `json:"window_start_time,omitempty"`
}

type DataSourceAccountSettingV2EffectiveAutomaticClusterUpdateWorkspaceMaintenanceWindow struct {
	WeekDayBasedSchedule *DataSourceAccountSettingV2EffectiveAutomaticClusterUpdateWorkspaceMaintenanceWindowWeekDayBasedSchedule `json:"week_day_based_schedule,omitempty"`
}

type DataSourceAccountSettingV2EffectiveAutomaticClusterUpdateWorkspace struct {
	CanToggle                       bool                                                                                 `json:"can_toggle,omitempty"`
	Enabled                         bool                                                                                 `json:"enabled,omitempty"`
	EnablementDetails               *DataSourceAccountSettingV2EffectiveAutomaticClusterUpdateWorkspaceEnablementDetails `json:"enablement_details,omitempty"`
	MaintenanceWindow               *DataSourceAccountSettingV2EffectiveAutomaticClusterUpdateWorkspaceMaintenanceWindow `json:"maintenance_window,omitempty"`
	RestartEvenIfNoUpdatesAvailable bool                                                                                 `json:"restart_even_if_no_updates_available,omitempty"`
}

type DataSourceAccountSettingV2EffectiveBooleanVal struct {
	Value bool `json:"value,omitempty"`
}

type DataSourceAccountSettingV2EffectiveDefaultDataSecurityMode struct {
	Status string `json:"status"`
}

type DataSourceAccountSettingV2EffectiveIntegerVal struct {
	Value int `json:"value,omitempty"`
}

type DataSourceAccountSettingV2EffectivePersonalCompute struct {
	Value string `json:"value,omitempty"`
}

type DataSourceAccountSettingV2EffectiveRestrictWorkspaceAdmins struct {
	Status string `json:"status"`
}

type DataSourceAccountSettingV2EffectiveStringVal struct {
	Value string `json:"value,omitempty"`
}

type DataSourceAccountSettingV2IntegerVal struct {
	Value int `json:"value,omitempty"`
}

type DataSourceAccountSettingV2PersonalCompute struct {
	Value string `json:"value,omitempty"`
}

type DataSourceAccountSettingV2RestrictWorkspaceAdmins struct {
	Status string `json:"status"`
}

type DataSourceAccountSettingV2StringVal struct {
	Value string `json:"value,omitempty"`
}

type DataSourceAccountSettingV2 struct {
	AibiDashboardEmbeddingAccessPolicy             *DataSourceAccountSettingV2AibiDashboardEmbeddingAccessPolicy             `json:"aibi_dashboard_embedding_access_policy,omitempty"`
	AibiDashboardEmbeddingApprovedDomains          *DataSourceAccountSettingV2AibiDashboardEmbeddingApprovedDomains          `json:"aibi_dashboard_embedding_approved_domains,omitempty"`
	AutomaticClusterUpdateWorkspace                *DataSourceAccountSettingV2AutomaticClusterUpdateWorkspace                `json:"automatic_cluster_update_workspace,omitempty"`
	BooleanVal                                     *DataSourceAccountSettingV2BooleanVal                                     `json:"boolean_val,omitempty"`
	DefaultDataSecurityMode                        *DataSourceAccountSettingV2DefaultDataSecurityMode                        `json:"default_data_security_mode,omitempty"`
	EffectiveAibiDashboardEmbeddingAccessPolicy    *DataSourceAccountSettingV2EffectiveAibiDashboardEmbeddingAccessPolicy    `json:"effective_aibi_dashboard_embedding_access_policy,omitempty"`
	EffectiveAibiDashboardEmbeddingApprovedDomains *DataSourceAccountSettingV2EffectiveAibiDashboardEmbeddingApprovedDomains `json:"effective_aibi_dashboard_embedding_approved_domains,omitempty"`
	EffectiveAutomaticClusterUpdateWorkspace       *DataSourceAccountSettingV2EffectiveAutomaticClusterUpdateWorkspace       `json:"effective_automatic_cluster_update_workspace,omitempty"`
	EffectiveBooleanVal                            *DataSourceAccountSettingV2EffectiveBooleanVal                            `json:"effective_boolean_val,omitempty"`
	EffectiveDefaultDataSecurityMode               *DataSourceAccountSettingV2EffectiveDefaultDataSecurityMode               `json:"effective_default_data_security_mode,omitempty"`
	EffectiveIntegerVal                            *DataSourceAccountSettingV2EffectiveIntegerVal                            `json:"effective_integer_val,omitempty"`
	EffectivePersonalCompute                       *DataSourceAccountSettingV2EffectivePersonalCompute                       `json:"effective_personal_compute,omitempty"`
	EffectiveRestrictWorkspaceAdmins               *DataSourceAccountSettingV2EffectiveRestrictWorkspaceAdmins               `json:"effective_restrict_workspace_admins,omitempty"`
	EffectiveStringVal                             *DataSourceAccountSettingV2EffectiveStringVal                             `json:"effective_string_val,omitempty"`
	IntegerVal                                     *DataSourceAccountSettingV2IntegerVal                                     `json:"integer_val,omitempty"`
	Name                                           string                                                                    `json:"name,omitempty"`
	PersonalCompute                                *DataSourceAccountSettingV2PersonalCompute                                `json:"personal_compute,omitempty"`
	RestrictWorkspaceAdmins                        *DataSourceAccountSettingV2RestrictWorkspaceAdmins                        `json:"restrict_workspace_admins,omitempty"`
	StringVal                                      *DataSourceAccountSettingV2StringVal                                      `json:"string_val,omitempty"`
}
