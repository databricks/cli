// Generated from Databricks Terraform provider schema. DO NOT EDIT.

package schema

type ResourceBudgetAlertConfigurationsActionConfigurations struct {
	ActionConfigurationId string `json:"action_configuration_id,omitempty"`
	ActionType            string `json:"action_type,omitempty"`
	Target                string `json:"target,omitempty"`
}

type ResourceBudgetAlertConfigurations struct {
	AlertConfigurationId string                                                  `json:"alert_configuration_id,omitempty"`
	QuantityThreshold    string                                                  `json:"quantity_threshold,omitempty"`
	QuantityType         string                                                  `json:"quantity_type,omitempty"`
	TimePeriod           string                                                  `json:"time_period,omitempty"`
	TriggerType          string                                                  `json:"trigger_type,omitempty"`
	ActionConfigurations []ResourceBudgetAlertConfigurationsActionConfigurations `json:"action_configurations,omitempty"`
}

type ResourceBudgetFilterTagsValue struct {
	Operator string   `json:"operator,omitempty"`
	Values   []string `json:"values,omitempty"`
}

type ResourceBudgetFilterTags struct {
	Key   string                         `json:"key,omitempty"`
	Value *ResourceBudgetFilterTagsValue `json:"value,omitempty"`
}

type ResourceBudgetFilterWorkspaceId struct {
	Operator string `json:"operator,omitempty"`
	Values   []int  `json:"values,omitempty"`
}

type ResourceBudgetFilter struct {
	Tags        []ResourceBudgetFilterTags       `json:"tags,omitempty"`
	WorkspaceId *ResourceBudgetFilterWorkspaceId `json:"workspace_id,omitempty"`
}

type ResourceBudget struct {
	AccountId             string                              `json:"account_id,omitempty"`
	BudgetConfigurationId string                              `json:"budget_configuration_id,omitempty"`
	CreateTime            int                                 `json:"create_time,omitempty"`
	DisplayName           string                              `json:"display_name,omitempty"`
	Id                    string                              `json:"id,omitempty"`
	UpdateTime            int                                 `json:"update_time,omitempty"`
	AlertConfigurations   []ResourceBudgetAlertConfigurations `json:"alert_configurations,omitempty"`
	Filter                *ResourceBudgetFilter               `json:"filter,omitempty"`
}
