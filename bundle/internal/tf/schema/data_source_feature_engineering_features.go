// Generated from Databricks Terraform provider schema. DO NOT EDIT.

package schema

type DataSourceFeatureEngineeringFeaturesFeaturesEntities struct {
	Name string `json:"name"`
}

type DataSourceFeatureEngineeringFeaturesFeaturesFunctionAggregationFunctionApproxCountDistinct struct {
	Input      string `json:"input"`
	RelativeSd int    `json:"relative_sd,omitempty"`
}

type DataSourceFeatureEngineeringFeaturesFeaturesFunctionAggregationFunctionApproxPercentile struct {
	Accuracy   int    `json:"accuracy,omitempty"`
	Input      string `json:"input"`
	Percentile int    `json:"percentile"`
}

type DataSourceFeatureEngineeringFeaturesFeaturesFunctionAggregationFunctionAvg struct {
	Input string `json:"input"`
}

type DataSourceFeatureEngineeringFeaturesFeaturesFunctionAggregationFunctionCountFunction struct {
	Input string `json:"input"`
}

type DataSourceFeatureEngineeringFeaturesFeaturesFunctionAggregationFunctionFirst struct {
	Input string `json:"input"`
}

type DataSourceFeatureEngineeringFeaturesFeaturesFunctionAggregationFunctionLast struct {
	Input string `json:"input"`
}

type DataSourceFeatureEngineeringFeaturesFeaturesFunctionAggregationFunctionMax struct {
	Input string `json:"input"`
}

type DataSourceFeatureEngineeringFeaturesFeaturesFunctionAggregationFunctionMin struct {
	Input string `json:"input"`
}

type DataSourceFeatureEngineeringFeaturesFeaturesFunctionAggregationFunctionStddevPop struct {
	Input string `json:"input"`
}

type DataSourceFeatureEngineeringFeaturesFeaturesFunctionAggregationFunctionStddevSamp struct {
	Input string `json:"input"`
}

type DataSourceFeatureEngineeringFeaturesFeaturesFunctionAggregationFunctionSum struct {
	Input string `json:"input"`
}

type DataSourceFeatureEngineeringFeaturesFeaturesFunctionAggregationFunctionTimeWindowContinuous struct {
	Offset         string `json:"offset,omitempty"`
	WindowDuration string `json:"window_duration"`
}

type DataSourceFeatureEngineeringFeaturesFeaturesFunctionAggregationFunctionTimeWindowSliding struct {
	SlideDuration  string `json:"slide_duration"`
	WindowDuration string `json:"window_duration"`
}

type DataSourceFeatureEngineeringFeaturesFeaturesFunctionAggregationFunctionTimeWindowTumbling struct {
	WindowDuration string `json:"window_duration"`
}

type DataSourceFeatureEngineeringFeaturesFeaturesFunctionAggregationFunctionTimeWindow struct {
	Continuous *DataSourceFeatureEngineeringFeaturesFeaturesFunctionAggregationFunctionTimeWindowContinuous `json:"continuous,omitempty"`
	Sliding    *DataSourceFeatureEngineeringFeaturesFeaturesFunctionAggregationFunctionTimeWindowSliding    `json:"sliding,omitempty"`
	Tumbling   *DataSourceFeatureEngineeringFeaturesFeaturesFunctionAggregationFunctionTimeWindowTumbling   `json:"tumbling,omitempty"`
}

type DataSourceFeatureEngineeringFeaturesFeaturesFunctionAggregationFunctionVarPop struct {
	Input string `json:"input"`
}

type DataSourceFeatureEngineeringFeaturesFeaturesFunctionAggregationFunctionVarSamp struct {
	Input string `json:"input"`
}

