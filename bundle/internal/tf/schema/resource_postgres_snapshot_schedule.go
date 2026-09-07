// Generated from Databricks Terraform provider schema. DO NOT EDIT.

package schema

type ResourcePostgresSnapshotScheduleProviderConfig struct {
	WorkspaceId string `json:"workspace_id,omitempty"`
}

type ResourcePostgresSnapshotScheduleScheduleDailySchedule struct {
	Hour int `json:"hour,omitempty"`
}

type ResourcePostgresSnapshotScheduleScheduleMonthlySchedule struct {
	Day  int `json:"day"`
	Hour int `json:"hour,omitempty"`
}

type ResourcePostgresSnapshotScheduleScheduleWeeklySchedule struct {
	DayOfWeek string `json:"day_of_week"`
	Hour      int    `json:"hour,omitempty"`
}

type ResourcePostgresSnapshotScheduleSchedule struct {
	DailySchedule   *ResourcePostgresSnapshotScheduleScheduleDailySchedule   `json:"daily_schedule,omitempty"`
	MonthlySchedule *ResourcePostgresSnapshotScheduleScheduleMonthlySchedule `json:"monthly_schedule,omitempty"`
	Retention       string                                                   `json:"retention"`
	WeeklySchedule  *ResourcePostgresSnapshotScheduleScheduleWeeklySchedule  `json:"weekly_schedule,omitempty"`
}

type ResourcePostgresSnapshotSchedule struct {
	Name           string                                          `json:"name,omitempty"`
	ProviderConfig *ResourcePostgresSnapshotScheduleProviderConfig `json:"provider_config,omitempty"`
	Schedule       []ResourcePostgresSnapshotScheduleSchedule      `json:"schedule,omitempty"`
}
