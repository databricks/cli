import { Bundle, VariableOr } from "@databricks/bundles";
import { Volume as BaseVolume, type VolumeParams } from "@databricks/bundles/volumes";
import { Workspace } from "@databricks/bundles";

export class Volume extends BaseVolume {
  readonly name: VariableOr<string>;

  constructor(bundle: Bundle, params: VolumeParams) {
    if (bundle.mode === "development") {
      params.name = `dev-${params.name}`;
    }
    super(params);
    this.name = params.name;
  }

  get fullName() {
    return `${this.data.catalog_name}.${this.data.schema_name}.${this.name}`;
  }
}
