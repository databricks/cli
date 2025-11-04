import {
  SecretScope as BaseSecretScope,
  type SecretScopeParams,
} from "../../generated/secret_scopes/index.js";
import type { Bundle, VariableOr } from "../core/index.js";
import { Workspace } from "../core/workspace.js";

export class SecretScope extends BaseSecretScope {
  constructor(name: string, bundle: Bundle, params: Partial<SecretScopeParams> = {}) {
    const defaultParams: Partial<SecretScopeParams> = {
      backend_type: "DATABRICKS",
    };

    if (!params.name) {
      params.name = name;
    }

    if (bundle.isDevelopment) {
      params.name = `dev-${Workspace.currentUser.domainFriendlyName.toString()}-${params.name.toString()}`;
    }
    super(name, { ...defaultParams, ...params, name: params.name });
  }

  get name(): VariableOr<string> {
    return this.data.name;
  }
}
