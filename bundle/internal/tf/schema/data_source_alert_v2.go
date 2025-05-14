// Generated from Databricks Terraform provider schema. DO NOT EDIT.

package schema

type DataSourceAlertV2EvaluationNotificationSubscriptions struct {
	DestinationId string `json:"destination_id,omitempty"`
	UserEmail     string `json:"user_email,omitempty"`
}

type DataSourceAlertV2EvaluationNotification struct {
	NotifyOnOk       bool                                                   `json:"notify_on_ok,omitempty"`
	RetriggerSeconds int                                                    `json:"retrigger_seconds,omitempty"`
	Subscriptions    []DataSourceAlertV2EvaluationNotificationSubscriptions `json:"subscriptions,omitempty"`
}

type DataSourceAlertV2EvaluationSource struct {
	Aggregation string `json:"aggregation,omitempty"`
	Display     string `json:"display,omitempty"`
	Name        string `json:"name,omitempty"`
}

type DataSourceAlertV2EvaluationThresholdColumn struct {
	Aggregation string `json:"aggregation,omitempty"`
	Display     string `json:"display,omitempty"`
	Name        string `json:"name,omitempty"`
}

type DataSourceAlertV2EvaluationThresholdValue struct {
	BoolValue   bool   `json:"bool_value,omitempty"`
	DoubleValue int    `json:"double_value,omitempty"`
	StringValue string `json:"string_value,omitempty"`
}

type DataSourceAlertV2EvaluationThreshold struct {
	Column *DataSourceAlertV2EvaluationThresholdColumn `json:"column,omitempty"`
	Value  *DataSourceAlertV2EvaluationThresholdValue  `json:"value,omitempty"`
}

type DataSourceAlertV2Evaluation struct {
	ComparisonOperator string                                   `json:"comparison_operator,omitempty"`
	EmptyResultState   string                                   `json:"empty_result_state,omitempty"`
	LastEvaluatedAt    string                                   `json:"last_evaluated_at,omitempty"`
	Notification       *DataSourceAlertV2EvaluationNotification `json:"notification,omitempty"`
	Source             *DataSourceAlertV2EvaluationSource       `json:"source,omitempty"`
	State              string                                   `json:"state,omitempty"`
	Threshold          *DataSourceAlertV2EvaluationThreshold    `json:"threshold,omitempty"`
}

type DataSourceAlertV2Schedule struct {
	PauseStatus        string `json:"pause_status,omitempty"`
	QuartzCronSchedule string `json:"quartz_cron_schedule,omitempty"`
	TimezoneId         string `json:"timezone_id,omitempty"`
}

type DataSourceAlertV2 struct {
	CreateTime        string                       `json:"create_time,omitempty"`
	CustomDescription string                       `json:"custom_description,omitempty"`
	CustomSummary     string                       `json:"custom_summary,omitempty"`
	DisplayName       string                       `json:"display_name,omitempty"`
	Evaluation        *DataSourceAlertV2Evaluation `json:"evaluation,omitempty"`
	Id                string                       `json:"id,omitempty"`
	LifecycleState    string                       `json:"lifecycle_state,omitempty"`
	OwnerUserName     string                       `json:"owner_user_name,omitempty"`
	ParentPath        string                       `json:"parent_path,omitempty"`
	QueryText         string                       `json:"query_text,omitempty"`
	RunAsUserName     string                       `json:"run_as_user_name,omitempty"`
	Schedule          *DataSourceAlertV2Schedule   `json:"schedule,omitempty"`
	UpdateTime        string                       `json:"update_time,omitempty"`
	WarehouseId       string                       `json:"warehouse_id,omitempty"`
}
