import { Bundle, VariableOr } from "@databricks/bundles";
import { Volume as BaseVolume, type VolumeParams } from "@databricks/bundles/volumes";

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
