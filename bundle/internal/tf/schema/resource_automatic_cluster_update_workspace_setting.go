// Generated from Databricks Terraform provider schema. DO NOT EDIT.

package schema

type ResourceAutomaticClusterUpdateWorkspaceSettingAutomaticClusterUpdateWorkspaceMaintenanceWindowWeekDayBasedScheduleWindowStartTime struct {
	Hours   int `json:"hours"`
	Minutes int `json:"minutes"`
}

type ResourceAutomaticClusterUpdateWorkspaceSettingAutomaticClusterUpdateWorkspaceMaintenanceWindowWeekDayBasedSchedule struct {
	DayOfWeek       string                                                                                                                             `json:"day_of_week"`
	Frequency       string                                                                                                                             `json:"frequency"`
	WindowStartTime *ResourceAutomaticClusterUpdateWorkspaceSettingAutomaticClusterUpdateWorkspaceMaintenanceWindowWeekDayBasedScheduleWindowStartTime `json:"window_start_time,omitempty"`
}

type ResourceAutomaticClusterUpdateWorkspaceSettingAutomaticClusterUpdateWorkspaceMaintenanceWindow struct {
	WeekDayBasedSchedule *ResourceAutomaticClusterUpdateWorkspaceSettingAutomaticClusterUpdateWorkspaceMaintenanceWindowWeekDayBasedSchedule `json:"week_day_based_schedule,omitempty"`
}

type ResourceAutomaticClusterUpdateWorkspaceSettingAutomaticClusterUpdateWorkspace struct {
	CanToggle                       bool                                                                                            `json:"can_toggle,omitempty"`
	Enabled                         bool                                                                                            `json:"enabled"`
	EnablementDetails               []any                                                                                           `json:"enablement_details,omitempty"`
	RestartEvenIfNoUpdatesAvailable bool                                                                                            `json:"restart_even_if_no_updates_available,omitempty"`
	MaintenanceWindow               *ResourceAutomaticClusterUpdateWorkspaceSettingAutomaticClusterUpdateWorkspaceMaintenanceWindow `json:"maintenance_window,omitempty"`
}

type ResourceAutomaticClusterUpdateWorkspaceSetting struct {
	Etag                            string                                                                         `json:"etag,omitempty"`
	Id                              string                                                                         `json:"id,omitempty"`
	SettingName                     string                                                                         `json:"setting_name,omitempty"`
	AutomaticClusterUpdateWorkspace *ResourceAutomaticClusterUpdateWorkspaceSettingAutomaticClusterUpdateWorkspace `json:"automatic_cluster_update_workspace,omitempty"`
}
