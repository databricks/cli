// Generated from Databricks Terraform provider schema. DO NOT EDIT.

package schema

type ResourceAutomaticClusterUpdateWorkspaceSettingAutomaticClusterUpdateWorkspaceEnablementDetails struct {
	ForcedForComplianceMode           bool `json:"forced_for_compliance_mode,omitempty"`
	UnavailableForDisabledEntitlement bool `json:"unavailable_for_disabled_entitlement,omitempty"`
	UnavailableForNonEnterpriseTier   bool `json:"unavailable_for_non_enterprise_tier,omitempty"`
}

type ResourceAutomaticClusterUpdateWorkspaceSettingAutomaticClusterUpdateWorkspaceMaintenanceWindowWeekDayBasedScheduleWindowStartTime struct {
	Hours   int `json:"hours,omitempty"`
	Minutes int `json:"minutes,omitempty"`
}

type ResourceAutomaticClusterUpdateWorkspaceSettingAutomaticClusterUpdateWorkspaceMaintenanceWindowWeekDayBasedSchedule struct {
	DayOfWeek       string                                                                                                                             `json:"day_of_week,omitempty"`
	Frequency       string                                                                                                                             `json:"frequency,omitempty"`
	WindowStartTime *ResourceAutomaticClusterUpdateWorkspaceSettingAutomaticClusterUpdateWorkspaceMaintenanceWindowWeekDayBasedScheduleWindowStartTime `json:"window_start_time,omitempty"`
}

type ResourceAutomaticClusterUpdateWorkspaceSettingAutomaticClusterUpdateWorkspaceMaintenanceWindow struct {
	WeekDayBasedSchedule *ResourceAutomaticClusterUpdateWorkspaceSettingAutomaticClusterUpdateWorkspaceMaintenanceWindowWeekDayBasedSchedule `json:"week_day_based_schedule,omitempty"`
}

type ResourceAutomaticClusterUpdateWorkspaceSettingAutomaticClusterUpdateWorkspace struct {
	CanToggle                       bool                                                                                            `json:"can_toggle,omitempty"`
	Enabled                         bool                                                                                            `json:"enabled,omitempty"`
	RestartEvenIfNoUpdatesAvailable bool                                                                                            `json:"restart_even_if_no_updates_available,omitempty"`
	EnablementDetails               *ResourceAutomaticClusterUpdateWorkspaceSettingAutomaticClusterUpdateWorkspaceEnablementDetails `json:"enablement_details,omitempty"`
	MaintenanceWindow               *ResourceAutomaticClusterUpdateWorkspaceSettingAutomaticClusterUpdateWorkspaceMaintenanceWindow `json:"maintenance_window,omitempty"`
}

type ResourceAutomaticClusterUpdateWorkspaceSetting struct {
	Etag                            string                                                                         `json:"etag,omitempty"`
	Id                              string                                                                         `json:"id,omitempty"`
	SettingName                     string                                                                         `json:"setting_name,omitempty"`
	AutomaticClusterUpdateWorkspace *ResourceAutomaticClusterUpdateWorkspaceSettingAutomaticClusterUpdateWorkspace `json:"automatic_cluster_update_workspace,omitempty"`
}
