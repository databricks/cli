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

type DataSourceFeatureEngineeringFeaturesFeaturesLineageContextJobContext struct {
	JobId    int `json:"job_id,omitempty"`
	JobRunId int `json:"job_run_id,omitempty"`
}

type DataSourceFeatureEngineeringFeaturesFeaturesLineageContext struct {
	JobContext *DataSourceFeatureEngineeringFeaturesFeaturesLineageContextJobContext `json:"job_context,omitempty"`
	NotebookId int                                                                   `json:"notebook_id,omitempty"`
}

type DataSourceFeatureEngineeringFeaturesFeaturesSourceDeltaTableSource struct {
	EntityColumns    []string `json:"entity_columns"`
	FullName         string   `json:"full_name"`
	TimeseriesColumn string   `json:"timeseries_column"`
}

type DataSourceFeatureEngineeringFeaturesFeaturesSourceKafkaSourceEntityColumnIdentifiers struct {
	VariantExprPath string `json:"variant_expr_path"`
}

type DataSourceFeatureEngineeringFeaturesFeaturesSourceKafkaSourceTimeseriesColumnIdentifier struct {
	VariantExprPath string `json:"variant_expr_path"`
}

type DataSourceFeatureEngineeringFeaturesFeaturesSourceKafkaSource struct {
	EntityColumnIdentifiers    []DataSourceFeatureEngineeringFeaturesFeaturesSourceKafkaSourceEntityColumnIdentifiers   `json:"entity_column_identifiers,omitempty"`
	Name                       string                                                                                   `json:"name"`
	TimeseriesColumnIdentifier *DataSourceFeatureEngineeringFeaturesFeaturesSourceKafkaSourceTimeseriesColumnIdentifier `json:"timeseries_column_identifier,omitempty"`
}

type DataSourceFeatureEngineeringFeaturesFeaturesSource struct {
	DeltaTableSource *DataSourceFeatureEngineeringFeaturesFeaturesSourceDeltaTableSource `json:"delta_table_source,omitempty"`
	KafkaSource      *DataSourceFeatureEngineeringFeaturesFeaturesSourceKafkaSource      `json:"kafka_source,omitempty"`
}

type DataSourceFeatureEngineeringFeaturesFeaturesTimeWindowContinuous struct {
	Offset         string `json:"offset,omitempty"`
	WindowDuration string `json:"window_duration"`
}

type DataSourceFeatureEngineeringFeaturesFeaturesTimeWindowSliding struct {
	SlideDuration  string `json:"slide_duration"`
	WindowDuration string `json:"window_duration"`
}

type DataSourceFeatureEngineeringFeaturesFeaturesTimeWindowTumbling struct {
	WindowDuration string `json:"window_duration"`
}

type DataSourceFeatureEngineeringFeaturesFeaturesTimeWindow struct {
	Continuous *DataSourceFeatureEngineeringFeaturesFeaturesTimeWindowContinuous `json:"continuous,omitempty"`
	Sliding    *DataSourceFeatureEngineeringFeaturesFeaturesTimeWindowSliding    `json:"sliding,omitempty"`
	Tumbling   *DataSourceFeatureEngineeringFeaturesFeaturesTimeWindowTumbling   `json:"tumbling,omitempty"`
}

type DataSourceFeatureEngineeringFeaturesFeatures struct {
	Description     string                                                      `json:"description,omitempty"`
	FilterCondition string                                                      `json:"filter_condition,omitempty"`
	FullName        string                                                      `json:"full_name"`
	Function        *DataSourceFeatureEngineeringFeaturesFeaturesFunction       `json:"function,omitempty"`
	Inputs          []string                                                    `json:"inputs,omitempty"`
	LineageContext  *DataSourceFeatureEngineeringFeaturesFeaturesLineageContext `json:"lineage_context,omitempty"`
	Source          *DataSourceFeatureEngineeringFeaturesFeaturesSource         `json:"source,omitempty"`
	TimeWindow      *DataSourceFeatureEngineeringFeaturesFeaturesTimeWindow     `json:"time_window,omitempty"`
}

type DataSourceFeatureEngineeringFeatures struct {
	Features []DataSourceFeatureEngineeringFeaturesFeatures `json:"features,omitempty"`
	PageSize int                                            `json:"page_size,omitempty"`
}
