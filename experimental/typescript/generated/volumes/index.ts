/**
 * Volume resource types for Databricks Asset Bundles
 *
 * Auto-generated from JSON Schema. Do not edit manually.
 */

import { Resource } from "../../src/core/resource.js";
import type { VariableOr } from "../../src/core/variable.js";

export interface VolumeParams {
  /**
   * The name of the catalog where the schema and the volume are
   */
  catalog_name: VariableOr<string>;
  /**
   * The comment attached to the volume
   */
  comment?: VariableOr<string>;
  grants?: VariableOr<VolumeGrant[]>;
  /**
   * Lifecycle is a struct that contains the lifecycle settings for a resource. It controls the behavior of the resource when it is deployed or destroyed.
   */
  lifecycle?: VariableOr<Lifecycle>;
  /**
   * The name of the volume
   */
  name: VariableOr<string>;
  /**
   * The name of the schema where the volume is
   */
  schema_name: VariableOr<string>;
  /**
   * The storage location on the cloud
   */
  storage_location?: VariableOr<string>;
  volume_type?: VariableOr<VolumeType>;
}

export class Volume extends Resource<VolumeParams> {
  constructor(params: VolumeParams) {
    super(params, "volumes");
  }
}

/**
 * The type of the volume. An external volume is located in the specified external location. A managed volume is located in the default location which is specified by the parent schema, or the parent catalog, or the Metastore. [Learn more](https://docs.databricks.com/aws/en/volumes/managed-vs-external)
 */
export type VolumeType =
  | "EXTERNAL"
  | "MANAGED";

export interface Lifecycle {
  /**
   * Lifecycle setting to prevent the resource from being destroyed.
   */
  prevent_destroy?: VariableOr<boolean>;
}

export interface VolumeGrant {
  principal: VariableOr<string>;
  privileges: VariableOr<VolumeGrantPrivilege[]>;
}

export type VolumeGrantPrivilege =
  | "ALL_PRIVILEGES"
  | "APPLY_TAG"
  | "MANAGE"
  | "READ_VOLUME"
  | "WRITE_VOLUME";
