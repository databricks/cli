/**
 * QualityMonitor resource types for Databricks Asset Bundles
 *
 * Auto-generated from JSON Schema. Do not edit manually.
 */

import { Resource } from "../../src/core/resource.js";
import type { VariableOr } from "../../src/core/variable.js";

export interface QualityMonitorParams {
  /**
   * [Create:REQ Update:IGN] Field for specifying the absolute path to a custom directory to store data-monitoring
   * assets. Normally prepopulated to a default user location via UI and Python APIs.
   */
  assets_dir: VariableOr<string>;
  /**
   * [Create:OPT Update:OPT] Baseline table name.
   * Baseline data is used to compute drift from the data in the monitored `table_name`.
   * The baseline table and the monitored table shall have the same schema.
   */
  baseline_table_name?: VariableOr<string>;
  /**
   * [Create:OPT Update:OPT] Custom metrics.
   */
  custom_metrics?: VariableOr<MonitorMetric[]>;
  /**
   * [Create:OPT Update:OPT] Data classification related config.
   */
  data_classification_config?: VariableOr<MonitorDataClassificationConfig>;
  inference_log?: VariableOr<MonitorInferenceLog>;
  /**
   * [Create:ERR Update:IGN] The latest error message for a monitor failure.
   */
  latest_monitor_failure_msg?: VariableOr<string>;
  /**
   * Lifecycle is a struct that contains the lifecycle settings for a resource. It controls the behavior of the resource when it is deployed or destroyed.
   */
  lifecycle?: VariableOr<Lifecycle>;
  /**
   * [Create:OPT Update:OPT] Field for specifying notification settings.
   */
  notifications?: VariableOr<MonitorNotifications>;
  /**
   * [Create:REQ Update:REQ] Schema where output tables are created. Needs to be in 2-level format {catalog}.{schema}
   */
  output_schema_name: VariableOr<string>;
  /**
   * [Create:OPT Update:OPT] The monitor schedule.
   */
  schedule?: VariableOr<MonitorCronSchedule>;
  /**
   * Whether to skip creating a default dashboard summarizing data quality metrics.
   */
  skip_builtin_dashboard?: VariableOr<boolean>;
  /**
   * [Create:OPT Update:OPT] List of column expressions to slice data with for targeted analysis. The data is grouped by
   * each expression independently, resulting in a separate slice for each predicate and its
   * complements. For example `slicing_exprs=[“col_1”, “col_2 > 10”]` will generate the following
   * slices: two slices for `col_2 > 10` (True and False), and one slice per unique value in
   * `col1`. For high-cardinality columns, only the top 100 unique values by frequency will
   * generate slices.
   */
  slicing_exprs?: VariableOr<string[]>;
  /**
   * Configuration for monitoring snapshot tables.
   */
  snapshot?: VariableOr<MonitorSnapshot>;
  table_name: VariableOr<string>;
  /**
   * Configuration for monitoring time series tables.
   */
  time_series?: VariableOr<MonitorTimeSeries>;
  /**
   * Optional argument to specify the warehouse for dashboard creation. If not specified, the first running
   * warehouse will be used.
   */
  warehouse_id?: VariableOr<string>;
}

export class QualityMonitor extends Resource<QualityMonitorParams> {
  constructor(name: string, params: QualityMonitorParams) {
    super(name, params, "quality_monitors");
  }
}

export interface MonitorCronSchedule {
  /**
   * Read only field that indicates whether a schedule is paused or not.
   */
  pause_status?: VariableOr<MonitorCronSchedulePauseStatus>;
  /**
   * The expression that determines when to run the monitor. See [examples](https://www.quartz-scheduler.org/documentation/quartz-2.3.0/tutorials/crontrigger.html).
   */
  quartz_cron_expression: VariableOr<string>;
  /**
   * The timezone id (e.g., ``PST``) in which to evaluate the quartz expression.
   */
  timezone_id: VariableOr<string>;
}

/**
 * Source link: https://src.dev.databricks.com/databricks/universe/-/blob/elastic-spark-common/api/messages/schedule.proto
 * Monitoring workflow schedule pause status.
 */
export type MonitorCronSchedulePauseStatus =
  | "UNSPECIFIED"
  | "UNPAUSED"
  | "PAUSED";

/**
 * Data classification related configuration.
 */
export interface MonitorDataClassificationConfig {
  /**
   * Whether to enable data classification.
   */
  enabled?: VariableOr<boolean>;
}

export interface MonitorDestination {
  /**
   * The list of email addresses to send the notification to. A maximum of 5 email addresses is supported.
   */
  email_addresses?: VariableOr<string[]>;
}

