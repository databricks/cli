// Generated from Databricks Terraform provider schema. DO NOT EDIT.

package schema

type ResourceQueryParameterDateRangeValueDateRangeValue struct {
	End   string `json:"end"`
	Start string `json:"start"`
}

type ResourceQueryParameterDateRangeValue struct {
	DynamicDateRangeValue string                                              `json:"dynamic_date_range_value,omitempty"`
	Precision             string                                              `json:"precision,omitempty"`
	StartDayOfWeek        int                                                 `json:"start_day_of_week,omitempty"`
	DateRangeValue        *ResourceQueryParameterDateRangeValueDateRangeValue `json:"date_range_value,omitempty"`
}

type ResourceQueryParameterDateValue struct {
	DateValue        string `json:"date_value,omitempty"`
	DynamicDateValue string `json:"dynamic_date_value,omitempty"`
	Precision        string `json:"precision,omitempty"`
}

type ResourceQueryParameterEnumValueMultiValuesOptions struct {
	Prefix    string `json:"prefix,omitempty"`
	Separator string `json:"separator,omitempty"`
	Suffix    string `json:"suffix,omitempty"`
}

type ResourceQueryParameterEnumValue struct {
	EnumOptions        string                                             `json:"enum_options,omitempty"`
	Values             []string                                           `json:"values,omitempty"`
	MultiValuesOptions *ResourceQueryParameterEnumValueMultiValuesOptions `json:"multi_values_options,omitempty"`
}

type ResourceQueryParameterNumericValue struct {
	Value int `json:"value"`
}

type ResourceQueryParameterQueryBackedValueMultiValuesOptions struct {
	Prefix    string `json:"prefix,omitempty"`
	Separator string `json:"separator,omitempty"`
	Suffix    string `json:"suffix,omitempty"`
}

type ResourceQueryParameterQueryBackedValue struct {
	QueryId            string                                                    `json:"query_id"`
	Values             []string                                                  `json:"values,omitempty"`
	MultiValuesOptions *ResourceQueryParameterQueryBackedValueMultiValuesOptions `json:"multi_values_options,omitempty"`
}

type ResourceQueryParameterTextValue struct {
	Value string `json:"value"`
}

type ResourceQueryParameter struct {
	Name             string                                  `json:"name"`
	Title            string                                  `json:"title,omitempty"`
	DateRangeValue   *ResourceQueryParameterDateRangeValue   `json:"date_range_value,omitempty"`
	DateValue        *ResourceQueryParameterDateValue        `json:"date_value,omitempty"`
	EnumValue        *ResourceQueryParameterEnumValue        `json:"enum_value,omitempty"`
	NumericValue     *ResourceQueryParameterNumericValue     `json:"numeric_value,omitempty"`
	QueryBackedValue *ResourceQueryParameterQueryBackedValue `json:"query_backed_value,omitempty"`
	TextValue        *ResourceQueryParameterTextValue        `json:"text_value,omitempty"`
}

type ResourceQuery struct {
	ApplyAutoLimit       bool                     `json:"apply_auto_limit,omitempty"`
	Catalog              string                   `json:"catalog,omitempty"`
	CreateTime           string                   `json:"create_time,omitempty"`
	Description          string                   `json:"description,omitempty"`
	DisplayName          string                   `json:"display_name"`
	Id                   string                   `json:"id,omitempty"`
	LastModifierUserName string                   `json:"last_modifier_user_name,omitempty"`
	LifecycleState       string                   `json:"lifecycle_state,omitempty"`
	OwnerUserName        string                   `json:"owner_user_name,omitempty"`
	ParentPath           string                   `json:"parent_path,omitempty"`
	QueryText            string                   `json:"query_text"`
	RunAsMode            string                   `json:"run_as_mode,omitempty"`
	Schema               string                   `json:"schema,omitempty"`
	Tags                 []string                 `json:"tags,omitempty"`
	UpdateTime           string                   `json:"update_time,omitempty"`
	WarehouseId          string                   `json:"warehouse_id"`
	Parameter            []ResourceQueryParameter `json:"parameter,omitempty"`
}
