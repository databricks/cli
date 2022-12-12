// Generated from Databricks Terraform provider schema. DO NOT EDIT.

package schema

type ResourceSqlQueryParameterDate struct {
	Value string `json:"value"`
}

type ResourceSqlQueryParameterDateRange struct {
	Value string `json:"value"`
}

type ResourceSqlQueryParameterDatetime struct {
	Value string `json:"value"`
}

type ResourceSqlQueryParameterDatetimeRange struct {
	Value string `json:"value"`
}

type ResourceSqlQueryParameterDatetimesec struct {
	Value string `json:"value"`
}

type ResourceSqlQueryParameterDatetimesecRange struct {
	Value string `json:"value"`
}

type ResourceSqlQueryParameterEnumMultiple struct {
	Prefix    string `json:"prefix"`
	Separator string `json:"separator"`
	Suffix    string `json:"suffix"`
}

type ResourceSqlQueryParameterEnum struct {
	Options  []string                               `json:"options"`
	Value    string                                 `json:"value,omitempty"`
	Values   []string                               `json:"values,omitempty"`
	Multiple *ResourceSqlQueryParameterEnumMultiple `json:"multiple,omitempty"`
}

type ResourceSqlQueryParameterNumber struct {
	Value int `json:"value"`
}

type ResourceSqlQueryParameterQueryMultiple struct {
	Prefix    string `json:"prefix"`
	Separator string `json:"separator"`
	Suffix    string `json:"suffix"`
}

type ResourceSqlQueryParameterQuery struct {
	QueryId  string                                  `json:"query_id"`
	Value    string                                  `json:"value,omitempty"`
	Values   []string                                `json:"values,omitempty"`
	Multiple *ResourceSqlQueryParameterQueryMultiple `json:"multiple,omitempty"`
}

type ResourceSqlQueryParameterText struct {
	Value string `json:"value"`
}

type ResourceSqlQueryParameter struct {
	Name             string                                     `json:"name"`
	Title            string                                     `json:"title,omitempty"`
	Date             *ResourceSqlQueryParameterDate             `json:"date,omitempty"`
	DateRange        *ResourceSqlQueryParameterDateRange        `json:"date_range,omitempty"`
	Datetime         *ResourceSqlQueryParameterDatetime         `json:"datetime,omitempty"`
	DatetimeRange    *ResourceSqlQueryParameterDatetimeRange    `json:"datetime_range,omitempty"`
	Datetimesec      *ResourceSqlQueryParameterDatetimesec      `json:"datetimesec,omitempty"`
	DatetimesecRange *ResourceSqlQueryParameterDatetimesecRange `json:"datetimesec_range,omitempty"`
	Enum             *ResourceSqlQueryParameterEnum             `json:"enum,omitempty"`
	Number           *ResourceSqlQueryParameterNumber           `json:"number,omitempty"`
	Query            *ResourceSqlQueryParameterQuery            `json:"query,omitempty"`
	Text             *ResourceSqlQueryParameterText             `json:"text,omitempty"`
}

type ResourceSqlQueryScheduleContinuous struct {
	IntervalSeconds int    `json:"interval_seconds"`
	UntilDate       string `json:"until_date,omitempty"`
}

type ResourceSqlQueryScheduleDaily struct {
	IntervalDays int    `json:"interval_days"`
	TimeOfDay    string `json:"time_of_day"`
	UntilDate    string `json:"until_date,omitempty"`
}

type ResourceSqlQueryScheduleWeekly struct {
	DayOfWeek     string `json:"day_of_week"`
	IntervalWeeks int    `json:"interval_weeks"`
	TimeOfDay     string `json:"time_of_day"`
	UntilDate     string `json:"until_date,omitempty"`
}

type ResourceSqlQuerySchedule struct {
	Continuous *ResourceSqlQueryScheduleContinuous `json:"continuous,omitempty"`
	Daily      *ResourceSqlQueryScheduleDaily      `json:"daily,omitempty"`
	Weekly     *ResourceSqlQueryScheduleWeekly     `json:"weekly,omitempty"`
}

type ResourceSqlQuery struct {
	DataSourceId string                      `json:"data_source_id"`
	Description  string                      `json:"description,omitempty"`
	Id           string                      `json:"id,omitempty"`
	Name         string                      `json:"name"`
	Query        string                      `json:"query"`
	RunAsRole    string                      `json:"run_as_role,omitempty"`
	Tags         []string                    `json:"tags,omitempty"`
	Parameter    []ResourceSqlQueryParameter `json:"parameter,omitempty"`
	Schedule     *ResourceSqlQuerySchedule   `json:"schedule,omitempty"`
}
