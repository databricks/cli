// Generated from Databricks Terraform provider schema. DO NOT EDIT.

package schema

type DataSourceAlertsV2AlertsEffectiveRunAs struct {
	ServicePrincipalName string `json:"service_principal_name,omitempty"`
	UserName             string `json:"user_name,omitempty"`
}

type DataSourceAlertsV2AlertsEvaluationNotificationSubscriptions struct {
	DestinationId string `json:"destination_id,omitempty"`
	UserEmail     string `json:"user_email,omitempty"`
}

type DataSourceAlertsV2AlertsEvaluationNotification struct {
	NotifyOnOk       bool                                                          `json:"notify_on_ok,omitempty"`
	RetriggerSeconds int                                                           `json:"retrigger_seconds,omitempty"`
	Subscriptions    []DataSourceAlertsV2AlertsEvaluationNotificationSubscriptions `json:"subscriptions,omitempty"`
}

type DataSourceAlertsV2AlertsEvaluationSource struct {
	Aggregation string `json:"aggregation,omitempty"`
	Display     string `json:"display,omitempty"`
	Name        string `json:"name,omitempty"`
}

type DataSourceAlertsV2AlertsEvaluationThresholdColumn struct {
	Aggregation string `json:"aggregation,omitempty"`
	Display     string `json:"display,omitempty"`
	Name        string `json:"name,omitempty"`
}

type DataSourceAlertsV2AlertsEvaluationThresholdValue struct {
	BoolValue   bool   `json:"bool_value,omitempty"`
	DoubleValue int    `json:"double_value,omitempty"`
	StringValue string `json:"string_value,omitempty"`
}

type DataSourceAlertsV2AlertsEvaluationThreshold struct {
	Column *DataSourceAlertsV2AlertsEvaluationThresholdColumn `json:"column,omitempty"`
	Value  *DataSourceAlertsV2AlertsEvaluationThresholdValue  `json:"value,omitempty"`
}

type DataSourceAlertsV2AlertsEvaluation struct {
	ComparisonOperator string                                          `json:"comparison_operator,omitempty"`
	EmptyResultState   string                                          `json:"empty_result_state,omitempty"`
	LastEvaluatedAt    string                                          `json:"last_evaluated_at,omitempty"`
	Notification       *DataSourceAlertsV2AlertsEvaluationNotification `json:"notification,omitempty"`
	Source             *DataSourceAlertsV2AlertsEvaluationSource       `json:"source,omitempty"`
	State              string                                          `json:"state,omitempty"`
	Threshold          *DataSourceAlertsV2AlertsEvaluationThreshold    `json:"threshold,omitempty"`
}

type DataSourceAlertsV2AlertsRunAs struct {
	ServicePrincipalName string `json:"service_principal_name,omitempty"`
	UserName             string `json:"user_name,omitempty"`
}

type DataSourceAlertsV2AlertsSchedule struct {
	PauseStatus        string `json:"pause_status,omitempty"`
	QuartzCronSchedule string `json:"quartz_cron_schedule,omitempty"`
	TimezoneId         string `json:"timezone_id,omitempty"`
}

type DataSourceAlertsV2Alerts struct {
	CreateTime        string                                  `json:"create_time,omitempty"`
	CustomDescription string                                  `json:"custom_description,omitempty"`
	CustomSummary     string                                  `json:"custom_summary,omitempty"`
	DisplayName       string                                  `json:"display_name,omitempty"`
	EffectiveRunAs    *DataSourceAlertsV2AlertsEffectiveRunAs `json:"effective_run_as,omitempty"`
	Evaluation        *DataSourceAlertsV2AlertsEvaluation     `json:"evaluation,omitempty"`
	Id                string                                  `json:"id"`
	LifecycleState    string                                  `json:"lifecycle_state,omitempty"`
	OwnerUserName     string                                  `json:"owner_user_name,omitempty"`
	ParentPath        string                                  `json:"parent_path,omitempty"`
	QueryText         string                                  `json:"query_text,omitempty"`
	RunAs             *DataSourceAlertsV2AlertsRunAs          `json:"run_as,omitempty"`
	RunAsUserName     string                                  `json:"run_as_user_name,omitempty"`
	Schedule          *DataSourceAlertsV2AlertsSchedule       `json:"schedule,omitempty"`
	UpdateTime        string                                  `json:"update_time,omitempty"`
	WarehouseId       string                                  `json:"warehouse_id,omitempty"`
}

type DataSourceAlertsV2 struct {
	Alerts []DataSourceAlertsV2Alerts `json:"alerts,omitempty"`
}
