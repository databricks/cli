// Code ported from github.com/databricks/databricks-sdk-go /service/catalog/model.go.

package qualitymonitors

import (
	"fmt"
)

type CancelRefreshRequest struct {
	RefreshId int64 `json:"-" url:"-"`
	// UC table name in format `catalog.schema.table_name`. table_name is case
	// insensitive and spaces are disallowed.
	TableName string `json:"-" url:"-"`
}

type CreateMonitor struct {
	// [Create:REQ Update:IGN] Field for specifying the absolute path to a
	// custom directory to store data-monitoring assets. Normally prepopulated
	// to a default user location via UI and Python APIs.
	AssetsDir string `json:"assets_dir"`
	// [Create:OPT Update:OPT] Baseline table name. Baseline data is used to
	// compute drift from the data in the monitored `table_name`. The baseline
	// table and the monitored table shall have the same schema.
	BaselineTableName string `json:"baseline_table_name,omitempty"`
	// [Create:OPT Update:OPT] Custom metrics.
	CustomMetrics []MonitorMetric `json:"custom_metrics,omitempty"`
	// [Create:OPT Update:OPT] Data classification related config.
	DataClassificationConfig *MonitorDataClassificationConfig `json:"data_classification_config,omitempty"`

	InferenceLog *MonitorInferenceLog `json:"inference_log,omitempty"`
	// [Create:ERR Update:IGN] The latest error message for a monitor failure.
	LatestMonitorFailureMsg string `json:"latest_monitor_failure_msg,omitempty"`
	// [Create:OPT Update:OPT] Field for specifying notification settings.
	Notifications *MonitorNotifications `json:"notifications,omitempty"`
	// [Create:REQ Update:REQ] Schema where output tables are created. Needs to
	// be in 2-level format {catalog}.{schema}
	OutputSchemaName string `json:"output_schema_name"`
	// [Create:OPT Update:OPT] The monitor schedule.
	Schedule *MonitorCronSchedule `json:"schedule,omitempty"`
	// Whether to skip creating a default dashboard summarizing data quality
	// metrics.
	SkipBuiltinDashboard bool `json:"skip_builtin_dashboard,omitempty"`
	// [Create:OPT Update:OPT] List of column expressions to slice data with for
	// targeted analysis. The data is grouped by each expression independently,
	// resulting in a separate slice for each predicate and its complements. For
	// example `slicing_exprs=[“col_1”, “col_2 > 10”]` will generate the
	// following slices: two slices for `col_2 > 10` (True and False), and one
	// slice per unique value in `col1`. For high-cardinality columns, only the
	// top 100 unique values by frequency will generate slices.
	SlicingExprs []string `json:"slicing_exprs,omitempty"`
	// Configuration for monitoring snapshot tables.
	Snapshot *MonitorSnapshot `json:"snapshot,omitempty"`
	// UC table name in format `catalog.schema.table_name`. This field
	// corresponds to the {full_table_name_arg} arg in the endpoint path.
	TableName string `json:"-" url:"-"`
	// Configuration for monitoring time series tables.
	TimeSeries *MonitorTimeSeries `json:"time_series,omitempty"`
	// Optional argument to specify the warehouse for dashboard creation. If not
	// specified, the first running warehouse will be used.
	WarehouseId string `json:"warehouse_id,omitempty"`
}

type DeleteMonitorResponse struct{}

type DeleteQualityMonitorRequest struct {
	// UC table name in format `catalog.schema.table_name`. This field
	// corresponds to the {full_table_name_arg} arg in the endpoint path.
	TableName string `json:"-" url:"-"`
}

type GetQualityMonitorRequest struct {
	// UC table name in format `catalog.schema.table_name`. This field
	// corresponds to the {full_table_name_arg} arg in the endpoint path.
	TableName string `json:"-" url:"-"`
}

type GetRefreshRequest struct {
	// ID of the refresh.
	RefreshId int64 `json:"-" url:"-"`
	// Full name of the table.
	TableName string `json:"-" url:"-"`
}

