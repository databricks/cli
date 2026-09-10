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

type DataSourceFeatureEngineeringFeatureFunctionAggregationFunctionFirstDistinct struct {
	Input string `json:"input"`
	N     int    `json:"n"`
}

type DataSourceFeatureEngineeringFeatureFunctionAggregationFunctionFirstN struct {
	Input string `json:"input"`
	N     int    `json:"n"`
}

type DataSourceFeatureEngineeringFeatureFunctionAggregationFunctionLast struct {
	Input string `json:"input"`
}

type DataSourceFeatureEngineeringFeatureFunctionAggregationFunctionLastDistinct struct {
	Input string `json:"input"`
	N     int    `json:"n"`
}

type DataSourceFeatureEngineeringFeatureFunctionAggregationFunctionLastN struct {
	Input string `json:"input"`
	N     int    `json:"n"`
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

type DataSourceFeatureEngineeringFeatureFunctionAggregationFunctionTimeWindowRolling struct {
	Delay          string `json:"delay,omitempty"`
	WindowDuration string `json:"window_duration,omitempty"`
}

type DataSourceFeatureEngineeringFeatureFunctionAggregationFunctionTimeWindowSawtooth struct {
	Delay          string `json:"delay,omitempty"`
	WindowDuration string `json:"window_duration,omitempty"`
}

type DataSourceFeatureEngineeringFeatureFunctionAggregationFunctionTimeWindowSliding struct {
	Delay          string `json:"delay,omitempty"`
	Offset         string `json:"offset,omitempty"`
	SlideDuration  string `json:"slide_duration"`
	WindowDuration string `json:"window_duration,omitempty"`
}

type DataSourceFeatureEngineeringFeatureFunctionAggregationFunctionTimeWindowTumbling struct {
	Delay          string `json:"delay,omitempty"`
	Offset         string `json:"offset,omitempty"`
	WindowDuration string `json:"window_duration"`
}

type DataSourceFeatureEngineeringFeatureFunctionAggregationFunctionTimeWindow struct {
	Rolling   *DataSourceFeatureEngineeringFeatureFunctionAggregationFunctionTimeWindowRolling  `json:"rolling,omitempty"`
	Sawtooth  *DataSourceFeatureEngineeringFeatureFunctionAggregationFunctionTimeWindowSawtooth `json:"sawtooth,omitempty"`
	Sliding   *DataSourceFeatureEngineeringFeatureFunctionAggregationFunctionTimeWindowSliding  `json:"sliding,omitempty"`
	StartTime string                                                                            `json:"start_time,omitempty"`
	Tumbling  *DataSourceFeatureEngineeringFeatureFunctionAggregationFunctionTimeWindowTumbling `json:"tumbling,omitempty"`
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
	FirstDistinct       *DataSourceFeatureEngineeringFeatureFunctionAggregationFunctionFirstDistinct       `json:"first_distinct,omitempty"`
	FirstN              *DataSourceFeatureEngineeringFeatureFunctionAggregationFunctionFirstN              `json:"first_n,omitempty"`
	Last                *DataSourceFeatureEngineeringFeatureFunctionAggregationFunctionLast                `json:"last,omitempty"`
	LastDistinct        *DataSourceFeatureEngineeringFeatureFunctionAggregationFunctionLastDistinct        `json:"last_distinct,omitempty"`
	LastN               *DataSourceFeatureEngineeringFeatureFunctionAggregationFunctionLastN               `json:"last_n,omitempty"`
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

type DataSourceFeatureEngineeringFeatureFunctionCustomUdfInputBindings struct {
	Column    string `json:"column"`
	Parameter string `json:"parameter"`
}

type DataSourceFeatureEngineeringFeatureFunctionCustomUdf struct {
	FunctionPath  string                                                              `json:"function_path"`
	InputBindings []DataSourceFeatureEngineeringFeatureFunctionCustomUdfInputBindings `json:"input_bindings,omitempty"`
}

type DataSourceFeatureEngineeringFeatureFunction struct {
	AggregationFunction *DataSourceFeatureEngineeringFeatureFunctionAggregationFunction `json:"aggregation_function,omitempty"`
	ColumnSelection     *DataSourceFeatureEngineeringFeatureFunctionColumnSelection     `json:"column_selection,omitempty"`
	CustomUdf           *DataSourceFeatureEngineeringFeatureFunctionCustomUdf           `json:"custom_udf,omitempty"`
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
	DataframeSchema   string `json:"dataframe_schema,omitempty"`
	FilterCondition   string `json:"filter_condition,omitempty"`
	FullName          string `json:"full_name"`
	TransformationSql string `json:"transformation_sql,omitempty"`
}

type DataSourceFeatureEngineeringFeatureSourceKafkaSource struct {
	FilterCondition string `json:"filter_condition,omitempty"`
	Name            string `json:"name"`
}

type DataSourceFeatureEngineeringFeatureSourceLateness struct {
	SettlingDelay string `json:"settling_delay,omitempty"`
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

type DataSourceFeatureEngineeringFeatureSourceStreamSource struct {
	DataframeSchema   string `json:"dataframe_schema,omitempty"`
	FilterCondition   string `json:"filter_condition,omitempty"`
	FullName          string `json:"full_name"`
	TransformationSql string `json:"transformation_sql,omitempty"`
}

type DataSourceFeatureEngineeringFeatureSource struct {
	DeltaTableSource *DataSourceFeatureEngineeringFeatureSourceDeltaTableSource `json:"delta_table_source,omitempty"`
	KafkaSource      *DataSourceFeatureEngineeringFeatureSourceKafkaSource      `json:"kafka_source,omitempty"`
	Lateness         *DataSourceFeatureEngineeringFeatureSourceLateness         `json:"lateness,omitempty"`
	RequestSource    *DataSourceFeatureEngineeringFeatureSourceRequestSource    `json:"request_source,omitempty"`
	StreamSource     *DataSourceFeatureEngineeringFeatureSourceStreamSource     `json:"stream_source,omitempty"`
}

type DataSourceFeatureEngineeringFeatureTimeseriesColumn struct {
	Name string `json:"name"`
}

type DataSourceFeatureEngineeringFeature struct {
	CatalogName      string                                               `json:"catalog_name,omitempty"`
	CreatedAt        string                                               `json:"created_at,omitempty"`
	CreatedBy        string                                               `json:"created_by,omitempty"`
	Description      string                                               `json:"description,omitempty"`
	Entities         []DataSourceFeatureEngineeringFeatureEntities        `json:"entities,omitempty"`
	FullName         string                                               `json:"full_name"`
	Function         *DataSourceFeatureEngineeringFeatureFunction         `json:"function,omitempty"`
	LineageContext   *DataSourceFeatureEngineeringFeatureLineageContext   `json:"lineage_context,omitempty"`
	Name             string                                               `json:"name,omitempty"`
	ProviderConfig   *DataSourceFeatureEngineeringFeatureProviderConfig   `json:"provider_config,omitempty"`
	SchemaName       string                                               `json:"schema_name,omitempty"`
	Source           *DataSourceFeatureEngineeringFeatureSource           `json:"source,omitempty"`
	TimeseriesColumn *DataSourceFeatureEngineeringFeatureTimeseriesColumn `json:"timeseries_column,omitempty"`
}
