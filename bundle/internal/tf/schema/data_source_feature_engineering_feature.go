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

type DataSourceFeatureEngineeringFeatureTimeWindow struct {
	Duration string `json:"duration"`
	Offset   string `json:"offset,omitempty"`
}

type DataSourceFeatureEngineeringFeature struct {
	Description string                                         `json:"description,omitempty"`
	FullName    string                                         `json:"full_name"`
	Function    *DataSourceFeatureEngineeringFeatureFunction   `json:"function,omitempty"`
	Inputs      []string                                       `json:"inputs"`
	Source      *DataSourceFeatureEngineeringFeatureSource     `json:"source,omitempty"`
	TimeWindow  *DataSourceFeatureEngineeringFeatureTimeWindow `json:"time_window,omitempty"`
}