type ListRefreshesRequest struct {
	// UC table name in format `catalog.schema.table_name`. table_name is case
	// insensitive and spaces are disallowed.
	TableName string `json:"-" url:"-"`
}

type MonitorCronSchedule struct {
	// Read only field that indicates whether a schedule is paused or not.
	PauseStatus MonitorCronSchedulePauseStatus `json:"pause_status,omitempty"`
	// The expression that determines when to run the monitor. See [examples].
	//
	// [examples]: https://www.quartz-scheduler.org/documentation/quartz-2.3.0/tutorials/crontrigger.html
	QuartzCronExpression string `json:"quartz_cron_expression"`
	// The timezone id (e.g., ``PST``) in which to evaluate the quartz
	// expression.
	TimezoneId string `json:"timezone_id"`
}

// Source link:
// https://src.dev.databricks.com/databricks/universe/-/blob/elastic-spark-common/api/messages/schedule.proto
// Monitoring workflow schedule pause status.
type MonitorCronSchedulePauseStatus string

const MonitorCronSchedulePauseStatusPaused MonitorCronSchedulePauseStatus = `PAUSED`

const MonitorCronSchedulePauseStatusUnpaused MonitorCronSchedulePauseStatus = `UNPAUSED`

const MonitorCronSchedulePauseStatusUnspecified MonitorCronSchedulePauseStatus = `UNSPECIFIED`

// String representation for [fmt.Print]
func (f *MonitorCronSchedulePauseStatus) String() string {
	return string(*f)
}

// Set raw string value and validate it against allowed values
func (f *MonitorCronSchedulePauseStatus) Set(v string) error {
	switch v {
	case `PAUSED`, `UNPAUSED`, `UNSPECIFIED`:
		*f = MonitorCronSchedulePauseStatus(v)
		return nil
	default:
		return fmt.Errorf(`value "%s" is not one of "PAUSED", "UNPAUSED", "UNSPECIFIED"`, v)
	}
}

// Values returns all possible values for MonitorCronSchedulePauseStatus.
//
// There is no guarantee on the order of the values in the slice.
func (f *MonitorCronSchedulePauseStatus) Values() []MonitorCronSchedulePauseStatus {
	return []MonitorCronSchedulePauseStatus{
		MonitorCronSchedulePauseStatusPaused,
		MonitorCronSchedulePauseStatusUnpaused,
		MonitorCronSchedulePauseStatusUnspecified,
	}
}

// Type always returns MonitorCronSchedulePauseStatus to satisfy [pflag.Value] interface
func (f *MonitorCronSchedulePauseStatus) Type() string {
	return "MonitorCronSchedulePauseStatus"
}

// Data classification related configuration.
type MonitorDataClassificationConfig struct {
	// Whether to enable data classification.
	Enabled bool `json:"enabled,omitempty"`
}

type MonitorDestination struct {
	// The list of email addresses to send the notification to. A maximum of 5
	// email addresses is supported.
	EmailAddresses []string `json:"email_addresses,omitempty"`
}

type MonitorInferenceLog struct {
	// List of granularities to use when aggregating data into time windows
	// based on their timestamp.
	Granularities []string `json:"granularities"`
	// Column for the label.
	LabelCol string `json:"label_col,omitempty"`
	// Column for the model identifier.
	ModelIdCol string `json:"model_id_col"`
	// Column for the prediction.
	PredictionCol string `json:"prediction_col"`
	// Column for prediction probabilities
	PredictionProbaCol string `json:"prediction_proba_col,omitempty"`
	// Problem type the model aims to solve.
	ProblemType MonitorInferenceLogProblemType `json:"problem_type"`
	// Column for the timestamp.
	TimestampCol string `json:"timestamp_col"`
}

type MonitorInferenceLogProblemType string

const MonitorInferenceLogProblemTypeProblemTypeClassification MonitorInferenceLogProblemType = `PROBLEM_TYPE_CLASSIFICATION`

const MonitorInferenceLogProblemTypeProblemTypeRegression MonitorInferenceLogProblemType = `PROBLEM_TYPE_REGRESSION`

