// Generated from Databricks Terraform provider schema. DO NOT EDIT.

package schema

type DataSourceFeatureEngineeringFeatureEntities struct {
	Name string `json:"name"`
}

type DataSourceFeatureEngineeringFeatureFunctionAggregationFunctionApproxCountDistinct struct {
	Input      string `json:"input"`
	RelativeSd int    `json:"relative_sd,omitempty"`
}

type DataSourceFeatureEngineeringFeatureFunctionAggregationFunctionApproxPercentile struct {
	Accuracy   int    `json:"accuracy,omitempty"`
	Input      string `json:"input"`
	Percentile int    `json:"percentile"`
}

type DataSourceFeatureEngineeringFeatureFunctionAggregationFunctionAvg struct {
	Input string `json:"input"`
}

type DataSourceFeatureEngineeringFeatureFunctionAggregationFunctionCountFunction struct {
	Input string `json:"input"`
}

type DataSourceFeatureEngineeringFeatureFunctionAggregationFunctionFirst struct {
	Input string `json:"input"`
}

type DataSourceFeatureEngineeringFeatureFunctionAggregationFunctionLast struct {
	Input string `json:"input"`
}

type DataSourceFeatureEngineeringFeatureFunctionAggregationFunctionMax struct {
	Input string `json:"input"`
}

type DataSourceFeatureEngineeringFeatureFunctionAggregationFunctionMin struct {
	Input string `json:"input"`
}

type DataSourceFeatureEngineeringFeatureFunctionAggregationFunctionStddevPop struct {
	Input string `json:"input"`
}

type DataSourceFeatureEngineeringFeatureFunctionAggregationFunctionStddevSamp struct {
	Input string `json:"input"`
}

type DataSourceFeatureEngineeringFeatureFunctionAggregationFunctionSum struct {
	Input string `json:"input"`
}

type DataSourceFeatureEngineeringFeatureFunctionAggregationFunctionTimeWindowContinuous struct {
	Offset         string `json:"offset,omitempty"`
	WindowDuration string `json:"window_duration"`
}

type DataSourceFeatureEngineeringFeatureFunctionAggregationFunctionTimeWindowSliding struct {
	SlideDuration  string `json:"slide_duration"`
	WindowDuration string `json:"window_duration"`
}

type DataSourceFeatureEngineeringFeatureFunctionAggregationFunctionTimeWindowTumbling struct {
	WindowDuration string `json:"window_duration"`
}

type DataSourceFeatureEngineeringFeatureFunctionAggregationFunctionTimeWindow struct {
	Continuous *DataSourceFeatureEngineeringFeatureFunctionAggregationFunctionTimeWindowContinuous `json:"continuous,omitempty"`
	Sliding    *DataSourceFeatureEngineeringFeatureFunctionAggregationFunctionTimeWindowSliding    `json:"sliding,omitempty"`
	Tumbling   *DataSourceFeatureEngineeringFeatureFunctionAggregationFunctionTimeWindowTumbling   `json:"tumbling,omitempty"`
}

type DataSourceFeatureEngineeringFeatureFunctionAggregationFunctionVarPop struct {
	Input string `json:"input"`
}

type DataSourceFeatureEngineeringFeatureFunctionAggregationFunctionVarSamp struct {
	Input string `json:"input"`
}

type DataSourceFeatureEngineeringFeatureFunctionAggregationFunction struct {
	ApproxCountDistinct *DataSourceFeatureEngineeringFeatureFunctionAggregationFunctionApproxCountDistinct `json:"approx_count_distinct,omitempty"`
	ApproxPercentile    *DataSourceFeatureEngineeringFeatureFunctionAggregationFunctionApproxPercentile    `json:"approx_percentile,omitempty"`
	Avg                 *DataSourceFeatureEngineeringFeatureFunctionAggregationFunctionAvg                 `json:"avg,omitempty"`
	CountFunction       *DataSourceFeatureEngineeringFeatureFunctionAggregationFunctionCountFunction       `json:"count_function,omitempty"`
	First               *DataSourceFeatureEngineeringFeatureFunctionAggregationFunctionFirst               `json:"first,omitempty"`
	Last                *DataSourceFeatureEngineeringFeatureFunctionAggregationFunctionLast                `json:"last,omitempty"`
	Max                 *DataSourceFeatureEngineeringFeatureFunctionAggregationFunctionMax                 `json:"max,omitempty"`
	Min                 *DataSourceFeatureEngineeringFeatureFunctionAggregationFunctionMin                 `json:"min,omitempty"`
	StddevPop           *DataSourceFeatureEngineeringFeatureFunctionAggregationFunctionStddevPop           `json:"stddev_pop,omitempty"`
	StddevSamp          *DataSourceFeatureEngineeringFeatureFunctionAggregationFunctionStddevSamp          `json:"stddev_samp,omitempty"`
	Sum                 *DataSourceFeatureEngineeringFeatureFunctionAggregationFunctionSum                 `json:"sum,omitempty"`
	TimeWindow          *DataSourceFeatureEngineeringFeatureFunctionAggregationFunctionTimeWindow          `json:"time_window,omitempty"`
	VarPop              *DataSourceFeatureEngineeringFeatureFunctionAggregationFunctionVarPop              `json:"var_pop,omitempty"`
	VarSamp             *DataSourceFeatureEngineeringFeatureFunctionAggregationFunctionVarSamp             `json:"var_samp,omitempty"`
}

