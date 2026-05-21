// Generated from Databricks Terraform provider schema. DO NOT EDIT.

package schema

type ResourceFeatureEngineeringFeatureEntities struct {
	Name string `json:"name"`
}

type ResourceFeatureEngineeringFeatureFunctionAggregationFunctionApproxCountDistinct struct {
	Input      string `json:"input"`
	RelativeSd int    `json:"relative_sd,omitempty"`
}

type ResourceFeatureEngineeringFeatureFunctionAggregationFunctionApproxPercentile struct {
	Accuracy   int    `json:"accuracy,omitempty"`
	Input      string `json:"input"`
	Percentile int    `json:"percentile"`
}

type ResourceFeatureEngineeringFeatureFunctionAggregationFunctionAvg struct {
	Input string `json:"input"`
}

type ResourceFeatureEngineeringFeatureFunctionAggregationFunctionCountFunction struct {
	Input string `json:"input"`
}

type ResourceFeatureEngineeringFeatureFunctionAggregationFunctionFirst struct {
	Input string `json:"input"`
}

type ResourceFeatureEngineeringFeatureFunctionAggregationFunctionLast struct {
	Input string `json:"input"`
}

type ResourceFeatureEngineeringFeatureFunctionAggregationFunctionMax struct {
	Input string `json:"input"`
}

type ResourceFeatureEngineeringFeatureFunctionAggregationFunctionMin struct {
	Input string `json:"input"`
}

type ResourceFeatureEngineeringFeatureFunctionAggregationFunctionStddevPop struct {
	Input string `json:"input"`
}

type ResourceFeatureEngineeringFeatureFunctionAggregationFunctionStddevSamp struct {
	Input string `json:"input"`
}

type ResourceFeatureEngineeringFeatureFunctionAggregationFunctionSum struct {
	Input string `json:"input"`
}

type ResourceFeatureEngineeringFeatureFunctionAggregationFunctionTimeWindowContinuous struct {
	Offset         string `json:"offset,omitempty"`
	WindowDuration string `json:"window_duration"`
}

type ResourceFeatureEngineeringFeatureFunctionAggregationFunctionTimeWindowSliding struct {
	SlideDuration  string `json:"slide_duration"`
	WindowDuration string `json:"window_duration"`
}

type ResourceFeatureEngineeringFeatureFunctionAggregationFunctionTimeWindowTumbling struct {
	WindowDuration string `json:"window_duration"`
}

type ResourceFeatureEngineeringFeatureFunctionAggregationFunctionTimeWindow struct {
	Continuous *ResourceFeatureEngineeringFeatureFunctionAggregationFunctionTimeWindowContinuous `json:"continuous,omitempty"`
	Sliding    *ResourceFeatureEngineeringFeatureFunctionAggregationFunctionTimeWindowSliding    `json:"sliding,omitempty"`
	Tumbling   *ResourceFeatureEngineeringFeatureFunctionAggregationFunctionTimeWindowTumbling   `json:"tumbling,omitempty"`
}

type ResourceFeatureEngineeringFeatureFunctionAggregationFunctionVarPop struct {
	Input string `json:"input"`
}

type ResourceFeatureEngineeringFeatureFunctionAggregationFunctionVarSamp struct {
	Input string `json:"input"`
}

type ResourceFeatureEngineeringFeatureFunctionAggregationFunction struct {
	ApproxCountDistinct *ResourceFeatureEngineeringFeatureFunctionAggregationFunctionApproxCountDistinct `json:"approx_count_distinct,omitempty"`
	ApproxPercentile    *ResourceFeatureEngineeringFeatureFunctionAggregationFunctionApproxPercentile    `json:"approx_percentile,omitempty"`
	Avg                 *ResourceFeatureEngineeringFeatureFunctionAggregationFunctionAvg                 `json:"avg,omitempty"`
	CountFunction       *ResourceFeatureEngineeringFeatureFunctionAggregationFunctionCountFunction       `json:"count_function,omitempty"`
	First               *ResourceFeatureEngineeringFeatureFunctionAggregationFunctionFirst               `json:"first,omitempty"`
	Last                *ResourceFeatureEngineeringFeatureFunctionAggregationFunctionLast                `json:"last,omitempty"`
	Max                 *ResourceFeatureEngineeringFeatureFunctionAggregationFunctionMax                 `json:"max,omitempty"`
	Min                 *ResourceFeatureEngineeringFeatureFunctionAggregationFunctionMin                 `json:"min,omitempty"`
	StddevPop           *ResourceFeatureEngineeringFeatureFunctionAggregationFunctionStddevPop           `json:"stddev_pop,omitempty"`
	StddevSamp          *ResourceFeatureEngineeringFeatureFunctionAggregationFunctionStddevSamp          `json:"stddev_samp,omitempty"`
	Sum                 *ResourceFeatureEngineeringFeatureFunctionAggregationFunctionSum                 `json:"sum,omitempty"`
	TimeWindow          *ResourceFeatureEngineeringFeatureFunctionAggregationFunctionTimeWindow          `json:"time_window,omitempty"`
	VarPop              *ResourceFeatureEngineeringFeatureFunctionAggregationFunctionVarPop              `json:"var_pop,omitempty"`
	VarSamp             *ResourceFeatureEngineeringFeatureFunctionAggregationFunctionVarSamp             `json:"var_samp,omitempty"`
}