export interface MonitorInferenceLog {
  /**
   * Granularities for aggregating data into time windows based on their timestamp. Valid values are 5 minutes, 30 minutes, 1 hour, 1 day, n weeks, 1 month, or 1 year.
   */
  granularities: VariableOr<string[]>;
  /**
   * Column for the label.
   */
  label_col?: VariableOr<string>;
  /**
   * Column for the model identifier.
   */
  model_id_col: VariableOr<string>;
  /**
   * Column for the prediction.
   */
  prediction_col: VariableOr<string>;
  /**
   * Column for prediction probabilities
   */
  prediction_proba_col?: VariableOr<string>;
  /**
   * Problem type the model aims to solve.
   */
  problem_type: VariableOr<MonitorInferenceLogProblemType>;
  /**
   * Column for the timestamp.
   */
  timestamp_col: VariableOr<string>;
}

export type MonitorInferenceLogProblemType =
  | "PROBLEM_TYPE_CLASSIFICATION"
  | "PROBLEM_TYPE_REGRESSION";

/**
 * Custom metric definition.
 */
export interface MonitorMetric {
  /**
   * Jinja template for a SQL expression that specifies how to compute the metric. See [create metric definition](https://docs.databricks.com/en/lakehouse-monitoring/custom-metrics.html#create-definition).
   */
  definition: VariableOr<string>;
  /**
   * A list of column names in the input table the metric should be computed for.
   * Can use ``":table"`` to indicate that the metric needs information from multiple columns.
   */
  input_columns: VariableOr<string[]>;
  /**
   * Name of the metric in the output tables.
   */
  name: VariableOr<string>;
  /**
   * The output type of the custom metric.
   */
  output_data_type: VariableOr<string>;
  /**
   * Can only be one of ``"CUSTOM_METRIC_TYPE_AGGREGATE"``, ``"CUSTOM_METRIC_TYPE_DERIVED"``, or ``"CUSTOM_METRIC_TYPE_DRIFT"``.
   * The ``"CUSTOM_METRIC_TYPE_AGGREGATE"`` and ``"CUSTOM_METRIC_TYPE_DERIVED"`` metrics
   * are computed on a single table, whereas the ``"CUSTOM_METRIC_TYPE_DRIFT"`` compare metrics across
   * baseline and input table, or across the two consecutive time windows.
   * - CUSTOM_METRIC_TYPE_AGGREGATE: only depend on the existing columns in your table
   * - CUSTOM_METRIC_TYPE_DERIVED: depend on previously computed aggregate metrics
   * - CUSTOM_METRIC_TYPE_DRIFT:  depend on previously computed aggregate or derived metrics
   */
  type: VariableOr<MonitorMetricType>;
}

/**
 * Can only be one of ``\"CUSTOM_METRIC_TYPE_AGGREGATE\"``, ``\"CUSTOM_METRIC_TYPE_DERIVED\"``, or ``\"CUSTOM_METRIC_TYPE_DRIFT\"``.
 * The ``\"CUSTOM_METRIC_TYPE_AGGREGATE\"`` and ``\"CUSTOM_METRIC_TYPE_DERIVED\"`` metrics
 * are computed on a single table, whereas the ``\"CUSTOM_METRIC_TYPE_DRIFT\"`` compare metrics across
 * baseline and input table, or across the two consecutive time windows.
 * - CUSTOM_METRIC_TYPE_AGGREGATE: only depend on the existing columns in your table
 * - CUSTOM_METRIC_TYPE_DERIVED: depend on previously computed aggregate metrics
 * - CUSTOM_METRIC_TYPE_DRIFT:  depend on previously computed aggregate or derived metrics
 */
export type MonitorMetricType =
  | "CUSTOM_METRIC_TYPE_AGGREGATE"
  | "CUSTOM_METRIC_TYPE_DERIVED"
  | "CUSTOM_METRIC_TYPE_DRIFT";

export interface MonitorNotifications {
  /**
   * Destinations to send notifications on failure/timeout.
   */
  on_failure?: VariableOr<MonitorDestination>;
  /**
   * Destinations to send notifications on new classification tag detected.
   */
  on_new_classification_tag_detected?: VariableOr<MonitorDestination>;
}

/**
 * Snapshot analysis configuration
 */
export interface MonitorSnapshot {
}

/**
 * Time series analysis configuration.
 */
export interface MonitorTimeSeries {
  /**
   * Granularities for aggregating data into time windows based on their timestamp. Valid values are 5 minutes, 30 minutes, 1 hour, 1 day, n weeks, 1 month, or 1 year.
   */
  granularities: VariableOr<string[]>;
  /**
   * Column for the timestamp.
   */
  timestamp_col: VariableOr<string>;
}

export interface Lifecycle {
  /**
   * Lifecycle setting to prevent the resource from being destroyed.
   */
  prevent_destroy?: VariableOr<boolean>;
}
