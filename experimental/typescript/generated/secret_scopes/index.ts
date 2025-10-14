/**
 * SecretScope resource types for Databricks Asset Bundles
 *
 * Auto-generated from JSON Schema. Do not edit manually.
 */

import { Resource } from "../../src/core/resource.js";
import type { VariableOr } from "../../src/core/variable.js";

export interface SecretScopeParams {
  /**
   * The backend type the scope will be created with. If not specified, will default to `DATABRICKS`
   */
  backend_type?: VariableOr<ScopeBackendType>;
  /**
   * The metadata for the secret scope if the `backend_type` is `AZURE_KEYVAULT`
   */
  keyvault_metadata?: VariableOr<AzureKeyVaultSecretScopeMetadata>;
  /**
   * Lifecycle is a struct that contains the lifecycle settings for a resource. It controls the behavior of the resource when it is deployed or destroyed.
   */
  lifecycle?: VariableOr<Lifecycle>;
  /**
   * Scope name requested by the user. Scope names are unique.
   */
  name: VariableOr<string>;
  /**
   * The permissions to apply to the secret scope. Permissions are managed via secret scope ACLs.
   */
  permissions?: VariableOr<SecretScopePermission[]>;
}

export class SecretScope extends Resource<SecretScopeParams> {
  constructor(name: string, params: SecretScopeParams) {
    super(name, params, "secret_scopes");
  }
}

export interface Lifecycle {
  /**
   * Lifecycle setting to prevent the resource from being destroyed.
   */
  prevent_destroy?: VariableOr<boolean>;
}

export interface SecretScopePermission {
  /**
   * The name of the group that has the permission set in level. This field translates to a `principal` field in secret scope ACL.
   */
  group_name?: VariableOr<string>;
  /**
   * The allowed permission for user, group, service principal defined for this permission.
   */
  level: VariableOr<SecretScopePermissionLevel>;
  /**
   * The application ID of an active service principal. This field translates to a `principal` field in secret scope ACL.
   */
  service_principal_name?: VariableOr<string>;
  /**
   * The name of the user that has the permission set in level. This field translates to a `principal` field in secret scope ACL.
   */
  user_name?: VariableOr<string>;
}

export type SecretScopePermissionLevel =
  | "READ"
  | "WRITE"
  | "MANAGE";

/**
 * The metadata of the Azure KeyVault for a secret scope of type `AZURE_KEYVAULT`
 */
export interface AzureKeyVaultSecretScopeMetadata {
  /**
   * The DNS of the KeyVault
   */
  dns_name: VariableOr<string>;
  /**
   * The resource id of the azure KeyVault that user wants to associate the scope with.
   */
  resource_id: VariableOr<string>;
}

/**
 * The types of secret scope backends in the Secret Manager. Azure KeyVault backed secret scopes
 * will be supported in a later release.
 */
export type ScopeBackendType =
  | "DATABRICKS"
  | "AZURE_KEYVAULT";
