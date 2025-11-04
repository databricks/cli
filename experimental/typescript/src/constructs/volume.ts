/**
 * High-level Volume construct with enhanced functionality.
 *
 * Extends the generated Volume resource with:
 * - Development mode name prefixing
 * - Full name computation for Unity Catalog references
 */
import { Bundle } from "../core/bundle.js";
import type { VariableOr } from "../core/variable.js";
import { Volume as BaseVolume, type VolumeParams } from "../../generated/volumes/index.js";

/**
 * Enhanced Volume construct.
 *
 * Features:
 * - Adds dev- prefix to volume name in development mode
 * - Provides `fullName` property with catalog.schema.volume format
 *
 * @example
 * ```typescript
 * const volume = new Volume("landing-zone", bundle, {
 *   name: "landing-zone",
 *   catalog_name: "main",
 *   schema_name: "default",
 *   volume_type: "MANAGED",
 * });
 *
 * // Use the full name in other resources (e.g., Apps)
 * app.addResource(volume, "READ_VOLUME");
 * ```
 */
export class Volume extends BaseVolume {
  /**
   * The resource name (possibly a variable reference)
   */
  readonly resourceName: VariableOr<string>;

  /**
   * Creates a new Volume construct.
   *
   * @param name - The name of the volume in the bundle
   * @param bundle - The bundle context
   * @param params - Volume parameters
   */
  constructor(
    name: string,
    bundle: Bundle,
    params: Omit<VolumeParams, "name"> & { name?: VariableOr<string> }
  ) {
    const defaultParams: Partial<VolumeParams> = {
      volume_type: "MANAGED",
    };

    if (!params.name) {
      params.name = name;
    }

    if (bundle.isDevelopment) {
      params.name = `dev-${params.name.toString()}`;
    }
    super(name, { ...defaultParams, ...params, name: params.name });
    this.resourceName = params.name;
  }

  /**
   * Returns the full Unity Catalog name of the volume.
   *
   * Format: `catalog_name.schema_name.volume_name`
   *
   * This can be used to reference the volume in other resources that
   * require a fully qualified Unity Catalog name.
   */
  get fullName(): string {
    return `${this.data.catalog_name.toString()}.${this.data.schema_name.toString()}.${this.resourceName.toString()}`;
  }
}