// String representation for [fmt.Print]
func (f *MonitorInferenceLogProblemType) String() string {
	return string(*f)
}

// Set raw string value and validate it against allowed values
func (f *MonitorInferenceLogProblemType) Set(v string) error {
	switch v {
	case `PROBLEM_TYPE_CLASSIFICATION`, `PROBLEM_TYPE_REGRESSION`:
		*f = MonitorInferenceLogProblemType(v)
		return nil
	default:
		return fmt.Errorf(`value "%s" is not one of "PROBLEM_TYPE_CLASSIFICATION", "PROBLEM_TYPE_REGRESSION"`, v)
	}
}

// Values returns all possible values for MonitorInferenceLogProblemType.
//
// There is no guarantee on the order of the values in the slice.
func (f *MonitorInferenceLogProblemType) Values() []MonitorInferenceLogProblemType {
	return []MonitorInferenceLogProblemType{
		MonitorInferenceLogProblemTypeProblemTypeClassification,
		MonitorInferenceLogProblemTypeProblemTypeRegression,
	}
}

// Type always returns MonitorInferenceLogProblemType to satisfy [pflag.Value] interface
func (f *MonitorInferenceLogProblemType) Type() string {
	return "MonitorInferenceLogProblemType"
}

type MonitorInfo struct {
	// [Create:REQ Update:IGN] Field for specifying the absolute path to a
	// custom directory to store data-monitoring assets. Normally prepopulated
	// to a default user location via UI and Python APIs.
	AssetsDir string `json:"assets_dir,omitempty"`
	// [Create:OPT Update:OPT] Baseline table name. Baseline data is used to
	// compute drift from the data in the monitored `table_name`. The baseline
	// table and the monitored table shall have the same schema.
	BaselineTableName string `json:"baseline_table_name,omitempty"`
	// [Create:OPT Update:OPT] Custom metrics.
	CustomMetrics []MonitorMetric `json:"custom_metrics,omitempty"`
	// [Create:ERR Update:OPT] Id of dashboard that visualizes the computed
	// metrics. This can be empty if the monitor is in PENDING state.
	DashboardId string `json:"dashboard_id,omitempty"`
	// [Create:OPT Update:OPT] Data classification related config.
	DataClassificationConfig *MonitorDataClassificationConfig `json:"data_classification_config,omitempty"`
	// [Create:ERR Update:IGN] Table that stores drift metrics data. Format:
	// `catalog.schema.table_name`.
	DriftMetricsTableName string `json:"drift_metrics_table_name"`

	InferenceLog *MonitorInferenceLog `json:"inference_log,omitempty"`
	// [Create:ERR Update:IGN] The latest error message for a monitor failure.
	LatestMonitorFailureMsg string `json:"latest_monitor_failure_msg,omitempty"`
	// [Create:ERR Update:IGN] Represents the current monitor configuration
	// version in use. The version will be represented in a numeric fashion
	// (1,2,3...). The field has flexibility to take on negative values, which
	// can indicate corrupted monitor_version numbers.
	MonitorVersion int64 `json:"monitor_version"`
	// [Create:OPT Update:OPT] Field for specifying notification settings.
	Notifications *MonitorNotifications `json:"notifications,omitempty"`
	// [Create:REQ Update:REQ] Schema where output tables are created. Needs to
	// be in 2-level format {catalog}.{schema}
	OutputSchemaName string `json:"output_schema_name"`
	// [Create:ERR Update:IGN] Table that stores profile metrics data. Format:
	// `catalog.schema.table_name`.
	ProfileMetricsTableName string `json:"profile_metrics_table_name"`
	// [Create:OPT Update:OPT] The monitor schedule.
	Schedule *MonitorCronSchedule `json:"schedule,omitempty"`
	// [Create:OPT Update:OPT] List of column expressions to slice data with for
	// targeted analysis. The data is grouped by each expression independently,
	// resulting in a separate slice for each predicate and its complements. For
	// example `slicing_exprs=[“col_1”, “col_2 > 10”]` will generate the
	// following slices: two slices for `col_2 > 10` (True and False), and one
	// slice per unique value in `col1`. For high-cardinality columns, only the
	// top 100 unique values by frequency will generate slices.
	SlicingExprs []string `json:"slicing_exprs,omitempty"`
	// Configuration for monitoring snapshot tables.
	Snapshot *MonitorSnapshot `json:"snapshot,omitempty"`
	// [Create:ERR Update:IGN] The monitor status.
	Status MonitorInfoStatus `json:"status"`
	// [Create:ERR Update:IGN] UC table to monitor. Format:
	// `catalog.schema.table_name`
	TableName string `json:"table_name"`
	// Configuration for monitoring time series tables.
	TimeSeries *MonitorTimeSeries `json:"time_series,omitempty"`
}

