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

type ResourceFeatureEngineeringFeatureFunctionAggregationFunctionFirstDistinct struct {
	Input string `json:"input"`
	N     int    `json:"n"`
}

type ResourceFeatureEngineeringFeatureFunctionAggregationFunctionFirstN struct {
	Input string `json:"input"`
	N     int    `json:"n"`
}

type ResourceFeatureEngineeringFeatureFunctionAggregationFunctionLast struct {
	Input string `json:"input"`
}

type ResourceFeatureEngineeringFeatureFunctionAggregationFunctionLastDistinct struct {
	Input string `json:"input"`
	N     int    `json:"n"`
}

type ResourceFeatureEngineeringFeatureFunctionAggregationFunctionLastN struct {
	Input string `json:"input"`
	N     int    `json:"n"`
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

type ResourceFeatureEngineeringFeatureFunctionAggregationFunctionTimeWindowRolling struct {
	Delay          string `json:"delay,omitempty"`
	WindowDuration string `json:"window_duration,omitempty"`
}

type ResourceFeatureEngineeringFeatureFunctionAggregationFunctionTimeWindowSawtooth struct {
	Delay          string `json:"delay,omitempty"`
	WindowDuration string `json:"window_duration,omitempty"`
}

type ResourceFeatureEngineeringFeatureFunctionAggregationFunctionTimeWindowSliding struct {
	Delay          string `json:"delay,omitempty"`
	Offset         string `json:"offset,omitempty"`
	SlideDuration  string `json:"slide_duration"`
	WindowDuration string `json:"window_duration,omitempty"`
}

type ResourceFeatureEngineeringFeatureFunctionAggregationFunctionTimeWindowTumbling struct {
	Delay          string `json:"delay,omitempty"`
	Offset         string `json:"offset,omitempty"`
	WindowDuration string `json:"window_duration"`
}

type ResourceFeatureEngineeringFeatureFunctionAggregationFunctionTimeWindow struct {
	Rolling   *ResourceFeatureEngineeringFeatureFunctionAggregationFunctionTimeWindowRolling  `json:"rolling,omitempty"`
	Sawtooth  *ResourceFeatureEngineeringFeatureFunctionAggregationFunctionTimeWindowSawtooth `json:"sawtooth,omitempty"`
	Sliding   *ResourceFeatureEngineeringFeatureFunctionAggregationFunctionTimeWindowSliding  `json:"sliding,omitempty"`
	StartTime string                                                                          `json:"start_time,omitempty"`
	Tumbling  *ResourceFeatureEngineeringFeatureFunctionAggregationFunctionTimeWindowTumbling `json:"tumbling,omitempty"`
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
	FirstDistinct       *ResourceFeatureEngineeringFeatureFunctionAggregationFunctionFirstDistinct       `json:"first_distinct,omitempty"`
	FirstN              *ResourceFeatureEngineeringFeatureFunctionAggregationFunctionFirstN              `json:"first_n,omitempty"`
	Last                *ResourceFeatureEngineeringFeatureFunctionAggregationFunctionLast                `json:"last,omitempty"`
	LastDistinct        *ResourceFeatureEngineeringFeatureFunctionAggregationFunctionLastDistinct        `json:"last_distinct,omitempty"`
	LastN               *ResourceFeatureEngineeringFeatureFunctionAggregationFunctionLastN               `json:"last_n,omitempty"`
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

type ResourceFeatureEngineeringFeatureFunctionCustomUdfInputBindings struct {
	Column    string `json:"column"`
	Parameter string `json:"parameter"`
}

type ResourceFeatureEngineeringFeatureFunctionCustomUdf struct {
	FunctionPath  string                                                            `json:"function_path"`
	InputBindings []ResourceFeatureEngineeringFeatureFunctionCustomUdfInputBindings `json:"input_bindings,omitempty"`
}

type ResourceFeatureEngineeringFeatureFunction struct {
	AggregationFunction *ResourceFeatureEngineeringFeatureFunctionAggregationFunction `json:"aggregation_function,omitempty"`
	ColumnSelection     *ResourceFeatureEngineeringFeatureFunctionColumnSelection     `json:"column_selection,omitempty"`
	CustomUdf           *ResourceFeatureEngineeringFeatureFunctionCustomUdf           `json:"custom_udf,omitempty"`
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
	DataframeSchema   string `json:"dataframe_schema,omitempty"`
	FilterCondition   string `json:"filter_condition,omitempty"`
	FullName          string `json:"full_name"`
	TransformationSql string `json:"transformation_sql,omitempty"`
}

type ResourceFeatureEngineeringFeatureSourceKafkaSource struct {
	FilterCondition string `json:"filter_condition,omitempty"`
	Name            string `json:"name"`
}

type ResourceFeatureEngineeringFeatureSourceLateness struct {
	SettlingDelay string `json:"settling_delay,omitempty"`
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

type ResourceFeatureEngineeringFeatureSourceStreamSource struct {
	DataframeSchema   string `json:"dataframe_schema,omitempty"`
	FilterCondition   string `json:"filter_condition,omitempty"`
	FullName          string `json:"full_name"`
	TransformationSql string `json:"transformation_sql,omitempty"`
}

type ResourceFeatureEngineeringFeatureSource struct {
	DeltaTableSource *ResourceFeatureEngineeringFeatureSourceDeltaTableSource `json:"delta_table_source,omitempty"`
	KafkaSource      *ResourceFeatureEngineeringFeatureSourceKafkaSource      `json:"kafka_source,omitempty"`
	Lateness         *ResourceFeatureEngineeringFeatureSourceLateness         `json:"lateness,omitempty"`
	RequestSource    *ResourceFeatureEngineeringFeatureSourceRequestSource    `json:"request_source,omitempty"`
	StreamSource     *ResourceFeatureEngineeringFeatureSourceStreamSource     `json:"stream_source,omitempty"`
}

type ResourceFeatureEngineeringFeatureTimeseriesColumn struct {
	Name string `json:"name"`
}

type ResourceFeatureEngineeringFeature struct {
	CatalogName      string                                             `json:"catalog_name,omitempty"`
	CreatedAt        string                                             `json:"created_at,omitempty"`
	CreatedBy        string                                             `json:"created_by,omitempty"`
	Description      string                                             `json:"description,omitempty"`
	Entities         []ResourceFeatureEngineeringFeatureEntities        `json:"entities,omitempty"`
	FullName         string                                             `json:"full_name"`
	Function         *ResourceFeatureEngineeringFeatureFunction         `json:"function,omitempty"`
	LineageContext   *ResourceFeatureEngineeringFeatureLineageContext   `json:"lineage_context,omitempty"`
	Name             string                                             `json:"name,omitempty"`
	ProviderConfig   *ResourceFeatureEngineeringFeatureProviderConfig   `json:"provider_config,omitempty"`
	SchemaName       string                                             `json:"schema_name,omitempty"`
	Source           *ResourceFeatureEngineeringFeatureSource           `json:"source,omitempty"`
	TimeseriesColumn *ResourceFeatureEngineeringFeatureTimeseriesColumn `json:"timeseries_column,omitempty"`
}