type DataSourceFeatureEngineeringFeatureFunctionColumnSelection struct {
	Column string `json:"column"`
}

type DataSourceFeatureEngineeringFeatureFunctionExtraParameters struct {
	Key   string `json:"key"`
	Value string `json:"value"`
}

type DataSourceFeatureEngineeringFeatureFunction struct {
	AggregationFunction *DataSourceFeatureEngineeringFeatureFunctionAggregationFunction `json:"aggregation_function,omitempty"`
	ColumnSelection     *DataSourceFeatureEngineeringFeatureFunctionColumnSelection     `json:"column_selection,omitempty"`
	ExtraParameters     []DataSourceFeatureEngineeringFeatureFunctionExtraParameters    `json:"extra_parameters,omitempty"`
	FunctionType        string                                                          `json:"function_type,omitempty"`
}

type DataSourceFeatureEngineeringFeatureLineageContextJobContext struct {
	JobId    int `json:"job_id,omitempty"`
	JobRunId int `json:"job_run_id,omitempty"`
}

type DataSourceFeatureEngineeringFeatureLineageContext struct {
	JobContext *DataSourceFeatureEngineeringFeatureLineageContextJobContext `json:"job_context,omitempty"`
	NotebookId int                                                          `json:"notebook_id,omitempty"`
}

type DataSourceFeatureEngineeringFeatureProviderConfig struct {
	WorkspaceId string `json:"workspace_id,omitempty"`
}

type DataSourceFeatureEngineeringFeatureSourceDeltaTableSource struct {
	DataframeSchema   string   `json:"dataframe_schema,omitempty"`
	EntityColumns     []string `json:"entity_columns,omitempty"`
	FilterCondition   string   `json:"filter_condition,omitempty"`
	FullName          string   `json:"full_name"`
	TimeseriesColumn  string   `json:"timeseries_column,omitempty"`
	TransformationSql string   `json:"transformation_sql,omitempty"`
}

type DataSourceFeatureEngineeringFeatureSourceKafkaSourceEntityColumnIdentifiers struct {
	VariantExprPath string `json:"variant_expr_path"`
}

type DataSourceFeatureEngineeringFeatureSourceKafkaSourceTimeseriesColumnIdentifier struct {
	VariantExprPath string `json:"variant_expr_path"`
}

type DataSourceFeatureEngineeringFeatureSourceKafkaSource struct {
	EntityColumnIdentifiers    []DataSourceFeatureEngineeringFeatureSourceKafkaSourceEntityColumnIdentifiers   `json:"entity_column_identifiers,omitempty"`
	FilterCondition            string                                                                          `json:"filter_condition,omitempty"`
	Name                       string                                                                          `json:"name"`
	TimeseriesColumnIdentifier *DataSourceFeatureEngineeringFeatureSourceKafkaSourceTimeseriesColumnIdentifier `json:"timeseries_column_identifier,omitempty"`
}

type DataSourceFeatureEngineeringFeatureSourceRequestSourceFlatSchemaFields struct {
	DataType string `json:"data_type"`
	Name     string `json:"name"`
}

type DataSourceFeatureEngineeringFeatureSourceRequestSourceFlatSchema struct {
	Fields []DataSourceFeatureEngineeringFeatureSourceRequestSourceFlatSchemaFields `json:"fields,omitempty"`
}

type DataSourceFeatureEngineeringFeatureSourceRequestSource struct {
	FlatSchema *DataSourceFeatureEngineeringFeatureSourceRequestSourceFlatSchema `json:"flat_schema,omitempty"`
}

type DataSourceFeatureEngineeringFeatureSource struct {
	DeltaTableSource *DataSourceFeatureEngineeringFeatureSourceDeltaTableSource `json:"delta_table_source,omitempty"`
	KafkaSource      *DataSourceFeatureEngineeringFeatureSourceKafkaSource      `json:"kafka_source,omitempty"`
	RequestSource    *DataSourceFeatureEngineeringFeatureSourceRequestSource    `json:"request_source,omitempty"`
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

type DataSourceFeatureEngineeringFeatureTimeseriesColumn struct {
	Name string `json:"name"`
}

type DataSourceFeatureEngineeringFeature struct {
	Description      string                                               `json:"description,omitempty"`
	Entities         []DataSourceFeatureEngineeringFeatureEntities        `json:"entities,omitempty"`
	FilterCondition  string                                               `json:"filter_condition,omitempty"`
	FullName         string                                               `json:"full_name"`
	Function         *DataSourceFeatureEngineeringFeatureFunction         `json:"function,omitempty"`
	Inputs           []string                                             `json:"inputs,omitempty"`
	LineageContext   *DataSourceFeatureEngineeringFeatureLineageContext   `json:"lineage_context,omitempty"`
	ProviderConfig   *DataSourceFeatureEngineeringFeatureProviderConfig   `json:"provider_config,omitempty"`
	Source           *DataSourceFeatureEngineeringFeatureSource           `json:"source,omitempty"`
	TimeWindow       *DataSourceFeatureEngineeringFeatureTimeWindow       `json:"time_window,omitempty"`
	TimeseriesColumn *DataSourceFeatureEngineeringFeatureTimeseriesColumn `json:"timeseries_column,omitempty"`
}
