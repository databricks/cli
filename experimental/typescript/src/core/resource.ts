/**
 * Base class for all Databricks resources.
 *
 * All resource types (Job, Pipeline, Schema, Volume, etc.) should extend this class.
 */
import { transformToJSON } from "./transform.js";
import { type ResourceType } from "./resources.js";

export class Resource<T> {  
  
  public readonly type: ResourceType;
  public readonly dabsName: string;

  constructor(dabsName: string, public readonly data: T, type: ResourceType) {
    this.type = type;
    this.dabsName = dabsName;
  }

  toJSON() {
    return transformToJSON(this.data);
  }
}