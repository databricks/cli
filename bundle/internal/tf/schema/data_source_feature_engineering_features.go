// Generated from Databricks Terraform provider schema. DO NOT EDIT.

package schema

type DataSourceFeatureEngineeringFeaturesFeaturesFunctionExtraParameters struct {
	Key   string `json:"key"`
	Value string `json:"value"`
}

type DataSourceFeatureEngineeringFeaturesFeaturesFunction struct {
	ExtraParameters []DataSourceFeatureEngineeringFeaturesFeaturesFunctionExtraParameters `json:"extra_parameters,omitempty"`
	FunctionType    string                                                                `json:"function_type"`
}

type DataSourceFeatureEngineeringFeaturesFeaturesSourceDeltaTableSource struct {
	EntityColumns    []string `json:"entity_columns"`
	FullName         string   `json:"full_name"`
	TimeseriesColumn string   `json:"timeseries_column"`
}

type DataSourceFeatureEngineeringFeaturesFeaturesSource struct {
	DeltaTableSource *DataSourceFeatureEngineeringFeaturesFeaturesSourceDeltaTableSource `json:"delta_table_source,omitempty"`
}

type DataSourceFeatureEngineeringFeaturesFeaturesTimeWindow struct {
	Duration string `json:"duration"`
	Offset   string `json:"offset,omitempty"`
}

type DataSourceFeatureEngineeringFeaturesFeatures struct {
	Description string                                                  `json:"description,omitempty"`
	FullName    string                                                  `json:"full_name"`
	Function    *DataSourceFeatureEngineeringFeaturesFeaturesFunction   `json:"function,omitempty"`
	Inputs      []string                                                `json:"inputs"`
	Source      *DataSourceFeatureEngineeringFeaturesFeaturesSource     `json:"source,omitempty"`
	TimeWindow  *DataSourceFeatureEngineeringFeaturesFeaturesTimeWindow `json:"time_window,omitempty"`
}

type DataSourceFeatureEngineeringFeatures struct {
	Features []DataSourceFeatureEngineeringFeaturesFeatures `json:"features,omitempty"`
}