type DataSourceFeatureEngineeringFeaturesFeaturesFunctionAggregationFunction struct {
	ApproxCountDistinct *DataSourceFeatureEngineeringFeaturesFeaturesFunctionAggregationFunctionApproxCountDistinct `json:"approx_count_distinct,omitempty"`
	ApproxPercentile    *DataSourceFeatureEngineeringFeaturesFeaturesFunctionAggregationFunctionApproxPercentile    `json:"approx_percentile,omitempty"`
	Avg                 *DataSourceFeatureEngineeringFeaturesFeaturesFunctionAggregationFunctionAvg                 `json:"avg,omitempty"`
	CountFunction       *DataSourceFeatureEngineeringFeaturesFeaturesFunctionAggregationFunctionCountFunction       `json:"count_function,omitempty"`
	First               *DataSourceFeatureEngineeringFeaturesFeaturesFunctionAggregationFunctionFirst               `json:"first,omitempty"`
	Last                *DataSourceFeatureEngineeringFeaturesFeaturesFunctionAggregationFunctionLast                `json:"last,omitempty"`
	Max                 *DataSourceFeatureEngineeringFeaturesFeaturesFunctionAggregationFunctionMax                 `json:"max,omitempty"`
	Min                 *DataSourceFeatureEngineeringFeaturesFeaturesFunctionAggregationFunctionMin                 `json:"min,omitempty"`
	StddevPop           *DataSourceFeatureEngineeringFeaturesFeaturesFunctionAggregationFunctionStddevPop           `json:"stddev_pop,omitempty"`
	StddevSamp          *DataSourceFeatureEngineeringFeaturesFeaturesFunctionAggregationFunctionStddevSamp          `json:"stddev_samp,omitempty"`
	Sum                 *DataSourceFeatureEngineeringFeaturesFeaturesFunctionAggregationFunctionSum                 `json:"sum,omitempty"`
	TimeWindow          *DataSourceFeatureEngineeringFeaturesFeaturesFunctionAggregationFunctionTimeWindow          `json:"time_window,omitempty"`
	VarPop              *DataSourceFeatureEngineeringFeaturesFeaturesFunctionAggregationFunctionVarPop              `json:"var_pop,omitempty"`
	VarSamp             *DataSourceFeatureEngineeringFeaturesFeaturesFunctionAggregationFunctionVarSamp             `json:"var_samp,omitempty"`
}

type DataSourceFeatureEngineeringFeaturesFeaturesFunctionColumnSelection struct {
	Column string `json:"column"`
}

type DataSourceFeatureEngineeringFeaturesFeaturesFunctionExtraParameters struct {
	Key   string `json:"key"`
	Value string `json:"value"`
}

type DataSourceFeatureEngineeringFeaturesFeaturesFunction struct {
	AggregationFunction *DataSourceFeatureEngineeringFeaturesFeaturesFunctionAggregationFunction `json:"aggregation_function,omitempty"`
	ColumnSelection     *DataSourceFeatureEngineeringFeaturesFeaturesFunctionColumnSelection     `json:"column_selection,omitempty"`
	ExtraParameters     []DataSourceFeatureEngineeringFeaturesFeaturesFunctionExtraParameters    `json:"extra_parameters,omitempty"`
	FunctionType        string                                                                   `json:"function_type,omitempty"`
}

type DataSourceFeatureEngineeringFeaturesFeaturesLineageContextJobContext struct {
	JobId    int `json:"job_id,omitempty"`
	JobRunId int `json:"job_run_id,omitempty"`
}

type DataSourceFeatureEngineeringFeaturesFeaturesLineageContext struct {
	JobContext *DataSourceFeatureEngineeringFeaturesFeaturesLineageContextJobContext `json:"job_context,omitempty"`
	NotebookId int                                                                   `json:"notebook_id,omitempty"`
}

type DataSourceFeatureEngineeringFeaturesFeaturesProviderConfig struct {
	WorkspaceId string `json:"workspace_id,omitempty"`
}

type DataSourceFeatureEngineeringFeaturesFeaturesSourceDeltaTableSource struct {
	DataframeSchema   string   `json:"dataframe_schema,omitempty"`
	EntityColumns     []string `json:"entity_columns,omitempty"`
	FilterCondition   string   `json:"filter_condition,omitempty"`
	FullName          string   `json:"full_name"`
	TimeseriesColumn  string   `json:"timeseries_column,omitempty"`
	TransformationSql string   `json:"transformation_sql,omitempty"`
}