type MonitorInfoStatus string

const MonitorInfoStatusMonitorStatusActive MonitorInfoStatus = `MONITOR_STATUS_ACTIVE`

const MonitorInfoStatusMonitorStatusDeletePending MonitorInfoStatus = `MONITOR_STATUS_DELETE_PENDING`

const MonitorInfoStatusMonitorStatusError MonitorInfoStatus = `MONITOR_STATUS_ERROR`

const MonitorInfoStatusMonitorStatusFailed MonitorInfoStatus = `MONITOR_STATUS_FAILED`

const MonitorInfoStatusMonitorStatusPending MonitorInfoStatus = `MONITOR_STATUS_PENDING`

// String representation for [fmt.Print]
func (f *MonitorInfoStatus) String() string {
	return string(*f)
}

// Set raw string value and validate it against allowed values
func (f *MonitorInfoStatus) Set(v string) error {
	switch v {
	case `MONITOR_STATUS_ACTIVE`, `MONITOR_STATUS_DELETE_PENDING`, `MONITOR_STATUS_ERROR`, `MONITOR_STATUS_FAILED`, `MONITOR_STATUS_PENDING`:
		*f = MonitorInfoStatus(v)
		return nil
	default:
		return fmt.Errorf(`value "%s" is not one of "MONITOR_STATUS_ACTIVE", "MONITOR_STATUS_DELETE_PENDING", "MONITOR_STATUS_ERROR", "MONITOR_STATUS_FAILED", "MONITOR_STATUS_PENDING"`, v)
	}
}

// Values returns all possible values for MonitorInfoStatus.
//
// There is no guarantee on the order of the values in the slice.
func (f *MonitorInfoStatus) Values() []MonitorInfoStatus {
	return []MonitorInfoStatus{
		MonitorInfoStatusMonitorStatusActive,
		MonitorInfoStatusMonitorStatusDeletePending,
		MonitorInfoStatusMonitorStatusError,
		MonitorInfoStatusMonitorStatusFailed,
		MonitorInfoStatusMonitorStatusPending,
	}
}

// Type always returns MonitorInfoStatus to satisfy [pflag.Value] interface
func (f *MonitorInfoStatus) Type() string {
	return "MonitorInfoStatus"
}

// Custom metric definition.
type MonitorMetric struct {
	// Jinja template for a SQL expression that specifies how to compute the
	// metric. See [create metric definition].
	//
	// [create metric definition]: https://docs.databricks.com/en/lakehouse-monitoring/custom-metrics.html#create-definition
	Definition string `json:"definition"`
	// A list of column names in the input table the metric should be computed
	// for. Can use ``":table"`` to indicate that the metric needs information
	// from multiple columns.
	InputColumns []string `json:"input_columns"`
	// Name of the metric in the output tables.
	Name string `json:"name"`
	// The output type of the custom metric.
	OutputDataType string `json:"output_data_type"`
	// Can only be one of ``"CUSTOM_METRIC_TYPE_AGGREGATE"``,
	// ``"CUSTOM_METRIC_TYPE_DERIVED"``, or ``"CUSTOM_METRIC_TYPE_DRIFT"``. The
	// ``"CUSTOM_METRIC_TYPE_AGGREGATE"`` and ``"CUSTOM_METRIC_TYPE_DERIVED"``
	// metrics are computed on a single table, whereas the
	// ``"CUSTOM_METRIC_TYPE_DRIFT"`` compare metrics across baseline and input
	// table, or across the two consecutive time windows. -
	// CUSTOM_METRIC_TYPE_AGGREGATE: only depend on the existing columns in your
	// table - CUSTOM_METRIC_TYPE_DERIVED: depend on previously computed
	// aggregate metrics - CUSTOM_METRIC_TYPE_DRIFT: depend on previously
	// computed aggregate or derived metrics
	Type MonitorMetricType `json:"type"`
}

