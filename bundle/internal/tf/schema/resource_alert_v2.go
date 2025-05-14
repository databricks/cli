// Generated from Databricks Terraform provider schema. DO NOT EDIT.

package schema

type ResourceAlertV2EvaluationNotificationSubscriptions struct {
	DestinationId string `json:"destination_id,omitempty"`
	UserEmail     string `json:"user_email,omitempty"`
}

type ResourceAlertV2EvaluationNotification struct {
	NotifyOnOk       bool                                                 `json:"notify_on_ok,omitempty"`
	RetriggerSeconds int                                                  `json:"retrigger_seconds,omitempty"`
	Subscriptions    []ResourceAlertV2EvaluationNotificationSubscriptions `json:"subscriptions,omitempty"`
}

type ResourceAlertV2EvaluationSource struct {
	Aggregation string `json:"aggregation,omitempty"`
	Display     string `json:"display,omitempty"`
	Name        string `json:"name,omitempty"`
}

type ResourceAlertV2EvaluationThresholdColumn struct {
	Aggregation string `json:"aggregation,omitempty"`
	Display     string `json:"display,omitempty"`
	Name        string `json:"name,omitempty"`
}

type ResourceAlertV2EvaluationThresholdValue struct {
	BoolValue   bool   `json:"bool_value,omitempty"`
	DoubleValue int    `json:"double_value,omitempty"`
	StringValue string `json:"string_value,omitempty"`
}

type ResourceAlertV2EvaluationThreshold struct {
	Column *ResourceAlertV2EvaluationThresholdColumn `json:"column,omitempty"`
	Value  *ResourceAlertV2EvaluationThresholdValue  `json:"value,omitempty"`
}

type ResourceAlertV2Evaluation struct {
	ComparisonOperator string                                 `json:"comparison_operator,omitempty"`
	EmptyResultState   string                                 `json:"empty_result_state,omitempty"`
	LastEvaluatedAt    string                                 `json:"last_evaluated_at,omitempty"`
	Notification       *ResourceAlertV2EvaluationNotification `json:"notification,omitempty"`
	Source             *ResourceAlertV2EvaluationSource       `json:"source,omitempty"`
	State              string                                 `json:"state,omitempty"`
	Threshold          *ResourceAlertV2EvaluationThreshold    `json:"threshold,omitempty"`
}

type ResourceAlertV2Schedule struct {
	PauseStatus        string `json:"pause_status,omitempty"`
	QuartzCronSchedule string `json:"quartz_cron_schedule,omitempty"`
	TimezoneId         string `json:"timezone_id,omitempty"`
}

type ResourceAlertV2 struct {
	CreateTime        string                     `json:"create_time,omitempty"`
	CustomDescription string                     `json:"custom_description,omitempty"`
	CustomSummary     string                     `json:"custom_summary,omitempty"`
	DisplayName       string                     `json:"display_name,omitempty"`
	Evaluation        *ResourceAlertV2Evaluation `json:"evaluation,omitempty"`
	Id                string                     `json:"id,omitempty"`
	LifecycleState    string                     `json:"lifecycle_state,omitempty"`
	OwnerUserName     string                     `json:"owner_user_name,omitempty"`
	ParentPath        string                     `json:"parent_path,omitempty"`
	QueryText         string                     `json:"query_text,omitempty"`
	RunAsUserName     string                     `json:"run_as_user_name,omitempty"`
	Schedule          *ResourceAlertV2Schedule   `json:"schedule,omitempty"`
	UpdateTime        string                     `json:"update_time,omitempty"`
	WarehouseId       string                     `json:"warehouse_id,omitempty"`
}
