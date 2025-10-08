/**
 * Pipeline resource types for Databricks Asset Bundles
 *
 * These are placeholder types that will be replaced by auto-generated types
 * from OpenAPI specifications in the future.
 */

import type { Resource } from "../../src/core/resource.js";
import type { VariableOr } from "../../src/core/variable.js";

/**
 * Represents a Databricks Delta Live Tables Pipeline resource
 */
export interface Pipeline extends Resource {
  /**
   * Optional user-defined name for the pipeline
   */
  name?: VariableOr<string>;

  /**
   * Storage location for tables
   */
  storage?: VariableOr<string>;

  /**
   * Configuration for the pipeline
   */
  configuration?: VariableOr<Record<string, string>>;

  /**
   * Compute specifications
   */
  clusters?: VariableOr<PipelineCluster[]>;

  /**
   * Libraries for the pipeline
   */
  libraries?: VariableOr<PipelineLibrary[]>;

  /**
   * Target schema
   */
  target?: VariableOr<string>;

  /**
   * Catalog for Unity Catalog
   */
  catalog?: VariableOr<string>;

  /**
   * Continuous or triggered
   */
  continuous?: VariableOr<boolean>;

  /**
   * Development mode
   */
  development?: VariableOr<boolean>;

  /**
   * Photon acceleration
   */
  photon?: VariableOr<boolean>;

  /**
   * Pipeline edition
   */
  edition?: VariableOr<string>;

  /**
   * Channel for preview features
   */
  channel?: VariableOr<string>;

  /**
   * Notifications
   */
  notifications?: VariableOr<PipelineNotification[]>;
}

/**
 * Pipeline cluster configuration
 */
export interface PipelineCluster {
  /**
   * Cluster label
   */
  label?: VariableOr<string>;

  /**
   * Number of workers
   */
  num_workers?: VariableOr<number>;

  /**
   * Autoscale configuration
   */
  autoscale?: VariableOr<PipelineClusterAutoscale>;

  /**
   * Node type ID
   */
  node_type_id?: VariableOr<string>;

  /**
   * Driver node type ID
   */
  driver_node_type_id?: VariableOr<string>;

  /**
   * Instance pool ID
   */
  instance_pool_id?: VariableOr<string>;

  /**
   * Custom tags
   */
  custom_tags?: VariableOr<Record<string, string>>;

  /**
   * Spark configuration
   */
  spark_conf?: VariableOr<Record<string, string>>;

  /**
   * Spark environment variables
   */
  spark_env_vars?: VariableOr<Record<string, string>>;

  /**
   * Init scripts
   */
  init_scripts?: VariableOr<PipelineInitScriptInfo[]>;

  /**
   * Policy ID
   */
  policy_id?: VariableOr<string>;
}

/**
 * Pipeline autoscale configuration
 */
export interface PipelineClusterAutoscale {
  min_workers: VariableOr<number>;
  max_workers: VariableOr<number>;
  mode?: VariableOr<string>;
}

/**
 * Pipeline init script
 */
export interface PipelineInitScriptInfo {
  dbfs?: VariableOr<{
    destination: VariableOr<string>;
  }>;
  workspace?: VariableOr<{
    destination: VariableOr<string>;
  }>;
  volumes?: VariableOr<{
    destination: VariableOr<string>;
  }>;
}

/**
 * Pipeline library
 */
export interface PipelineLibrary {
  /**
   * Notebook path
   */
  notebook?: VariableOr<{
    path: VariableOr<string>;
  }>;

  /**
   * File path
   */
  file?: VariableOr<{
    path: VariableOr<string>;
  }>;

  /**
   * JAR file
   */
  jar?: VariableOr<string>;

  /**
   * Maven coordinates
   */
  maven?: VariableOr<{
    coordinates: VariableOr<string>;
    repo?: VariableOr<string>;
    exclusions?: VariableOr<string[]>;
  }>;
}

/**
 * Pipeline notification
 */
export interface PipelineNotification {
  /**
   * Email addresses
   */
  email_recipients?: VariableOr<string[]>;

  /**
   * Alerts
   */
  alerts?: VariableOr<string[]>;
}

/**
 * Helper function to create a Pipeline with type safety
 */
export function createPipeline(config: Pipeline): Pipeline {
  return config;
}