// Can only be one of “\"CUSTOM_METRIC_TYPE_AGGREGATE\"“,
// “\"CUSTOM_METRIC_TYPE_DERIVED\"“, or “\"CUSTOM_METRIC_TYPE_DRIFT\"“. The
// “\"CUSTOM_METRIC_TYPE_AGGREGATE\"“ and “\"CUSTOM_METRIC_TYPE_DERIVED\"“
// metrics are computed on a single table, whereas the
// “\"CUSTOM_METRIC_TYPE_DRIFT\"“ compare metrics across baseline and input
// table, or across the two consecutive time windows. -
// CUSTOM_METRIC_TYPE_AGGREGATE: only depend on the existing columns in your
// table - CUSTOM_METRIC_TYPE_DERIVED: depend on previously computed aggregate
// metrics - CUSTOM_METRIC_TYPE_DRIFT: depend on previously computed aggregate
// or derived metrics
type MonitorMetricType string

const MonitorMetricTypeCustomMetricTypeAggregate MonitorMetricType = `CUSTOM_METRIC_TYPE_AGGREGATE`

const MonitorMetricTypeCustomMetricTypeDerived MonitorMetricType = `CUSTOM_METRIC_TYPE_DERIVED`

const MonitorMetricTypeCustomMetricTypeDrift MonitorMetricType = `CUSTOM_METRIC_TYPE_DRIFT`

// String representation for [fmt.Print]
func (f *MonitorMetricType) String() string {
	return string(*f)
}

// Set raw string value and validate it against allowed values
func (f *MonitorMetricType) Set(v string) error {
	switch v {
	case `CUSTOM_METRIC_TYPE_AGGREGATE`, `CUSTOM_METRIC_TYPE_DERIVED`, `CUSTOM_METRIC_TYPE_DRIFT`:
		*f = MonitorMetricType(v)
		return nil
	default:
		return fmt.Errorf(`value "%s" is not one of "CUSTOM_METRIC_TYPE_AGGREGATE", "CUSTOM_METRIC_TYPE_DERIVED", "CUSTOM_METRIC_TYPE_DRIFT"`, v)
	}
}

// Values returns all possible values for MonitorMetricType.
//
// There is no guarantee on the order of the values in the slice.
func (f *MonitorMetricType) Values() []MonitorMetricType {
	return []MonitorMetricType{
		MonitorMetricTypeCustomMetricTypeAggregate,
		MonitorMetricTypeCustomMetricTypeDerived,
		MonitorMetricTypeCustomMetricTypeDrift,
	}
}

// Type always returns MonitorMetricType to satisfy [pflag.Value] interface
func (f *MonitorMetricType) Type() string {
	return "MonitorMetricType"
}

type MonitorNotifications struct {
	// Destinations to send notifications on failure/timeout.
	OnFailure *MonitorDestination `json:"on_failure,omitempty"`
	// Destinations to send notifications on new classification tag detected.
	OnNewClassificationTagDetected *MonitorDestination `json:"on_new_classification_tag_detected,omitempty"`
}

type MonitorRefreshInfo struct {
	// Time at which refresh operation completed (milliseconds since 1/1/1970
	// UTC).
	EndTimeMs int64 `json:"end_time_ms,omitempty"`
	// An optional message to give insight into the current state of the job
	// (e.g. FAILURE messages).
	Message string `json:"message,omitempty"`
	// Unique id of the refresh operation.
	RefreshId int64 `json:"refresh_id"`
	// Time at which refresh operation was initiated (milliseconds since
	// 1/1/1970 UTC).
	StartTimeMs int64 `json:"start_time_ms"`
	// The current state of the refresh.
	State MonitorRefreshInfoState `json:"state"`
	// The method by which the refresh was triggered.
	Trigger MonitorRefreshInfoTrigger `json:"trigger,omitempty"`
}