type DataSourceFeatureEngineeringFeaturesFeaturesSourceKafkaSourceEntityColumnIdentifiers struct {
	VariantExprPath string `json:"variant_expr_path"`
}

type DataSourceFeatureEngineeringFeaturesFeaturesSourceKafkaSourceTimeseriesColumnIdentifier struct {
	VariantExprPath string `json:"variant_expr_path"`
}

type DataSourceFeatureEngineeringFeaturesFeaturesSourceKafkaSource struct {
	EntityColumnIdentifiers    []DataSourceFeatureEngineeringFeaturesFeaturesSourceKafkaSourceEntityColumnIdentifiers   `json:"entity_column_identifiers,omitempty"`
	FilterCondition            string                                                                                   `json:"filter_condition,omitempty"`
	Name                       string                                                                                   `json:"name"`
	TimeseriesColumnIdentifier *DataSourceFeatureEngineeringFeaturesFeaturesSourceKafkaSourceTimeseriesColumnIdentifier `json:"timeseries_column_identifier,omitempty"`
}

type DataSourceFeatureEngineeringFeaturesFeaturesSourceRequestSourceFlatSchemaFields struct {
	DataType string `json:"data_type"`
	Name     string `json:"name"`
}

type DataSourceFeatureEngineeringFeaturesFeaturesSourceRequestSourceFlatSchema struct {
	Fields []DataSourceFeatureEngineeringFeaturesFeaturesSourceRequestSourceFlatSchemaFields `json:"fields,omitempty"`
}

type DataSourceFeatureEngineeringFeaturesFeaturesSourceRequestSource struct {
	FlatSchema *DataSourceFeatureEngineeringFeaturesFeaturesSourceRequestSourceFlatSchema `json:"flat_schema,omitempty"`
}

type DataSourceFeatureEngineeringFeaturesFeaturesSource struct {
	DeltaTableSource *DataSourceFeatureEngineeringFeaturesFeaturesSourceDeltaTableSource `json:"delta_table_source,omitempty"`
	KafkaSource      *DataSourceFeatureEngineeringFeaturesFeaturesSourceKafkaSource      `json:"kafka_source,omitempty"`
	RequestSource    *DataSourceFeatureEngineeringFeaturesFeaturesSourceRequestSource    `json:"request_source,omitempty"`
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

type DataSourceFeatureEngineeringFeaturesFeaturesTimeseriesColumn struct {
	Name string `json:"name"`
}

type DataSourceFeatureEngineeringFeaturesFeatures struct {
	Description      string                                                        `json:"description,omitempty"`
	Entities         []DataSourceFeatureEngineeringFeaturesFeaturesEntities        `json:"entities,omitempty"`
	FilterCondition  string                                                        `json:"filter_condition,omitempty"`
	FullName         string                                                        `json:"full_name"`
	Function         *DataSourceFeatureEngineeringFeaturesFeaturesFunction         `json:"function,omitempty"`
	Inputs           []string                                                      `json:"inputs,omitempty"`
	LineageContext   *DataSourceFeatureEngineeringFeaturesFeaturesLineageContext   `json:"lineage_context,omitempty"`
	ProviderConfig   *DataSourceFeatureEngineeringFeaturesFeaturesProviderConfig   `json:"provider_config,omitempty"`
	Source           *DataSourceFeatureEngineeringFeaturesFeaturesSource           `json:"source,omitempty"`
	TimeWindow       *DataSourceFeatureEngineeringFeaturesFeaturesTimeWindow       `json:"time_window,omitempty"`
	TimeseriesColumn *DataSourceFeatureEngineeringFeaturesFeaturesTimeseriesColumn `json:"timeseries_column,omitempty"`
}

type DataSourceFeatureEngineeringFeaturesProviderConfig struct {
	WorkspaceId string `json:"workspace_id,omitempty"`
}

type DataSourceFeatureEngineeringFeatures struct {
	Features       []DataSourceFeatureEngineeringFeaturesFeatures      `json:"features,omitempty"`
	PageSize       int                                                 `json:"page_size,omitempty"`
	ProviderConfig *DataSourceFeatureEngineeringFeaturesProviderConfig `json:"provider_config,omitempty"`
}
