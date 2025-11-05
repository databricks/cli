// Generated from Databricks Terraform provider schema. DO NOT EDIT.

package schema

type ResourceFeatureEngineeringFeatureFunctionExtraParameters struct {
	Key   string `json:"key"`
	Value string `json:"value"`
}

type ResourceFeatureEngineeringFeatureFunction struct {
	ExtraParameters []ResourceFeatureEngineeringFeatureFunctionExtraParameters `json:"extra_parameters,omitempty"`
	FunctionType    string                                                     `json:"function_type"`
}

type ResourceFeatureEngineeringFeatureSourceDeltaTableSource struct {
	EntityColumns    []string `json:"entity_columns"`
	FullName         string   `json:"full_name"`
	TimeseriesColumn string   `json:"timeseries_column"`
}

type ResourceFeatureEngineeringFeatureSource struct {
	DeltaTableSource *ResourceFeatureEngineeringFeatureSourceDeltaTableSource `json:"delta_table_source,omitempty"`
}

type ResourceFeatureEngineeringFeatureTimeWindowContinuous struct {
	Offset         string `json:"offset,omitempty"`
	WindowDuration string `json:"window_duration"`
}

type ResourceFeatureEngineeringFeatureTimeWindowSliding struct {
	SlideDuration  string `json:"slide_duration"`
	WindowDuration string `json:"window_duration"`
}

type ResourceFeatureEngineeringFeatureTimeWindowTumbling struct {
	WindowDuration string `json:"window_duration"`
}

type ResourceFeatureEngineeringFeatureTimeWindow struct {
	Continuous *ResourceFeatureEngineeringFeatureTimeWindowContinuous `json:"continuous,omitempty"`
	Sliding    *ResourceFeatureEngineeringFeatureTimeWindowSliding    `json:"sliding,omitempty"`
	Tumbling   *ResourceFeatureEngineeringFeatureTimeWindowTumbling   `json:"tumbling,omitempty"`
}

type ResourceFeatureEngineeringFeature struct {
	Description     string                                       `json:"description,omitempty"`
	FilterCondition string                                       `json:"filter_condition,omitempty"`
	FullName        string                                       `json:"full_name"`
	Function        *ResourceFeatureEngineeringFeatureFunction   `json:"function,omitempty"`
	Inputs          []string                                     `json:"inputs"`
	Source          *ResourceFeatureEngineeringFeatureSource     `json:"source,omitempty"`
	TimeWindow      *ResourceFeatureEngineeringFeatureTimeWindow `json:"time_window,omitempty"`
}
