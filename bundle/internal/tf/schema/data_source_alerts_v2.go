// Generated from Databricks Terraform provider schema. DO NOT EDIT.

package schema

type DataSourceAlertsV2ResultsEvaluationNotificationSubscriptions struct {
	DestinationId string `json:"destination_id,omitempty"`
	UserEmail     string `json:"user_email,omitempty"`
}

type DataSourceAlertsV2ResultsEvaluationNotification struct {
	NotifyOnOk       bool                                                           `json:"notify_on_ok,omitempty"`
	RetriggerSeconds int                                                            `json:"retrigger_seconds,omitempty"`
	Subscriptions    []DataSourceAlertsV2ResultsEvaluationNotificationSubscriptions `json:"subscriptions,omitempty"`
}

type DataSourceAlertsV2ResultsEvaluationSource struct {
	Aggregation string `json:"aggregation,omitempty"`
	Display     string `json:"display,omitempty"`
	Name        string `json:"name,omitempty"`
}

type DataSourceAlertsV2ResultsEvaluationThresholdColumn struct {
	Aggregation string `json:"aggregation,omitempty"`
	Display     string `json:"display,omitempty"`
	Name        string `json:"name,omitempty"`
}

type DataSourceAlertsV2ResultsEvaluationThresholdValue struct {
	BoolValue   bool   `json:"bool_value,omitempty"`
	DoubleValue int    `json:"double_value,omitempty"`
	StringValue string `json:"string_value,omitempty"`
}

type DataSourceAlertsV2ResultsEvaluationThreshold struct {
	Column *DataSourceAlertsV2ResultsEvaluationThresholdColumn `json:"column,omitempty"`
	Value  *DataSourceAlertsV2ResultsEvaluationThresholdValue  `json:"value,omitempty"`
}

type DataSourceAlertsV2ResultsEvaluation struct {
	ComparisonOperator string                                           `json:"comparison_operator,omitempty"`
	EmptyResultState   string                                           `json:"empty_result_state,omitempty"`
	LastEvaluatedAt    string                                           `json:"last_evaluated_at,omitempty"`
	Notification       *DataSourceAlertsV2ResultsEvaluationNotification `json:"notification,omitempty"`
	Source             *DataSourceAlertsV2ResultsEvaluationSource       `json:"source,omitempty"`
	State              string                                           `json:"state,omitempty"`
	Threshold          *DataSourceAlertsV2ResultsEvaluationThreshold    `json:"threshold,omitempty"`
}

type DataSourceAlertsV2ResultsSchedule struct {
	PauseStatus        string `json:"pause_status,omitempty"`
	QuartzCronSchedule string `json:"quartz_cron_schedule,omitempty"`
	TimezoneId         string `json:"timezone_id,omitempty"`
}

type DataSourceAlertsV2Results struct {
	CreateTime        string                               `json:"create_time,omitempty"`
	CustomDescription string                               `json:"custom_description,omitempty"`
	CustomSummary     string                               `json:"custom_summary,omitempty"`
	DisplayName       string                               `json:"display_name,omitempty"`
	Evaluation        *DataSourceAlertsV2ResultsEvaluation `json:"evaluation,omitempty"`
	Id                string                               `json:"id,omitempty"`
	LifecycleState    string                               `json:"lifecycle_state,omitempty"`
	OwnerUserName     string                               `json:"owner_user_name,omitempty"`
	ParentPath        string                               `json:"parent_path,omitempty"`
	QueryText         string                               `json:"query_text,omitempty"`
	RunAsUserName     string                               `json:"run_as_user_name,omitempty"`
	Schedule          *DataSourceAlertsV2ResultsSchedule   `json:"schedule,omitempty"`
	UpdateTime        string                               `json:"update_time,omitempty"`
	WarehouseId       string                               `json:"warehouse_id,omitempty"`
}

type DataSourceAlertsV2 struct {
	Results []DataSourceAlertsV2Results `json:"results,omitempty"`
}