type ResourceFeatureEngineeringFeatureFunctionColumnSelection struct {
	Column string `json:"column"`
}

type ResourceFeatureEngineeringFeatureFunctionExtraParameters struct {
	Key   string `json:"key"`
	Value string `json:"value"`
}

type ResourceFeatureEngineeringFeatureFunction struct {
	AggregationFunction *ResourceFeatureEngineeringFeatureFunctionAggregationFunction `json:"aggregation_function,omitempty"`
	ColumnSelection     *ResourceFeatureEngineeringFeatureFunctionColumnSelection     `json:"column_selection,omitempty"`
	ExtraParameters     []ResourceFeatureEngineeringFeatureFunctionExtraParameters    `json:"extra_parameters,omitempty"`
	FunctionType        string                                                        `json:"function_type,omitempty"`
}

type ResourceFeatureEngineeringFeatureLineageContextJobContext struct {
	JobId    int `json:"job_id,omitempty"`
	JobRunId int `json:"job_run_id,omitempty"`
}

type ResourceFeatureEngineeringFeatureLineageContext struct {
	JobContext *ResourceFeatureEngineeringFeatureLineageContextJobContext `json:"job_context,omitempty"`
	NotebookId int                                                        `json:"notebook_id,omitempty"`
}

type ResourceFeatureEngineeringFeatureProviderConfig struct {
	WorkspaceId string `json:"workspace_id,omitempty"`
}

type ResourceFeatureEngineeringFeatureSourceDeltaTableSource struct {
	DataframeSchema   string   `json:"dataframe_schema,omitempty"`
	EntityColumns     []string `json:"entity_columns,omitempty"`
	FilterCondition   string   `json:"filter_condition,omitempty"`
	FullName          string   `json:"full_name"`
	TimeseriesColumn  string   `json:"timeseries_column,omitempty"`
	TransformationSql string   `json:"transformation_sql,omitempty"`
}

type ResourceFeatureEngineeringFeatureSourceKafkaSourceEntityColumnIdentifiers struct {
	VariantExprPath string `json:"variant_expr_path"`
}

type ResourceFeatureEngineeringFeatureSourceKafkaSourceTimeseriesColumnIdentifier struct {
	VariantExprPath string `json:"variant_expr_path"`
}

type ResourceFeatureEngineeringFeatureSourceKafkaSource struct {
	EntityColumnIdentifiers    []ResourceFeatureEngineeringFeatureSourceKafkaSourceEntityColumnIdentifiers   `json:"entity_column_identifiers,omitempty"`
	FilterCondition            string                                                                        `json:"filter_condition,omitempty"`
	Name                       string                                                                        `json:"name"`
	TimeseriesColumnIdentifier *ResourceFeatureEngineeringFeatureSourceKafkaSourceTimeseriesColumnIdentifier `json:"timeseries_column_identifier,omitempty"`
}

type ResourceFeatureEngineeringFeatureSourceRequestSourceFlatSchemaFields struct {
	DataType string `json:"data_type"`
	Name     string `json:"name"`
}

type ResourceFeatureEngineeringFeatureSourceRequestSourceFlatSchema struct {
	Fields []ResourceFeatureEngineeringFeatureSourceRequestSourceFlatSchemaFields `json:"fields,omitempty"`
}

type ResourceFeatureEngineeringFeatureSourceRequestSource struct {
	FlatSchema *ResourceFeatureEngineeringFeatureSourceRequestSourceFlatSchema `json:"flat_schema,omitempty"`
}

type ResourceFeatureEngineeringFeatureSource struct {
	DeltaTableSource *ResourceFeatureEngineeringFeatureSourceDeltaTableSource `json:"delta_table_source,omitempty"`
	KafkaSource      *ResourceFeatureEngineeringFeatureSourceKafkaSource      `json:"kafka_source,omitempty"`
	RequestSource    *ResourceFeatureEngineeringFeatureSourceRequestSource    `json:"request_source,omitempty"`
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

type ResourceFeatureEngineeringFeatureTimeseriesColumn struct {
	Name string `json:"name"`
}

type ResourceFeatureEngineeringFeature struct {
	Description      string                                             `json:"description,omitempty"`
	Entities         []ResourceFeatureEngineeringFeatureEntities        `json:"entities,omitempty"`
	FilterCondition  string                                             `json:"filter_condition,omitempty"`
	FullName         string                                             `json:"full_name"`
	Function         *ResourceFeatureEngineeringFeatureFunction         `json:"function,omitempty"`
	Inputs           []string                                           `json:"inputs,omitempty"`
	LineageContext   *ResourceFeatureEngineeringFeatureLineageContext   `json:"lineage_context,omitempty"`
	ProviderConfig   *ResourceFeatureEngineeringFeatureProviderConfig   `json:"provider_config,omitempty"`
	Source           *ResourceFeatureEngineeringFeatureSource           `json:"source,omitempty"`
	TimeWindow       *ResourceFeatureEngineeringFeatureTimeWindow       `json:"time_window,omitempty"`
	TimeseriesColumn *ResourceFeatureEngineeringFeatureTimeseriesColumn `json:"timeseries_column,omitempty"`
}
