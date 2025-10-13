import { Bundle } from "../core/bundle.js";
import type { VariableOr } from "../core/variable.js";
import { Volume as BaseVolume, type VolumeParams } from "../../generated/volumes/index.js";

export class Volume extends BaseVolume {
  readonly resourceName: VariableOr<string>;

  constructor(name: string, bundle: Bundle, params: VolumeParams) {
    if (bundle.mode === "development") {
      params.name = `dev-${params.name}`;
    }
    super(name, params);
    this.resourceName = params.name;
  }

  get fullName() {
    return `${this.data.catalog_name}.${this.data.schema_name}.${this.resourceName}`;
  }
}
