// Generated from Databricks Terraform provider schema. DO NOT EDIT.

package schema

type DataSourcePostgresSnapshotScheduleProviderConfig struct {
	WorkspaceId string `json:"workspace_id,omitempty"`
}

type DataSourcePostgresSnapshotScheduleScheduleDailySchedule struct {
	Hour int `json:"hour,omitempty"`
}

type DataSourcePostgresSnapshotScheduleScheduleMonthlySchedule struct {
	Day  int `json:"day"`
	Hour int `json:"hour,omitempty"`
}

type DataSourcePostgresSnapshotScheduleScheduleWeeklySchedule struct {
	DayOfWeek string `json:"day_of_week"`
	Hour      int    `json:"hour,omitempty"`
}

type DataSourcePostgresSnapshotScheduleSchedule struct {
	DailySchedule   *DataSourcePostgresSnapshotScheduleScheduleDailySchedule   `json:"daily_schedule,omitempty"`
	MonthlySchedule *DataSourcePostgresSnapshotScheduleScheduleMonthlySchedule `json:"monthly_schedule,omitempty"`
	Retention       string                                                     `json:"retention"`
	WeeklySchedule  *DataSourcePostgresSnapshotScheduleScheduleWeeklySchedule  `json:"weekly_schedule,omitempty"`
}

type DataSourcePostgresSnapshotSchedule struct {
	Name           string                                            `json:"name"`
	ProviderConfig *DataSourcePostgresSnapshotScheduleProviderConfig `json:"provider_config,omitempty"`
	Schedule       []DataSourcePostgresSnapshotScheduleSchedule      `json:"schedule,omitempty"`
}
