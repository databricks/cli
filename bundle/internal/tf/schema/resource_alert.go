// Generated from Databricks Terraform provider schema. DO NOT EDIT.

package schema

type ResourceAlertConditionOperandColumn struct {
	Name string `json:"name"`
}

type ResourceAlertConditionOperand struct {
	Column *ResourceAlertConditionOperandColumn `json:"column,omitempty"`
}

type ResourceAlertConditionThresholdValue struct {
	BoolValue   bool   `json:"bool_value,omitempty"`
	DoubleValue int    `json:"double_value,omitempty"`
	StringValue string `json:"string_value,omitempty"`
}

type ResourceAlertConditionThreshold struct {
	Value *ResourceAlertConditionThresholdValue `json:"value,omitempty"`
}

type ResourceAlertCondition struct {
	EmptyResultState string                           `json:"empty_result_state,omitempty"`
	Op               string                           `json:"op"`
	Operand          *ResourceAlertConditionOperand   `json:"operand,omitempty"`
	Threshold        *ResourceAlertConditionThreshold `json:"threshold,omitempty"`
}

type ResourceAlert struct {
	CreateTime         string                  `json:"create_time,omitempty"`
	CustomBody         string                  `json:"custom_body,omitempty"`
	CustomSubject      string                  `json:"custom_subject,omitempty"`
	DisplayName        string                  `json:"display_name"`
	Id                 string                  `json:"id,omitempty"`
	LifecycleState     string                  `json:"lifecycle_state,omitempty"`
	NotifyOnOk         bool                    `json:"notify_on_ok,omitempty"`
	OwnerUserName      string                  `json:"owner_user_name,omitempty"`
	ParentPath         string                  `json:"parent_path,omitempty"`
	QueryId            string                  `json:"query_id"`
	SecondsToRetrigger int                     `json:"seconds_to_retrigger,omitempty"`
	State              string                  `json:"state,omitempty"`
	TriggerTime        string                  `json:"trigger_time,omitempty"`
	UpdateTime         string                  `json:"update_time,omitempty"`
	Condition          *ResourceAlertCondition `json:"condition,omitempty"`
}