// The current state of the refresh.
type MonitorRefreshInfoState string

const MonitorRefreshInfoStateCanceled MonitorRefreshInfoState = `CANCELED`

const MonitorRefreshInfoStateFailed MonitorRefreshInfoState = `FAILED`

const MonitorRefreshInfoStatePending MonitorRefreshInfoState = `PENDING`

const MonitorRefreshInfoStateRunning MonitorRefreshInfoState = `RUNNING`

const MonitorRefreshInfoStateSuccess MonitorRefreshInfoState = `SUCCESS`

const MonitorRefreshInfoStateUnknown MonitorRefreshInfoState = `UNKNOWN`

// String representation for [fmt.Print]
func (f *MonitorRefreshInfoState) String() string {
	return string(*f)
}

// Set raw string value and validate it against allowed values
func (f *MonitorRefreshInfoState) Set(v string) error {
	switch v {
	case `CANCELED`, `FAILED`, `PENDING`, `RUNNING`, `SUCCESS`, `UNKNOWN`:
		*f = MonitorRefreshInfoState(v)
		return nil
	default:
		return fmt.Errorf(`value "%s" is not one of "CANCELED", "FAILED", "PENDING", "RUNNING", "SUCCESS", "UNKNOWN"`, v)
	}
}

// Values returns all possible values for MonitorRefreshInfoState.
//
// There is no guarantee on the order of the values in the slice.
func (f *MonitorRefreshInfoState) Values() []MonitorRefreshInfoState {
	return []MonitorRefreshInfoState{
		MonitorRefreshInfoStateCanceled,
		MonitorRefreshInfoStateFailed,
		MonitorRefreshInfoStatePending,
		MonitorRefreshInfoStateRunning,
		MonitorRefreshInfoStateSuccess,
		MonitorRefreshInfoStateUnknown,
	}
}

// Type always returns MonitorRefreshInfoState to satisfy [pflag.Value] interface
func (f *MonitorRefreshInfoState) Type() string {
	return "MonitorRefreshInfoState"
}

type MonitorRefreshInfoTrigger string

const MonitorRefreshInfoTriggerManual MonitorRefreshInfoTrigger = `MANUAL`

const MonitorRefreshInfoTriggerSchedule MonitorRefreshInfoTrigger = `SCHEDULE`

const MonitorRefreshInfoTriggerUnknownTrigger MonitorRefreshInfoTrigger = `UNKNOWN_TRIGGER`

// String representation for [fmt.Print]
func (f *MonitorRefreshInfoTrigger) String() string {
	return string(*f)
}

// Set raw string value and validate it against allowed values
func (f *MonitorRefreshInfoTrigger) Set(v string) error {
	switch v {
	case `MANUAL`, `SCHEDULE`, `UNKNOWN_TRIGGER`:
		*f = MonitorRefreshInfoTrigger(v)
		return nil
	default:
		return fmt.Errorf(`value "%s" is not one of "MANUAL", "SCHEDULE", "UNKNOWN_TRIGGER"`, v)
	}
}

// Values returns all possible values for MonitorRefreshInfoTrigger.
//
// There is no guarantee on the order of the values in the slice.
func (f *MonitorRefreshInfoTrigger) Values() []MonitorRefreshInfoTrigger {
	return []MonitorRefreshInfoTrigger{
		MonitorRefreshInfoTriggerManual,
		MonitorRefreshInfoTriggerSchedule,
		MonitorRefreshInfoTriggerUnknownTrigger,
	}
}

// Type always returns MonitorRefreshInfoTrigger to satisfy [pflag.Value] interface
func (f *MonitorRefreshInfoTrigger) Type() string {
	return "MonitorRefreshInfoTrigger"
}

type MonitorRefreshListResponse struct {
	// List of refreshes.
	Refreshes []MonitorRefreshInfo `json:"refreshes,omitempty"`
}

