/**
 * ModelServingEndpoint resource types for Databricks Asset Bundles
 *
 * Auto-generated from JSON Schema. Do not edit manually.
 */

import type { Resource } from "../../src/core/resource.js";
import type { VariableOr } from "../../src/core/variable.js";

export interface ModelServingEndpoint extends Resource {
  /**
   * The AI Gateway configuration for the serving endpoint. NOTE: External model, provisioned throughput, and pay-per-token endpoints are fully supported; agent endpoints currently only support inference tables.
   */
  ai_gateway?: VariableOr<AiGatewayConfig>;
  /**
   * The budget policy to be applied to the serving endpoint.
   */
  budget_policy_id?: VariableOr<string>;
  /**
   * The core config of the serving endpoint.
   */
  config?: VariableOr<EndpointCoreConfigInput>;
  description?: VariableOr<string>;
  /**
   * Email notification settings.
   */
  email_notifications?: VariableOr<EmailNotifications>;
  /**
   * Lifecycle is a struct that contains the lifecycle settings for a resource. It controls the behavior of the resource when it is deployed or destroyed.
   */
  lifecycle?: VariableOr<Lifecycle>;
  /**
   * The name of the serving endpoint. This field is required and must be unique across a Databricks workspace.
   * An endpoint name can consist of alphanumeric characters, dashes, and underscores.
   */
  name: VariableOr<string>;
  permissions?: VariableOr<ModelServingEndpointPermission[]>;
  /**
   * Rate limits to be applied to the serving endpoint. NOTE: this field is deprecated, please use AI Gateway to manage rate limits.
   * @deprecated
   */
  rate_limits?: VariableOr<RateLimit[]>;
  /**
   * Enable route optimization for the serving endpoint.
   */
  route_optimized?: VariableOr<boolean>;
  /**
   * Tags to be attached to the serving endpoint and automatically propagated to billing logs.
   */
  tags?: VariableOr<EndpointTag[]>;
}

export interface Lifecycle extends Resource {
  /**
   * Lifecycle setting to prevent the resource from being destroyed.
   */
  prevent_destroy?: VariableOr<boolean>;
}

export interface ModelServingEndpointPermission extends Resource {
  group_name?: VariableOr<string>;
  level: VariableOr<ModelServingEndpointPermissionLevel>;
  service_principal_name?: VariableOr<string>;
  user_name?: VariableOr<string>;
}

export type ModelServingEndpointPermissionLevel =
  | "CAN_MANAGE"
  | "CAN_QUERY"
  | "CAN_VIEW";

/**
 * Helper function to create a ModelServingEndpoint with type safety
 */
export function createModelServingEndpoint(config: ModelServingEndpoint): ModelServingEndpoint {
  return config;
}
