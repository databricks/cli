// Generated from Databricks Terraform provider schema. DO NOT EDIT.

package schema

type DataSourceFeatureEngineeringFeatureFunctionExtraParameters struct {
	Key   string `json:"key"`
	Value string `json:"value"`
}

type DataSourceFeatureEngineeringFeatureFunction struct {
	ExtraParameters []DataSourceFeatureEngineeringFeatureFunctionExtraParameters `json:"extra_parameters,omitempty"`
	FunctionType    string                                                       `json:"function_type"`
}

type DataSourceFeatureEngineeringFeatureSourceDeltaTableSource struct {
	EntityColumns    []string `json:"entity_columns"`
	FullName         string   `json:"full_name"`
	TimeseriesColumn string   `json:"timeseries_column"`
}

type DataSourceFeatureEngineeringFeatureSource struct {
	DeltaTableSource *DataSourceFeatureEngineeringFeatureSourceDeltaTableSource `json:"delta_table_source,omitempty"`
}

type DataSourceFeatureEngineeringFeatureTimeWindowContinuous struct {
	Offset         string `json:"offset,omitempty"`
	WindowDuration string `json:"window_duration"`
}

type DataSourceFeatureEngineeringFeatureTimeWindowSliding struct {
	SlideDuration  string `json:"slide_duration"`
	WindowDuration string `json:"window_duration"`
}

type DataSourceFeatureEngineeringFeatureTimeWindowTumbling struct {
	WindowDuration string `json:"window_duration"`
}

type DataSourceFeatureEngineeringFeatureTimeWindow struct {
	Continuous *DataSourceFeatureEngineeringFeatureTimeWindowContinuous `json:"continuous,omitempty"`
	Sliding    *DataSourceFeatureEngineeringFeatureTimeWindowSliding    `json:"sliding,omitempty"`
	Tumbling   *DataSourceFeatureEngineeringFeatureTimeWindowTumbling   `json:"tumbling,omitempty"`
}

type DataSourceFeatureEngineeringFeature struct {
	Description     string                                         `json:"description,omitempty"`
	FilterCondition string                                         `json:"filter_condition,omitempty"`
	FullName        string                                         `json:"full_name"`
	Function        *DataSourceFeatureEngineeringFeatureFunction   `json:"function,omitempty"`
	Inputs          []string                                       `json:"inputs,omitempty"`
	Source          *DataSourceFeatureEngineeringFeatureSource     `json:"source,omitempty"`
	TimeWindow      *DataSourceFeatureEngineeringFeatureTimeWindow `json:"time_window,omitempty"`
}