// Snapshot analysis configuration
type MonitorSnapshot struct{}

// Time series analysis configuration.
type MonitorTimeSeries struct {
	// Granularities for aggregating data into time windows based on their
	// timestamp. Currently the following static granularities are supported:
	// {``\"5 minutes\"``, ``\"30 minutes\"``, ``\"1 hour\"``, ``\"1 day\"``,
	// ``\"\u003cn\u003e week(s)\"``, ``\"1 month\"``, ``\"1 year\"``}.
	Granularities []string `json:"granularities"`
	// Column for the timestamp.
	TimestampCol string `json:"timestamp_col"`
}

type RegenerateDashboardRequest struct {
	// UC table name in format `catalog.schema.table_name`. This field
	// corresponds to the {full_table_name_arg} arg in the endpoint path.
	TableName string `json:"-" url:"-"`
	// Optional argument to specify the warehouse for dashboard regeneration. If
	// not specified, the first running warehouse will be used.
	WarehouseId string `json:"warehouse_id,omitempty"`
}

type RegenerateDashboardResponse struct {
	DashboardId string `json:"dashboard_id,omitempty"`
	// Parent folder is equivalent to {assets_dir}/{tableName}
	ParentFolder string `json:"parent_folder,omitempty"`
}

type RunRefreshRequest struct {
	// UC table name in format `catalog.schema.table_name`. table_name is case
	// insensitive and spaces are disallowed.
	TableName string `json:"-" url:"-"`
}

type UpdateMonitor struct {
	// [Create:OPT Update:OPT] Baseline table name. Baseline data is used to
	// compute drift from the data in the monitored `table_name`. The baseline
	// table and the monitored table shall have the same schema.
	BaselineTableName string `json:"baseline_table_name,omitempty"`
	// [Create:OPT Update:OPT] Custom metrics.
	CustomMetrics []MonitorMetric `json:"custom_metrics,omitempty"`
	// [Create:ERR Update:OPT] Id of dashboard that visualizes the computed
	// metrics. This can be empty if the monitor is in PENDING state.
	DashboardId string `json:"dashboard_id,omitempty"`
	// [Create:OPT Update:OPT] Data classification related config.
	DataClassificationConfig *MonitorDataClassificationConfig `json:"data_classification_config,omitempty"`

	InferenceLog *MonitorInferenceLog `json:"inference_log,omitempty"`
	// [Create:ERR Update:IGN] The latest error message for a monitor failure.
	LatestMonitorFailureMsg string `json:"latest_monitor_failure_msg,omitempty"`
	// [Create:OPT Update:OPT] Field for specifying notification settings.
	Notifications *MonitorNotifications `json:"notifications,omitempty"`
	// [Create:REQ Update:REQ] Schema where output tables are created. Needs to
	// be in 2-level format {catalog}.{schema}
	OutputSchemaName string `json:"output_schema_name"`
	// [Create:OPT Update:OPT] The monitor schedule.
	Schedule *MonitorCronSchedule `json:"schedule,omitempty"`
	// [Create:OPT Update:OPT] List of column expressions to slice data with for
	// targeted analysis. The data is grouped by each expression independently,
	// resulting in a separate slice for each predicate and its complements. For
	// example `slicing_exprs=[“col_1”, “col_2 > 10”]` will generate the
	// following slices: two slices for `col_2 > 10` (True and False), and one
	// slice per unique value in `col1`. For high-cardinality columns, only the
	// top 100 unique values by frequency will generate slices.
	SlicingExprs []string `json:"slicing_exprs,omitempty"`
	// Configuration for monitoring snapshot tables.
	Snapshot *MonitorSnapshot `json:"snapshot,omitempty"`
	// UC table name in format `catalog.schema.table_name`. This field
	// corresponds to the {full_table_name_arg} arg in the endpoint path.
	TableName string `json:"-" url:"-"`
	// Configuration for monitoring time series tables.
	TimeSeries *MonitorTimeSeries `json:"time_series,omitempty"`
}
